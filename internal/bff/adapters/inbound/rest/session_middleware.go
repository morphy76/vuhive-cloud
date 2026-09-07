package rest

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/rs/zerolog"
)

type tokenClaimsPayload struct {
	Exp int64 `json:"exp"`
}

// isTokenExpiringSoon decodes the unverified JWT payload and checks if the token expires within window.
func isTokenExpiringSoon(tokenStr string, window time.Duration) bool {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return false
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	var claims tokenClaimsPayload
	if err := json.Unmarshal(payloadBytes, &claims); err != nil || claims.Exp <= 0 {
		return false
	}

	expTime := time.Unix(claims.Exp, 0)
	return time.Now().Add(window).After(expTime)
}

// SessionMiddleware extracts and verifies the client session cookie, transparently refreshes
// expiring access tokens via OIDC, and injects Authorization headers for downstream requests.
func SessionMiddleware(sessionService inbound.SessionService, oidcClient outbound.OIDCClient, cookieName string) gin.HandlerFunc {
	if cookieName == "" {
		cookieName = "vuhive_session"
	}

	return func(c *gin.Context) {
		start := time.Now()
		ctx := c.Request.Context()
		log := zerolog.Ctx(ctx).With().
			Str("op", "SessionMiddleware").
			Str("path", c.Request.URL.Path).
			Logger()

		cookieVal, err := c.Cookie(cookieName)
		if err != nil || cookieVal == "" {
			log.Debug().Msg("session cookie missing or empty")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		session, err := sessionService.GetSession(ctx, model.SessionID(cookieVal))
		if err != nil || session == nil || session.IsExpired() {
			log.Info().Err(err).Str("session_id", cookieVal).Msg("session missing or expired; clearing cookie")
			c.SetCookie(cookieName, "", -1, "/", "", false, true)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Check if access token is expired or expiring within 60 seconds
		if session.AccessToken != "" && isTokenExpiringSoon(session.AccessToken, 60*time.Second) {
			if oidcClient != nil && session.RefreshToken != "" {
				log.Debug().Str("session_id", string(session.ID)).Msg("access token expiring soon; refreshing transparently")

				tokenResp, refreshErr := oidcClient.RefreshToken(ctx, session.RefreshToken)
				if refreshErr != nil {
					log.Warn().Err(refreshErr).Str("session_id", string(session.ID)).Msg("failed transparent token refresh; invalidating session")
					_ = sessionService.RevokeSession(ctx, session.ID)
					c.SetCookie(cookieName, "", -1, "/", "", false, true)
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
					return
				}

				ttl := time.Duration(tokenResp.ExpiresIn) * time.Second
				if ttl <= 0 {
					ttl = session.ExpiresAt.Sub(session.UpdatedAt)
				}

				updatedSession, rotateErr := sessionService.RotateSessionTokens(ctx, inbound.RotateTokensCommand{
					SessionID:    session.ID,
					AccessToken:  tokenResp.AccessToken,
					RefreshToken: tokenResp.RefreshToken,
					IDToken:      tokenResp.IDToken,
					TTL:          ttl,
				})
				if rotateErr == nil && updatedSession != nil {
					session = updatedSession
					log.Info().Str("session_id", string(session.ID)).Msg("transparent token refresh succeeded")
				}
			}
		}

		// Inject Authorization header and context attributes
		if session.AccessToken != "" {
			c.Request.Header.Set("Authorization", "Bearer "+session.AccessToken)
		}

		c.Set("session", session)
		c.Set("user_id", session.UserID)
		c.Set("roles", session.Roles)

		log.Debug().
			Str("session_id", string(session.ID)).
			Str("user_id", session.UserID).
			Dur("duration_ms", time.Since(start)).
			Msg("session verified successfully")

		c.Next()
	}
}
