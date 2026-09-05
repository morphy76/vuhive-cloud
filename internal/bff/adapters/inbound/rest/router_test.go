package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
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
}
