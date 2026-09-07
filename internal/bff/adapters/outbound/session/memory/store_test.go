package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/session/memory"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemorySessionStore_InterfaceCompliance(t *testing.T) {
	var _ outbound.SessionStore = (*memory.MemorySessionStore)(nil)
}

func TestMemorySessionStore_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemorySessionStore()

	session, err := model.NewClientSession("sess-1", "user-100", 1*time.Hour,
		model.WithKeycloakSID("kc-sid-1"),
		model.WithTokens("access-1", "refresh-1", "id-1"),
		model.WithRoles([]string{"viewer"}),
		model.WithMetadata(map[string]string{"env": "test"}),
	)
	require.NoError(t, err)

	err = store.Create(ctx, session)
	require.NoError(t, err)

	// Retrieve session
	retrieved, err := store.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.KeycloakSID, retrieved.KeycloakSID)
	assert.Equal(t, session.AccessToken, retrieved.AccessToken)
	assert.Equal(t, session.RefreshToken, retrieved.RefreshToken)
	assert.Equal(t, session.IDToken, retrieved.IDToken)
	assert.Equal(t, session.Roles, retrieved.Roles)
	assert.Equal(t, session.Metadata, retrieved.Metadata)

	// Verify deep copy: mutating returned session does not affect stored session
	retrieved.AccessToken = "mutated-token"
	retrieved.Roles[0] = "mutated-role"
	retrieved.Metadata["env"] = "mutated-env"

	secondGet, err := store.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "access-1", secondGet.AccessToken)
	assert.Equal(t, "viewer", secondGet.Roles[0])
	assert.Equal(t, "test", secondGet.Metadata["env"])
}

func TestMemorySessionStore_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemorySessionStore()

	_, err := store.Get(ctx, model.SessionID("non-existent"))
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
}

func TestMemorySessionStore_Create_NilSession(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemorySessionStore()

	err := store.Create(ctx, nil)
	assert.ErrorIs(t, err, model.ErrInvalidParameter)
}

func TestMemorySessionStore_Update(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemorySessionStore()

	session, err := model.NewClientSession("sess-update", "user-1", 10*time.Minute,
		model.WithTokens("acc-old", "ref-old", "id-old"),
	)
	require.NoError(t, err)
	require.NoError(t, store.Create(ctx, session))

	// Rotate tokens on aggregate
	err = session.RotateTokens("acc-new", "ref-new", "id-new", 20*time.Minute)
	require.NoError(t, err)

	err = store.Update(ctx, session)
	require.NoError(t, err)

	updated, err := store.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "acc-new", updated.AccessToken)
	assert.Equal(t, "ref-new", updated.RefreshToken)

	// Update non-existent session
	nonExistent, err := model.NewClientSession("sess-ghost", "user-ghost", 10*time.Minute)
	require.NoError(t, err)
	err = store.Update(ctx, nonExistent)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
}

func TestMemorySessionStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemorySessionStore()

	session, err := model.NewClientSession("sess-del", "user-1", 10*time.Minute)
	require.NoError(t, err)
	require.NoError(t, store.Create(ctx, session))

	err = store.Delete(ctx, session.ID)
	require.NoError(t, err)

	_, err = store.Get(ctx, session.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)

	// Delete non-existent is idempotent / no error
	err = store.Delete(ctx, model.SessionID("non-existent"))
	assert.NoError(t, err)
}

func TestMemorySessionStore_DeleteByKeycloakSID(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemorySessionStore()

	s1, err := model.NewClientSession("s1", "u1", 1*time.Hour, model.WithKeycloakSID("kc-target"))
	require.NoError(t, err)
	s2, err := model.NewClientSession("s2", "u1", 1*time.Hour, model.WithKeycloakSID("kc-target"))
	require.NoError(t, err)
	s3, err := model.NewClientSession("s3", "u2", 1*time.Hour, model.WithKeycloakSID("kc-other"))
	require.NoError(t, err)

	require.NoError(t, store.Create(ctx, s1))
	require.NoError(t, store.Create(ctx, s2))
	require.NoError(t, store.Create(ctx, s3))

	err = store.DeleteByKeycloakSID(ctx, "kc-target")
	require.NoError(t, err)

	_, err = store.Get(ctx, s1.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
	_, err = store.Get(ctx, s2.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)

	// Other session remains intact
	remaining, err := store.Get(ctx, s3.ID)
	require.NoError(t, err)
	assert.Equal(t, model.SessionID("s3"), remaining.ID)
}

func TestMemorySessionStore_DeleteByUserID(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemorySessionStore()

	s1, err := model.NewClientSession("s1", "user-alpha", 1*time.Hour)
	require.NoError(t, err)
	s2, err := model.NewClientSession("s2", "user-alpha", 1*time.Hour)
	require.NoError(t, err)
	s3, err := model.NewClientSession("s3", "user-beta", 1*time.Hour)
	require.NoError(t, err)

	require.NoError(t, store.Create(ctx, s1))
	require.NoError(t, store.Create(ctx, s2))
	require.NoError(t, store.Create(ctx, s3))

	err = store.DeleteByUserID(ctx, "user-alpha")
	require.NoError(t, err)

	_, err = store.Get(ctx, s1.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
	_, err = store.Get(ctx, s2.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)

	remaining, err := store.Get(ctx, s3.ID)
	require.NoError(t, err)
	assert.Equal(t, model.SessionID("s3"), remaining.ID)
}

func TestMemorySessionStore_DeleteExpired(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemorySessionStore()

	// Expired session 1
	s1, err := model.NewClientSession("s1", "u1", 10*time.Millisecond)
	require.NoError(t, err)
	// Expired session 2
	s2, err := model.NewClientSession("s2", "u2", 10*time.Millisecond)
	require.NoError(t, err)
	// Active session
	s3, err := model.NewClientSession("s3", "u3", 1*time.Hour)
	require.NoError(t, err)

	require.NoError(t, store.Create(ctx, s1))
	require.NoError(t, store.Create(ctx, s2))
	require.NoError(t, store.Create(ctx, s3))

	time.Sleep(20 * time.Millisecond)

	deletedCount, err := store.DeleteExpired(ctx, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(2), deletedCount)

	_, err = store.Get(ctx, s1.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)
	_, err = store.Get(ctx, s2.ID)
	assert.ErrorIs(t, err, model.ErrSessionNotFound)

	remaining, err := store.Get(ctx, s3.ID)
	require.NoError(t, err)
	assert.Equal(t, model.SessionID("s3"), remaining.ID)
}
