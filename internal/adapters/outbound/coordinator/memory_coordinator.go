package coordinator

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

type activeSession struct {
	mu           sync.Mutex
	session      *model.BarrierSession
	releaseCh    chan struct{}
	abortCh      chan struct{}
	releasedOnce sync.Once
	abortedOnce  sync.Once
}

// MemoryBarrierCoordinator implements outbound.BarrierCoordinatorPort using thread-safe in-memory channels.
type MemoryBarrierCoordinator struct {
	mu       sync.Mutex
	sessions map[string]*activeSession
}

// NewMemoryBarrierCoordinator constructs a new MemoryBarrierCoordinator.
func NewMemoryBarrierCoordinator() *MemoryBarrierCoordinator {
	return &MemoryBarrierCoordinator{
		sessions: make(map[string]*activeSession),
	}
}

// Await registers the worker, signals readiness, and blocks until all peers arrive, or until timeout/abort.
func (c *MemoryBarrierCoordinator) Await(
	ctx context.Context,
	runID, workerID string,
	totalWorkers int,
	timeout, releaseDelay time.Duration,
) (*model.BarrierSession, error) {
	trimmedRunID := strings.TrimSpace(runID)
	trimmedWorkerID := strings.TrimSpace(workerID)

	act := c.getOrCreate(trimmedRunID, totalWorkers, timeout, releaseDelay)

	act.mu.Lock()
	sess := act.session

	if sess.Status() == model.BarrierStatusAborted {
		act.mu.Unlock()
		return nil, model.ErrBarrierAborted
	}
	if sess.Status() == model.BarrierStatusTimedOut {
		act.mu.Unlock()
		return nil, model.ErrBarrierTimeout
	}
	if sess.Status() == model.BarrierStatusReleased {
		act.mu.Unlock()
		return sess, nil
	}

	// Register worker if not already registered
	_ = sess.RegisterWorker(trimmedWorkerID)

	// Mark worker ready
	allReady, err := sess.MarkWorkerReady(trimmedWorkerID)
	if err != nil {
		act.mu.Unlock()
		return nil, err
	}

	releaseCh := act.releaseCh
	abortCh := act.abortCh

	if allReady {
		// All workers have arrived! Release barrier
		_, _ = sess.Release()
		act.releasedOnce.Do(func() {
			close(releaseCh)
		})
		act.mu.Unlock()
		return sess, nil
	}

	act.mu.Unlock()

	// Wait for release, abort, timeout, or context cancellation
	timer := time.NewTimer(sess.Timeout())
	defer timer.Stop()

	select {
	case <-releaseCh:
		act.mu.Lock()
		defer act.mu.Unlock()
		return act.session, nil

	case <-abortCh:
		return nil, model.ErrBarrierAborted

	case <-timer.C:
		act.mu.Lock()
		_ = act.session.TimeoutSession()
		act.mu.Unlock()
		return nil, model.ErrBarrierTimeout

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Abort transitions the barrier to ABORTED and broadcasts the abort to all waiters.
func (c *MemoryBarrierCoordinator) Abort(ctx context.Context, runID, workerID, reason string) error {
	trimmedRunID := strings.TrimSpace(runID)

	c.mu.Lock()
	act, exists := c.sessions[trimmedRunID]
	c.mu.Unlock()

	if !exists {
		return model.ErrBarrierNotFound
	}

	act.mu.Lock()
	defer act.mu.Unlock()

	_ = act.session.MarkWorkerAborted(workerID, reason)
	act.abortedOnce.Do(func() {
		close(act.abortCh)
	})

	return nil
}

// Get retrieves the current session state.
func (c *MemoryBarrierCoordinator) Get(ctx context.Context, runID string) (*model.BarrierSession, error) {
	trimmedRunID := strings.TrimSpace(runID)

	c.mu.Lock()
	act, exists := c.sessions[trimmedRunID]
	c.mu.Unlock()

	if !exists {
		return nil, model.ErrBarrierNotFound
	}

	act.mu.Lock()
	defer act.mu.Unlock()

	return act.session, nil
}

func (c *MemoryBarrierCoordinator) getOrCreate(
	runID string,
	totalWorkers int,
	timeout, releaseDelay time.Duration,
) *activeSession {
	c.mu.Lock()
	defer c.mu.Unlock()

	if act, exists := c.sessions[runID]; exists {
		return act
	}

	sess, _ := model.NewBarrierSession(runID, totalWorkers, timeout, releaseDelay)
	act := &activeSession{
		session:   sess,
		releaseCh: make(chan struct{}),
		abortCh:   make(chan struct{}),
	}
	c.sessions[runID] = act
	return act
}

// Compile-time static interface verification
var _ outbound.BarrierCoordinatorPort = (*MemoryBarrierCoordinator)(nil)
