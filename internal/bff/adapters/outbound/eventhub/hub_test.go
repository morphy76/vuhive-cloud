package eventhub_test

import (
	"context"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/eventhub"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface assertion
var _ outbound.EventStreamHub = (*eventhub.Hub)(nil)

func TestHub_SubscribeAndBroadcast(t *testing.T) {
	hub := eventhub.NewHub(16)
	defer func() { _ = hub.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch1, unsub1, err := hub.Subscribe(ctx)
	require.NoError(t, err)
	defer unsub1()

	ch2, unsub2, err := hub.Subscribe(ctx)
	require.NoError(t, err)
	defer unsub2()

	assert.Equal(t, 2, hub.ClientCount())

	now := time.Now().UTC()
	ev, err := model.NewSystemHeartbeatEvent("hb-1", model.SystemHeartbeatPayload{
		Status:        "UP",
		ActiveClients: 2,
		ActiveRuns:    0,
		Timestamp:     now,
	})
	require.NoError(t, err)

	err = hub.Broadcast(ctx, ev)
	require.NoError(t, err)

	select {
	case received1 := <-ch1:
		assert.Equal(t, "hb-1", received1.ID)
		assert.Equal(t, model.EventSystemHeartbeat, received1.Event)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for client 1 to receive event")
	}

	select {
	case received2 := <-ch2:
		assert.Equal(t, "hb-1", received2.ID)
		assert.Equal(t, model.EventSystemHeartbeat, received2.Event)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for client 2 to receive event")
	}
}

func TestHub_UnsubscribeCallback(t *testing.T) {
	hub := eventhub.NewHub(16)
	defer func() { _ = hub.Close() }()

	ctx := context.Background()
	ch, unsub, err := hub.Subscribe(ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, hub.ClientCount())

	// Unsubscribe explicitly
	unsub()
	assert.Equal(t, 0, hub.ClientCount())

	// Verify channel is closed
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after unsubscribe")

	// Idempotent call to unsub should not panic
	assert.NotPanics(t, func() {
		unsub()
	})
}

func TestHub_ContextCancellationCleanup(t *testing.T) {
	hub := eventhub.NewHub(16)
	defer func() { _ = hub.Close() }()

	clientCtx, clientCancel := context.WithCancel(context.Background())
	ch, _, err := hub.Subscribe(clientCtx)
	require.NoError(t, err)
	assert.Equal(t, 1, hub.ClientCount())

	// Cancel client context
	clientCancel()

	// Wait briefly for monitoring goroutine to clean up
	require.Eventually(t, func() bool {
		return hub.ClientCount() == 0
	}, 1*time.Second, 20*time.Millisecond)

	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after context cancellation")
}

func TestHub_SlowClientNonBlockingDrop(t *testing.T) {
	// Hub with buffer size 1
	hub := eventhub.NewHub(1)
	defer func() { _ = hub.Close() }()

	ctx := context.Background()
	ch, unsub, err := hub.Subscribe(ctx)
	require.NoError(t, err)
	defer unsub()

	// Send 3 events without reading
	for i := 1; i <= 3; i++ {
		ev, _ := model.NewServerSentEvent("ev-test", model.EventSystemHeartbeat, []byte(`{"status":"UP"}`), time.Now().UTC())
		err := hub.Broadcast(ctx, ev)
		assert.NoError(t, err)
	}

	// Should not block or hang; first event is in buffer
	select {
	case <-ch:
		// read one
	default:
		t.Fatal("expected at least one event in channel")
	}
}

func TestHub_Close(t *testing.T) {
	hub := eventhub.NewHub(16)
	ctx := context.Background()

	ch1, _, err := hub.Subscribe(ctx)
	require.NoError(t, err)

	err = hub.Close()
	require.NoError(t, err)

	// Subscribers should have their channels closed
	_, ok := <-ch1
	assert.False(t, ok, "channel should be closed when hub is closed")
	assert.Equal(t, 0, hub.ClientCount())

	// Subscribing after close should return error
	_, _, err = hub.Subscribe(ctx)
	assert.Error(t, err)

	// Broadcasting after close should return error or no-op
	ev, _ := model.NewServerSentEvent("ev-after-close", model.EventSystemHeartbeat, []byte(`{}`), time.Now().UTC())
	err = hub.Broadcast(ctx, ev)
	assert.Error(t, err)
}
