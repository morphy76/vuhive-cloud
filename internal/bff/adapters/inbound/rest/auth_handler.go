package rest

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/rs/zerolog"
)

// AuthHandlerConfig defines configuration for the BFF authentication handler.
type AuthHandlerConfig struct {
	SessionService inbound.SessionService
	OIDCClient     outbound.OIDCClient
	CookieName     string
	CookiePath     string
	CookieDomain   string
	CookieSecure   bool
	RedirectURI    string
	PostLoginURL   string
	SessionTTL     time.Duration
}

// AuthHandler handles authentication flows: login redirect with PKCE, code callback, logout, user profile, and backchannel logout.
type AuthHandler struct {
	cfg AuthHandlerConfig
}

// NewAuthHandler constructs an initialized AuthHandler.
func NewAuthHandler(cfg AuthHandlerConfig) *AuthHandler {
	if cfg.CookieName == "" {
		cfg.CookieName = "vuhive_session"
	}
	if cfg.CookiePath == "" {
		cfg.CookiePath = "/"
	}
	if cfg.PostLoginURL == "" {
		cfg.PostLoginURL = "/"
	}
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 24 * time.Hour
	}

	return &AuthHandler{cfg: cfg}
}

type tokenClaimsExtraction struct {
	Sub               string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	Sid               string `json:"sid"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`
}

func parseJWTClaims(rawToken string) *tokenClaimsExtraction {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return nil
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}

	var claims tokenClaimsExtraction
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil
	}
	return &claims
}

func generateRandomHex(n int) string {
	bytes := make([]byte, n)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Login initiates the OIDC authorization code flow with PKCE, setting temporary state cookies and redirecting to Keycloak.
func (h *AuthHandler) Login(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	log := zerolog.Ctx(ctx).With().Str("op", "AuthHandler.Login").Logger()
	log.Debug().Msg("initiating login flow")

	if h.cfg.OIDCClient == nil {
		log.Error().Msg("oidc client not configured")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "oidc client not configured"})
		return
	}

	pkce, err := h.cfg.OIDCClient.GeneratePKCE()
	if err != nil {
		log.Error().Err(err).Msg("failed generating pkce pair")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed generating pkce"})
		return
	}

	state := generateRandomHex(16)
	nonce := generateRandomHex(16)

	authURL, err := h.cfg.OIDCClient.BuildAuthorizationURL(
		state,
		nonce,
		h.cfg.RedirectURI,
		pkce,
		"openid", "profile", "email", "roles",
	)
	if err != nil {
		log.Error().Err(err).Msg("failed building authorization url")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed building authorization url"})
		return
	}

	// Set temporary cookies for CSRF state, nonce, and PKCE code verifier (valid for 10 minutes)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("vuhive_auth_state", state, 600, h.cfg.CookiePath, h.cfg.CookieDomain, h.cfg.CookieSecure, true)
	c.SetCookie("vuhive_auth_verifier", pkce.Verifier, 600, h.cfg.CookiePath, h.cfg.CookieDomain, h.cfg.CookieSecure, true)
	c.SetCookie("vuhive_auth_nonce", nonce, 600, h.cfg.CookiePath, h.cfg.CookieDomain, h.cfg.CookieSecure, true)

	log.Info().
		Dur("duration_ms", time.Since(start)).
		Msg("redirecting to keycloak authorization endpoint")

	c.Redirect(http.StatusFound, authURL)
}

// Callback handles the OIDC redirect with authorization code and state, creating a session and setting HttpOnly cookie.
func (h *AuthHandler) Callback(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	log := zerolog.Ctx(ctx).With().Str("op", "AuthHandler.Callback").Logger()
	log.Debug().Msg("handling oidc callback")

	code := c.Query("code")
	state := c.Query("state")

	savedState, err := c.Cookie("vuhive_auth_state")
	if err != nil || savedState == "" || savedState != state {
		log.Warn().Msg("state parameter missing or mismatched")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state parameter"})
		return
	}

	verifier, err := c.Cookie("vuhive_auth_verifier")
	if err != nil || verifier == "" {
		log.Warn().Msg("pkce verifier cookie missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing pkce verifier"})
		return
	}

	// Clear temporary auth cookies
	c.SetCookie("vuhive_auth_state", "", -1, h.cfg.CookiePath, h.cfg.CookieDomain, h.cfg.CookieSecure, true)
	c.SetCookie("vuhive_auth_verifier", "", -1, h.cfg.CookiePath, h.cfg.CookieDomain, h.cfg.CookieSecure, true)
	c.SetCookie("vuhive_auth_nonce", "", -1, h.cfg.CookiePath, h.cfg.CookieDomain, h.cfg.CookieSecure, true)

	if code == "" {
		log.Warn().Msg("authorization code missing from callback query")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing authorization code"})
		return
	}

	tokenResp, err := h.cfg.OIDCClient.ExchangeCode(ctx, code, verifier, h.cfg.RedirectURI)
	if err != nil {
		log.Error().Err(err).Msg("code exchange failed")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "code exchange failed"})
		return
	}

	// Extract user identity, roles, and Keycloak session ID (sid)
	userID := "user"
	var kcSID string
	var roles []string

	tokenForClaims := tokenResp.IDToken
	if tokenForClaims == "" {
		tokenForClaims = tokenResp.AccessToken
	}
	if claims := parseJWTClaims(tokenForClaims); claims != nil {
		if claims.PreferredUsername != "" {
			userID = claims.PreferredUsername
		} else if claims.Sub != "" {
			userID = claims.Sub
		}
		kcSID = claims.Sid
		roles = claims.RealmAccess.Roles
	}

	ttl := time.Duration(tokenResp.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = h.cfg.SessionTTL
	}

	sessionID := uuid.New().String()
	cmd := inbound.CreateSessionCommand{
		SessionID:    sessionID,
		UserID:       userID,
		KeycloakSID:  kcSID,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		Roles:        roles,
		TTL:          ttl,
	}

	session, err := h.cfg.SessionService.CreateSession(ctx, cmd)
	if err != nil {
		log.Error().Err(err).Msg("failed creating session after code exchange")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed creating session"})
		return
	}

	// Set secure HttpOnly session cookie
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		h.cfg.CookieName,
		string(session.ID),
		int(ttl.Seconds()),
		h.cfg.CookiePath,
		h.cfg.CookieDomain,
		h.cfg.CookieSecure,
		true,
	)

	log.Info().
		Str("session_id", string(session.ID)).
		Str("user_id", session.UserID).
		Dur("duration_ms", time.Since(start)).
		Msg("completed login callback; redirecting to application")

	c.Redirect(http.StatusFound, h.cfg.PostLoginURL)
}

// Logout terminates the session in the store, revokes tokens with Keycloak, and clears the session cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	log := zerolog.Ctx(ctx).With().Str("op", "AuthHandler.Logout").Logger()
	log.Debug().Msg("handling logout request")

	cookieVal, err := c.Cookie(h.cfg.CookieName)
	if err == nil && cookieVal != "" {
		if session, err := h.cfg.SessionService.GetSession(ctx, model.SessionID(cookieVal)); err == nil && session != nil {
			if h.cfg.OIDCClient != nil && session.RefreshToken != "" {
				_ = h.cfg.OIDCClient.RevokeToken(ctx, session.RefreshToken, "refresh_token")
			}
			_ = h.cfg.SessionService.RevokeSession(ctx, session.ID)
		}
	}

	// Clear session cookie
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cfg.CookieName, "", -1, h.cfg.CookiePath, h.cfg.CookieDomain, h.cfg.CookieSecure, true)

	log.Info().
		Dur("duration_ms", time.Since(start)).
		Msg("completed user logout")

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Me returns the sanitized identity and roles of the authenticated caller.
func (h *AuthHandler) Me(c *gin.Context) {
	sessVal, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	session, ok := sessVal.(*model.ClientSession)
	if !ok || session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":    session.UserID,
		"roles":      session.Roles,
		"expires_at": session.ExpiresAt,
		"metadata":   session.Metadata,
	})
}

// BackchannelLogout validates signed OIDC logout tokens from Keycloak and terminates all matching sessions.
func (h *AuthHandler) BackchannelLogout(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	log := zerolog.Ctx(ctx).With().Str("op", "AuthHandler.BackchannelLogout").Logger()
	log.Debug().Msg("handling oidc backchannel logout webhook")

	logoutToken := c.PostForm("logout_token")
	if logoutToken == "" {
		log.Warn().Msg("missing logout_token form value")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing logout_token"})
		return
	}

	if h.cfg.OIDCClient == nil {
		log.Error().Msg("oidc client not configured for backchannel logout")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "oidc client not configured"})
		return
	}

	claims, err := h.cfg.OIDCClient.VerifyLogoutToken(ctx, logoutToken)
	if err != nil {
		log.Warn().Err(err).Msg("failed verifying logout token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid logout_token"})
		return
	}

	if claims.SessionID != "" {
		if err := h.cfg.SessionService.RevokeByKeycloakSID(ctx, claims.SessionID); err != nil {
			log.Error().Err(err).Str("sid", claims.SessionID).Msg("failed revoking sessions by keycloak sid")
		}
	}

	c.Header("Cache-Control", "no-cache, no-store")
	log.Info().
		Str("sid", claims.SessionID).
		Dur("duration_ms", time.Since(start)).
		Msg("completed backchannel logout")

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
