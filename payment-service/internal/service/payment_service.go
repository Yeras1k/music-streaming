package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"

	"github.com/music-streaming/payment-service/internal/domain"
	"github.com/music-streaming/payment-service/pkg/events"
)

type PaymentService struct {
	subscriptionRepo domain.SubscriptionRepository
	paymentRepo      domain.PaymentRepository
	planRepo         domain.PricingPlanRepository
	couponRepo       domain.CouponRepository
	events           *events.EventPublisher
	redis            *redis.Client
}

func NewPaymentService(
	subscriptionRepo domain.SubscriptionRepository,
	paymentRepo domain.PaymentRepository,
	planRepo domain.PricingPlanRepository,
	couponRepo domain.CouponRepository,
	events *events.EventPublisher,
	redis *redis.Client,
) *PaymentService {
	return &PaymentService{
		subscriptionRepo: subscriptionRepo,
		paymentRepo:      paymentRepo,
		planRepo:         planRepo,
		couponRepo:       couponRepo,
		events:           events,
		redis:            redis,
	}
}

func (s *PaymentService) CreateSubscription(ctx context.Context, userID, planID, paymentMethodID string) (*domain.Subscription, error) {
	// Check rate limit
	if err := s.checkRateLimit(ctx, userID); err != nil {
		return nil, err
	}

	// Get plan details
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}

	// Check for existing subscription
	existing, _ := s.subscriptionRepo.GetByUserID(ctx, userID)
	if existing != nil && existing.Status == "active" {
		return nil, fmt.Errorf("user already has an active subscription")
	}

	// Create subscription
	subscription := &domain.Subscription{
		ID:        uuid.New().String(),
		UserID:    userID,
		PlanID:    plan.ID,
		PlanName:  plan.Name,
		Status:    "active",
		Price:     plan.Price,
		Currency:  plan.Currency,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(0, 1, 0),
	}

	if err := s.subscriptionRepo.Create(ctx, subscription); err != nil {
		return nil, err
	}

	// Process initial payment
	payment := &domain.PaymentTransaction{
		ID:            uuid.New().String(),
		UserID:        userID,
		SubscriptionID: &subscription.ID,
		Amount:        plan.Price,
		Currency:      plan.Currency,
		Status:        "completed",
		PaymentMethod: paymentMethodID,
		Description:   fmt.Sprintf("%s subscription", plan.Name),
		ReceiptURL:    fmt.Sprintf("https://payments.musicstreaming.com/receipts/%s.pdf", uuid.New().String()),
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	// Cache subscription
	s.cacheSubscription(ctx, subscription)

	// Publish events
	s.events.PublishSubscriptionCreated(ctx, domain.NewSubscriptionCreatedEvent(
		subscription.ID, userID, plan.Name, plan.Price,
	))
	s.events.PublishPaymentCompleted(ctx, domain.NewPaymentCompletedEvent(
		payment.ID, userID, payment.Amount, payment.Currency,
	))

	return subscription, nil
}

func (s *PaymentService) CancelSubscription(ctx context.Context, subscriptionID, userID string) error {
	if err := s.subscriptionRepo.Cancel(ctx, subscriptionID, userID); err != nil {
		return err
	}

	// Invalidate cache
	s.redis.Del(ctx, fmt.Sprintf("sub:%s", userID))

	return nil
}

func (s *PaymentService) GetSubscription(ctx context.Context, userID string) (*domain.Subscription, error) {
	// Try cache first
	if cached, err := s.getCachedSubscription(ctx, userID); err == nil {
		return cached, nil
	}

	sub, err := s.subscriptionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache for future
	s.cacheSubscription(ctx, sub)

	return sub, nil
}

func (s *PaymentService) ProcessPayment(ctx context.Context, userID string, amount float64, currency, paymentMethodID, description string) (*domain.PaymentTransaction, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}

	// Check rate limit
	if err := s.checkRateLimit(ctx, userID); err != nil {
		return nil, err
	}

	// Create transaction record
	transaction := &domain.PaymentTransaction{
		ID:            uuid.New().String(),
		UserID:        userID,
		Amount:        amount,
		Currency:      currency,
		Status:        "pending",
		PaymentMethod: paymentMethodID,
		Description:   description,
	}

	if err := s.paymentRepo.Create(ctx, transaction); err != nil {
		return nil, err
	}

	// Simulate payment processing with retries
	if err := s.processWithRetry(ctx, transaction); err != nil {
		transaction.Status = "failed"
		s.paymentRepo.Update(ctx, transaction)
		return nil, err
	}

	transaction.Status = "completed"
	transaction.ReceiptURL = fmt.Sprintf("https://payments.musicstreaming.com/receipts/%s.pdf", transaction.ID)

	if err := s.paymentRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}

	// Publish event
	s.events.PublishPaymentCompleted(ctx, domain.NewPaymentCompletedEvent(
		transaction.ID, userID, amount, currency,
	))

	return transaction, nil
}

func (s *PaymentService) GetPaymentHistory(ctx context.Context, userID string, page, pageSize int32) ([]domain.PaymentTransaction, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.paymentRepo.GetByUserID(ctx, userID, page, pageSize)
}

func (s *PaymentService) GetPricingPlans(ctx context.Context) ([]domain.PricingPlan, error) {
	return s.planRepo.GetAll(ctx)
}

func (s *PaymentService) ApplyCoupon(ctx context.Context, userID, couponCode string) (float64, error) {
	coupon, err := s.couponRepo.GetByCode(ctx, couponCode)
	if err != nil {
		return 0, domain.ErrCouponExpired
	}

	if coupon.ExpiresAt.Before(time.Now()) {
		return 0, domain.ErrCouponExpired
	}

	if coupon.UsedBy != nil {
		return 0, domain.ErrCouponAlreadyUsed
	}

	if err := s.couponRepo.Apply(ctx, couponCode, userID); err != nil {
		return 0, err
	}

	return coupon.Discount, nil
}

func (s *PaymentService) checkRateLimit(ctx context.Context, userID string) error {
	key := fmt.Sprintf("rate:payment:%s", userID)
	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return nil
	}
	if count > 10 {
		return fmt.Errorf("rate limit exceeded")
	}
	if count == 1 {
		s.redis.Expire(ctx, key, time.Hour)
	}
	return nil
}

func (s *PaymentService) cacheSubscription(ctx context.Context, sub *domain.Subscription) {
	data := fmt.Sprintf(`{"id":"%s","status":"%s","end_date":%d}`,
		sub.ID, sub.Status, sub.EndDate.Unix())
	s.redis.Set(ctx, fmt.Sprintf("sub:%s", sub.UserID), data, 5*time.Minute)
}

func (s *PaymentService) getCachedSubscription(ctx context.Context, userID string) (*domain.Subscription, error) {
	data, err := s.redis.Get(ctx, fmt.Sprintf("sub:%s", userID)).Result()
	if err != nil {
		return nil, err
	}
	// Parse cached data (simplified)
	return &domain.Subscription{UserID: userID, Status: "active"}, nil
}

func (s *PaymentService) processWithRetry(ctx context.Context, transaction *domain.PaymentTransaction) error {
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := s.callPaymentGateway(transaction); err == nil {
			return nil
		}
		time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
	}
	return domain.ErrPaymentFailed
}

func (s *PaymentService) callPaymentGateway(transaction *domain.PaymentTransaction) error {
	time.Sleep(500 * time.Millisecond)

	// Simulate 95% success rate
	if time.Now().UnixNano()%100 < 95 {
		return nil
	}
	return fmt.Errorf("payment gateway declined")
}