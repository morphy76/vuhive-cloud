package service

import (
	"context"
	"fmt"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/rs/zerolog"
)

var _ inbound.SessionService = (*SessionService)(nil)

// SessionServiceConfig configures session timeouts and sliding write-throttling behavior.
type SessionServiceConfig struct {
	SessionTTL       time.Duration // Default session inactivity timeout (default: 24h)
	SlidingThreshold time.Duration // Sliding expiration update threshold (default: 15m)
}

// SessionService orchestrates client session lifecycles, sliding expiration, and token rotation.
type SessionService struct {
	store outbound.SessionStore
	cfg   SessionServiceConfig
}

// NewSessionService constructs an initialized SessionService.
func NewSessionService(store outbound.SessionStore, cfgs ...SessionServiceConfig) *SessionService {
	cfg := SessionServiceConfig{
		SessionTTL:       24 * time.Hour,
		SlidingThreshold: 15 * time.Minute,
	}
	if len(cfgs) > 0 {
		c := cfgs[0]
		if c.SessionTTL > 0 {
			cfg.SessionTTL = c.SessionTTL
		}
		if c.SlidingThreshold > 0 {
			cfg.SlidingThreshold = c.SlidingThreshold
		}
	}

	return &SessionService{
		store: store,
		cfg:   cfg,
	}
}

// CreateSession initiates and persists a new client session aggregate.
func (s *SessionService) CreateSession(ctx context.Context, cmd inbound.CreateSessionCommand) (*model.ClientSession, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "SessionService.CreateSession").
		Str("session_id", cmd.SessionID).
		Str("user_id", cmd.UserID).
		Logger()
	log.Debug().Msg("starting session creation")

	ttl := cmd.TTL
	if ttl <= 0 {
		ttl = s.cfg.SessionTTL
	}

	var opts []model.SessionOption
	if cmd.KeycloakSID != "" {
		opts = append(opts, model.WithKeycloakSID(cmd.KeycloakSID))
	}
	if cmd.AccessToken != "" || cmd.RefreshToken != "" || cmd.IDToken != "" {
		opts = append(opts, model.WithTokens(cmd.AccessToken, cmd.RefreshToken, cmd.IDToken))
	}
	if len(cmd.Roles) > 0 {
		opts = append(opts, model.WithRoles(cmd.Roles))
	}
	if len(cmd.Metadata) > 0 {
		opts = append(opts, model.WithMetadata(cmd.Metadata))
	}

	session, err := model.NewClientSession(cmd.SessionID, cmd.UserID, ttl, opts...)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed session domain validation")
		return nil, err
	}

	if err := s.store.Create(ctx, session); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed persisting session to store")
		return nil, err
	}

	log.Info().
		Str("session_id", string(session.ID)).
		Str("user_id", session.UserID).
		Time("expires_at", session.ExpiresAt).
		Dur("duration_ms", time.Since(start)).
		Msg("completed session creation")

	return session, nil
}

// GetSession retrieves an active session by ID, applying sliding expiration write-throttling.
func (s *SessionService) GetSession(ctx context.Context, id model.SessionID) (*model.ClientSession, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "SessionService.GetSession").
		Str("session_id", string(id)).
		Logger()
	log.Debug().Msg("starting session lookup")

	if id == "" {
		err := fmt.Errorf("%w: session ID cannot be empty", model.ErrInvalidParameter)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid session ID")
		return nil, err
	}

	session, err := s.store.Get(ctx, id)
	if err != nil {
		log.Info().Err(err).Dur("duration_ms", time.Since(start)).Msg("session lookup failed or not found")
		return nil, err
	}

	now := time.Now()
	if now.After(session.ExpiresAt) {
		log.Info().
			Time("expires_at", session.ExpiresAt).
			Dur("duration_ms", time.Since(start)).
			Msg("session has expired")
		return nil, model.ErrSessionExpired
	}

	// Sliding expiration optimization: only touch and update store if sliding threshold has passed
	if now.Sub(session.UpdatedAt) >= s.cfg.SlidingThreshold {
		ttl := session.ExpiresAt.Sub(session.UpdatedAt)
		if ttl <= 0 {
			ttl = s.cfg.SessionTTL
		}

		if err := session.Touch(ttl); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed touching session")
			return nil, err
		}

		if err := s.store.Update(ctx, session); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed persisting touched session to store")
			return nil, err
		}

		log.Debug().
			Time("new_expires_at", session.ExpiresAt).
			Msg("sliding expiration extended in store")
	}

	log.Info().
		Str("session_id", string(session.ID)).
		Str("user_id", session.UserID).
		Dur("duration_ms", time.Since(start)).
		Msg("completed session lookup")

	return session, nil
}

// RotateSessionTokens updates OAuth2/OIDC tokens and extends session TTL.
func (s *SessionService) RotateSessionTokens(ctx context.Context, cmd inbound.RotateTokensCommand) (*model.ClientSession, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "SessionService.RotateSessionTokens").
		Str("session_id", string(cmd.SessionID)).
		Logger()
	log.Debug().Msg("starting session token rotation")

	if cmd.SessionID == "" {
		err := fmt.Errorf("%w: session ID cannot be empty", model.ErrInvalidParameter)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid session ID")
		return nil, err
	}

	session, err := s.store.Get(ctx, cmd.SessionID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed querying session for rotation")
		return nil, err
	}

	if session.IsExpired() {
		log.Info().Dur("duration_ms", time.Since(start)).Msg("cannot rotate tokens on expired session")
		return nil, model.ErrSessionExpired
	}

	ttl := cmd.TTL
	if ttl <= 0 {
		ttl = session.ExpiresAt.Sub(session.UpdatedAt)
	}
	if ttl <= 0 {
		ttl = s.cfg.SessionTTL
	}

	if err := session.RotateTokens(cmd.AccessToken, cmd.RefreshToken, cmd.IDToken, ttl); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed rotating session tokens in domain aggregate")
		return nil, err
	}

	if err := s.store.Update(ctx, session); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed persisting rotated tokens to store")
		return nil, err
	}

	log.Info().
		Str("session_id", string(session.ID)).
		Str("user_id", session.UserID).
		Time("new_expires_at", session.ExpiresAt).
		Dur("duration_ms", time.Since(start)).
		Msg("completed session token rotation")

	return session, nil
}

// RotateTokens is a convenience helper method to rotate tokens without manually constructing a RotateTokensCommand.
func (s *SessionService) RotateTokens(ctx context.Context, id model.SessionID, accessToken, refreshToken, idToken string, ttl time.Duration) (*model.ClientSession, error) {
	return s.RotateSessionTokens(ctx, inbound.RotateTokensCommand{
		SessionID:    id,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		TTL:          ttl,
	})
}

// RevokeSession terminates a session immediately by ID.
func (s *SessionService) RevokeSession(ctx context.Context, id model.SessionID) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "SessionService.RevokeSession").
		Str("session_id", string(id)).
		Logger()
	log.Debug().Msg("starting session revocation")

	if id == "" {
		err := fmt.Errorf("%w: session ID cannot be empty", model.ErrInvalidParameter)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid session ID")
		return err
	}

	if err := s.store.Delete(ctx, id); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting session from store")
		return err
	}

	log.Info().
		Str("session_id", string(id)).
		Dur("duration_ms", time.Since(start)).
		Msg("completed session revocation")

	return nil
}

// RevokeByKeycloakSID terminates all sessions associated with a Keycloak session ID (for backchannel logout).
func (s *SessionService) RevokeByKeycloakSID(ctx context.Context, sid string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "SessionService.RevokeByKeycloakSID").
		Str("keycloak_sid", sid).
		Logger()
	log.Debug().Msg("starting revocation by Keycloak SID")

	if sid == "" {
		err := fmt.Errorf("%w: keycloak SID cannot be empty", model.ErrInvalidParameter)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid Keycloak SID")
		return err
	}

	if err := s.store.DeleteByKeycloakSID(ctx, sid); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting sessions by Keycloak SID")
		return err
	}

	log.Info().
		Str("keycloak_sid", sid).
		Dur("duration_ms", time.Since(start)).
		Msg("completed revocation by Keycloak SID")

	return nil
}
