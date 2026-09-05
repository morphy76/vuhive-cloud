package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/service"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockControlPlaneClient is a test mock satisfying outbound.ControlPlaneClient.
type MockControlPlaneClient struct {
	mock.Mock
}

func (m *MockControlPlaneClient) CheckHealth(ctx context.Context) (*outbound.ControlPlaneHealth, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.ControlPlaneHealth), args.Error(1)
}

func (m *MockControlPlaneClient) GetVersion(ctx context.Context) (*outbound.ControlPlaneVersion, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.ControlPlaneVersion), args.Error(1)
}

// MockCache is a test mock satisfying outbound.CachePort.
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).([]byte), args.Bool(1), args.Error(2)
}

func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func TestBFFService_GetStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("successful status aggregation with healthy control plane", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)

		mockCP.On("CheckHealth", mock.Anything).Return(&outbound.ControlPlaneHealth{
			Status:    "UP",
			Timestamp: time.Now(),
		}, nil)
		mockCP.On("GetVersion", mock.Anything).Return(&outbound.ControlPlaneVersion{
			Version:   "0.0.1",
			Commit:    "abcdef",
			BuildTime: "2026-09-05T10:00:00Z",
		}, nil)

		svc := service.NewBFFService(mockCP, mockCache, "0.1.0")
		status, err := svc.GetStatus(ctx)

		require.NoError(t, err)
		assert.Equal(t, "UP", status.BFFStatus)
		assert.Equal(t, "0.1.0", status.BFFVersion)
		assert.Equal(t, "UP", status.ControlPlaneStatus)
		assert.Equal(t, "0.0.1", status.ControlPlaneVersion)
		mockCP.AssertExpectations(t)
	})

	t.Run("degraded status when control plane is unavailable", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)

		mockCP.On("CheckHealth", mock.Anything).Return(nil, errors.New("connection refused"))

		svc := service.NewBFFService(mockCP, mockCache, "0.1.0")
		status, err := svc.GetStatus(ctx)

		require.NoError(t, err)
		assert.Equal(t, "UP", status.BFFStatus)
		assert.Equal(t, "DOWN", status.ControlPlaneStatus)
		assert.Equal(t, "", status.ControlPlaneVersion)
		mockCP.AssertExpectations(t)
	})
}

func TestBFFService_SessionLifecycle(t *testing.T) {
	ctx := context.Background()

	t.Run("create session successfully", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)

		cmd := inbound.CreateSessionCommand{
			SessionID: "sess-abc",
			UserID:    "user-1",
			TTL:       30 * time.Minute,
			Metadata:  map[string]string{"theme": "dark"},
		}

		mockCache.On("Set", mock.Anything, "session:sess-abc", mock.Anything, 30*time.Minute).Return(nil)

		svc := service.NewBFFService(mockCP, mockCache, "0.1.0")
		session, err := svc.CreateSession(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, model.SessionID("sess-abc"), session.ID)
		assert.Equal(t, "user-1", session.UserID)
		mockCache.AssertExpectations(t)
	})

	t.Run("create session with invalid parameters", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)

		cmd := inbound.CreateSessionCommand{
			SessionID: "",
			UserID:    "user-1",
			TTL:       30 * time.Minute,
		}

		svc := service.NewBFFService(mockCP, mockCache, "0.1.0")
		_, err := svc.CreateSession(ctx, cmd)

		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("get existing active session", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)

		activeSession, err := model.NewClientSession("sess-existing", "user-2", 1*time.Hour)
		require.NoError(t, err)
		sessionBytes, err := json.Marshal(activeSession)
		require.NoError(t, err)

		mockCache.On("Get", mock.Anything, "session:sess-existing").Return(sessionBytes, true, nil)

		svc := service.NewBFFService(mockCP, mockCache, "0.1.0")
		session, err := svc.GetSession(ctx, "sess-existing")

		require.NoError(t, err)
		assert.Equal(t, model.SessionID("sess-existing"), session.ID)
		assert.Equal(t, "user-2", session.UserID)
		mockCache.AssertExpectations(t)
	})

	t.Run("get non-existent session returns ErrSessionNotFound", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)

		mockCache.On("Get", mock.Anything, "session:missing").Return(nil, false, nil)

		svc := service.NewBFFService(mockCP, mockCache, "0.1.0")
		_, err := svc.GetSession(ctx, "missing")

		assert.ErrorIs(t, err, model.ErrSessionNotFound)
		mockCache.AssertExpectations(t)
	})
}
