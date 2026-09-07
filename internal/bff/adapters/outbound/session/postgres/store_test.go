package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/session/crypto"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/session/postgres"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresSessionStore_InterfaceCompliance(t *testing.T) {
	var _ outbound.SessionStore = (*postgres.PostgresSessionStore)(nil)
}

// mockDB implements postgres.DB interface for unit tests without external DB.
type mockDB struct {
	execFunc     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	queryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (m *mockDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, sql, args...)
	}
	return pgconn.NewCommandTag(""), nil
}

func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, sql, args...)
	}
	return &mockRow{err: pgx.ErrNoRows}
}

type mockRow struct {
	values []any
	err    error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, d := range dest {
		if i < len(r.values) {
			switch ptr := d.(type) {
			case *string:
				if v, ok := r.values[i].(string); ok {
					*ptr = v
				}
			case *[]byte:
				if v, ok := r.values[i].([]byte); ok {
					*ptr = v
				}
			case *time.Time:
				if v, ok := r.values[i].(time.Time); ok {
					*ptr = v
				}
			}
		}
	}
	return nil
}

func TestPostgresSessionStore_Create_NilSession(t *testing.T) {
	ctx := context.Background()
	store := postgres.NewPostgresSessionStore(&mockDB{})

	err := store.Create(ctx, nil)
	assert.ErrorIs(t, err, model.ErrInvalidParameter)
}

func TestPostgresSessionStore_Create_Success(t *testing.T) {
	ctx := context.Background()
	var capturedSQL string
	var capturedArgs []any

	db := &mockDB{
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			capturedSQL = sql
			capturedArgs = args
			return pgconn.NewCommandTag("INSERT 0 1"), nil
		},
	}

	store := postgres.NewPostgresSessionStore(db)

	session, err := model.NewClientSession("sess-1", "user-1", 1*time.Hour,
		model.WithKeycloakSID("kc-1"),
		model.WithTokens("acc-token", "ref-token", "id-token"),
		model.WithRoles([]string{"admin"}),
		model.WithMetadata(map[string]string{"k": "v"}),
	)
	require.NoError(t, err)

	err = store.Create(ctx, session)
	require.NoError(t, err)

	assert.Contains(t, capturedSQL, "INSERT INTO bff_sessions")
	require.Len(t, capturedArgs, 11)
	assert.Equal(t, "sess-1", capturedArgs[0])
	assert.Equal(t, "user-1", capturedArgs[1])
	assert.Equal(t, "kc-1", capturedArgs[2])
	assert.Equal(t, "acc-token", capturedArgs[3])
	assert.Equal(t, "ref-token", capturedArgs[4])
	assert.Equal(t, "id-token", capturedArgs[5])
}

func TestPostgresSessionStore_Create_WithCipherEncryption(t *testing.T) {
	ctx := context.Background()
	var capturedArgs []any

	db := &mockDB{
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			capturedArgs = args
			return pgconn.NewCommandTag("INSERT 0 1"), nil
		},
	}

	cipher, err := crypto.NewTokenCipherFromPassphrase("test-encryption-key-passphrase")
	require.NoError(t, err)

	store := postgres.NewPostgresSessionStore(db, postgres.WithCipher(cipher))

	session, err := model.NewClientSession("sess-enc", "user-1", 1*time.Hour,
		model.WithTokens("secret-access-token", "secret-refresh-token", "secret-id-token"),
	)
	require.NoError(t, err)

	err = store.Create(ctx, session)
	require.NoError(t, err)

	require.Len(t, capturedArgs, 11)
	// Stored tokens in args must NOT match plaintext
	assert.NotEqual(t, "secret-access-token", capturedArgs[3])
	assert.NotEqual(t, "secret-refresh-token", capturedArgs[4])
	assert.NotEqual(t, "secret-id-token", capturedArgs[5])

	// But decrypting them must yield the plaintext
	decAcc, err := cipher.Decrypt(capturedArgs[3].(string))
	require.NoError(t, err)
	assert.Equal(t, "secret-access-token", decAcc)

	decRef, err := cipher.Decrypt(capturedArgs[4].(string))
	require.NoError(t, err)
	assert.Equal(t, "secret-refresh-token", decRef)
}

func TestPostgresSessionStore_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	db := &mockDB{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{err: pgx.ErrNoRows}
		},
	}

	store := postgres.NewPostgresSessionStore(db)

	_, err := store.Get(ctx, model.SessionID("missing"))
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
}

func TestPostgresSessionStore_Get_Success(t *testing.T) {
	ctx := context.Background()
	cipher, err := crypto.NewTokenCipherFromPassphrase("test-encryption-key-passphrase")
	require.NoError(t, err)

	encAcc, _ := cipher.Encrypt("decrypted-acc")
	encRef, _ := cipher.Encrypt("decrypted-ref")
	encID, _ := cipher.Encrypt("decrypted-id")

	now := time.Now().UTC()
	expiry := now.Add(1 * time.Hour)

	db := &mockDB{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{
				values: []any{
					"sess-get-1",
					"user-get-1",
					"kc-get-1",
					encAcc,
					encRef,
					encID,
					[]byte(`["operator"]`),
					[]byte(`{"tenant":"acme"}`),
					now,
					now,
					expiry,
				},
			}
		},
	}

	store := postgres.NewPostgresSessionStore(db, postgres.WithCipher(cipher))

	sess, err := store.Get(ctx, model.SessionID("sess-get-1"))
	require.NoError(t, err)
	assert.Equal(t, model.SessionID("sess-get-1"), sess.ID)
	assert.Equal(t, "user-get-1", sess.UserID)
	assert.Equal(t, "kc-get-1", sess.KeycloakSID)
	assert.Equal(t, "decrypted-acc", sess.AccessToken)
	assert.Equal(t, "decrypted-ref", sess.RefreshToken)
	assert.Equal(t, "decrypted-id", sess.IDToken)
	assert.Equal(t, []string{"operator"}, sess.Roles)
	assert.Equal(t, "acme", sess.Metadata["tenant"])
}

func TestPostgresSessionStore_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("nil session", func(t *testing.T) {
		store := postgres.NewPostgresSessionStore(&mockDB{})
		err := store.Update(ctx, nil)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("not found returns ErrSessionNotFound", func(t *testing.T) {
		db := &mockDB{
			execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("UPDATE 0"), nil
			},
		}
		store := postgres.NewPostgresSessionStore(db)
		sess, err := model.NewClientSession("sess-1", "user-1", 1*time.Hour)
		require.NoError(t, err)

		err = store.Update(ctx, sess)
		assert.ErrorIs(t, err, model.ErrSessionNotFound)
	})

	t.Run("successful update", func(t *testing.T) {
		db := &mockDB{
			execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("UPDATE 1"), nil
			},
		}
		store := postgres.NewPostgresSessionStore(db)
		sess, err := model.NewClientSession("sess-1", "user-1", 1*time.Hour)
		require.NoError(t, err)

		err = store.Update(ctx, sess)
		assert.NoError(t, err)
	})
}

func TestPostgresSessionStore_Delete(t *testing.T) {
	ctx := context.Background()
	var capturedSQL string
	var capturedArgs []any

	db := &mockDB{
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			capturedSQL = sql
			capturedArgs = args
			return pgconn.NewCommandTag("DELETE 1"), nil
		},
	}

	store := postgres.NewPostgresSessionStore(db)
	err := store.Delete(ctx, model.SessionID("sess-to-del"))
	require.NoError(t, err)

	assert.Contains(t, capturedSQL, "DELETE FROM bff_sessions WHERE id = $1")
	assert.Equal(t, []any{"sess-to-del"}, capturedArgs)
}

func TestPostgresSessionStore_DeleteByKeycloakSID(t *testing.T) {
	ctx := context.Background()
	var capturedSQL string
	var capturedArgs []any

	db := &mockDB{
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			capturedSQL = sql
			capturedArgs = args
			return pgconn.NewCommandTag("DELETE 2"), nil
		},
	}

	store := postgres.NewPostgresSessionStore(db)
	err := store.DeleteByKeycloakSID(ctx, "kc-sid-target")
	require.NoError(t, err)

	assert.Contains(t, capturedSQL, "DELETE FROM bff_sessions WHERE keycloak_sid = $1")
	assert.Equal(t, []any{"kc-sid-target"}, capturedArgs)
}

func TestPostgresSessionStore_DeleteByUserID(t *testing.T) {
	ctx := context.Background()
	var capturedSQL string
	var capturedArgs []any

	db := &mockDB{
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			capturedSQL = sql
			capturedArgs = args
			return pgconn.NewCommandTag("DELETE 3"), nil
		},
	}

	store := postgres.NewPostgresSessionStore(db)
	err := store.DeleteByUserID(ctx, "user-target")
	require.NoError(t, err)

	assert.Contains(t, capturedSQL, "DELETE FROM bff_sessions WHERE user_id = $1")
	assert.Equal(t, []any{"user-target"}, capturedArgs)
}

func TestPostgresSessionStore_DeleteExpired(t *testing.T) {
	ctx := context.Background()
	var capturedSQL string
	var capturedArgs []any

	db := &mockDB{
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			capturedSQL = sql
			capturedArgs = args
			return pgconn.NewCommandTag("DELETE 5"), nil
		},
	}

	cutoff := time.Now()
	store := postgres.NewPostgresSessionStore(db)
	count, err := store.DeleteExpired(ctx, cutoff)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)

	assert.Contains(t, capturedSQL, "DELETE FROM bff_sessions WHERE expires_at < $1")
	assert.Equal(t, []any{cutoff}, capturedArgs)
}

func TestPostgresSessionStore_DatabaseError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("connection failed")
	db := &mockDB{
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, expectedErr
		},
	}

	store := postgres.NewPostgresSessionStore(db)
	err := store.Delete(ctx, model.SessionID("sess-1"))
	assert.ErrorIs(t, err, expectedErr)
}
