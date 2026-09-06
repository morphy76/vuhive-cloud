package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/rs/zerolog"
)

var _ inbound.BFFService = (*BFFService)(nil)

// BFFService orchestrates Backend-For-Frontend use cases and status aggregation.
type BFFService struct {
	controlPlane outbound.ControlPlaneClient
	cache        outbound.CachePort
	version      string
}

// NewBFFService creates an instance of the BFF use case orchestrator.
func NewBFFService(cp outbound.ControlPlaneClient, cache outbound.CachePort, version string) *BFFService {
	return &BFFService{
		controlPlane: cp,
		cache:        cache,
		version:      version,
	}
}

// GetStatus aggregates the health and runtime version of the BFF and the upstream control plane.
func (s *BFFService) GetStatus(ctx context.Context) (*inbound.SystemStatus, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "GetStatus").Logger()
	log.Debug().Msg("starting get status aggregation")

	status := &inbound.SystemStatus{
		BFFStatus:           "UP",
		BFFVersion:          s.version,
		ControlPlaneStatus:  "UNKNOWN",
		ControlPlaneVersion: "",
		Timestamp:           time.Now().UTC(),
	}

	if s.controlPlane != nil {
		cpHealth, err := s.controlPlane.CheckHealth(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("upstream control plane is unreachable or reported error")
			status.ControlPlaneStatus = "DOWN"
		} else if cpHealth != nil {
			status.ControlPlaneStatus = cpHealth.Status
		}

		if status.ControlPlaneStatus == "UP" {
			cpVersion, err := s.controlPlane.GetVersion(ctx)
			if err == nil && cpVersion != nil {
				status.ControlPlaneVersion = cpVersion.Version
			}
		}
	}

	log.Info().
		Str("bff_status", status.BFFStatus).
		Str("control_plane_status", status.ControlPlaneStatus).
		Dur("duration_ms", time.Since(start)).
		Msg("completed get status aggregation")

	return status, nil
}

// CreateSession initiates and persists a client session aggregate.
func (s *BFFService) CreateSession(ctx context.Context, cmd inbound.CreateSessionCommand) (*model.ClientSession, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "CreateSession").
		Str("session_id", cmd.SessionID).
		Str("user_id", cmd.UserID).
		Logger()
	log.Debug().Msg("starting session creation")

	session, err := model.NewClientSession(cmd.SessionID, cmd.UserID, cmd.TTL)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed session domain validation")
		return nil, err
	}

	if cmd.Metadata != nil {
		session.Metadata = cmd.Metadata
	}

	if s.cache != nil {
		data, marshalErr := json.Marshal(session)
		if marshalErr != nil {
			log.Error().Err(marshalErr).Dur("duration_ms", time.Since(start)).Msg("failed serializing session")
			return nil, fmt.Errorf("%w: failed to serialize session: %v", model.ErrInternal, marshalErr)
		}

		cacheKey := fmt.Sprintf("session:%s", session.ID)
		if setErr := s.cache.Set(ctx, cacheKey, data, cmd.TTL); setErr != nil {
			log.Error().Err(setErr).Dur("duration_ms", time.Since(start)).Msg("failed persisting session to cache")
			return nil, fmt.Errorf("%w: failed to cache session: %v", model.ErrInternal, setErr)
		}
	}

	log.Info().
		Str("session_id", string(session.ID)).
		Dur("duration_ms", time.Since(start)).
		Msg("completed session creation")

	return session, nil
}

// GetSession retrieves and validates an existing active client session.
func (s *BFFService) GetSession(ctx context.Context, id model.SessionID) (*model.ClientSession, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "GetSession").
		Str("session_id", string(id)).
		Logger()
	log.Debug().Msg("starting session lookup")

	if id == "" {
		err := fmt.Errorf("%w: session ID cannot be empty", model.ErrInvalidParameter)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid session ID")
		return nil, err
	}

	if s.cache == nil {
		log.Error().Dur("duration_ms", time.Since(start)).Msg("cache adapter not configured")
		return nil, model.ErrSessionNotFound
	}

	cacheKey := fmt.Sprintf("session:%s", id)
	data, found, err := s.cache.Get(ctx, cacheKey)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("error retrieving session from cache")
		return nil, fmt.Errorf("%w: failed reading session cache: %v", model.ErrInternal, err)
	}
	if !found {
		log.Info().Dur("duration_ms", time.Since(start)).Msg("session not found in cache")
		return nil, model.ErrSessionNotFound
	}

	var session model.ClientSession
	if err := json.Unmarshal(data, &session); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deserializing session")
		return nil, fmt.Errorf("%w: corrupted session data: %v", model.ErrInternal, err)
	}

	if session.IsExpired() {
		log.Info().Dur("duration_ms", time.Since(start)).Msg("session expired")
		_ = s.cache.Delete(ctx, cacheKey)
		return nil, model.ErrSessionNotFound
	}

	log.Info().
		Str("session_id", string(session.ID)).
		Dur("duration_ms", time.Since(start)).
		Msg("completed session lookup")

	return &session, nil
}

// GetDashboard aggregates telemetry across control plane APIs concurrently to build the dashboard overview.
func (s *BFFService) GetDashboard(ctx context.Context) (*inbound.DashboardOverview, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "GetDashboard").Logger()
	log.Debug().Msg("starting dashboard composite aggregation")

	overview := &inbound.DashboardOverview{
		BFFStatus:           "UP",
		BFFVersion:          s.version,
		ControlPlaneStatus:  "UNKNOWN",
		ControlPlaneVersion: "",
		ActiveRunsCount:     0,
		RecentSuites:        []outbound.SuiteSummary{},
		ProfilesSummary:     []outbound.ProfileSummary{},
		Timestamp:           time.Now().UTC(),
	}

	if s.controlPlane == nil {
		overview.ControlPlaneStatus = "DOWN"
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("control plane client not configured")
		return overview, nil
	}

	var (
		wg              sync.WaitGroup
		mu              sync.Mutex
		cpStatus        = "UP"
		cpVersion       = ""
		activeRunsCount int64
		recentSuites    []outbound.SuiteSummary
		profilesSummary []outbound.ProfileSummary
	)

	// 1. Health & Version check
	wg.Add(1)
	go func() {
		defer wg.Done()
		health, err := s.controlPlane.CheckHealth(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("control plane health check failed")
			mu.Lock()
			cpStatus = "DOWN"
			mu.Unlock()
			return
		}
		if health != nil {
			mu.Lock()
			cpStatus = health.Status
			mu.Unlock()
		}

		ver, err := s.controlPlane.GetVersion(ctx)
		if err == nil && ver != nil {
			mu.Lock()
			cpVersion = ver.Version
			mu.Unlock()
		}
	}()

	// 2. Active runs count
	wg.Add(1)
	go func() {
		defer wg.Done()
		count, err := s.controlPlane.GetActiveRunsCount(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("failed querying active runs count")
			return
		}
		mu.Lock()
		activeRunsCount = count
		mu.Unlock()
	}()

	// 3. Recent test suites
	wg.Add(1)
	go func() {
		defer wg.Done()
		suites, err := s.controlPlane.ListRecentSuites(ctx, 5)
		if err != nil {
			log.Warn().Err(err).Msg("failed querying recent suites")
			return
		}
		if suites != nil {
			mu.Lock()
			recentSuites = suites
			mu.Unlock()
		}
	}()

	// 4. Runner profiles
	wg.Add(1)
	go func() {
		defer wg.Done()
		profiles, err := s.controlPlane.ListProfiles(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("failed querying runner profiles")
			return
		}
		if profiles != nil {
			mu.Lock()
			profilesSummary = profiles
			mu.Unlock()
		}
	}()

	wg.Wait()

	overview.ControlPlaneStatus = cpStatus
	overview.ControlPlaneVersion = cpVersion
	overview.ActiveRunsCount = activeRunsCount
	if recentSuites != nil {
		overview.RecentSuites = recentSuites
	}
	if profilesSummary != nil {
		overview.ProfilesSummary = profilesSummary
		overview.ProfilesCount = len(profilesSummary)
	}

	log.Info().
		Str("control_plane_status", overview.ControlPlaneStatus).
		Int64("active_runs", overview.ActiveRunsCount).
		Int("suites_count", len(overview.RecentSuites)).
		Int("profiles_count", overview.ProfilesCount).
		Dur("duration_ms", time.Since(start)).
		Msg("completed dashboard composite aggregation")

	return overview, nil
}

// GetRunDetail aggregates run metadata, indexed KPIs, and direct presigned S3 artifact links.
func (s *BFFService) GetRunDetail(ctx context.Context, id string) (*inbound.RunDetailComposite, error) {
	start := time.Now()
	runID := strings.TrimSpace(id)
	log := zerolog.Ctx(ctx).With().
		Str("op", "GetRunDetail").
		Str("run_id", runID).
		Logger()
	log.Debug().Msg("starting run detail composite aggregation")

	if runID == "" {
		err := fmt.Errorf("%w: run id cannot be empty", model.ErrInvalidParameter)
		log.Warn().Err(err).Msg("invalid run id argument")
		return nil, err
	}

	if s.controlPlane == nil {
		log.Error().Dur("duration_ms", time.Since(start)).Msg("control plane client not configured")
		return nil, model.ErrControlPlaneUnavailable
	}

	run, err := s.controlPlane.GetRun(ctx, runID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed retrieving run detail")
		return nil, err
	}

	var (
		wg        sync.WaitGroup
		reportURL string
		logsURL   string
	)

	// Fetch artifact presigned URLs concurrently
	wg.Add(2)
	go func() {
		defer wg.Done()
		u, err := s.controlPlane.GetRunReportURL(ctx, runID)
		if err != nil {
			log.Warn().Err(err).Msg("failed retrieving presigned report URL")
			return
		}
		reportURL = u
	}()

	go func() {
		defer wg.Done()
		u, err := s.controlPlane.GetRunLogsURL(ctx, runID)
		if err != nil {
			log.Warn().Err(err).Msg("failed retrieving presigned logs URL")
			return
		}
		logsURL = u
	}()

	wg.Wait()

	composite := &inbound.RunDetailComposite{
		RunDetail: *run,
		ArtifactLinks: inbound.ArtifactLinks{
			ReportURL: reportURL,
			LogsURL:   logsURL,
		},
	}

	log.Info().
		Str("status", composite.Status).
		Bool("has_report_url", composite.ArtifactLinks.ReportURL != "").
		Bool("has_logs_url", composite.ArtifactLinks.LogsURL != "").
		Dur("duration_ms", time.Since(start)).
		Msg("completed run detail composite aggregation")

	return composite, nil
}
