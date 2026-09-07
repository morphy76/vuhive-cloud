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

// MockSuitesUseCase mocks inbound.SuitesUseCase
type MockSuitesUseCase struct {
	mock.Mock
}

func (m *MockSuitesUseCase) CreateSuite(ctx context.Context, name, description string) (*model.TestSuite, error) {
	args := m.Called(ctx, name, description)
	if s := args.Get(0); s != nil {
		return s.(*model.TestSuite), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSuitesUseCase) GetSuite(ctx context.Context, id string) (*model.TestSuite, error) {
	args := m.Called(ctx, id)
	if s := args.Get(0); s != nil {
		return s.(*model.TestSuite), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSuitesUseCase) ListSuites(ctx context.Context) ([]*model.TestSuite, error) {
	args := m.Called(ctx)
	if s := args.Get(0); s != nil {
		return s.([]*model.TestSuite), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSuitesUseCase) UpdateSuite(ctx context.Context, id string, cmd inbound.UpdateSuiteCommand) (*model.TestSuite, error) {
	args := m.Called(ctx, id, cmd)
	if s := args.Get(0); s != nil {
		return s.(*model.TestSuite), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSuitesUseCase) DeleteSuite(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockSuitesUseCase) ArchiveSuite(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func setupSuiteTestRouter(mockUC *MockSuitesUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := rest.NewSuiteHandler(mockUC)

	v1 := router.Group("/api/v1/suites")
	{
		v1.POST("", handler.CreateSuite)
		v1.GET("", handler.ListSuites)
		v1.GET("/:id", handler.GetSuite)
		v1.PUT("/:id", handler.UpdateSuite)
		v1.DELETE("/:id", handler.DeleteSuite)
	}
	return router
}

func TestSuiteHandler_CreateSuite(t *testing.T) {
	t.Run("creates suite returning HTTP 201 Created", func(t *testing.T) {
		mockUC := new(MockSuitesUseCase)
		suite, _ := model.NewTestSuite("checkout-test", "Checkout load scenarios")
		mockUC.On("CreateSuite", mock.Anything, "checkout-test", "Checkout load scenarios").Return(suite, nil).Once()

		router := setupSuiteTestRouter(mockUC)

		body := []byte(`{"name":"checkout-test","description":"Checkout load scenarios"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/suites", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp rest.SuiteResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, suite.ID(), resp.ID)
		assert.Equal(t, "checkout-test", resp.Name)
		assert.Equal(t, "Checkout load scenarios", resp.Description)
		assert.Equal(t, "DRAFT", resp.State)
		mockUC.AssertExpectations(t)
	})

	t.Run("returns HTTP 400 when name is missing", func(t *testing.T) {
		mockUC := new(MockSuitesUseCase)
		router := setupSuiteTestRouter(mockUC)

		body := []byte(`{"description":"Missing name"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/suites", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns HTTP 409 when suite name already exists", func(t *testing.T) {
		mockUC := new(MockSuitesUseCase)
		mockUC.On("CreateSuite", mock.Anything, "duplicate-suite", "").Return(nil, model.ErrConflict).Once()

		router := setupSuiteTestRouter(mockUC)

		body := []byte(`{"name":"duplicate-suite"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/suites", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		mockUC.AssertExpectations(t)
	})
}

func TestSuiteHandler_GetSuite(t *testing.T) {
	t.Run("returns HTTP 200 with suite details", func(t *testing.T) {
		mockUC := new(MockSuitesUseCase)
		suite, _ := model.NewTestSuite("suite-1", "desc-1")
		mockUC.On("GetSuite", mock.Anything, suite.ID()).Return(suite, nil).Once()

		router := setupSuiteTestRouter(mockUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/"+suite.ID(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.SuiteResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, suite.ID(), resp.ID)
		assert.Equal(t, "suite-1", resp.Name)
	})

	t.Run("returns HTTP 404 when suite not found", func(t *testing.T) {
		mockUC := new(MockSuitesUseCase)
		mockUC.On("GetSuite", mock.Anything, "unknown-id").Return(nil, model.ErrNotFound).Once()

		router := setupSuiteTestRouter(mockUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/unknown-id", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSuiteHandler_ListSuites(t *testing.T) {
	t.Run("returns HTTP 200 with list of suites", func(t *testing.T) {
		mockUC := new(MockSuitesUseCase)
		s1, _ := model.NewTestSuite("suite-1", "desc-1")
		s2, _ := model.NewTestSuite("suite-2", "desc-2")
		mockUC.On("ListSuites", mock.Anything).Return([]*model.TestSuite{s1, s2}, nil).Once()

		router := setupSuiteTestRouter(mockUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.SuiteListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Count)
		assert.Len(t, resp.Suites, 2)
	})
}

func TestSuiteHandler_UpdateSuite(t *testing.T) {
	t.Run("returns HTTP 200 with updated suite details", func(t *testing.T) {
		mockUC := new(MockSuitesUseCase)
		suite, _ := model.NewTestSuite("new-name", "new-desc")
		_ = suite.Activate()

		activeState := model.TestSuiteStateActive
		mockUC.On("UpdateSuite", mock.Anything, "s-1", inbound.UpdateSuiteCommand{
			Name:        "new-name",
			Description: "new-desc",
			State:       &activeState,
		}).Return(suite, nil).Once()

		router := setupSuiteTestRouter(mockUC)

		body := []byte(`{"name":"new-name","description":"new-desc","state":"ACTIVE"}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/suites/s-1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.SuiteResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "new-name", resp.Name)
		assert.Equal(t, "ACTIVE", resp.State)
	})

	t.Run("returns HTTP 404 when suite does not exist", func(t *testing.T) {
		mockUC := new(MockSuitesUseCase)
		mockUC.On("UpdateSuite", mock.Anything, "missing", mock.Anything).Return(nil, model.ErrNotFound).Once()

		router := setupSuiteTestRouter(mockUC)

		body := []byte(`{"name":"new-name"}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/suites/missing", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSuiteHandler_DeleteSuite(t *testing.T) {
	t.Run("returns HTTP 204 on successful deletion", func(t *testing.T) {
		mockUC := new(MockSuitesUseCase)
		mockUC.On("DeleteSuite", mock.Anything, "suite-1").Return(nil).Once()

		router := setupSuiteTestRouter(mockUC)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/suites/suite-1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns HTTP 404 when suite to delete is not found", func(t *testing.T) {
		mockUC := new(MockSuitesUseCase)
		mockUC.On("DeleteSuite", mock.Anything, "missing-suite").Return(model.ErrNotFound).Once()

		router := setupSuiteTestRouter(mockUC)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/suites/missing-suite", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
