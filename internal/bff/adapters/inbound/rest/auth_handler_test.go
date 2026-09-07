package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSessionSvc := new(MockSessionService)
	mockOIDC := new(MockOIDCClient)

	mockOIDC.On("GeneratePKCE").Return(&outbound.PKCEPair{
		Verifier:  "pkce-verifier-123",
		Challenge: "pkce-challenge-456",
		Method:    "S256",
	}, nil)

	mockOIDC.On("BuildAuthorizationURL", mock.Anything, mock.Anything, "http://localhost:8081/api/v1/bff/auth/callback", mock.Anything, mock.Anything).
		Return("https://auth.example.com/realms/vuhive/protocol/openid-connect/auth?client_id=vuhive-cloud-bff", nil)

	authHandler := rest.NewAuthHandler(rest.AuthHandlerConfig{
		SessionService: mockSessionSvc,
		OIDCClient:     mockOIDC,
		RedirectURI:    "http://localhost:8081/api/v1/bff/auth/callback",
		CookieName:     "vuhive_session",
	})

	router := gin.New()
	router.GET("/api/v1/bff/auth/login", authHandler.Login)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/bff/auth/login", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "https://auth.example.com")

	// Verify temporary auth cookies were set
	cookies := rec.Result().Cookies()
	var hasState, hasVerifier bool
	for _, c := range cookies {
		if c.Name == "vuhive_auth_state" && c.Value != "" {
			hasState = true
		}
		if c.Name == "vuhive_auth_verifier" && c.Value == "pkce-verifier-123" {
			hasVerifier = true
		}
	}
	assert.True(t, hasState, "expected vuhive_auth_state cookie")
	assert.True(t, hasVerifier, "expected vuhive_auth_verifier cookie")
	mockOIDC.AssertExpectations(t)
}

func TestAuthHandler_Callback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful callback creates session and sets cookie", func(t *testing.T) {
		mockSessionSvc := new(MockSessionService)
		mockOIDC := new(MockOIDCClient)

		// Dummy tokens
		idToken := makeDummyJWT(time.Now().Add(1 * time.Hour))
		mockOIDC.On("ExchangeCode", mock.Anything, "valid-auth-code", "saved-verifier", "http://localhost:8081/api/v1/bff/auth/callback").
			Return(&outbound.TokenResponse{
				AccessToken:  "acc-tok-1",
				RefreshToken: "ref-tok-1",
				IDToken:      idToken,
				TokenType:    "Bearer",
				ExpiresIn:    3600,
			}, nil)

		mockSessionSvc.On("CreateSession", mock.Anything, mock.MatchedBy(func(cmd inbound.CreateSessionCommand) bool {
			return cmd.AccessToken == "acc-tok-1" && cmd.RefreshToken == "ref-tok-1"
		})).Return(&model.ClientSession{
			ID:          "sess-created-1",
			UserID:      "test-user",
			AccessToken: "acc-tok-1",
		}, nil)

		authHandler := rest.NewAuthHandler(rest.AuthHandlerConfig{
			SessionService: mockSessionSvc,
			OIDCClient:     mockOIDC,
			RedirectURI:    "http://localhost:8081/api/v1/bff/auth/callback",
			CookieName:     "vuhive_session",
		})

		router := gin.New()
		router.GET("/api/v1/bff/auth/callback", authHandler.Callback)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/bff/auth/callback?code=valid-auth-code&state=expected-state", nil)
		req.AddCookie(&http.Cookie{Name: "vuhive_auth_state", Value: "expected-state"})
		req.AddCookie(&http.Cookie{Name: "vuhive_auth_verifier", Value: "saved-verifier"})

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/", rec.Header().Get("Location"))

		// Check session cookie set
		cookies := rec.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "vuhive_session" {
				sessionCookie = c
			}
		}
		require.NotNil(t, sessionCookie)
		assert.Equal(t, "sess-created-1", sessionCookie.Value)
		assert.True(t, sessionCookie.HttpOnly)

		mockOIDC.AssertExpectations(t)
		mockSessionSvc.AssertExpectations(t)
	})

	t.Run("state mismatch returns 400 Bad Request", func(t *testing.T) {
		mockSessionSvc := new(MockSessionService)
		mockOIDC := new(MockOIDCClient)

		authHandler := rest.NewAuthHandler(rest.AuthHandlerConfig{
			SessionService: mockSessionSvc,
			OIDCClient:     mockOIDC,
			CookieName:     "vuhive_session",
		})

		router := gin.New()
		router.GET("/api/v1/bff/auth/callback", authHandler.Callback)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/bff/auth/callback?code=auth-code&state=bad-state", nil)
		req.AddCookie(&http.Cookie{Name: "vuhive_auth_state", Value: "different-state"})

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSessionSvc := new(MockSessionService)
	mockOIDC := new(MockOIDCClient)

	sess, err := model.NewClientSession("sess-logout", "user-1", 1*time.Hour,
		model.WithTokens("acc", "ref-to-revoke", "id"),
	)
	require.NoError(t, err)

	mockSessionSvc.On("GetSession", mock.Anything, model.SessionID("sess-logout")).
		Return(sess, nil)

	mockOIDC.On("RevokeToken", mock.Anything, "ref-to-revoke", "refresh_token").
		Return(nil)

	mockSessionSvc.On("RevokeSession", mock.Anything, model.SessionID("sess-logout")).
		Return(nil)

	authHandler := rest.NewAuthHandler(rest.AuthHandlerConfig{
		SessionService: mockSessionSvc,
		OIDCClient:     mockOIDC,
		CookieName:     "vuhive_session",
	})

	router := gin.New()
	router.POST("/api/v1/bff/auth/logout", authHandler.Logout)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/bff/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "vuhive_session", Value: "sess-logout"})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Check cookie cleared
	cookies := rec.Result().Cookies()
	var cleared bool
	for _, c := range cookies {
		if c.Name == "vuhive_session" && c.MaxAge < 0 {
			cleared = true
		}
	}
	assert.True(t, cleared, "expected session cookie to be cleared")
	mockOIDC.AssertExpectations(t)
	mockSessionSvc.AssertExpectations(t)
}

func TestAuthHandler_Me(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSessionSvc := new(MockSessionService)
	mockOIDC := new(MockOIDCClient)

	sess, err := model.NewClientSession("sess-me", "user-123", 1*time.Hour,
		model.WithTokens("secret-access-tok", "secret-refresh-tok", "secret-id-tok"),
		model.WithRoles([]string{"vuhive-admin", "vuhive-developer"}),
		model.WithMetadata(map[string]string{"theme": "dark"}),
	)
	require.NoError(t, err)

	mockSessionSvc.On("GetSession", mock.Anything, model.SessionID("sess-me")).
		Return(sess, nil)

	authHandler := rest.NewAuthHandler(rest.AuthHandlerConfig{
		SessionService: mockSessionSvc,
		OIDCClient:     mockOIDC,
		CookieName:     "vuhive_session",
	})

	router := gin.New()
	router.GET("/api/v1/bff/auth/me",
		rest.SessionMiddleware(mockSessionSvc, mockOIDC, "vuhive_session"),
		authHandler.Me,
	)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/bff/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "vuhive_session", Value: "sess-me"})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "user-123", resp["user_id"])
	roles := resp["roles"].([]interface{})
	assert.Contains(t, roles, "vuhive-admin")
	assert.Contains(t, roles, "vuhive-developer")

	// Ensure raw tokens are NEVER present in response
	assert.Nil(t, resp["access_token"])
	assert.Nil(t, resp["refresh_token"])
	assert.Nil(t, resp["id_token"])
}

func TestAuthHandler_BackchannelLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid logout token terminates sessions matching keycloak sid", func(t *testing.T) {
		mockSessionSvc := new(MockSessionService)
		mockOIDC := new(MockOIDCClient)

		mockOIDC.On("VerifyLogoutToken", mock.Anything, "raw-signed-logout-token").
			Return(&outbound.LogoutTokenClaims{
				SessionID: "kc-sid-777",
				TokenID:   "jti-123",
			}, nil)

		mockSessionSvc.On("RevokeByKeycloakSID", mock.Anything, "kc-sid-777").
			Return(nil)

		authHandler := rest.NewAuthHandler(rest.AuthHandlerConfig{
			SessionService: mockSessionSvc,
			OIDCClient:     mockOIDC,
			CookieName:     "vuhive_session",
		})

		router := gin.New()
		router.POST("/api/v1/bff/auth/backchannel-logout", authHandler.BackchannelLogout)

		form := url.Values{}
		form.Set("logout_token", "raw-signed-logout-token")

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/bff/auth/backchannel-logout", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "no-cache, no-store", rec.Header().Get("Cache-Control"))
		mockOIDC.AssertExpectations(t)
		mockSessionSvc.AssertExpectations(t)
	})

	t.Run("missing logout token returns 400 Bad Request", func(t *testing.T) {
		mockSessionSvc := new(MockSessionService)
		mockOIDC := new(MockOIDCClient)

		authHandler := rest.NewAuthHandler(rest.AuthHandlerConfig{
			SessionService: mockSessionSvc,
			OIDCClient:     mockOIDC,
		})

		router := gin.New()
		router.POST("/api/v1/bff/auth/backchannel-logout", authHandler.BackchannelLogout)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/bff/auth/backchannel-logout", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid logout token returns 400 Bad Request", func(t *testing.T) {
		mockSessionSvc := new(MockSessionService)
		mockOIDC := new(MockOIDCClient)

		mockOIDC.On("VerifyLogoutToken", mock.Anything, "bad-token").
			Return(nil, model.ErrUnauthorized)

		authHandler := rest.NewAuthHandler(rest.AuthHandlerConfig{
			SessionService: mockSessionSvc,
			OIDCClient:     mockOIDC,
		})

		router := gin.New()
		router.POST("/api/v1/bff/auth/backchannel-logout", authHandler.BackchannelLogout)

		form := url.Values{}
		form.Set("logout_token", "bad-token")

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/bff/auth/backchannel-logout", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
