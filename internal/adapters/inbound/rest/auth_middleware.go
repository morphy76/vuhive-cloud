package rest

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// ClaimsContextKey is the context key under which verified *model.Claims is stored in gin.Context.
const ClaimsContextKey = "claims"

// AuthMiddleware inspects Authorization header, verifies JWT bearer token using TokenVerifierPort,
// and injects the resulting *model.Claims into gin.Context.
func AuthMiddleware(verifier outbound.TokenVerifierPort, optional bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			if optional {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error: "unauthorized: authentication required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error: "unauthorized: malformed authorization header, expected 'Bearer <token>'",
			})
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error: "unauthorized: empty bearer token",
			})
			return
		}

		if verifier == nil {
			c.Next()
			return
		}

		claims, err := verifier.VerifyToken(c.Request.Context(), tokenString)
		if err != nil {
			reqLogger := zerolog.Ctx(c.Request.Context())
			reqLogger.Warn().Err(err).Msg("token verification failed")

			status := http.StatusUnauthorized
			if errors.Is(err, model.ErrForbidden) {
				status = http.StatusForbidden
			}
			c.AbortWithStatusJSON(status, ErrorResponse{
				Error: err.Error(),
			})
			return
		}

		c.Set(ClaimsContextKey, claims)
		c.Next()
	}
}

// GetClaims extracts verified *model.Claims from gin.Context if present.
func GetClaims(c *gin.Context) *model.Claims {
	if val, exists := c.Get(ClaimsContextKey); exists {
		if claims, ok := val.(*model.Claims); ok {
			return claims
		}
	}
	return nil
}

// RequireRole guards an endpoint by asserting that authenticated claims satisfy at least one of the required roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error: "unauthorized: authentication required",
			})
			return
		}

		if !claims.HasAnyRole(roles...) {
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{
				Error: "forbidden: insufficient permissions",
			})
			return
		}

		c.Next()
	}
}

// RequireGroup guards an endpoint by asserting that authenticated claims belong to at least one of the required groups.
func RequireGroup(groups ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error: "unauthorized: authentication required",
			})
			return
		}

		// Admins satisfy any group requirement
		if claims.HasRole(model.RoleAdmin) {
			c.Next()
			return
		}

		inGroup := false
		for _, g := range groups {
			if claims.InGroup(g) {
				inGroup = true
				break
			}
		}

		if !inGroup {
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{
				Error: "forbidden: insufficient group membership",
			})
			return
		}

		c.Next()
	}
}
