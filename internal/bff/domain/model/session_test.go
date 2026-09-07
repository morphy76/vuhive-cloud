package model_test

import (
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientSession(t *testing.T) {
	t.Run("valid session creation default", func(t *testing.T) {
		session, err := model.NewClientSession("sess-123", "user-456", 1*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, model.SessionID("sess-123"), session.ID)
		assert.Equal(t, "user-456", session.UserID)
		assert.False(t, session.IsExpired())
		assert.WithinDuration(t, time.Now().Add(1*time.Hour), session.ExpiresAt, 2*time.Second)
		assert.WithinDuration(t, time.Now(), session.CreatedAt, 2*time.Second)
		assert.WithinDuration(t, time.Now(), session.UpdatedAt, 2*time.Second)
		assert.NotNil(t, session.Metadata)
	})

	t.Run("valid session creation with functional options", func(t *testing.T) {
		meta := map[string]string{"user_agent": "Mozilla/5.0", "client_ip": "192.168.1.1"}
		roles := []string{"admin", "operator"}
		session, err := model.NewClientSession("sess-opt", "user-789", 30*time.Minute,
			model.WithKeycloakSID("kc-sid-abc"),
			model.WithTokens("acc-token-1", "ref-token-1", "id-token-1"),
			model.WithRoles(roles),
			model.WithMetadata(meta),
		)
		require.NoError(t, err)
		assert.Equal(t, model.SessionID("sess-opt"), session.ID)
		assert.Equal(t, "user-789", session.UserID)
		assert.Equal(t, "kc-sid-abc", session.KeycloakSID)
		assert.Equal(t, "acc-token-1", session.AccessToken)
		assert.Equal(t, "ref-token-1", session.RefreshToken)
		assert.Equal(t, "id-token-1", session.IDToken)
		assert.Equal(t, roles, session.Roles)
		assert.Equal(t, meta, session.Metadata)
	})

	t.Run("empty session ID", func(t *testing.T) {
		_, err := model.NewClientSession("", "user-456", 1*time.Hour)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("empty user ID", func(t *testing.T) {
		_, err := model.NewClientSession("sess-123", "", 1*time.Hour)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("zero or negative TTL", func(t *testing.T) {
		_, err := model.NewClientSession("sess-123", "user-456", 0)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)

		_, err = model.NewClientSession("sess-123", "user-456", -10*time.Second)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("session expiration check", func(t *testing.T) {
		session, err := model.NewClientSession("sess-expired", "user-1", 10*time.Millisecond)
		require.NoError(t, err)
		time.Sleep(15 * time.Millisecond)
		assert.True(t, session.IsExpired())
	})
}

func TestClientSession_RotateTokens(t *testing.T) {
	t.Run("successful token rotation", func(t *testing.T) {
		session, err := model.NewClientSession("sess-1", "user-1", 10*time.Minute,
			model.WithTokens("old-access", "old-refresh", "old-id"),
		)
		require.NoError(t, err)

		time.Sleep(5 * time.Millisecond)
		err = session.RotateTokens("new-access", "new-refresh", "new-id", 30*time.Minute)
		require.NoError(t, err)

		assert.Equal(t, "new-access", session.AccessToken)
		assert.Equal(t, "new-refresh", session.RefreshToken)
		assert.Equal(t, "new-id", session.IDToken)
		assert.WithinDuration(t, time.Now().Add(30*time.Minute), session.ExpiresAt, 2*time.Second)
		assert.True(t, session.UpdatedAt.After(session.CreatedAt))
	})

	t.Run("empty access token returns error", func(t *testing.T) {
		session, err := model.NewClientSession("sess-1", "user-1", 10*time.Minute)
		require.NoError(t, err)

		err = session.RotateTokens("", "new-refresh", "new-id", 30*time.Minute)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("empty refresh token returns error", func(t *testing.T) {
		session, err := model.NewClientSession("sess-1", "user-1", 10*time.Minute)
		require.NoError(t, err)

		err = session.RotateTokens("new-access", "", "new-id", 30*time.Minute)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("non-positive ttl returns error", func(t *testing.T) {
		session, err := model.NewClientSession("sess-1", "user-1", 10*time.Minute)
		require.NoError(t, err)

		err = session.RotateTokens("new-access", "new-refresh", "new-id", 0)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)

		err = session.RotateTokens("new-access", "new-refresh", "new-id", -5*time.Minute)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("rotating on expired session returns ErrSessionExpired", func(t *testing.T) {
		session, err := model.NewClientSession("sess-1", "user-1", 10*time.Millisecond)
		require.NoError(t, err)
		time.Sleep(15 * time.Millisecond)

		err = session.RotateTokens("new-access", "new-refresh", "new-id", 30*time.Minute)
		assert.ErrorIs(t, err, model.ErrSessionExpired)
	})
}

func TestClientSession_Touch(t *testing.T) {
	t.Run("successful touch extends expiration", func(t *testing.T) {
		session, err := model.NewClientSession("sess-1", "user-1", 5*time.Minute)
		require.NoError(t, err)

		time.Sleep(5 * time.Millisecond)
		err = session.Touch(1 * time.Hour)
		require.NoError(t, err)

		assert.WithinDuration(t, time.Now().Add(1*time.Hour), session.ExpiresAt, 2*time.Second)
		assert.True(t, session.UpdatedAt.After(session.CreatedAt))
	})

	t.Run("touch with non-positive ttl returns error", func(t *testing.T) {
		session, err := model.NewClientSession("sess-1", "user-1", 5*time.Minute)
		require.NoError(t, err)

		err = session.Touch(0)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("touch on expired session returns ErrSessionExpired", func(t *testing.T) {
		session, err := model.NewClientSession("sess-1", "user-1", 10*time.Millisecond)
		require.NoError(t, err)
		time.Sleep(15 * time.Millisecond)

		err = session.Touch(1 * time.Hour)
		assert.ErrorIs(t, err, model.ErrSessionExpired)
	})
}

func TestClientSession_Revoke(t *testing.T) {
	t.Run("revoke immediately marks session as expired", func(t *testing.T) {
		session, err := model.NewClientSession("sess-1", "user-1", 1*time.Hour)
		require.NoError(t, err)
		assert.False(t, session.IsExpired())

		session.Revoke()
		assert.True(t, session.IsExpired())
	})
}
