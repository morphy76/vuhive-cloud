package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// MockSecretsUseCase mocks inbound.SecretsUseCase
type MockSecretsUseCase struct {
	mock.Mock
}

var _ inbound.SecretsUseCase = (*MockSecretsUseCase)(nil)

func (m *MockSecretsUseCase) CreateSecret(ctx context.Context, cmd inbound.CreateSecretCommand) (*model.Secret, error) {
	args := m.Called(ctx, cmd)
	if s := args.Get(0); s != nil {
		return s.(*model.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSecretsUseCase) ListSecrets(ctx context.Context, suiteID string) ([]*model.Secret, error) {
	args := m.Called(ctx, suiteID)
	if s := args.Get(0); s != nil {
		return s.([]*model.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSecretsUseCase) UpdateSecret(ctx context.Context, cmd inbound.UpdateSecretCommand) (*model.Secret, error) {
	args := m.Called(ctx, cmd)
	if s := args.Get(0); s != nil {
		return s.(*model.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSecretsUseCase) DeleteSecret(ctx context.Context, suiteID, secretID string) error {
	return m.Called(ctx, suiteID, secretID).Error(0)
}

func (m *MockSecretsUseCase) GetDecryptedSecrets(ctx context.Context, suiteID string, keys []string) (map[string]string, error) {
	args := m.Called(ctx, suiteID, keys)
	if s := args.Get(0); s != nil {
		return s.(map[string]string), args.Error(1)
	}
	return nil, args.Error(1)
}

func setupSecretTestRouter(secretsUC inbound.SecretsUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := rest.NewSecretHandler(secretsUC)

	v1 := router.Group("/api/v1")
	{
		suites := v1.Group("/suites")
		{
			suites.POST("/:id/secrets", handler.CreateSecret)
			suites.GET("/:id/secrets", handler.ListSecrets)
			suites.PUT("/:id/secrets/:secretId", handler.UpdateSecret)
			suites.DELETE("/:id/secrets/:secretId", handler.DeleteSecret)
		}
	}
	return router
}

func TestSecretHandler_CreateSecret(t *testing.T) {
	t.Run("successfully creates secret and returns 201", func(t *testing.T) {
		mockUC := new(MockSecretsUseCase)
		router := setupSecretTestRouter(mockUC)

		suiteID := "suite-123"
		sec, err := model.NewSecret(suiteID, "API_KEY", []byte("encrypted-payload"))
		require.NoError(t, err)

		mockUC.On("CreateSecret", mock.Anything, inbound.CreateSecretCommand{
			SuiteID: suiteID,
			Key:     "API_KEY",
			Value:   "secret-val",
		}).Return(sec, nil)

		payload := rest.CreateSecretRequest{
			Key:   "API_KEY",
			Value: "secret-val",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/suites/"+suiteID+"/secrets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp rest.SecretResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "API_KEY", resp.Key)
		assert.Equal(t, suiteID, resp.SuiteID)
		mockUC.AssertExpectations(t)
	})

	t.Run("returns 400 when body binding fails", func(t *testing.T) {
		mockUC := new(MockSecretsUseCase)
		router := setupSecretTestRouter(mockUC)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/suites/s1/secrets", bytes.NewBufferString("invalid-json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSecretHandler_ListSecrets(t *testing.T) {
	t.Run("returns 200 with secrets list", func(t *testing.T) {
		mockUC := new(MockSecretsUseCase)
		router := setupSecretTestRouter(mockUC)

		suiteID := "suite-123"
		sec, err := model.NewSecret(suiteID, "API_KEY", []byte("enc"))
		require.NoError(t, err)

		mockUC.On("ListSecrets", mock.Anything, suiteID).Return([]*model.Secret{sec}, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/"+suiteID+"/secrets", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.SecretListResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Count)
		assert.Len(t, resp.Secrets, 1)
		assert.Equal(t, "API_KEY", resp.Secrets[0].Key)
		mockUC.AssertExpectations(t)
	})
}

func TestSecretHandler_UpdateSecret(t *testing.T) {
	t.Run("successfully updates secret and returns 200", func(t *testing.T) {
		mockUC := new(MockSecretsUseCase)
		router := setupSecretTestRouter(mockUC)

		suiteID := "suite-123"
		sec, err := model.NewSecret(suiteID, "API_KEY", []byte("updated-enc"))
		require.NoError(t, err)

		mockUC.On("UpdateSecret", mock.Anything, inbound.UpdateSecretCommand{
			SuiteID:  suiteID,
			SecretID: sec.ID(),
			Value:    "new-secret-val",
		}).Return(sec, nil)

		payload := rest.UpdateSecretRequest{
			Value: "new-secret-val",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/suites/"+suiteID+"/secrets/"+sec.ID(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.SecretResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "API_KEY", resp.Key)
		mockUC.AssertExpectations(t)
	})
}

func TestSecretHandler_DeleteSecret(t *testing.T) {
	t.Run("successfully deletes secret and returns 204", func(t *testing.T) {
		mockUC := new(MockSecretsUseCase)
		router := setupSecretTestRouter(mockUC)

		suiteID := "suite-123"
		secretID := "sec-456"
		mockUC.On("DeleteSecret", mock.Anything, suiteID, secretID).Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/suites/"+suiteID+"/secrets/"+secretID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		mockUC.AssertExpectations(t)
	})
}

func TestSecretHandler_DisabledAndNilHandling(t *testing.T) {
	t.Run("nil usecase returns 501 Not Implemented", func(t *testing.T) {
		router := setupSecretTestRouter(nil)

		// POST
		req := httptest.NewRequest(http.MethodPost, "/api/v1/suites/s1/secrets", bytes.NewBufferString(`{"key":"A","value":"b"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "SECRETS_ENCRYPTION_KEY")

		// GET
		req = httptest.NewRequest(http.MethodGet, "/api/v1/suites/s1/secrets", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "SECRETS_ENCRYPTION_KEY")

		// PUT
		req = httptest.NewRequest(http.MethodPut, "/api/v1/suites/s1/secrets/sec1", bytes.NewBufferString(`{"value":"b"}`))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "SECRETS_ENCRYPTION_KEY")

		// DELETE
		req = httptest.NewRequest(http.MethodDelete, "/api/v1/suites/s1/secrets/sec1", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "SECRETS_ENCRYPTION_KEY")
	})

	t.Run("nil receiver returns 501 Not Implemented without panic", func(t *testing.T) {
		var nilHandler *rest.SecretHandler
		router := gin.New()
		router.POST("/test/secrets", nilHandler.CreateSecret)
		router.GET("/test/secrets", nilHandler.ListSecrets)
		router.PUT("/test/secrets/:id", nilHandler.UpdateSecret)
		router.DELETE("/test/secrets/:id", nilHandler.DeleteSecret)

		req := httptest.NewRequest(http.MethodPost, "/test/secrets", bytes.NewBufferString(`{"key":"A","value":"b"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "SECRETS_ENCRYPTION_KEY")

		req = httptest.NewRequest(http.MethodGet, "/test/secrets", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotImplemented, w.Code)

		req = httptest.NewRequest(http.MethodPut, "/test/secrets/1", bytes.NewBufferString(`{"value":"b"}`))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotImplemented, w.Code)

		req = httptest.NewRequest(http.MethodDelete, "/test/secrets/1", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotImplemented, w.Code)
	})
}

func TestRouter_SecretsDisabledRouting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suitesUC := new(MockSuitesUseCase)

	router := rest.SetupRouterWithConfig(rest.RouterConfig{
		SuitesUC:  suitesUC,
		SecretsUC: nil, // explicitly unconfigured
	})

	for _, method := range []string{http.MethodPost, http.MethodGet} {
		req := httptest.NewRequest(method, "/api/v1/suites/s1/secrets", bytes.NewBufferString(`{"key":"A","value":"b"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotImplemented, w.Code, "expected 501 Not Implemented for method %s", method)
		assert.Contains(t, w.Body.String(), "SECRETS_ENCRYPTION_KEY")
	}

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/api/v1/suites/s1/secrets/sec1", bytes.NewBufferString(`{"value":"b"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotImplemented, w.Code, "expected 501 Not Implemented for method %s", method)
		assert.Contains(t, w.Body.String(), "SECRETS_ENCRYPTION_KEY")
	}
}
