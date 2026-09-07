package rest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTokenVerifier struct {
	mock.Mock
}

func (m *MockTokenVerifier) VerifyToken(ctx context.Context, tokenString string) (*model.Claims, error) {
	args := m.Called(ctx, tokenString)
	if claims, ok := args.Get(0).(*model.Claims); ok {
		return claims, args.Error(1)
	}
	return nil, args.Error(1)
}

var _ outbound.TokenVerifierPort = (*MockTokenVerifier)(nil)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing authorization header returns 401", func(t *testing.T) {
		mockVerifier := new(MockTokenVerifier)
		r := gin.New()
		r.Use(rest.AuthMiddleware(mockVerifier, false))
		r.GET("/protected", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "unauthorized")
	})

	t.Run("malformed authorization header returns 401", func(t *testing.T) {
		mockVerifier := new(MockTokenVerifier)
		r := gin.New()
		r.Use(rest.AuthMiddleware(mockVerifier, false))
		r.GET("/protected", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		mockVerifier := new(MockTokenVerifier)
		mockVerifier.On("VerifyToken", mock.Anything, "invalid-token").Return(nil, model.ErrInvalidToken)

		r := gin.New()
		r.Use(rest.AuthMiddleware(mockVerifier, false))
		r.GET("/protected", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("valid token sets claims and passes through", func(t *testing.T) {
		mockVerifier := new(MockTokenVerifier)
		claims := model.NewClaims("u-1", "alice", "alice@example.com", []string{model.RoleDeveloper}, nil, time.Now().Add(time.Hour))
		mockVerifier.On("VerifyToken", mock.Anything, "valid-token").Return(claims, nil)

		var capturedClaims *model.Claims
		r := gin.New()
		r.Use(rest.AuthMiddleware(mockVerifier, false))
		r.GET("/protected", func(c *gin.Context) {
			capturedClaims = rest.GetClaims(c)
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotNil(t, capturedClaims)
		assert.Equal(t, "alice", capturedClaims.Username())
	})

	t.Run("optional auth mode allows unauthenticated requests", func(t *testing.T) {
		mockVerifier := new(MockTokenVerifier)
		r := gin.New()
		r.Use(rest.AuthMiddleware(mockVerifier, true))
		r.GET("/optional", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/optional", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	setupRouteWithClaims := func(claims *model.Claims, roles ...string) *httptest.ResponseRecorder {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			if claims != nil {
				c.Set(rest.ClaimsContextKey, claims)
			}
			c.Next()
		})
		r.GET("/role-guarded", rest.RequireRole(roles...), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/role-guarded", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		w := setupRouteWithClaims(nil, model.RoleDeveloper)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("insufficient role returns 403", func(t *testing.T) {
		claims := model.NewClaims("u-1", "viewer", "v@example.com", []string{model.RoleViewer}, nil, time.Now().Add(time.Hour))
		w := setupRouteWithClaims(claims, model.RoleDeveloper)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "forbidden")
	})

	t.Run("matching direct role returns 200", func(t *testing.T) {
		claims := model.NewClaims("u-2", "dev", "d@example.com", []string{model.RoleDeveloper}, nil, time.Now().Add(time.Hour))
		w := setupRouteWithClaims(claims, model.RoleDeveloper)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("admin role satisfies developer requirement returns 200", func(t *testing.T) {
		claims := model.NewClaims("u-3", "admin", "a@example.com", []string{model.RoleAdmin}, nil, time.Now().Add(time.Hour))
		w := setupRouteWithClaims(claims, model.RoleDeveloper)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("group mapping satisfies role returns 200", func(t *testing.T) {
		claims := model.NewClaims("u-4", "devgrp", "dg@example.com", nil, []string{"/developers"}, time.Now().Add(time.Hour))
		w := setupRouteWithClaims(claims, model.RoleDeveloper)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
