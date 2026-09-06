package eventhub

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/rs/zerolog"
)

var _ outbound.EventStreamHub = (*Hub)(nil)

var (
	// ErrHubClosed is returned when operations are attempted on a closed Hub.
	ErrHubClosed = errors.New("event hub is closed")
)

type clientSubscriber struct {
	id     uint64
	ch     chan model.ServerSentEvent
	stopCh chan struct{}
	once   sync.Once
}

// Hub manages active SSE client subscribers and thread-safe event fan-out.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[uint64]*clientSubscriber
	nextID      atomic.Uint64
	bufferSize  int
	closed      bool
}

// NewHub constructs an initialized in-memory Hub.
func NewHub(bufferSize int) *Hub {
	if bufferSize <= 0 {
		bufferSize = 32
	}
	return &Hub{
		subscribers: make(map[uint64]*clientSubscriber),
		bufferSize:  bufferSize,
	}
}

// Subscribe registers a new subscriber channel.
// It returns the receive-only channel, an idempotent unsubscribe function, or an error if the hub is closed.
func (h *Hub) Subscribe(ctx context.Context) (<-chan model.ServerSentEvent, func(), error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "Hub.Subscribe").Logger()
	log.Debug().Msg("registering new SSE subscriber")

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("cannot subscribe to closed hub")
		return nil, nil, ErrHubClosed
	}

	clientID := h.nextID.Add(1)
	sub := &clientSubscriber{
		id:     clientID,
		ch:     make(chan model.ServerSentEvent, h.bufferSize),
		stopCh: make(chan struct{}),
	}
	h.subscribers[clientID] = sub
	currentCount := len(h.subscribers)
	h.mu.Unlock()

	unsubscribe := func() {
		sub.once.Do(func() {
			close(sub.stopCh)
			h.mu.Lock()
			delete(h.subscribers, sub.id)
			h.mu.Unlock()
			close(sub.ch)
		})
	}

	// Actively monitor context cancellation to unregister and prevent goroutine leaks
	if ctx != nil && ctx.Done() != nil {
		go func() {
			select {
			case <-ctx.Done():
				unsubscribe()
			case <-sub.stopCh:
				// Normal unsubscribe already occurred
			}
		}()
	}

	log.Info().
		Uint64("client_id", clientID).
		Int("active_clients", currentCount).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully subscribed client to SSE stream")

	return sub.ch, unsubscribe, nil
}

// Broadcast distributes an event to all registered clients without blocking on slow consumers.
func (h *Hub) Broadcast(ctx context.Context, event model.ServerSentEvent) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "Hub.Broadcast").
		Str("event_id", event.ID).
		Str("event_type", event.Event).
		Logger()
	log.Debug().Msg("broadcasting event to SSE subscribers")

	h.mu.RLock()
	if h.closed {
		h.mu.RUnlock()
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("cannot broadcast on closed hub")
		return ErrHubClosed
	}

	// Snapshot active subscribers
	subs := make([]*clientSubscriber, 0, len(h.subscribers))
	for _, s := range h.subscribers {
		subs = append(subs, s)
	}
	h.mu.RUnlock()

	var dropped int
	for _, sub := range subs {
		select {
		case sub.ch <- event:
		default:
			dropped++
		}
	}

	if dropped > 0 {
		log.Warn().
			Int("dropped_count", dropped).
			Int("total_subscribers", len(subs)).
			Msg("slow clients detected, dropped buffered events")
	}

	log.Info().
		Int("subscribers", len(subs)).
		Int("dropped", dropped).
		Dur("duration_ms", time.Since(start)).
		Msg("completed event broadcast")

	return nil
}

// ClientCount returns the current number of active subscribers.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}

// Close closes the hub and terminates all active subscriptions cleanly.
func (h *Hub) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true

	subs := make([]*clientSubscriber, 0, len(h.subscribers))
	for _, s := range h.subscribers {
		subs = append(subs, s)
	}
	h.subscribers = make(map[uint64]*clientSubscriber)
	h.mu.Unlock()

	for _, sub := range subs {
		sub.once.Do(func() {
			close(sub.stopCh)
			close(sub.ch)
		})
	}

	return nil
}

// String provides a human-readable representation of Hub state.
func (h *Hub) String() string {
	return fmt.Sprintf("SSEHub(active_clients=%d)", h.ClientCount())
}
