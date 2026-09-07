package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/session/crypto"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/rs/zerolog"
)

var _ outbound.SessionStore = (*PostgresSessionStore)(nil)

// DB defines the minimal interface required for PostgreSQL operations.
// Satisfied by *pgxpool.Pool, *pgx.Conn, and mock DB implementations.
type DB interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

// Option configures optional parameters on PostgresSessionStore.
type Option func(*PostgresSessionStore)

// WithCipher configures the AES-256-GCM token encryption cipher.
func WithCipher(c *crypto.TokenCipher) Option {
	return func(s *PostgresSessionStore) {
		s.cipher = c
	}
}

// PostgresSessionStore implements outbound.SessionStore using PostgreSQL and pgx.
type PostgresSessionStore struct {
	db     DB
	cipher *crypto.TokenCipher
}

// NewPostgresSessionStore constructs an initialized PostgresSessionStore instance.
func NewPostgresSessionStore(db DB, opts ...Option) *PostgresSessionStore {
	store := &PostgresSessionStore{
		db: db,
	}
	for _, opt := range opts {
		opt(store)
	}
	return store
}

func (s *PostgresSessionStore) encrypt(plaintext string) (string, error) {
	if s.cipher == nil || plaintext == "" {
		return plaintext, nil
	}
	return s.cipher.Encrypt(plaintext)
}

func (s *PostgresSessionStore) decrypt(ciphertext string) (string, error) {
	if s.cipher == nil || ciphertext == "" {
		return ciphertext, nil
	}
	return s.cipher.Decrypt(ciphertext)
}

// Create persists a new client session in PostgreSQL, encrypting tokens if configured.
func (s *PostgresSessionStore) Create(ctx context.Context, session *model.ClientSession) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "PostgresSessionStore.Create").Logger()
	log.Debug().Msg("starting postgres session creation")

	if session == nil {
		err := model.ErrInvalidParameter
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("cannot create nil session")
		return err
	}

	encAccess, err := s.encrypt(session.AccessToken)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed encrypting access token")
		return fmt.Errorf("%w: failed encrypting access token: %v", model.ErrInternal, err)
	}

	encRefresh, err := s.encrypt(session.RefreshToken)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed encrypting refresh token")
		return fmt.Errorf("%w: failed encrypting refresh token: %v", model.ErrInternal, err)
	}

	encID, err := s.encrypt(session.IDToken)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed encrypting id token")
		return fmt.Errorf("%w: failed encrypting id token: %v", model.ErrInternal, err)
	}

	rolesJSON, err := json.Marshal(session.Roles)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed marshaling roles")
		return fmt.Errorf("%w: failed marshaling roles: %v", model.ErrInternal, err)
	}

	metaJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed marshaling metadata")
		return fmt.Errorf("%w: failed marshaling metadata: %v", model.ErrInternal, err)
	}

	query := `
		INSERT INTO bff_sessions (
			id, user_id, keycloak_sid, access_token, refresh_token, id_token,
			roles, metadata, created_at, updated_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = s.db.Exec(ctx, query,
		string(session.ID),
		session.UserID,
		session.KeycloakSID,
		encAccess,
		encRefresh,
		encID,
		rolesJSON,
		metaJSON,
		session.CreatedAt,
		session.UpdatedAt,
		session.ExpiresAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed executing insert session")
		return MapError(err)
	}

	log.Info().
		Str("session_id", string(session.ID)).
		Str("user_id", session.UserID).
		Dur("duration_ms", time.Since(start)).
		Msg("completed postgres session creation")
	return nil
}

// Get retrieves a client session by ID from PostgreSQL, decrypting tokens if configured.
func (s *PostgresSessionStore) Get(ctx context.Context, id model.SessionID) (*model.ClientSession, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "PostgresSessionStore.Get").
		Str("session_id", string(id)).
		Logger()
	log.Debug().Msg("starting postgres session lookup")

	if id == "" {
		err := model.ErrInvalidParameter
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("empty session ID")
		return nil, err
	}

	query := `
		SELECT id, user_id, keycloak_sid, access_token, refresh_token, id_token,
		       roles, metadata, created_at, updated_at, expires_at
		FROM bff_sessions
		WHERE id = $1
	`

	var (
		sessID, userID, kcSID, encAccess, encRefresh, encID string
		rolesRaw, metaRaw                                   []byte
		createdAt, updatedAt, expiresAt                     time.Time
	)

	row := s.db.QueryRow(ctx, query, string(id))
	err := row.Scan(
		&sessID,
		&userID,
		&kcSID,
		&encAccess,
		&encRefresh,
		&encID,
		&rolesRaw,
		&metaRaw,
		&createdAt,
		&updatedAt,
		&expiresAt,
	)
	if err != nil {
		mapped := MapError(err)
		if mapped == model.ErrSessionNotFound {
			log.Info().Dur("duration_ms", time.Since(start)).Msg("session not found in database")
		} else {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed querying session")
		}
		return nil, mapped
	}

	access, err := s.decrypt(encAccess)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed decrypting access token")
		return nil, fmt.Errorf("%w: failed decrypting access token: %v", model.ErrInternal, err)
	}

	refresh, err := s.decrypt(encRefresh)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed decrypting refresh token")
		return nil, fmt.Errorf("%w: failed decrypting refresh token: %v", model.ErrInternal, err)
	}

	idToken, err := s.decrypt(encID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed decrypting id token")
		return nil, fmt.Errorf("%w: failed decrypting id token: %v", model.ErrInternal, err)
	}

	var roles []string
	if len(rolesRaw) > 0 {
		if err := json.Unmarshal(rolesRaw, &roles); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed unmarshaling roles")
			return nil, fmt.Errorf("%w: failed unmarshaling roles: %v", model.ErrInternal, err)
		}
	}

	meta := make(map[string]string)
	if len(metaRaw) > 0 {
		if err := json.Unmarshal(metaRaw, &meta); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed unmarshaling metadata")
			return nil, fmt.Errorf("%w: failed unmarshaling metadata: %v", model.ErrInternal, err)
		}
	}

	session := &model.ClientSession{
		ID:           model.SessionID(sessID),
		UserID:       userID,
		KeycloakSID:  kcSID,
		AccessToken:  access,
		RefreshToken: refresh,
		IDToken:      idToken,
		Roles:        roles,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		ExpiresAt:    expiresAt,
		Metadata:     meta,
	}

	log.Info().
		Str("user_id", userID).
		Dur("duration_ms", time.Since(start)).
		Msg("completed postgres session lookup")
	return session, nil
}

// Update updates an existing client session in PostgreSQL, encrypting updated tokens.
func (s *PostgresSessionStore) Update(ctx context.Context, session *model.ClientSession) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "PostgresSessionStore.Update").
		Logger()
	log.Debug().Msg("starting postgres session update")

	if session == nil {
		err := model.ErrInvalidParameter
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("cannot update nil session")
		return err
	}

	encAccess, err := s.encrypt(session.AccessToken)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed encrypting access token")
		return fmt.Errorf("%w: failed encrypting access token: %v", model.ErrInternal, err)
	}

	encRefresh, err := s.encrypt(session.RefreshToken)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed encrypting refresh token")
		return fmt.Errorf("%w: failed encrypting refresh token: %v", model.ErrInternal, err)
	}

	encID, err := s.encrypt(session.IDToken)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed encrypting id token")
		return fmt.Errorf("%w: failed encrypting id token: %v", model.ErrInternal, err)
	}

	rolesJSON, err := json.Marshal(session.Roles)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed marshaling roles")
		return fmt.Errorf("%w: failed marshaling roles: %v", model.ErrInternal, err)
	}

	metaJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed marshaling metadata")
		return fmt.Errorf("%w: failed marshaling metadata: %v", model.ErrInternal, err)
	}

	query := `
		UPDATE bff_sessions
		SET user_id = $2,
		    keycloak_sid = $3,
		    access_token = $4,
		    refresh_token = $5,
		    id_token = $6,
		    roles = $7,
		    metadata = $8,
		    updated_at = $9,
		    expires_at = $10
		WHERE id = $1
	`

	cmdTag, err := s.db.Exec(ctx, query,
		string(session.ID),
		session.UserID,
		session.KeycloakSID,
		encAccess,
		encRefresh,
		encID,
		rolesJSON,
		metaJSON,
		session.UpdatedAt,
		session.ExpiresAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed updating session")
		return MapError(err)
	}

	if cmdTag.RowsAffected() == 0 {
		log.Info().
			Str("session_id", string(session.ID)).
			Dur("duration_ms", time.Since(start)).
			Msg("session to update not found")
		return model.ErrSessionNotFound
	}

	log.Info().
		Str("session_id", string(session.ID)).
		Dur("duration_ms", time.Since(start)).
		Msg("completed postgres session update")
	return nil
}

// Delete removes a session by ID from PostgreSQL.
func (s *PostgresSessionStore) Delete(ctx context.Context, id model.SessionID) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "PostgresSessionStore.Delete").
		Str("session_id", string(id)).
		Logger()
	log.Debug().Msg("starting postgres session deletion")

	query := `DELETE FROM bff_sessions WHERE id = $1`
	_, err := s.db.Exec(ctx, query, string(id))
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting session")
		return MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed postgres session deletion")
	return nil
}

// DeleteByKeycloakSID terminates all sessions with the given Keycloak session ID (for backchannel logout).
func (s *PostgresSessionStore) DeleteByKeycloakSID(ctx context.Context, sid string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "PostgresSessionStore.DeleteByKeycloakSID").
		Str("keycloak_sid", sid).
		Logger()
	log.Debug().Msg("starting deletion by keycloak SID")

	query := `DELETE FROM bff_sessions WHERE keycloak_sid = $1`
	cmdTag, err := s.db.Exec(ctx, query, sid)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting by keycloak SID")
		return MapError(err)
	}

	log.Info().
		Int64("deleted_count", cmdTag.RowsAffected()).
		Dur("duration_ms", time.Since(start)).
		Msg("completed deletion by keycloak SID")
	return nil
}

// DeleteByUserID terminates all sessions belonging to the specified user.
func (s *PostgresSessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "PostgresSessionStore.DeleteByUserID").
		Str("user_id", userID).
		Logger()
	log.Debug().Msg("starting deletion by user ID")

	query := `DELETE FROM bff_sessions WHERE user_id = $1`
	cmdTag, err := s.db.Exec(ctx, query, userID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting by user ID")
		return MapError(err)
	}

	log.Info().
		Int64("deleted_count", cmdTag.RowsAffected()).
		Dur("duration_ms", time.Since(start)).
		Msg("completed deletion by user ID")
	return nil
}

// DeleteExpired removes all sessions that expired before the specified cutoff time.
func (s *PostgresSessionStore) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "PostgresSessionStore.DeleteExpired").
		Time("cutoff", before).
		Logger()
	log.Debug().Msg("starting expired sessions deletion")

	query := `DELETE FROM bff_sessions WHERE expires_at < $1`
	cmdTag, err := s.db.Exec(ctx, query, before)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting expired sessions")
		return 0, MapError(err)
	}

	count := cmdTag.RowsAffected()
	log.Info().
		Int64("deleted_count", count).
		Dur("duration_ms", time.Since(start)).
		Msg("completed expired sessions deletion")
	return count, nil
}
