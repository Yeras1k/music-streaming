package events

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/music-streaming/payment-service/internal/domain"
)

type EventPublisher struct {
	nc *nats.Conn
}

func NewEventPublisher(nc *nats.Conn) *EventPublisher {
	return &EventPublisher{nc: nc}
}

func (p *EventPublisher) PublishSubscriptionCreated(ctx context.Context, event *domain.SubscriptionCreatedEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal subscription_created event: %v", err)
		return
	}
	if err := p.nc.Publish("payment.events", data); err != nil {
		log.Printf("Failed to publish subscription_created event: %v", err)
	}
}

func (p *EventPublisher) PublishPaymentCompleted(ctx context.Context, event *domain.PaymentCompletedEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal payment_completed event: %v", err)
		return
	}
	if err := p.nc.Publish("payment.events", data); err != nil {
		log.Printf("Failed to publish payment_completed event: %v", err)
	}
}