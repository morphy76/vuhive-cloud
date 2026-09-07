package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/session/memory"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/service"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSessionCleaner_CleanOnce(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully deletes expired sessions from memory store", func(t *testing.T) {
		store := memory.NewMemorySessionStore()
		cleaner := service.NewSessionCleaner(store, service.SessionCleanerConfig{
			Interval: 10 * time.Minute,
		})

		// Create 2 expired sessions and 1 active session
		expired1, err := model.NewClientSession("sess-exp-1", "user-1", 10*time.Millisecond)
		require.NoError(t, err)
		expired2, err := model.NewClientSession("sess-exp-2", "user-2", 10*time.Millisecond)
		require.NoError(t, err)
		active, err := model.NewClientSession("sess-act-1", "user-3", 1*time.Hour)
		require.NoError(t, err)

		require.NoError(t, store.Create(ctx, expired1))
		require.NoError(t, store.Create(ctx, expired2))
		require.NoError(t, store.Create(ctx, active))

		time.Sleep(20 * time.Millisecond)

		deleted, err := cleaner.CleanOnce(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), deleted)

		// Assert expired sessions deleted
		_, err = store.Get(ctx, "sess-exp-1")
		assert.ErrorIs(t, err, model.ErrSessionNotFound)
		_, err = store.Get(ctx, "sess-exp-2")
		assert.ErrorIs(t, err, model.ErrSessionNotFound)

		// Assert active session remains
		act, err := store.Get(ctx, "sess-act-1")
		require.NoError(t, err)
		assert.Equal(t, model.SessionID("sess-act-1"), act.ID)
	})

	t.Run("store failure propagates error", func(t *testing.T) {
		mockStore := new(MockSessionStore)
		mockStore.On("DeleteExpired", mock.Anything, mock.Anything).Return(int64(0), errors.New("db failure"))

		cleaner := service.NewSessionCleaner(mockStore, service.SessionCleanerConfig{})
		_, err := cleaner.CleanOnce(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db failure")
		mockStore.AssertExpectations(t)
	})
}

func TestSessionCleaner_Start_GracefulShutdown(t *testing.T) {
	mockStore := new(MockSessionStore)
	mockStore.On("DeleteExpired", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()

	cleaner := service.NewSessionCleaner(mockStore, service.SessionCleanerConfig{
		Interval: 10 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- cleaner.Start(ctx)
	}()

	// Allow cleaner ticker to tick at least once
	time.Sleep(30 * time.Millisecond)

	// Trigger shutdown
	cancel()

	select {
	case err := <-errCh:
		assert.True(t, errors.Is(err, context.Canceled), "expected context.Canceled error on shutdown, got: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("cleaner failed to stop within 2 seconds; goroutine leaked")
	}
}
