package inbound

import (
	"context"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// AwaitRendezvousCommand contains parameters for a worker pod to join and block at the barrier.
type AwaitRendezvousCommand struct {
	RunID        string
	WorkerID     string
	TotalWorkers int
	Timeout      time.Duration
	ReleaseDelay time.Duration
}

// SignalAbortCommand contains parameters for aborting the barrier rendezvous.
type SignalAbortCommand struct {
	RunID    string
	WorkerID string
	Reason   string
}

// RendezvousResult represents the outcome when released from the barrier.
type RendezvousResult struct {
	Status          model.BarrierStatus
	RunID           string
	WorkerID        string
	TotalWorkers    int
	TargetStartTime time.Time
	StartIn         time.Duration
}

// BarrierUseCase defines the driving use case for distributed barrier rendezvous coordination.
type BarrierUseCase interface {
	AwaitRendezvous(ctx context.Context, cmd AwaitRendezvousCommand) (*RendezvousResult, error)
	SignalAbort(ctx context.Context, cmd SignalAbortCommand) error
	GetBarrierStatus(ctx context.Context, runID string) (*model.BarrierSession, error)
}
