package domain

import "time"

type SubscriptionCreatedEvent struct {
	Event          string `json:"event"`
	SubscriptionID string `json:"subscription_id"`
	UserID         string `json:"user_id"`
	PlanName       string `json:"plan_name"`
	Price          float64 `json:"price"`
	Timestamp      int64  `json:"timestamp"`
}

func NewSubscriptionCreatedEvent(subscriptionID, userID, planName string, price float64) *SubscriptionCreatedEvent {
	return &SubscriptionCreatedEvent{
		Event:          "subscription_created",
		SubscriptionID: subscriptionID,
		UserID:         userID,
		PlanName:       planName,
		Price:          price,
		Timestamp:      time.Now().Unix(),
	}
}

type PaymentCompletedEvent struct {
	Event         string  `json:"event"`
	TransactionID string  `json:"transaction_id"`
	UserID        string  `json:"user_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Timestamp     int64   `json:"timestamp"`
}

func NewPaymentCompletedEvent(transactionID, userID string, amount float64, currency string) *PaymentCompletedEvent {
	return &PaymentCompletedEvent{
		Event:         "payment_completed",
		TransactionID: transactionID,
		UserID:        userID,
		Amount:        amount,
		Currency:      currency,
		Timestamp:     time.Now().Unix(),
	}
}