package rest_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// MockSessionService satisfies inbound.SessionService.
type MockSessionService struct {
	mock.Mock
}

func (m *MockSessionService) CreateSession(ctx context.Context, cmd inbound.CreateSessionCommand) (*model.ClientSession, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ClientSession), args.Error(1)
}

func (m *MockSessionService) GetSession(ctx context.Context, id model.SessionID) (*model.ClientSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ClientSession), args.Error(1)
}

func (m *MockSessionService) RotateSessionTokens(ctx context.Context, cmd inbound.RotateTokensCommand) (*model.ClientSession, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ClientSession), args.Error(1)
}

func (m *MockSessionService) RevokeSession(ctx context.Context, id model.SessionID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionService) RevokeByKeycloakSID(ctx context.Context, sid string) error {
	args := m.Called(ctx, sid)
	return args.Error(0)
}

// MockOIDCClient satisfies outbound.OIDCClient.
type MockOIDCClient struct {
	mock.Mock
}

func (m *MockOIDCClient) GeneratePKCE() (*outbound.PKCEPair, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.PKCEPair), args.Error(1)
}

func (m *MockOIDCClient) BuildAuthorizationURL(state, nonce, redirectURI string, pkce *outbound.PKCEPair, scopes ...string) (string, error) {
	args := m.Called(state, nonce, redirectURI, pkce, scopes)
	return args.String(0), args.Error(1)
}

func (m *MockOIDCClient) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*outbound.TokenResponse, error) {
	args := m.Called(ctx, code, codeVerifier, redirectURI)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.TokenResponse), args.Error(1)
}

func (m *MockOIDCClient) RefreshToken(ctx context.Context, refreshToken string) (*outbound.TokenResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.TokenResponse), args.Error(1)
}

func (m *MockOIDCClient) RevokeToken(ctx context.Context, token, tokenTypeHint string) error {
	args := m.Called(ctx, token, tokenTypeHint)
	return args.Error(0)
}

func (m *MockOIDCClient) VerifyLogoutToken(ctx context.Context, rawLogoutToken string) (*outbound.LogoutTokenClaims, error) {
	args := m.Called(ctx, rawLogoutToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.LogoutTokenClaims), args.Error(1)
}

func makeDummyJWT(exp time.Time) string {
	hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d,"sub":"test-user"}`, exp.Unix())))
	return hdr + "." + payload + ".sig"
}

func TestSessionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing cookie returns 401 Unauthorized", func(t *testing.T) {
		mockSessionSvc := new(MockSessionService)
		mockOIDC := new(MockOIDCClient)

		router := gin.New()
		router.Use(rest.SessionMiddleware(mockSessionSvc, mockOIDC, "vuhive_session"))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid or expired session in store returns 401 and clears cookie", func(t *testing.T) {
		mockSessionSvc := new(MockSessionService)
		mockOIDC := new(MockOIDCClient)

		mockSessionSvc.On("GetSession", mock.Anything, model.SessionID("sess-invalid")).
			Return(nil, model.ErrSessionNotFound)

		router := gin.New()
		router.Use(rest.SessionMiddleware(mockSessionSvc, mockOIDC, "vuhive_session"))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "vuhive_session", Value: "sess-invalid"})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		// Check cookie was cleared (MaxAge < 0)
		cookies := rec.Result().Cookies()
		var found bool
		for _, c := range cookies {
			if c.Name == "vuhive_session" && c.MaxAge < 0 {
				found = true
			}
		}
		assert.True(t, found, "expected vuhive_session cookie to be cleared")
	})

	t.Run("active session injects Authorization header and context attributes", func(t *testing.T) {
		mockSessionSvc := new(MockSessionService)
		mockOIDC := new(MockOIDCClient)

		validJWT := makeDummyJWT(time.Now().Add(10 * time.Minute))
		sess, err := model.NewClientSession("sess-active", "user-1", 1*time.Hour,
			model.WithTokens(validJWT, "refresh-1", "id-1"),
			model.WithRoles([]string{"admin"}),
		)
		require.NoError(t, err)

		mockSessionSvc.On("GetSession", mock.Anything, model.SessionID("sess-active")).
			Return(sess, nil)

		var capturedAuthHeader string
		var capturedUser string
		var capturedRoles []string

		router := gin.New()
		router.Use(rest.SessionMiddleware(mockSessionSvc, mockOIDC, "vuhive_session"))
		router.GET("/protected", func(c *gin.Context) {
			capturedAuthHeader = c.GetHeader("Authorization")
			if u, ok := c.Get("user_id"); ok {
				capturedUser = u.(string)
			}
			if r, ok := c.Get("roles"); ok {
				capturedRoles = r.([]string)
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "vuhive_session", Value: "sess-active"})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "Bearer "+validJWT, capturedAuthHeader)
		assert.Equal(t, "user-1", capturedUser)
		assert.Equal(t, []string{"admin"}, capturedRoles)
	})

	t.Run("expiring access token triggers transparent refresh via OIDCClient", func(t *testing.T) {
		mockSessionSvc := new(MockSessionService)
		mockOIDC := new(MockOIDCClient)

		// Access token expiring in 10 seconds (less than 60s threshold)
		expiringJWT := makeDummyJWT(time.Now().Add(10 * time.Second))
		newAccessJWT := makeDummyJWT(time.Now().Add(5 * time.Minute))

		sess, err := model.NewClientSession("sess-expiring", "user-1", 1*time.Hour,
			model.WithTokens(expiringJWT, "old-refresh", "old-id"),
			model.WithRoles([]string{"developer"}),
		)
		require.NoError(t, err)

		mockSessionSvc.On("GetSession", mock.Anything, model.SessionID("sess-expiring")).
			Return(sess, nil)

		mockOIDC.On("RefreshToken", mock.Anything, "old-refresh").
			Return(&outbound.TokenResponse{
				AccessToken:  newAccessJWT,
				RefreshToken: "new-refresh",
				ExpiresIn:    300,
			}, nil)

		mockSessionSvc.On("RotateSessionTokens", mock.Anything, mock.MatchedBy(func(cmd inbound.RotateTokensCommand) bool {
			return cmd.SessionID == "sess-expiring" && cmd.AccessToken == newAccessJWT && cmd.RefreshToken == "new-refresh"
		})).Return(&model.ClientSession{
			ID:          "sess-expiring",
			UserID:      "user-1",
			AccessToken: newAccessJWT,
			Roles:       []string{"developer"},
		}, nil)

		var capturedAuthHeader string

		router := gin.New()
		router.Use(rest.SessionMiddleware(mockSessionSvc, mockOIDC, "vuhive_session"))
		router.GET("/protected", func(c *gin.Context) {
			capturedAuthHeader = c.GetHeader("Authorization")
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "vuhive_session", Value: "sess-expiring"})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "Bearer "+newAccessJWT, capturedAuthHeader)
		mockOIDC.AssertExpectations(t)
		mockSessionSvc.AssertExpectations(t)
	})

	t.Run("failed token refresh clears cookie and aborts with 401", func(t *testing.T) {
		mockSessionSvc := new(MockSessionService)
		mockOIDC := new(MockOIDCClient)

		expiringJWT := makeDummyJWT(time.Now().Add(10 * time.Second))
		sess, err := model.NewClientSession("sess-fail-ref", "user-1", 1*time.Hour,
			model.WithTokens(expiringJWT, "bad-refresh", ""),
		)
		require.NoError(t, err)

		mockSessionSvc.On("GetSession", mock.Anything, model.SessionID("sess-fail-ref")).
			Return(sess, nil)

		mockOIDC.On("RefreshToken", mock.Anything, "bad-refresh").
			Return(nil, model.ErrSessionExpired)

		mockSessionSvc.On("RevokeSession", mock.Anything, model.SessionID("sess-fail-ref")).
			Return(nil)

		router := gin.New()
		router.Use(rest.SessionMiddleware(mockSessionSvc, mockOIDC, "vuhive_session"))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "vuhive_session", Value: "sess-fail-ref"})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		// Check cookie cleared
		cookies := rec.Result().Cookies()
		var found bool
		for _, c := range cookies {
			if c.Name == "vuhive_session" && c.MaxAge < 0 {
				found = true
			}
		}
		assert.True(t, found)
	})
}
