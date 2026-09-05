package coordinator_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/coordinator"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryBarrierCoordinator_ConcurrentRendezvous(t *testing.T) {
	t.Parallel()

	coord := coordinator.NewMemoryBarrierCoordinator()
	runID := "run-rendezvous-concurrent"
	totalWorkers := 5
	releaseDelay := 100 * time.Millisecond
	timeout := 5 * time.Second

	var wg sync.WaitGroup
	var releaseCount int32
	results := make([]*model.BarrierSession, totalWorkers)
	errors := make([]error, totalWorkers)

	for i := 0; i < totalWorkers; i++ {
		wg.Add(1)
		workerIdx := i
		workerID := "worker-" + string(rune('A'+i))

		go func() {
			defer wg.Done()
			// Simulate staggered worker startup
			time.Sleep(time.Duration(workerIdx*20) * time.Millisecond)

			session, err := coord.Await(context.Background(), runID, workerID, totalWorkers, timeout, releaseDelay)
			results[workerIdx] = session
			errors[workerIdx] = err
			if err == nil && session.IsReleased() {
				atomic.AddInt32(&releaseCount, 1)
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(totalWorkers), releaseCount)
	for i := 0; i < totalWorkers; i++ {
		require.NoError(t, errors[i])
		require.NotNil(t, results[i])
		assert.Equal(t, model.BarrierStatusReleased, results[i].Status())
		assert.NotNil(t, results[i].TargetStartTime())
		// TargetStartTime should be identical across all workers
		assert.Equal(t, results[0].TargetStartTime().UnixNano(), results[i].TargetStartTime().UnixNano())
	}
}

func TestMemoryBarrierCoordinator_Abort(t *testing.T) {
	t.Parallel()

	coord := coordinator.NewMemoryBarrierCoordinator()
	runID := "run-abort-test"
	totalWorkers := 3
	timeout := 5 * time.Second

	var wg sync.WaitGroup
	errors := make([]error, 2)

	// Launch 2 workers that will block
	for i := 0; i < 2; i++ {
		wg.Add(1)
		idx := i
		workerID := "worker-" + string(rune('A'+i))
		go func() {
			defer wg.Done()
			_, err := coord.Await(context.Background(), runID, workerID, totalWorkers, timeout, 50*time.Millisecond)
			errors[idx] = err
		}()
	}

	// Small delay to ensure workers are waiting
	time.Sleep(50 * time.Millisecond)

	// Worker 3 fails preflight check and aborts
	err := coord.Abort(context.Background(), runID, "worker-C", "preflight binary check failed")
	require.NoError(t, err)

	wg.Wait()

	// Both waiting workers must unblock with ErrBarrierAborted
	assert.ErrorIs(t, errors[0], model.ErrBarrierAborted)
	assert.ErrorIs(t, errors[1], model.ErrBarrierAborted)

	// Status should be aborted
	session, err := coord.Get(context.Background(), runID)
	require.NoError(t, err)
	assert.Equal(t, model.BarrierStatusAborted, session.Status())
	assert.Equal(t, "preflight binary check failed", session.AbortReason())
}

func TestMemoryBarrierCoordinator_Timeout(t *testing.T) {
	t.Parallel()

	coord := coordinator.NewMemoryBarrierCoordinator()
	runID := "run-timeout-test"
	totalWorkers := 3
	timeout := 80 * time.Millisecond

	// Only 1 worker arrives, expecting 3
	session, err := coord.Await(context.Background(), runID, "worker-lonely", totalWorkers, timeout, 50*time.Millisecond)
	assert.ErrorIs(t, err, model.ErrBarrierTimeout)
	assert.Nil(t, session)

	// Status should be timed out
	sess, err := coord.Get(context.Background(), runID)
	require.NoError(t, err)
	assert.Equal(t, model.BarrierStatusTimedOut, sess.Status())
}

func TestMemoryBarrierCoordinator_ContextCancelled(t *testing.T) {
	t.Parallel()

	coord := coordinator.NewMemoryBarrierCoordinator()
	runID := "run-ctx-cancel"
	totalWorkers := 2

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	session, err := coord.Await(ctx, runID, "worker-1", totalWorkers, 5*time.Second, 50*time.Millisecond)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, session)
}
