package outbound

import (
	"context"

	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
)

// EventStreamHub defines the driven outbound port for real-time telemetry / SSE broadcasting.
type EventStreamHub interface {
	Broadcast(ctx context.Context, event model.ServerSentEvent) error
	Subscribe(ctx context.Context) (<-chan model.ServerSentEvent, func(), error)
	ClientCount() int
	Close() error
}
