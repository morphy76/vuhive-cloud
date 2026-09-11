package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockBFFService struct {
	mock.Mock
}

func (m *MockBFFService) GetStatus(ctx context.Context) (*inbound.SystemStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inbound.SystemStatus), args.Error(1)
}

func (m *MockBFFService) CreateSession(ctx context.Context, cmd inbound.CreateSessionCommand) (*model.ClientSession, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ClientSession), args.Error(1)
}

func (m *MockBFFService) GetSession(ctx context.Context, id model.SessionID) (*model.ClientSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ClientSession), args.Error(1)
}

func (m *MockBFFService) GetDashboard(ctx context.Context) (*inbound.DashboardOverview, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inbound.DashboardOverview), args.Error(1)
}

func (m *MockBFFService) GetRunDetail(ctx context.Context, id string) (*inbound.RunDetailComposite, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inbound.RunDetailComposite), args.Error(1)
}

func (m *MockBFFService) SubscribeEvents(ctx context.Context) (<-chan model.ServerSentEvent, func(), error) {
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

func (m *MockBFFService) BroadcastEvent(ctx context.Context, event model.ServerSentEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestRouter_Endpoints(t *testing.T) {
	mockSvc := new(MockBFFService)
	router := rest.SetupRouter(mockSvc, "0.1.0")

	t.Run("GET /healthz returns 200 OK", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "ok", resp["status"])
	})

	t.Run("GET /version returns 200 with version info", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/version", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "0.1.0", resp["version"])
	})

	t.Run("GET /api/v1/bff/status returns aggregated status", func(t *testing.T) {
		mockSvc.On("GetStatus", mock.Anything).Return(&inbound.SystemStatus{
			BFFStatus:           "UP",
			BFFVersion:          "0.1.0",
			ControlPlaneStatus:  "UP",
			ControlPlaneVersion: "0.0.1",
			Timestamp:           time.Now().UTC(),
		}, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/bff/status", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp rest.StatusResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "UP", resp.BFFStatus)
		assert.Equal(t, "UP", resp.ControlPlaneStatus)
		mockSvc.AssertExpectations(t)
	})

	t.Run("POST /api/v1/bff/sessions creates session", func(t *testing.T) {
		reqBody := rest.CreateSessionRequest{
			SessionID:  "sess-99",
			UserID:     "user-admin",
			TTLSeconds: 3600,
			Metadata:   map[string]string{"env": "test"},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		sess, err := model.NewClientSession("sess-99", "user-admin", 1*time.Hour)
		require.NoError(t, err)

		mockSvc.On("CreateSession", mock.Anything, mock.MatchedBy(func(cmd inbound.CreateSessionCommand) bool {
			return cmd.SessionID == "sess-99" && cmd.UserID == "user-admin"
		})).Return(sess, nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/bff/sessions", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var resp rest.SessionResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "sess-99", resp.ID)
		assert.Equal(t, "user-admin", resp.UserID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("GET /api/v1/bff/sessions/:id returns session", func(t *testing.T) {
		sess, err := model.NewClientSession("sess-99", "user-admin", 1*time.Hour)
		require.NoError(t, err)

		mockSvc.On("GetSession", mock.Anything, model.SessionID("sess-99")).Return(sess, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/bff/sessions/sess-99", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp rest.SessionResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "sess-99", resp.ID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("GET /api/v1/bff/sessions/:id returns 404 for missing session", func(t *testing.T) {
		mockSvc.On("GetSession", mock.Anything, model.SessionID("missing")).Return(nil, model.ErrSessionNotFound).Once()

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/bff/sessions/missing", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("GET / returns embedded index.html by default", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, rec.Body.String(), "vuhive-cloud Web Dashboard")
	})

	t.Run("GET /suites/42/runs returns embedded index.html fallback", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/suites/42/runs", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, rec.Body.String(), "vuhive-cloud Web Dashboard")
	})

	t.Run("GET /api/bff/v1/dashboard returns 200 with aggregated dashboard", func(t *testing.T) {
		mockSvc.On("GetDashboard", mock.Anything).Return(&inbound.DashboardOverview{
			BFFStatus:           "UP",
			BFFVersion:          "0.1.0",
			ControlPlaneStatus:  "UP",
			ControlPlaneVersion: "0.0.1",
			ActiveRunsCount:     3,
			RecentSuites: []outbound.SuiteSummary{
				{ID: "suite-1", Name: "Perf Suite", State: "ACTIVE"},
			},
			ProfilesCount: 1,
			ProfilesSummary: []outbound.ProfileSummary{
				{ID: "prof-1", Name: "Small Runner"},
			},
			Timestamp: time.Now().UTC(),
		}, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/api/bff/v1/dashboard", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp rest.DashboardResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "UP", resp.BFFStatus)
		assert.Equal(t, int64(3), resp.ActiveRunsCount)
		assert.Len(t, resp.RecentSuites, 1)
		assert.Equal(t, "Perf Suite", resp.RecentSuites[0].Name)
		mockSvc.AssertExpectations(t)
	})

	t.Run("GET /api/bff/v1/runs/:id returns 200 with composite run detail", func(t *testing.T) {
		mockSvc.On("GetRunDetail", mock.Anything, "run-888").Return(&inbound.RunDetailComposite{
			RunDetail: outbound.RunDetail{
				ID:          "run-888",
				SuiteID:     "suite-1",
				ArtifactID:  "art-1",
				Status:      "COMPLETED",
				S3ReportKey: "runs/run-888/summary.json",
				S3LogsKey:   "runs/run-888/run.log",
				Metrics: outbound.RunMetrics{
					TotalIterations: 500,
					AvgTPS:          120.4,
				},
			},
			ArtifactLinks: inbound.ArtifactLinks{
				ReportURL: "https://s3/report.json",
				LogsURL:   "https://s3/run.log",
			},
		}, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/api/bff/v1/runs/run-888", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp rest.RunDetailResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "run-888", resp.ID)
		assert.Equal(t, "COMPLETED", resp.Status)
		assert.Equal(t, 120.4, resp.Metrics.AvgTPS)
		assert.Equal(t, "https://s3/report.json", resp.ArtifactLinks.ReportURL)
		assert.Equal(t, "https://s3/run.log", resp.ArtifactLinks.LogsURL)
		mockSvc.AssertExpectations(t)
	})

	t.Run("GET /api/bff/v1/runs/:id returns 404 when run not found", func(t *testing.T) {
		mockSvc.On("GetRunDetail", mock.Anything, "non-existent").Return(nil, model.ErrRunNotFound).Once()

		req, _ := http.NewRequest(http.MethodGet, "/api/bff/v1/runs/non-existent", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		mockSvc.AssertExpectations(t)
	})
}

type proxyRoundTripFunc func(req *http.Request) (*http.Response, error)

func (f proxyRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRouter_TransparentProxy(t *testing.T) {
	mockTransport := proxyRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/api/v1/suites":
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(`{"suites":[{"id":"suite-proxy","name":"Proxy Suite"}]}`)),
				Request:    req,
			}, nil
		case "/api/v1/profiles":
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(`{"profiles":[{"id":"prof-proxy","name":"Proxy Profile"}]}`)),
				Request:    req,
			}, nil
		case "/api/v1/schedules":
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(`{"schedules":[{"id":"sched-proxy","name":"Proxy Schedule"}]}`)),
				Request:    req,
			}, nil
		case "/api/v1/runs":
			if req.Method == http.MethodPost {
				return &http.Response{
					StatusCode: http.StatusCreated,
					Status:     "201 Created",
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(bytes.NewBufferString(`{"id":"run-spawned","status":"QUEUED"}`)),
					Request:    req,
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(`{"runs":[],"total":0}`)),
				Request:    req,
			}, nil
		case "/api/v1/runs/run-spawned/logs":
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Body:       io.NopCloser(bytes.NewBufferString("[INFO] container started\n")),
				Request:    req,
			}, nil
		case "/api/v1/runs/run-spawned/report":
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(`{"suite_name":"Ecommerce Checkout Suite","status":"PASS"}`)),
				Request:    req,
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBufferString(`{"error":"not found"}`)),
				Request:    req,
			}, nil
		}
	})

	mockSvc := new(MockBFFService)
	router := rest.SetupRouterWithProxy(mockSvc, "0.1.0", "http://controlplane", mockTransport)

	t.Run("proxies GET /api/bff/v1/suites to /api/v1/suites", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/bff/v1/suites", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "suite-proxy")
	})

	t.Run("proxies GET /api/bff/v1/profiles to /api/v1/profiles", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/bff/v1/profiles", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "prof-proxy")
	})

	t.Run("proxies GET /api/bff/v1/schedules to /api/v1/schedules", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/bff/v1/schedules", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "sched-proxy")
	})

	t.Run("proxies POST /api/bff/v1/runs to /api/v1/runs", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/bff/v1/runs", bytes.NewBufferString(`{"suite_id":"suite-1"}`))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Body.String(), "run-spawned")
	})

	t.Run("proxies GET /api/bff/v1/runs/:id/logs to /api/v1/runs/:id/logs", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/bff/v1/runs/run-spawned/logs", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "[INFO] container started")
	})

	t.Run("proxies GET /api/bff/v1/runs/:id/report to /api/v1/runs/:id/report", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/bff/v1/runs/run-spawned/report", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ecommerce Checkout Suite")
	})
}

func TestRouter_Events(t *testing.T) {
	mockSvc := new(MockBFFService)
	router := rest.SetupRouter(mockSvc, "0.1.0")

	t.Run("GET /api/bff/v1/events establishes stream and sends events", func(t *testing.T) {
		eventsCh := make(chan model.ServerSentEvent, 2)
		unsubCalled := false
		unsub := func() {
			unsubCalled = true
		}

		mockSvc.On("SubscribeEvents", mock.Anything).Return(eventsCh, unsub, nil).Once()

		reqCtx, reqCancel := context.WithCancel(context.Background())
		defer reqCancel()

		req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, "/api/bff/v1/events", nil)
		rec := httptest.NewRecorder()

		// Send an event into channel before cancel
		now := time.Now().UTC()
		ev, _ := model.NewSystemHeartbeatEvent("hb-1", model.SystemHeartbeatPayload{
			Status:        "UP",
			ActiveClients: 1,
			ActiveRuns:    0,
			Timestamp:     now,
		})
		eventsCh <- ev

		go func() {
			time.Sleep(50 * time.Millisecond)
			reqCancel()
		}()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
		assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
		assert.Contains(t, rec.Body.String(), "id: hb-1\nevent: system_heartbeat\n")
		assert.True(t, unsubCalled, "unsubscribe should be called on disconnect")
		mockSvc.AssertExpectations(t)
	})

	t.Run("GET /api/v1/bff/events legacy alias works", func(t *testing.T) {
		eventsCh := make(chan model.ServerSentEvent, 1)
		unsub := func() {}

		mockSvc.On("SubscribeEvents", mock.Anything).Return(eventsCh, unsub, nil).Once()

		reqCtx, reqCancel := context.WithCancel(context.Background())
		req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, "/api/v1/bff/events", nil)
		rec := httptest.NewRecorder()

		go func() {
			time.Sleep(20 * time.Millisecond)
			reqCancel()
		}()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
		mockSvc.AssertExpectations(t)
	})
}

