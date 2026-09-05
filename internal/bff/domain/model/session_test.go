package model_test

import (
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientSession(t *testing.T) {
	t.Run("valid session creation", func(t *testing.T) {
		session, err := model.NewClientSession("sess-123", "user-456", 1*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, model.SessionID("sess-123"), session.ID)
		assert.Equal(t, "user-456", session.UserID)
		assert.False(t, session.IsExpired())
		assert.WithinDuration(t, time.Now().Add(1*time.Hour), session.ExpiresAt, 2*time.Second)
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
