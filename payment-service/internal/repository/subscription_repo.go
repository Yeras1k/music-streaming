package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/music-streaming/payment-service/internal/domain"
)

type SubscriptionModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string    `gorm:"type:uuid;not null;index"`
	PlanID    string    `gorm:"not null"`
	PlanName  string    `gorm:"not null"`
	Status    string    `gorm:"default:active"`
	Price     float64   `gorm:"not null"`
	Currency  string    `gorm:"default:USD"`
	StartDate time.Time `gorm:"not null"`
	EndDate   time.Time `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (m *SubscriptionModel) ToDomain() *domain.Subscription {
	return &domain.Subscription{
		ID:        m.ID,
		UserID:    m.UserID,
		PlanID:    m.PlanID,
		PlanName:  m.PlanName,
		Status:    m.Status,
		Price:     m.Price,
		Currency:  m.Currency,
		StartDate: m.StartDate,
		EndDate:   m.EndDate,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type subscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) domain.SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) Create(ctx context.Context, sub *domain.Subscription) error {
	model := &SubscriptionModel{
		ID:        sub.ID,
		UserID:    sub.UserID,
		PlanID:    sub.PlanID,
		PlanName:  sub.PlanName,
		Status:    sub.Status,
		Price:     sub.Price,
		Currency:  sub.Currency,
		StartDate: sub.StartDate,
		EndDate:   sub.EndDate,
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *subscriptionRepository) GetByID(ctx context.Context, id string) (*domain.Subscription, error) {
	var model SubscriptionModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *subscriptionRepository) GetByUserID(ctx context.Context, userID string) (*domain.Subscription, error) {
	var model SubscriptionModel
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, "active").
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *subscriptionRepository) Update(ctx context.Context, sub *domain.Subscription) error {
	return r.db.WithContext(ctx).Model(&SubscriptionModel{}).
		Where("id = ?", sub.ID).
		Updates(map[string]interface{}{
			"status":     sub.Status,
			"end_date":   sub.EndDate,
			"updated_at": time.Now(),
		}).Error
}

func (r *subscriptionRepository) Cancel(ctx context.Context, id, userID string) error {
	return r.db.WithContext(ctx).Model(&SubscriptionModel{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("status", "cancelled").Error
}