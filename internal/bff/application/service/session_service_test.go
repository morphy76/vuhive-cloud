package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/session/memory"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/service"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockSessionStore is a test double for outbound.SessionStore.
type MockSessionStore struct {
	mock.Mock
}

func (m *MockSessionStore) Create(ctx context.Context, s *model.ClientSession) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockSessionStore) Get(ctx context.Context, id model.SessionID) (*model.ClientSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ClientSession), args.Error(1)
}

func (m *MockSessionStore) Update(ctx context.Context, s *model.ClientSession) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockSessionStore) Delete(ctx context.Context, id model.SessionID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionStore) DeleteByKeycloakSID(ctx context.Context, sid string) error {
	args := m.Called(ctx, sid)
	return args.Error(0)
}

func (m *MockSessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionStore) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func TestSessionService_CreateSession(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully create session with full options", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store)

		cmd := inbound.CreateSessionCommand{
			SessionID:    "sess-123",
			UserID:       "user-456",
			KeycloakSID:  "kc-sid-789",
			AccessToken:  "access-tok",
			RefreshToken: "refresh-tok",
			IDToken:      "id-tok",
			Roles:        []string{"admin", "runner"},
			TTL:          2 * time.Hour,
			Metadata:     map[string]string{"ip": "127.0.0.1"},
		}

		session, err := svc.CreateSession(ctx, cmd)
		require.NoError(t, err)
		require.NotNil(t, session)

		assert.Equal(t, model.SessionID("sess-123"), session.ID)
		assert.Equal(t, "user-456", session.UserID)
		assert.Equal(t, "kc-sid-789", session.KeycloakSID)
		assert.Equal(t, "access-tok", session.AccessToken)
		assert.Equal(t, "refresh-tok", session.RefreshToken)
		assert.Equal(t, "id-tok", session.IDToken)
		assert.Equal(t, []string{"admin", "runner"}, session.Roles)
		assert.Equal(t, map[string]string{"ip": "127.0.0.1"}, session.Metadata)
		assert.True(t, session.ExpiresAt.After(time.Now().Add(1*time.Hour)))

		// Verify persisted in store
		stored, err := store.Get(ctx, "sess-123")
		require.NoError(t, err)
		assert.Equal(t, session.ID, stored.ID)
	})

	t.Run("default TTL applied when TTL is zero or negative", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store, service.SessionServiceConfig{
			SessionTTL: 12 * time.Hour,
		})

		cmd := inbound.CreateSessionCommand{
			SessionID: "sess-default-ttl",
			UserID:    "user-1",
			TTL:       0,
		}

		session, err := svc.CreateSession(ctx, cmd)
		require.NoError(t, err)
		assert.True(t, session.ExpiresAt.After(time.Now().Add(11*time.Hour)))
	})

	t.Run("invalid parameters return ErrInvalidParameter", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store)

		_, err := svc.CreateSession(ctx, inbound.CreateSessionCommand{
			SessionID: "",
			UserID:    "user-1",
		})
		assert.ErrorIs(t, err, model.ErrInvalidParameter)

		_, err = svc.CreateSession(ctx, inbound.CreateSessionCommand{
			SessionID: "sess-1",
			UserID:    "",
		})
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})
}

func TestSessionService_GetSession(t *testing.T) {
	ctx := context.Background()

	t.Run("empty session ID returns ErrInvalidParameter", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store)

		_, err := svc.GetSession(ctx, "")
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("non-existent session returns ErrSessionNotFound", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store)

		_, err := svc.GetSession(ctx, "non-existent")
		assert.ErrorIs(t, err, model.ErrSessionNotFound)
	})

	t.Run("expired session returns ErrSessionExpired", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store)

		sess, err := model.NewClientSession("sess-expired", "user-1", 10*time.Millisecond)
		require.NoError(t, err)
		time.Sleep(20 * time.Millisecond)
		require.NoError(t, store.Create(ctx, sess))

		_, err = svc.GetSession(ctx, "sess-expired")
		assert.ErrorIs(t, err, model.ErrSessionExpired)
	})

	t.Run("sliding expiration write-throttling: read within threshold does NOT update store", func(t *testing.T) {
		mockStore := new(MockSessionStore)
		slidingThreshold := 15 * time.Minute
		svc := service.NewSessionService(mockStore, service.SessionServiceConfig{
			SessionTTL:       24 * time.Hour,
			SlidingThreshold: slidingThreshold,
		})

		now := time.Now()
		// Session was updated 5 minutes ago (less than sliding threshold of 15m)
		sess, err := model.NewClientSession("sess-recent", "user-1", 24*time.Hour)
		require.NoError(t, err)
		sess.UpdatedAt = now.Add(-5 * time.Minute)
		sess.ExpiresAt = now.Add(23 * time.Hour + 55*time.Minute)

		mockStore.On("Get", mock.Anything, model.SessionID("sess-recent")).Return(sess, nil)
		// mockStore.Update must NOT be called

		retrieved, err := svc.GetSession(ctx, "sess-recent")
		require.NoError(t, err)
		assert.Equal(t, model.SessionID("sess-recent"), retrieved.ID)
		mockStore.AssertExpectations(t)
	})

	t.Run("sliding expiration write-throttling: read at or after threshold touches and updates store", func(t *testing.T) {
		mockStore := new(MockSessionStore)
		slidingThreshold := 15 * time.Minute
		svc := service.NewSessionService(mockStore, service.SessionServiceConfig{
			SessionTTL:       24 * time.Hour,
			SlidingThreshold: slidingThreshold,
		})

		now := time.Now()
		// Session was updated 20 minutes ago (greater than 15m threshold)
		sess, err := model.NewClientSession("sess-touch", "user-1", 24*time.Hour)
		require.NoError(t, err)
		sess.UpdatedAt = now.Add(-20 * time.Minute)
		sess.ExpiresAt = now.Add(23 * time.Hour + 40*time.Minute)

		mockStore.On("Get", mock.Anything, model.SessionID("sess-touch")).Return(sess, nil)
		mockStore.On("Update", mock.Anything, mock.MatchedBy(func(s *model.ClientSession) bool {
			return s.ID == "sess-touch" && s.UpdatedAt.After(now.Add(-1*time.Second))
		})).Return(nil)

		retrieved, err := svc.GetSession(ctx, "sess-touch")
		require.NoError(t, err)
		assert.Equal(t, model.SessionID("sess-touch"), retrieved.ID)
		mockStore.AssertExpectations(t)
	})
}

func TestSessionService_RotateSessionTokens(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully rotate tokens", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store)

		sess, err := svc.CreateSession(ctx, inbound.CreateSessionCommand{
			SessionID:    "sess-rotate",
			UserID:       "user-1",
			AccessToken:  "old-access",
			RefreshToken: "old-refresh",
			TTL:          1 * time.Hour,
		})
		require.NoError(t, err)

		cmd := inbound.RotateTokensCommand{
			SessionID:    sess.ID,
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			IDToken:      "new-id",
			TTL:          2 * time.Hour,
		}

		updated, err := svc.RotateSessionTokens(ctx, cmd)
		require.NoError(t, err)
		assert.Equal(t, "new-access", updated.AccessToken)
		assert.Equal(t, "new-refresh", updated.RefreshToken)
		assert.Equal(t, "new-id", updated.IDToken)

		// Verify store holds the rotated tokens
		stored, err := store.Get(ctx, sess.ID)
		require.NoError(t, err)
		assert.Equal(t, "new-access", stored.AccessToken)
		assert.Equal(t, "new-refresh", stored.RefreshToken)
	})

	t.Run("rotate tokens on expired session fails with ErrSessionExpired", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store)

		sess, err := model.NewClientSession("sess-expired-rot", "user-1", 10*time.Millisecond)
		require.NoError(t, err)
		time.Sleep(20 * time.Millisecond)
		require.NoError(t, store.Create(ctx, sess))

		_, err = svc.RotateSessionTokens(ctx, inbound.RotateTokensCommand{
			SessionID:    sess.ID,
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			TTL:          1 * time.Hour,
		})
		assert.ErrorIs(t, err, model.ErrSessionExpired)
	})

	t.Run("rotate tokens with invalid parameters", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store)

		_, err := svc.RotateSessionTokens(ctx, inbound.RotateTokensCommand{
			SessionID: "",
		})
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})
}

func TestSessionService_Revocation(t *testing.T) {
	ctx := context.Background()

	t.Run("revoke session by ID", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store)

		sess, err := svc.CreateSession(ctx, inbound.CreateSessionCommand{
			SessionID: "sess-revoke",
			UserID:    "user-1",
			TTL:       1 * time.Hour,
		})
		require.NoError(t, err)

		err = svc.RevokeSession(ctx, sess.ID)
		require.NoError(t, err)

		_, err = svc.GetSession(ctx, sess.ID)
		assert.ErrorIs(t, err, model.ErrSessionNotFound)
	})

	t.Run("revoke session with empty ID returns ErrInvalidParameter", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store)

		err := svc.RevokeSession(ctx, "")
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("revoke by Keycloak SID terminates all associated sessions", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store)

		_, err := svc.CreateSession(ctx, inbound.CreateSessionCommand{
			SessionID:   "sess-kc-1",
			UserID:      "user-1",
			KeycloakSID: "target-sid",
			TTL:         1 * time.Hour,
		})
		require.NoError(t, err)

		_, err = svc.CreateSession(ctx, inbound.CreateSessionCommand{
			SessionID:   "sess-kc-2",
			UserID:      "user-1",
			KeycloakSID: "target-sid",
			TTL:         1 * time.Hour,
		})
		require.NoError(t, err)

		_, err = svc.CreateSession(ctx, inbound.CreateSessionCommand{
			SessionID:   "sess-kc-3",
			UserID:      "user-2",
			KeycloakSID: "other-sid",
			TTL:         1 * time.Hour,
		})
		require.NoError(t, err)

		err = svc.RevokeByKeycloakSID(ctx, "target-sid")
		require.NoError(t, err)

		_, err = svc.GetSession(ctx, "sess-kc-1")
		assert.ErrorIs(t, err, model.ErrSessionNotFound)

		_, err = svc.GetSession(ctx, "sess-kc-2")
		assert.ErrorIs(t, err, model.ErrSessionNotFound)

		// Other session still exists
		other, err := svc.GetSession(ctx, "sess-kc-3")
		require.NoError(t, err)
		assert.Equal(t, model.SessionID("sess-kc-3"), other.ID)
	})

	t.Run("revoke by empty Keycloak SID returns ErrInvalidParameter", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		svc := service.NewSessionService(store)

		err := svc.RevokeByKeycloakSID(ctx, "")
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})
}

func TestSessionService_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemorySessionStore()
	svc := service.NewSessionService(store, service.SessionServiceConfig{
		SessionTTL:       1 * time.Hour,
		SlidingThreshold: 1 * time.Millisecond,
	})

	sess, err := svc.CreateSession(ctx, inbound.CreateSessionCommand{
		SessionID:    "sess-concurrent",
		UserID:       "user-conc",
		AccessToken:  "tok-0",
		RefreshToken: "ref-0",
		TTL:          1 * time.Hour,
	})
	require.NoError(t, err)

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = svc.GetSession(ctx, sess.ID)
		}()
		go func(idx int) {
			defer wg.Done()
			_, _ = svc.RotateTokens(ctx, sess.ID, "tok", "ref", "", 1*time.Hour)
		}(i)
	}

	wg.Wait()

	finalSess, err := svc.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, finalSess.ID)
}
