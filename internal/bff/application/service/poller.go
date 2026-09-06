package service

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

// PollerConfig configures background polling worker schedules.
type PollerConfig struct {
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
}

// Poller runs background event polling against the control plane and publishes SSE events.
type Poller struct {
	cp              outbound.ControlPlaneClient
	hub             outbound.EventStreamHub
	cfg             PollerConfig
	lastRunStatus   map[string]string
	lastBuildStatus map[string]string
	mu              sync.Mutex
	eventSeq        atomic.Uint64
}

// NewPoller constructs an initialized Poller.
func NewPoller(cp outbound.ControlPlaneClient, hub outbound.EventStreamHub, cfg PollerConfig) *Poller {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 15 * time.Second
	}

	return &Poller{
		cp:              cp,
		hub:             hub,
		cfg:             cfg,
		lastRunStatus:   make(map[string]string),
		lastBuildStatus: make(map[string]string),
	}
}

// PollOnce queries active runs and builds, broadcasting SSE events for detected status changes.
func (p *Poller) PollOnce(ctx context.Context) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "Poller.PollOnce").Logger()
	log.Debug().Msg("executing state polling cycle")

	if p.cp == nil || p.hub == nil {
		return nil
	}

	// 1. Poll Test Runs across active and recent sets
	runsByID := make(map[string]outbound.RunDetail)

	// Fetch active runs
	if runningRuns, err := p.cp.ListRuns(ctx, "RUNNING", 50); err == nil {
		for _, r := range runningRuns {
			runsByID[r.ID] = r
		}
	}
	if queuedRuns, err := p.cp.ListRuns(ctx, "QUEUED", 50); err == nil {
		for _, r := range queuedRuns {
			runsByID[r.ID] = r
		}
	}
	// Fetch recent runs to capture transitions to COMPLETED, FAILED, ABORTED
	if recentRuns, err := p.cp.ListRuns(ctx, "", 20); err == nil {
		for _, r := range recentRuns {
			runsByID[r.ID] = r
		}
	}

	p.mu.Lock()
	for runID, run := range runsByID {
		lastStatus, exists := p.lastRunStatus[runID]
		if !exists || lastStatus != run.Status {
			p.lastRunStatus[runID] = run.Status

			payload := model.RunStatusChangedPayload{
				RunID:          run.ID,
				SuiteID:        run.SuiteID,
				Status:         run.Status,
				PreviousStatus: lastStatus,
				K8sJobName:     run.K8sJobName,
				StartedAt:      run.StartedAt,
				FinishedAt:     run.FinishedAt,
				DurationMs:     run.DurationMs,
				ExitCode:       run.ExitCode,
				SLAPassed:      run.SLAPassed,
				Metrics: &model.RunMetrics{
					TotalIterations: run.Metrics.TotalIterations,
					TotalRequests:   run.Metrics.TotalRequests,
					AvgTPS:          run.Metrics.AvgTPS,
					P50DurationMs:   run.Metrics.P50DurationMs,
					P90DurationMs:   run.Metrics.P90DurationMs,
					P95DurationMs:   run.Metrics.P95DurationMs,
					P99DurationMs:   run.Metrics.P99DurationMs,
					ErrorRatePct:    run.Metrics.ErrorRatePct,
				},
				Timestamp: time.Now().UTC(),
			}

			eventID := fmt.Sprintf("run-%s-%d", run.ID, p.eventSeq.Add(1))
			ev, err := model.NewRunStatusChangedEvent(eventID, payload)
			if err == nil {
				_ = p.hub.Broadcast(ctx, ev)
			}
		}
	}

	// 2. Poll Test Suites and their compiled Artifacts
	suites, err := p.cp.ListRecentSuites(ctx, 10)
	if err == nil && suites != nil {
		for _, suite := range suites {
			artifacts, artErr := p.cp.ListArtifacts(ctx, suite.ID)
			if artErr != nil || artifacts == nil {
				continue
			}

			for _, art := range artifacts {
				lastStatus, exists := p.lastBuildStatus[art.ID]
				if !exists || lastStatus != art.Status {
					p.lastBuildStatus[art.ID] = art.Status

					payload := model.BuildStatusChangedPayload{
						ArtifactID:     art.ID,
						SuiteID:        art.SuiteID,
						Platform:       art.Platform,
						Status:         art.Status,
						PreviousStatus: lastStatus,
						S3BinaryKey:    art.S3BinaryKey,
						SHA256Checksum: art.SHA256Checksum,
						ErrorMessage:   art.ErrorMessage,
						Timestamp:      time.Now().UTC(),
					}

					eventID := fmt.Sprintf("build-%s-%d", art.ID, p.eventSeq.Add(1))
					ev, err := model.NewBuildStatusChangedEvent(eventID, payload)
					if err == nil {
						_ = p.hub.Broadcast(ctx, ev)
					}
				}
			}
		}
	}
	p.mu.Unlock()

	log.Info().
		Int("observed_runs", len(runsByID)).
		Dur("duration_ms", time.Since(start)).
		Msg("completed state polling cycle")

	return nil
}

// EmitHeartbeat sends a system_heartbeat event to all subscribers.
func (p *Poller) EmitHeartbeat(ctx context.Context) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "Poller.EmitHeartbeat").Logger()
	log.Debug().Msg("emitting system heartbeat")

	if p.hub == nil {
		return nil
	}

	activeClients := p.hub.ClientCount()
	var activeRuns int64
	if p.cp != nil {
		activeRuns, _ = p.cp.GetActiveRunsCount(ctx)
	}

	payload := model.SystemHeartbeatPayload{
		Status:        "UP",
		ActiveClients: activeClients,
		ActiveRuns:    activeRuns,
		Timestamp:     time.Now().UTC(),
	}

	eventID := fmt.Sprintf("hb-%d", p.eventSeq.Add(1))
	ev, err := model.NewSystemHeartbeatEvent(eventID, payload)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating heartbeat event")
		return err
	}

	if err := p.hub.Broadcast(ctx, ev); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed broadcasting heartbeat")
		return err
	}

	log.Info().
		Int("active_clients", activeClients).
		Int64("active_runs", activeRuns).
		Dur("duration_ms", time.Since(start)).
		Msg("completed heartbeat emission")

	return nil
}

// Start begins the background polling loop until the context is cancelled.
func (p *Poller) Start(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().
		Str("op", "Poller.Start").
		Dur("poll_interval", p.cfg.PollInterval).
		Dur("heartbeat_interval", p.cfg.HeartbeatInterval).
		Logger()
	log.Info().Msg("starting background event poller loop")

	pollTicker := time.NewTicker(p.cfg.PollInterval)
	defer pollTicker.Stop()

	heartbeatTicker := time.NewTicker(p.cfg.HeartbeatInterval)
	defer heartbeatTicker.Stop()

	// Initial poll and heartbeat on launch
	_ = p.PollOnce(ctx)
	_ = p.EmitHeartbeat(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("stopping background event poller loop")
			return ctx.Err()
		case <-pollTicker.C:
			if err := p.PollOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Warn().Err(err).Msg("error during state polling cycle")
			}
		case <-heartbeatTicker.C:
			if err := p.EmitHeartbeat(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Warn().Err(err).Msg("error during heartbeat emission")
			}
		}
	}
}
