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

func (m *MockControlPlaneClient) GetActiveRunsCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockControlPlaneClient) ListRecentSuites(ctx context.Context, limit int) ([]outbound.SuiteSummary, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]outbound.SuiteSummary), args.Error(1)
}

func (m *MockControlPlaneClient) ListProfiles(ctx context.Context) ([]outbound.ProfileSummary, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]outbound.ProfileSummary), args.Error(1)
}

func (m *MockControlPlaneClient) GetRun(ctx context.Context, id string) (*outbound.RunDetail, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.RunDetail), args.Error(1)
}

func (m *MockControlPlaneClient) GetRunReportURL(ctx context.Context, id string) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}

func (m *MockControlPlaneClient) GetRunLogsURL(ctx context.Context, id string) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}

func (m *MockControlPlaneClient) ListRuns(ctx context.Context, status string, limit int) ([]outbound.RunDetail, error) {
	args := m.Called(ctx, status, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]outbound.RunDetail), args.Error(1)
}

func (m *MockControlPlaneClient) ListArtifacts(ctx context.Context, suiteID string) ([]outbound.ArtifactDetail, error) {
	args := m.Called(ctx, suiteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]outbound.ArtifactDetail), args.Error(1)
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

func TestBFFService_GetDashboard(t *testing.T) {
	ctx := context.Background()

	t.Run("concurrent retrieval aggregates all dashboard metrics", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)

		mockCP.On("CheckHealth", mock.Anything).Return(&outbound.ControlPlaneHealth{
			Status:    "UP",
			Timestamp: time.Now(),
		}, nil)
		mockCP.On("GetVersion", mock.Anything).Return(&outbound.ControlPlaneVersion{
			Version: "1.0.0",
		}, nil)
		mockCP.On("GetActiveRunsCount", mock.Anything).Return(int64(7), nil)
		mockCP.On("ListRecentSuites", mock.Anything, 5).Return([]outbound.SuiteSummary{
			{ID: "suite-1", Name: "Stress Test", State: "ACTIVE"},
		}, nil)
		mockCP.On("ListProfiles", mock.Anything).Return([]outbound.ProfileSummary{
			{ID: "prof-1", Name: "Default Profile"},
			{ID: "prof-2", Name: "High Memory"},
		}, nil)

		svc := service.NewBFFService(mockCP, mockCache, "0.2.0")

		start := time.Now()
		dashboard, err := svc.GetDashboard(ctx)
		duration := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, dashboard)
		assert.Equal(t, "UP", dashboard.BFFStatus)
		assert.Equal(t, "0.2.0", dashboard.BFFVersion)
		assert.Equal(t, "UP", dashboard.ControlPlaneStatus)
		assert.Equal(t, "1.0.0", dashboard.ControlPlaneVersion)
		assert.Equal(t, int64(7), dashboard.ActiveRunsCount)
		assert.Len(t, dashboard.RecentSuites, 1)
		assert.Equal(t, 2, dashboard.ProfilesCount)
		assert.Len(t, dashboard.ProfilesSummary, 2)
		assert.Less(t, duration, 50*time.Millisecond, "dashboard aggregation must respond under 50ms")
		mockCP.AssertExpectations(t)
	})

	t.Run("partial degradation when some control plane queries fail", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)

		mockCP.On("CheckHealth", mock.Anything).Return(&outbound.ControlPlaneHealth{
			Status: "UP",
		}, nil)
		mockCP.On("GetVersion", mock.Anything).Return(nil, errors.New("version timeout"))
		mockCP.On("GetActiveRunsCount", mock.Anything).Return(int64(0), errors.New("runs unreachable"))
		mockCP.On("ListRecentSuites", mock.Anything, 5).Return(nil, errors.New("suites unreachable"))
		mockCP.On("ListProfiles", mock.Anything).Return(nil, errors.New("profiles unreachable"))

		svc := service.NewBFFService(mockCP, mockCache, "0.2.0")

		dashboard, err := svc.GetDashboard(ctx)
		require.NoError(t, err)
		assert.Equal(t, "UP", dashboard.BFFStatus)
		assert.Equal(t, "UP", dashboard.ControlPlaneStatus)
		assert.Equal(t, int64(0), dashboard.ActiveRunsCount)
		assert.Empty(t, dashboard.RecentSuites)
		assert.Equal(t, 0, dashboard.ProfilesCount)
	})
}

func TestBFFService_GetRunDetail(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully retrieves run with artifact links", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)

		runDetail := &outbound.RunDetail{
			ID:          "run-42",
			SuiteID:     "suite-1",
			ArtifactID:  "art-1",
			Status:      "COMPLETED",
			S3ReportKey: "runs/run-42/summary.json",
			S3LogsKey:   "runs/run-42/run.log",
			Metrics: outbound.RunMetrics{
				TotalRequests: 10000,
				AvgTPS:        500.0,
			},
		}

		mockCP.On("GetRun", mock.Anything, "run-42").Return(runDetail, nil)
		mockCP.On("GetRunReportURL", mock.Anything, "run-42").Return("https://s3/summary.json?token=xyz", nil)
		mockCP.On("GetRunLogsURL", mock.Anything, "run-42").Return("https://s3/run.log?token=xyz", nil)

		svc := service.NewBFFService(mockCP, mockCache, "0.2.0")
		detail, err := svc.GetRunDetail(ctx, "run-42")

		require.NoError(t, err)
		require.NotNil(t, detail)
		assert.Equal(t, "run-42", detail.ID)
		assert.Equal(t, "COMPLETED", detail.Status)
		assert.Equal(t, 500.0, detail.Metrics.AvgTPS)
		assert.Equal(t, "https://s3/summary.json?token=xyz", detail.ArtifactLinks.ReportURL)
		assert.Equal(t, "https://s3/run.log?token=xyz", detail.ArtifactLinks.LogsURL)
		mockCP.AssertExpectations(t)
	})

	t.Run("run not found maps to ErrRunNotFound", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)

		mockCP.On("GetRun", mock.Anything, "missing-run").Return(nil, model.ErrRunNotFound)

		svc := service.NewBFFService(mockCP, mockCache, "0.2.0")
		_, err := svc.GetRunDetail(ctx, "missing-run")

		assert.ErrorIs(t, err, model.ErrRunNotFound)
	})

	t.Run("empty run id returns ErrInvalidParameter", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)

		svc := service.NewBFFService(mockCP, mockCache, "0.2.0")
		_, err := svc.GetRunDetail(ctx, "   ")

		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})
}

// MockEventStreamHub satisfies outbound.EventStreamHub for testing.
type MockEventStreamHub struct {
	mock.Mock
}

func (m *MockEventStreamHub) Broadcast(ctx context.Context, event model.ServerSentEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventStreamHub) Subscribe(ctx context.Context) (<-chan model.ServerSentEvent, func(), error) {
	args := m.Called(ctx)
	ch := args.Get(0)
	var outCh <-chan model.ServerSentEvent
	if ch != nil {
		if c, ok := ch.(<-chan model.ServerSentEvent); ok {
			outCh = c
		} else if c, ok := ch.(chan model.ServerSentEvent); ok {
			outCh = c
		}
	}
	return outCh, args.Get(1).(func()), args.Error(2)
}

func (m *MockEventStreamHub) ClientCount() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockEventStreamHub) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestBFFService_Events(t *testing.T) {
	ctx := context.Background()

	t.Run("SubscribeEvents delegates to event hub", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)
		mockHub := new(MockEventStreamHub)

		ch := make(chan model.ServerSentEvent)
		unsub := func() {}

		mockHub.On("Subscribe", ctx).Return(ch, unsub, nil)

		svc := service.NewBFFService(mockCP, mockCache, "0.2.0", mockHub)
		subCh, subUnsub, err := svc.SubscribeEvents(ctx)
		require.NoError(t, err)
		assert.NotNil(t, subCh)
		assert.NotNil(t, subUnsub)
		mockHub.AssertExpectations(t)
	})

	t.Run("SubscribeEvents without hub returns error", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)

		svc := service.NewBFFService(mockCP, mockCache, "0.2.0")
		_, _, err := svc.SubscribeEvents(ctx)
		assert.Error(t, err)
	})

	t.Run("BroadcastEvent delegates to event hub", func(t *testing.T) {
		mockCP := new(MockControlPlaneClient)
		mockCache := new(MockCache)
		mockHub := new(MockEventStreamHub)

		ev, _ := model.NewServerSentEvent("ev-1", model.EventSystemHeartbeat, []byte(`{}`), time.Now())
		mockHub.On("Broadcast", ctx, ev).Return(nil)

		svc := service.NewBFFService(mockCP, mockCache, "0.2.0", mockHub)
		err := svc.BroadcastEvent(ctx, ev)
		require.NoError(t, err)
		mockHub.AssertExpectations(t)
	})
}
