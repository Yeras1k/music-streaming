package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrPaymentFailed         = errors.New("payment failed")
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrCouponExpired         = errors.New("coupon expired")
	ErrCouponAlreadyUsed     = errors.New("coupon already used")
)

type Subscription struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	PlanID    string    `json:"plan_id"`
	PlanName  string    `json:"plan_name"`
	Status    string    `json:"status"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PaymentTransaction struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	SubscriptionID *string   `json:"subscription_id,omitempty"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	PaymentMethod  string    `json:"payment_method"`
	Description    string    `json:"description"`
	ReceiptURL     string    `json:"receipt_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PricingPlan struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	Interval    string  `json:"interval"`
	Quality     int32   `json:"quality"`
	OfflineMode bool    `json:"offline_mode"`
}

type Coupon struct {
	ID        string     `json:"id"`
	Code      string     `json:"code"`
	Discount  float64    `json:"discount"`
	UsedBy    *string    `json:"used_by,omitempty"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type SubscriptionRepository interface {
	Create(ctx context.Context, sub *Subscription) error
	GetByID(ctx context.Context, id string) (*Subscription, error)
	GetByUserID(ctx context.Context, userID string) (*Subscription, error)
	Update(ctx context.Context, sub *Subscription) error
	Cancel(ctx context.Context, id, userID string) error
}

type PaymentRepository interface {
	Create(ctx context.Context, tx *PaymentTransaction) error
	GetByID(ctx context.Context, id string) (*PaymentTransaction, error)
	GetByUserID(ctx context.Context, userID string, page, pageSize int32) ([]PaymentTransaction, int64, error)
	Update(ctx context.Context, tx *PaymentTransaction) error
}

type PricingPlanRepository interface {
	GetAll(ctx context.Context) ([]PricingPlan, error)
	GetByID(ctx context.Context, id string) (*PricingPlan, error)
}

type CouponRepository interface {
	GetByCode(ctx context.Context, code string) (*Coupon, error)
	Apply(ctx context.Context, code, userID string) error
}