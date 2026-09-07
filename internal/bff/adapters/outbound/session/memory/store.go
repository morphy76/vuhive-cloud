package memory

import (
	"context"
	"sync"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/rs/zerolog"
)

var _ outbound.SessionStore = (*MemorySessionStore)(nil)

// MemorySessionStore provides an in-memory, thread-safe implementation of outbound.SessionStore.
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[model.SessionID]*model.ClientSession
}

// NewMemorySessionStore constructs an initialized MemorySessionStore.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[model.SessionID]*model.ClientSession),
	}
}

// Create persists a new client session aggregate in memory.
func (s *MemorySessionStore) Create(ctx context.Context, session *model.ClientSession) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "MemorySessionStore.Create").Logger()
	log.Debug().Msg("starting memory session creation")

	if session == nil {
		err := model.ErrInvalidParameter
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("cannot create nil session")
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.ID] = session.Clone()

	log.Info().
		Str("session_id", string(session.ID)).
		Str("user_id", session.UserID).
		Dur("duration_ms", time.Since(start)).
		Msg("completed memory session creation")
	return nil
}

// Get retrieves a client session by its ID, returning model.ErrSessionNotFound if missing.
func (s *MemorySessionStore) Get(ctx context.Context, id model.SessionID) (*model.ClientSession, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "MemorySessionStore.Get").
		Str("session_id", string(id)).
		Logger()
	log.Debug().Msg("starting memory session lookup")

	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[id]
	if !exists {
		log.Info().Dur("duration_ms", time.Since(start)).Msg("session not found in memory store")
		return nil, model.ErrSessionNotFound
	}

	log.Info().
		Str("user_id", session.UserID).
		Dur("duration_ms", time.Since(start)).
		Msg("completed memory session lookup")
	return session.Clone(), nil
}

// Update updates an existing client session in memory.
func (s *MemorySessionStore) Update(ctx context.Context, session *model.ClientSession) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "MemorySessionStore.Update").Logger()
	log.Debug().Msg("starting memory session update")

	if session == nil {
		err := model.ErrInvalidParameter
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("cannot update nil session")
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[session.ID]; !exists {
		log.Error().
			Str("session_id", string(session.ID)).
			Dur("duration_ms", time.Since(start)).
			Msg("cannot update non-existent session")
		return model.ErrSessionNotFound
	}

	s.sessions[session.ID] = session.Clone()

	log.Info().
		Str("session_id", string(session.ID)).
		Dur("duration_ms", time.Since(start)).
		Msg("completed memory session update")
	return nil
}

// Delete removes a session by ID. Deletion is idempotent.
func (s *MemorySessionStore) Delete(ctx context.Context, id model.SessionID) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "MemorySessionStore.Delete").
		Str("session_id", string(id)).
		Logger()
	log.Debug().Msg("starting memory session deletion")

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed memory session deletion")
	return nil
}

// DeleteByKeycloakSID terminates all sessions associated with a Keycloak session ID (for backchannel logout).
func (s *MemorySessionStore) DeleteByKeycloakSID(ctx context.Context, sid string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "MemorySessionStore.DeleteByKeycloakSID").
		Str("keycloak_sid", sid).
		Logger()
	log.Debug().Msg("starting deletion by keycloak SID")

	s.mu.Lock()
	defer s.mu.Unlock()

	var deletedCount int
	for id, sess := range s.sessions {
		if sess.KeycloakSID == sid {
			delete(s.sessions, id)
			deletedCount++
		}
	}

	log.Info().
		Int("deleted_count", deletedCount).
		Dur("duration_ms", time.Since(start)).
		Msg("completed deletion by keycloak SID")
	return nil
}

// DeleteByUserID terminates all sessions belonging to a specific user.
func (s *MemorySessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "MemorySessionStore.DeleteByUserID").
		Str("user_id", userID).
		Logger()
	log.Debug().Msg("starting deletion by user ID")

	s.mu.Lock()
	defer s.mu.Unlock()

	var deletedCount int
	for id, sess := range s.sessions {
		if sess.UserID == userID {
			delete(s.sessions, id)
			deletedCount++
		}
	}

	log.Info().
		Int("deleted_count", deletedCount).
		Dur("duration_ms", time.Since(start)).
		Msg("completed deletion by user ID")
	return nil
}

// DeleteExpired removes all sessions that expired before the specified cutoff time.
func (s *MemorySessionStore) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "MemorySessionStore.DeleteExpired").
		Time("cutoff", before).
		Logger()
	log.Debug().Msg("starting expired sessions deletion")

	s.mu.Lock()
	defer s.mu.Unlock()

	var count int64
	for id, sess := range s.sessions {
		if sess.ExpiresAt.Before(before) {
			delete(s.sessions, id)
			count++
		}
	}

	log.Info().
		Int64("deleted_count", count).
		Dur("duration_ms", time.Since(start)).
		Msg("completed expired sessions deletion")
	return count, nil
}
