package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/application/service"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBarrierCoordinator struct {
	awaitFn func(ctx context.Context, runID, workerID string, totalWorkers int, timeout, releaseDelay time.Duration) (*model.BarrierSession, error)
	abortFn func(ctx context.Context, runID, workerID, reason string) error
	getFn   func(ctx context.Context, runID string) (*model.BarrierSession, error)
}

func (m *mockBarrierCoordinator) Await(ctx context.Context, runID, workerID string, totalWorkers int, timeout, releaseDelay time.Duration) (*model.BarrierSession, error) {
	if m.awaitFn != nil {
		return m.awaitFn(ctx, runID, workerID, totalWorkers, timeout, releaseDelay)
	}
	return nil, nil
}

func (m *mockBarrierCoordinator) Abort(ctx context.Context, runID, workerID, reason string) error {
	if m.abortFn != nil {
		return m.abortFn(ctx, runID, workerID, reason)
	}
	return nil
}

func (m *mockBarrierCoordinator) Get(ctx context.Context, runID string) (*model.BarrierSession, error) {
	if m.getFn != nil {
		return m.getFn(ctx, runID)
	}
	return nil, nil
}

var _ outbound.BarrierCoordinatorPort = (*mockBarrierCoordinator)(nil)

func TestBarrierService_AwaitRendezvous(t *testing.T) {
	t.Parallel()

	t.Run("fails validation when runID is empty", func(t *testing.T) {
		t.Parallel()
		svc := service.NewBarrierService(&mockBarrierCoordinator{})
		res, err := svc.AwaitRendezvous(context.Background(), inbound.AwaitRendezvousCommand{
			RunID:        "",
			WorkerID:     "w-1",
			TotalWorkers: 2,
		})
		assert.ErrorIs(t, err, model.ErrValidation)
		assert.Nil(t, res)
	})

	t.Run("fails validation when workerID is empty", func(t *testing.T) {
		t.Parallel()
		svc := service.NewBarrierService(&mockBarrierCoordinator{})
		res, err := svc.AwaitRendezvous(context.Background(), inbound.AwaitRendezvousCommand{
			RunID:        "run-1",
			WorkerID:     "",
			TotalWorkers: 2,
		})
		assert.ErrorIs(t, err, model.ErrValidation)
		assert.Nil(t, res)
	})

	t.Run("successful rendezvous returns valid RendezvousResult", func(t *testing.T) {
		t.Parallel()
		target := time.Now().Add(500 * time.Millisecond)
		session, err := model.NewBarrierSession("run-1", 2, 10*time.Second, 500*time.Millisecond)
		require.NoError(t, err)
		require.NoError(t, session.RegisterWorker("w-1"))
		_, _ = session.MarkWorkerReady("w-1")
		require.NoError(t, session.RegisterWorker("w-2"))
		_, _ = session.MarkWorkerReady("w-2")
		_, _ = session.Release()

		mock := &mockBarrierCoordinator{
			awaitFn: func(ctx context.Context, runID, workerID string, totalWorkers int, timeout, releaseDelay time.Duration) (*model.BarrierSession, error) {
				return session, nil
			},
		}

		svc := service.NewBarrierService(mock)
		res, err := svc.AwaitRendezvous(context.Background(), inbound.AwaitRendezvousCommand{
			RunID:        "run-1",
			WorkerID:     "w-1",
			TotalWorkers: 2,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, model.BarrierStatusReleased, res.Status)
		assert.Equal(t, "run-1", res.RunID)
		assert.Equal(t, "w-1", res.WorkerID)
		assert.Equal(t, 2, res.TotalWorkers)
		assert.False(t, res.TargetStartTime.IsZero())
		_ = target
	})
}

func TestBarrierService_SignalAbort(t *testing.T) {
	t.Parallel()

	t.Run("fails validation when runID is empty", func(t *testing.T) {
		t.Parallel()
		svc := service.NewBarrierService(&mockBarrierCoordinator{})
		err := svc.SignalAbort(context.Background(), inbound.SignalAbortCommand{
			RunID:    "",
			WorkerID: "w-1",
			Reason:   "failed",
		})
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("delegates abort to coordinator", func(t *testing.T) {
		t.Parallel()
		aborted := false
		mock := &mockBarrierCoordinator{
			abortFn: func(ctx context.Context, runID, workerID, reason string) error {
				assert.Equal(t, "run-abort", runID)
				assert.Equal(t, "w-1", workerID)
				assert.Equal(t, "preflight check failed", reason)
				aborted = true
				return nil
			},
		}

		svc := service.NewBarrierService(mock)
		err := svc.SignalAbort(context.Background(), inbound.SignalAbortCommand{
			RunID:    "run-abort",
			WorkerID: "w-1",
			Reason:   "preflight check failed",
		})
		require.NoError(t, err)
		assert.True(t, aborted)
	})
}

func TestBarrierService_GetBarrierStatus(t *testing.T) {
	t.Parallel()

	session, err := model.NewBarrierSession("run-status", 3, 10*time.Second, 200*time.Millisecond)
	require.NoError(t, err)

	mock := &mockBarrierCoordinator{
		getFn: func(ctx context.Context, runID string) (*model.BarrierSession, error) {
			return session, nil
		},
	}

	svc := service.NewBarrierService(mock)
	got, err := svc.GetBarrierStatus(context.Background(), "run-status")
	require.NoError(t, err)
	assert.Equal(t, session.RunID(), got.RunID())
}
