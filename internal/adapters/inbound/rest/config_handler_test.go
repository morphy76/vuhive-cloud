package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConfigsUseCase mocks inbound.ConfigsUseCase
type MockConfigsUseCase struct {
	mock.Mock
}

func (m *MockConfigsUseCase) CreateConfig(ctx context.Context, cmd inbound.CreateConfigCommand) (*model.Configuration, error) {
	args := m.Called(ctx, cmd)
	if c := args.Get(0); c != nil {
		return c.(*model.Configuration), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockConfigsUseCase) GetConfig(ctx context.Context, suiteID, configID string) (*model.Configuration, error) {
	args := m.Called(ctx, suiteID, configID)
	if c := args.Get(0); c != nil {
		return c.(*model.Configuration), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockConfigsUseCase) ListConfigs(ctx context.Context, suiteID string) ([]*model.Configuration, error) {
	args := m.Called(ctx, suiteID)
	if c := args.Get(0); c != nil {
		return c.([]*model.Configuration), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockConfigsUseCase) DeleteConfig(ctx context.Context, suiteID, configID string) error {
	return m.Called(ctx, suiteID, configID).Error(0)
}

func (m *MockConfigsUseCase) UpdateConfig(ctx context.Context, cmd inbound.UpdateConfigCommand) (*model.Configuration, error) {
	args := m.Called(ctx, cmd)
	if c := args.Get(0); c != nil {
		return c.(*model.Configuration), args.Error(1)
	}
	return nil, args.Error(1)
}

func setupConfigTestRouter(mockUC *MockConfigsUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := rest.NewConfigHandler(mockUC)

	v1 := router.Group("/api/v1/suites/:id/configs")
	{
		v1.POST("", handler.CreateConfig)
		v1.GET("", handler.ListConfigs)
		v1.GET("/:configId", handler.GetConfig)
		v1.PUT("/:configId", handler.UpdateConfig)
		v1.DELETE("/:configId", handler.DeleteConfig)
	}
	return router
}

func TestConfigHandler_CreateConfig(t *testing.T) {
	t.Run("creates configuration returning HTTP 201 Created", func(t *testing.T) {
		mockUC := new(MockConfigsUseCase)
		cfg, _ := model.NewConfiguration("s-1", "staging", "vus: 10", "suites/s-1/configs/c-1.yaml", true)
		mockUC.On("CreateConfig", mock.Anything, inbound.CreateConfigCommand{
			SuiteID:     "s-1",
			Name:        "staging",
			ContentYAML: "vus: 10",
			IsDefault:   true,
		}).Return(cfg, nil).Once()

		router := setupConfigTestRouter(mockUC)

		body := []byte(`{"name":"staging","content_yaml":"vus: 10","is_default":true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/suites/s-1/configs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp rest.ConfigResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, cfg.ID(), resp.ID)
		assert.Equal(t, "s-1", resp.SuiteID)
		assert.Equal(t, "staging", resp.Name)
		assert.True(t, resp.IsDefault)
		mockUC.AssertExpectations(t)
	})

	t.Run("returns HTTP 400 when name is missing", func(t *testing.T) {
		mockUC := new(MockConfigsUseCase)
		router := setupConfigTestRouter(mockUC)

		body := []byte(`{"content_yaml":"vus: 10"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/suites/s-1/configs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns HTTP 404 when parent suite does not exist", func(t *testing.T) {
		mockUC := new(MockConfigsUseCase)
		mockUC.On("CreateConfig", mock.Anything, mock.Anything).Return(nil, model.ErrNotFound).Once()

		router := setupConfigTestRouter(mockUC)

		body := []byte(`{"name":"staging","content_yaml":"vus: 10"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/suites/missing/configs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestConfigHandler_GetConfig(t *testing.T) {
	t.Run("returns HTTP 200 with configuration details", func(t *testing.T) {
		mockUC := new(MockConfigsUseCase)
		cfg, _ := model.NewConfiguration("s-1", "staging", "vus: 10", "key", false)
		mockUC.On("GetConfig", mock.Anything, "s-1", cfg.ID()).Return(cfg, nil).Once()

		router := setupConfigTestRouter(mockUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/s-1/configs/"+cfg.ID(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.ConfigResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, cfg.ID(), resp.ID)
		assert.Equal(t, "staging", resp.Name)
	})

	t.Run("returns HTTP 404 when configuration not found", func(t *testing.T) {
		mockUC := new(MockConfigsUseCase)
		mockUC.On("GetConfig", mock.Anything, "s-1", "missing-cfg").Return(nil, model.ErrNotFound).Once()

		router := setupConfigTestRouter(mockUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/s-1/configs/missing-cfg", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestConfigHandler_ListConfigs(t *testing.T) {
	t.Run("returns HTTP 200 with list of configurations", func(t *testing.T) {
		mockUC := new(MockConfigsUseCase)
		c1, _ := model.NewConfiguration("s-1", "cfg-1", "yaml1", "k1", true)
		c2, _ := model.NewConfiguration("s-1", "cfg-2", "yaml2", "k2", false)
		mockUC.On("ListConfigs", mock.Anything, "s-1").Return([]*model.Configuration{c1, c2}, nil).Once()

		router := setupConfigTestRouter(mockUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/s-1/configs", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.ConfigListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Count)
		assert.Len(t, resp.Configs, 2)
	})
}

func TestConfigHandler_DeleteConfig(t *testing.T) {
	t.Run("returns HTTP 204 on successful deletion", func(t *testing.T) {
		mockUC := new(MockConfigsUseCase)
		mockUC.On("DeleteConfig", mock.Anything, "s-1", "c-1").Return(nil).Once()

		router := setupConfigTestRouter(mockUC)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/suites/s-1/configs/c-1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns HTTP 404 when configuration to delete is not found", func(t *testing.T) {
		mockUC := new(MockConfigsUseCase)
		mockUC.On("DeleteConfig", mock.Anything, "s-1", "missing-c").Return(model.ErrNotFound).Once()

		router := setupConfigTestRouter(mockUC)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/suites/s-1/configs/missing-c", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestConfigHandler_UpdateConfig(t *testing.T) {
	t.Run("updates configuration returning HTTP 200 OK", func(t *testing.T) {
		mockUC := new(MockConfigsUseCase)
		cfg, _ := model.NewConfiguration("s-1", "staging-updated", "vus: 20", "suites/s-1/configs/c-1.yaml", true)
		isDefault := true
		mockUC.On("UpdateConfig", mock.Anything, inbound.UpdateConfigCommand{
			SuiteID:     "s-1",
			ConfigID:    "c-1",
			Name:        "staging-updated",
			ContentYAML: "vus: 20",
			IsDefault:   &isDefault,
		}).Return(cfg, nil).Once()

		router := setupConfigTestRouter(mockUC)

		body := []byte(`{"name":"staging-updated","content_yaml":"vus: 20","is_default":true}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/suites/s-1/configs/c-1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.ConfigResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, cfg.ID(), resp.ID)
		assert.Equal(t, "staging-updated", resp.Name)
		assert.Equal(t, "vus: 20", resp.ContentYAML)
		assert.True(t, resp.IsDefault)
		mockUC.AssertExpectations(t)
	})

	t.Run("returns HTTP 400 on invalid JSON body", func(t *testing.T) {
		mockUC := new(MockConfigsUseCase)
		router := setupConfigTestRouter(mockUC)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/suites/s-1/configs/c-1", bytes.NewReader([]byte(`{"name":""}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns HTTP 404 when configuration to update is not found", func(t *testing.T) {
		mockUC := new(MockConfigsUseCase)
		mockUC.On("UpdateConfig", mock.Anything, mock.Anything).Return(nil, model.ErrNotFound).Once()

		router := setupConfigTestRouter(mockUC)

		body := []byte(`{"name":"name","content_yaml":"yaml"}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/suites/s-1/configs/missing", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
