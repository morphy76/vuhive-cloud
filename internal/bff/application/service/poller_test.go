package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/service"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPoller_PollOnce_RunStatusTransitions(t *testing.T) {
	ctx := context.Background()

	t.Run("detects run status changes and avoids duplicates", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockHub := new(MockEventStreamHub)

		var broadcastedEvents []model.ServerSentEvent
		var mu sync.Mutex

		mockHub.On("Broadcast", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			mu.Lock()
			defer mu.Unlock()
			broadcastedEvents = append(broadcastedEvents, args.Get(1).(model.ServerSentEvent))
		}).Return(nil)

		poller := service.NewPoller(mockCP, mockHub, service.PollerConfig{
			PollInterval:      100 * time.Millisecond,
			HeartbeatInterval: 1 * time.Second,
		})

		// First poll: run-1 is RUNNING
		mockCP.On("ListRuns", mock.Anything, "RUNNING", mock.Anything).Return([]outbound.RunDetail{
			{ID: "run-1", SuiteID: "suite-1", Status: "RUNNING"},
		}, nil).Once()
		mockCP.On("ListRuns", mock.Anything, "QUEUED", mock.Anything).Return([]outbound.RunDetail{}, nil).Once()
		mockCP.On("ListRuns", mock.Anything, "", mock.Anything).Return([]outbound.RunDetail{
			{ID: "run-1", SuiteID: "suite-1", Status: "RUNNING"},
		}, nil).Once()
		mockCP.On("ListRecentSuites", mock.Anything, mock.Anything).Return([]outbound.SuiteSummary{}, nil).Once()

		err := poller.PollOnce(ctx)
		require.NoError(t, err)

		mu.Lock()
		require.Len(t, broadcastedEvents, 1)
		assert.Equal(t, model.EventRunStatusChanged, broadcastedEvents[0].Event)
		mu.Unlock()

		// Second poll: run-1 still RUNNING (no duplicate event emitted)
		mockCP.On("ListRuns", mock.Anything, "RUNNING", mock.Anything).Return([]outbound.RunDetail{
			{ID: "run-1", SuiteID: "suite-1", Status: "RUNNING"},
		}, nil).Once()
		mockCP.On("ListRuns", mock.Anything, "QUEUED", mock.Anything).Return([]outbound.RunDetail{}, nil).Once()
		mockCP.On("ListRuns", mock.Anything, "", mock.Anything).Return([]outbound.RunDetail{
			{ID: "run-1", SuiteID: "suite-1", Status: "RUNNING"},
		}, nil).Once()
		mockCP.On("ListRecentSuites", mock.Anything, mock.Anything).Return([]outbound.SuiteSummary{}, nil).Once()

		err = poller.PollOnce(ctx)
		require.NoError(t, err)

		mu.Lock()
		assert.Len(t, broadcastedEvents, 1) // Still 1
		mu.Unlock()

		// Third poll: run-1 transitioned to COMPLETED
		mockCP.On("ListRuns", mock.Anything, "RUNNING", mock.Anything).Return([]outbound.RunDetail{}, nil).Once()
		mockCP.On("ListRuns", mock.Anything, "QUEUED", mock.Anything).Return([]outbound.RunDetail{}, nil).Once()
		mockCP.On("ListRuns", mock.Anything, "", mock.Anything).Return([]outbound.RunDetail{
			{ID: "run-1", SuiteID: "suite-1", Status: "COMPLETED"},
		}, nil).Once()
		mockCP.On("ListRecentSuites", mock.Anything, mock.Anything).Return([]outbound.SuiteSummary{}, nil).Once()

		err = poller.PollOnce(ctx)
		require.NoError(t, err)

		mu.Lock()
		require.Len(t, broadcastedEvents, 2)
		assert.Equal(t, model.EventRunStatusChanged, broadcastedEvents[1].Event)
		mu.Unlock()
	})
}

func TestPoller_PollOnce_BuildStatusTransitions(t *testing.T) {
	ctx := context.Background()

	t.Run("detects build status changes", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockHub := new(MockEventStreamHub)

		var broadcastedEvents []model.ServerSentEvent
		var mu sync.Mutex

		mockHub.On("Broadcast", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			mu.Lock()
			defer mu.Unlock()
			broadcastedEvents = append(broadcastedEvents, args.Get(1).(model.ServerSentEvent))
		}).Return(nil)

		poller := service.NewPoller(mockCP, mockHub, service.PollerConfig{
			PollInterval:      100 * time.Millisecond,
			HeartbeatInterval: 1 * time.Second,
		})

		mockCP.On("ListRuns", mock.Anything, mock.Anything, mock.Anything).Return([]outbound.RunDetail{}, nil)
		mockCP.On("ListRecentSuites", mock.Anything, mock.Anything).Return([]outbound.SuiteSummary{
			{ID: "suite-1", Name: "Suite One"},
		}, nil).Once()
		mockCP.On("ListArtifacts", mock.Anything, "suite-1").Return([]outbound.ArtifactDetail{
			{ID: "art-1", SuiteID: "suite-1", Platform: "linux/amd64", Status: "BUILDING"},
		}, nil).Once()

		err := poller.PollOnce(ctx)
		require.NoError(t, err)

		mu.Lock()
		require.Len(t, broadcastedEvents, 1)
		assert.Equal(t, model.EventBuildStatusChanged, broadcastedEvents[0].Event)
		mu.Unlock()

		// Next poll: art-1 transitions to READY
		mockCP.On("ListRecentSuites", mock.Anything, mock.Anything).Return([]outbound.SuiteSummary{
			{ID: "suite-1", Name: "Suite One"},
		}, nil).Once()
		mockCP.On("ListArtifacts", mock.Anything, "suite-1").Return([]outbound.ArtifactDetail{
			{ID: "art-1", SuiteID: "suite-1", Platform: "linux/amd64", Status: "READY"},
		}, nil).Once()

		err = poller.PollOnce(ctx)
		require.NoError(t, err)

		mu.Lock()
		require.Len(t, broadcastedEvents, 2)
		assert.Equal(t, model.EventBuildStatusChanged, broadcastedEvents[1].Event)
		mu.Unlock()
	})
}

func TestPoller_EmitHeartbeat(t *testing.T) {
	ctx := context.Background()

	t.Run("emits heartbeat with active clients and runs", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockHub := new(MockEventStreamHub)

		mockHub.On("ClientCount").Return(3)
		mockCP.On("GetActiveRunsCount", mock.Anything).Return(int64(1), nil)

		var capturedEvent model.ServerSentEvent
		mockHub.On("Broadcast", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			capturedEvent = args.Get(1).(model.ServerSentEvent)
		}).Return(nil).Once()

		poller := service.NewPoller(mockCP, mockHub, service.PollerConfig{})
		err := poller.EmitHeartbeat(ctx)
		require.NoError(t, err)

		assert.Equal(t, model.EventSystemHeartbeat, capturedEvent.Event)
		mockHub.AssertExpectations(t)
		mockCP.AssertExpectations(t)
	})
}

func TestPoller_Start_GracefulTeardown(t *testing.T) {
	mockCP := new(MockControlPlaneClient)
	mockHub := new(MockEventStreamHub)

	mockHub.On("ClientCount").Return(0).Maybe()
	mockCP.On("GetActiveRunsCount", mock.Anything).Return(int64(0), nil).Maybe()
	mockCP.On("ListRuns", mock.Anything, mock.Anything, mock.Anything).Return([]outbound.RunDetail{}, nil).Maybe()
	mockCP.On("ListRecentSuites", mock.Anything, mock.Anything).Return([]outbound.SuiteSummary{}, nil).Maybe()
	mockHub.On("Broadcast", mock.Anything, mock.Anything).Return(nil).Maybe()

	poller := service.NewPoller(mockCP, mockHub, service.PollerConfig{
		PollInterval:      50 * time.Millisecond,
		HeartbeatInterval: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- poller.Start(ctx)
	}()

	// Allow one iteration
	time.Sleep(70 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(1 * time.Second):
		t.Fatal("poller failed to stop cleanly within timeout")
	}
}
