package outbound

import (
	"context"

	"github.com/morphy76/vuhive-cloud/internal/domain/event"
)

// EventPublisher defines the driven port for publishing domain events.
type EventPublisher interface {
	Publish(ctx context.Context, evt event.DomainEvent) error
}
