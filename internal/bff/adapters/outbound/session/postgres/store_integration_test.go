//go:build integration

package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/postgres"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/session/crypto"
	bffpg "github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/session/postgres"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupIntegrationPostgres(t *testing.T) (*sql.DB, *pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("vuhive_bff_test"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "failed to start postgres testcontainer")

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	sqlDB, err := sql.Open("pgx", connStr)
	require.NoError(t, err, "failed to open sql.DB")

	// Apply migrations
	err = postgres.MigrateUp(ctx, sqlDB)
	require.NoError(t, err, "failed to run database migrations")

	poolConfig, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err, "failed to parse pgxpool config")

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err, "failed to create pgxpool")

	cleanup := func() {
		pool.Close()
		_ = sqlDB.Close()
		_ = pgContainer.Terminate(ctx)
	}

	return sqlDB, pool, cleanup
}

func TestPostgresSessionStore_Integration_FullLifecycle(t *testing.T) {
	_, pool, cleanup := setupIntegrationPostgres(t)
	defer cleanup()

	ctx := context.Background()
	cipher, err := crypto.NewTokenCipherFromPassphrase("test-encryption-key-for-integration")
	require.NoError(t, err)

	store := bffpg.NewPostgresSessionStore(pool, bffpg.WithCipher(cipher))

	// 1. Create Session
	session, err := model.NewClientSession("sess-integ-1", "user-integ-1", 1*time.Hour,
		model.WithKeycloakSID("kc-sid-integ-1"),
		model.WithTokens("access-jwt-1", "refresh-jwt-1", "id-jwt-1"),
		model.WithRoles([]string{"admin", "viewer"}),
		model.WithMetadata(map[string]string{"user_agent": "integration-test"}),
	)
	require.NoError(t, err)

	err = store.Create(ctx, session)
	require.NoError(t, err)

	// 2. Get Session and verify decrypted tokens
	retrieved, err := store.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, "user-integ-1", retrieved.UserID)
	assert.Equal(t, "kc-sid-integ-1", retrieved.KeycloakSID)
	assert.Equal(t, "access-jwt-1", retrieved.AccessToken)
	assert.Equal(t, "refresh-jwt-1", retrieved.RefreshToken)
	assert.Equal(t, "id-jwt-1", retrieved.IDToken)
	assert.Equal(t, []string{"admin", "viewer"}, retrieved.Roles)
	assert.Equal(t, "integration-test", retrieved.Metadata["user_agent"])

	// 3. Update Session with rotated tokens
	err = retrieved.RotateTokens("access-jwt-2", "refresh-jwt-2", "id-jwt-2", 2*time.Hour)
	require.NoError(t, err)

	err = store.Update(ctx, retrieved)
	require.NoError(t, err)

	updated, err := store.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "access-jwt-2", updated.AccessToken)
	assert.Equal(t, "refresh-jwt-2", updated.RefreshToken)

	// 4. Create second session with same KeycloakSID
	sess2, err := model.NewClientSession("sess-integ-2", "user-integ-1", 1*time.Hour,
		model.WithKeycloakSID("kc-sid-integ-1"),
	)
	require.NoError(t, err)
	require.NoError(t, store.Create(ctx, sess2))

	// 5. Backchannel logout by KeycloakSID deletes both sessions
	err = store.DeleteByKeycloakSID(ctx, "kc-sid-integ-1")
	require.NoError(t, err)

	_, err = store.Get(ctx, session.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
	_, err = store.Get(ctx, sess2.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
}
