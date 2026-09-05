package outbound

import (
	"context"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// BarrierCoordinatorPort defines the driven port for distributed start barrier coordination backends.
type BarrierCoordinatorPort interface {
	// Await blocks until all workers for runID are registered and ready, or until abort, timeout, or context cancellation.
	Await(ctx context.Context, runID, workerID string, totalWorkers int, timeout, releaseDelay time.Duration) (*model.BarrierSession, error)
	// Abort signals an early abort to all waiting workers in the barrier session.
	Abort(ctx context.Context, runID, workerID, reason string) error
	// Get retrieves the current barrier session for runID.
	Get(ctx context.Context, runID string) (*model.BarrierSession, error)
}
