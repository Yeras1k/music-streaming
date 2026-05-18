package events

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/music-streaming/user-service/internal/domain"
)

type EventPublisher struct {
	nc *nats.Conn
}

func NewEventPublisher(nc *nats.Conn) *EventPublisher {
	return &EventPublisher{nc: nc}
}

func (p *EventPublisher) PublishUserRegistered(ctx context.Context, event *domain.UserRegisteredEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal user_registered event: %v", err)
		return
	}
	if err := p.nc.Publish("user.events", data); err != nil {
		log.Printf("Failed to publish user_registered event: %v", err)
	}
}

func (p *EventPublisher) PublishUserDeleted(ctx context.Context, event *domain.UserDeletedEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal user_deleted event: %v", err)
		return
	}
	if err := p.nc.Publish("user.events", data); err != nil {
		log.Printf("Failed to publish user_deleted event: %v", err)
	}
}