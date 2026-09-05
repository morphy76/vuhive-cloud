package service

import (
	"context"
	"encoding/json"
	"fmt"
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
