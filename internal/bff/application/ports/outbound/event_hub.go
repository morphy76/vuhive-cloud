package outbound

import (
	"context"
)

// EventStreamHub defines the driven outbound port for real-time telemetry / SSE broadcasting.
type EventStreamHub interface {
	Broadcast(ctx context.Context, topic string, payload []byte) error
	Subscribe(ctx context.Context, topic string) (<-chan []byte, func(), error)
}
