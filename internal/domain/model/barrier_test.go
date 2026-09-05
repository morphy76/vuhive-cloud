package model_test

import (
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBarrierSession_Validation(t *testing.T) {
	t.Parallel()

	t.Run("fails when runID is empty", func(t *testing.T) {
		t.Parallel()
		session, err := model.NewBarrierSession("", 3, 30*time.Second, 200*time.Millisecond)
		assert.ErrorIs(t, err, model.ErrValidation)
		assert.Nil(t, session)
	})

	t.Run("fails when totalWorkers is less than 1", func(t *testing.T) {
		t.Parallel()
		session, err := model.NewBarrierSession("run-123", 0, 30*time.Second, 200*time.Millisecond)
		assert.ErrorIs(t, err, model.ErrInvalidWorkerCount)
		assert.Nil(t, session)
	})

	t.Run("defaults timeout and releaseDelay when non-positive", func(t *testing.T) {
		t.Parallel()
		session, err := model.NewBarrierSession("run-123", 2, 0, 0)
		require.NoError(t, err)
		require.NotNil(t, session)
		assert.Equal(t, model.BarrierStatusPending, session.Status())
		assert.Equal(t, 2, session.TotalWorkers())
		assert.Equal(t, model.DefaultBarrierTimeout, session.Timeout())
		assert.Equal(t, model.DefaultReleaseDelay, session.ReleaseDelay())
		assert.False(t, session.IsReleased())
		assert.False(t, session.IsTerminal())
	})
}

func TestBarrierSession_RegisterAndMarkReady(t *testing.T) {
	t.Parallel()

	session, err := model.NewBarrierSession("run-abc", 3, 10*time.Second, 300*time.Millisecond)
	require.NoError(t, err)

	// Register worker 1
	err = session.RegisterWorker("w-1")
	require.NoError(t, err)
	assert.Equal(t, 1, session.RegisteredCount())
	assert.Equal(t, 0, session.ReadyCount())

	// Duplicate registration should fail
	err = session.RegisterWorker("w-1")
	assert.ErrorIs(t, err, model.ErrWorkerAlreadyRegistered)

	// Mark worker 1 ready
	allReady, err := session.MarkWorkerReady("w-1")
	require.NoError(t, err)
	assert.False(t, allReady)
	assert.Equal(t, 1, session.ReadyCount())
	assert.Equal(t, model.BarrierStatusPending, session.Status())

	// Mark unknown worker ready should fail with ErrNotFound
	_, err = session.MarkWorkerReady("w-unknown")
	assert.ErrorIs(t, err, model.ErrNotFound)

	// Register & mark worker 2 ready
	require.NoError(t, session.RegisterWorker("w-2"))
	allReady, err = session.MarkWorkerReady("w-2")
	require.NoError(t, err)
	assert.False(t, allReady)
	assert.Equal(t, 2, session.ReadyCount())

	// Register & mark worker 3 ready (final worker)
	require.NoError(t, session.RegisterWorker("w-3"))
	allReady, err = session.MarkWorkerReady("w-3")
	require.NoError(t, err)
	assert.True(t, allReady)

	// Session should now be ready to release or automatically released
	targetTime, err := session.Release()
	require.NoError(t, err)
	require.NotNil(t, targetTime)
	assert.True(t, targetTime.After(time.Now()))
	assert.Equal(t, model.BarrierStatusReleased, session.Status())
	assert.True(t, session.IsReleased())
	assert.True(t, session.IsTerminal())

	// Attempting to release again should fail
	_, err = session.Release()
	assert.ErrorIs(t, err, model.ErrBarrierReleased)
}

func TestBarrierSession_AbortHandling(t *testing.T) {
	t.Parallel()

	session, err := model.NewBarrierSession("run-abort", 2, 10*time.Second, 100*time.Millisecond)
	require.NoError(t, err)

	require.NoError(t, session.RegisterWorker("w-1"))
	_, err = session.MarkWorkerReady("w-1")
	require.NoError(t, err)

	// Worker 2 registers and aborts due to preflight failure
	require.NoError(t, session.RegisterWorker("w-2"))
	err = session.MarkWorkerAborted("w-2", "preflight check failed: runner binary missing")
	require.NoError(t, err)

	assert.Equal(t, model.BarrierStatusAborted, session.Status())
	assert.True(t, session.IsTerminal())
	assert.Equal(t, "preflight check failed: runner binary missing", session.AbortReason())

	// Further operations should fail on aborted session
	err = session.RegisterWorker("w-3")
	assert.ErrorIs(t, err, model.ErrBarrierAborted)

	_, err = session.MarkWorkerReady("w-1")
	assert.ErrorIs(t, err, model.ErrBarrierAborted)
}

func TestBarrierSession_TimeoutHandling(t *testing.T) {
	t.Parallel()

	session, err := model.NewBarrierSession("run-timeout", 2, 50*time.Millisecond, 10*time.Millisecond)
	require.NoError(t, err)

	require.NoError(t, session.RegisterWorker("w-1"))
	_, err = session.MarkWorkerReady("w-1")
	require.NoError(t, err)

	assert.False(t, session.HasTimedOut(time.Now()))

	// Fast forward time past timeout
	future := time.Now().Add(100 * time.Millisecond)
	assert.True(t, session.HasTimedOut(future))

	err = session.TimeoutSession()
	require.NoError(t, err)
	assert.Equal(t, model.BarrierStatusTimedOut, session.Status())
	assert.True(t, session.IsTerminal())

	// Further operations should fail
	_, err = session.Release()
	assert.ErrorIs(t, err, model.ErrBarrierTimeout)
}
