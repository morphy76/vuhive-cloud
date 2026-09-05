package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// BarrierService implements inbound.BarrierUseCase for coordinating distributed pod startup rendezvous.
type BarrierService struct {
	coordinator outbound.BarrierCoordinatorPort
}

// NewBarrierService constructs a new BarrierService using the provided coordinator port.
func NewBarrierService(coordinator outbound.BarrierCoordinatorPort) *BarrierService {
	return &BarrierService{
		coordinator: coordinator,
	}
}

// AwaitRendezvous validates inputs, registers the worker in the barrier, and blocks until all peers are ready.
func (s *BarrierService) AwaitRendezvous(ctx context.Context, cmd inbound.AwaitRendezvousCommand) (*inbound.RendezvousResult, error) {
	start := time.Now()
	runID := strings.TrimSpace(cmd.RunID)
	workerID := strings.TrimSpace(cmd.WorkerID)

	log := zerolog.Ctx(ctx).With().
		Str("op", "BarrierService.AwaitRendezvous").
		Str("run_id", runID).
		Str("worker_id", workerID).
		Int("total_workers", cmd.TotalWorkers).
		Logger()
	log.Debug().Msg("starting barrier rendezvous await")

	if runID == "" {
		err := fmt.Errorf("%w: run_id is required", model.ErrValidation)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("barrier validation failed")
		return nil, err
	}
	if workerID == "" {
		err := fmt.Errorf("%w: worker_id is required", model.ErrValidation)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("barrier validation failed")
		return nil, err
	}
	if cmd.TotalWorkers < 1 {
		err := model.ErrInvalidWorkerCount
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("barrier validation failed")
		return nil, err
	}

	session, err := s.coordinator.Await(ctx, runID, workerID, cmd.TotalWorkers, cmd.Timeout, cmd.ReleaseDelay)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("barrier rendezvous failed or timed out")
		return nil, err
	}

	targetStartTime := time.Now().UTC()
	if session.TargetStartTime() != nil {
		targetStartTime = *session.TargetStartTime()
	}

	startIn := time.Until(targetStartTime)
	if startIn < 0 {
		startIn = 0
	}

	res := &inbound.RendezvousResult{
		Status:          session.Status(),
		RunID:           session.RunID(),
		WorkerID:        workerID,
		TotalWorkers:    session.TotalWorkers(),
		TargetStartTime: targetStartTime,
		StartIn:         startIn,
	}

	log.Info().
		Str("status", string(session.Status())).
		Time("target_start_time", targetStartTime).
		Dur("start_in_ms", startIn).
		Dur("duration_ms", time.Since(start)).
		Msg("completed barrier rendezvous await")

	return res, nil
}

// SignalAbort transitions the barrier to ABORTED, unblocking all participating workers.
func (s *BarrierService) SignalAbort(ctx context.Context, cmd inbound.SignalAbortCommand) error {
	start := time.Now()
	runID := strings.TrimSpace(cmd.RunID)
	workerID := strings.TrimSpace(cmd.WorkerID)
	reason := strings.TrimSpace(cmd.Reason)

	log := zerolog.Ctx(ctx).With().
		Str("op", "BarrierService.SignalAbort").
		Str("run_id", runID).
		Str("worker_id", workerID).
		Str("reason", reason).
		Logger()
	log.Debug().Msg("starting barrier abort signaling")

	if runID == "" {
		err := fmt.Errorf("%w: run_id is required", model.ErrValidation)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("abort validation failed")
		return err
	}

	if err := s.coordinator.Abort(ctx, runID, workerID, reason); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed signaling barrier abort")
		return err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed barrier abort signaling")
	return nil
}

// GetBarrierStatus retrieves the current barrier session for a run.
func (s *BarrierService) GetBarrierStatus(ctx context.Context, runID string) (*model.BarrierSession, error) {
	start := time.Now()
	trimmedRunID := strings.TrimSpace(runID)

	log := zerolog.Ctx(ctx).With().
		Str("op", "BarrierService.GetBarrierStatus").
		Str("run_id", trimmedRunID).
		Logger()
	log.Debug().Msg("fetching barrier status")

	if trimmedRunID == "" {
		err := fmt.Errorf("%w: run_id is required", model.ErrValidation)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("get barrier validation failed")
		return nil, err
	}

	session, err := s.coordinator.Get(ctx, trimmedRunID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching barrier session")
		return nil, err
	}

	log.Info().
		Str("status", string(session.Status())).
		Int("ready_workers", session.ReadyCount()).
		Int("total_workers", session.TotalWorkers()).
		Dur("duration_ms", time.Since(start)).
		Msg("completed barrier status retrieval")

	return session, nil
}

// Compile-time static interface verification
var _ inbound.BarrierUseCase = (*BarrierService)(nil)
