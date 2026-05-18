package events

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/music-streaming/music-service/internal/domain"
)

type EventPublisher struct {
	nc *nats.Conn
}

func NewEventPublisher(nc *nats.Conn) *EventPublisher {
	return &EventPublisher{nc: nc}
}

func (p *EventPublisher) PublishTrackUploaded(ctx context.Context, event *domain.TrackUploadedEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal track_uploaded event: %v", err)
		return
	}
	if err := p.nc.Publish("music.events", data); err != nil {
		log.Printf("Failed to publish track_uploaded event: %v", err)
	}
}