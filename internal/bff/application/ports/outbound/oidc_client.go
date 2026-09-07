package outbound

import (
	"context"
	"time"
)

// TokenResponse represents the OAuth 2.0 / OIDC token endpoint response payload.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // Expiration duration in seconds
	Scope        string `json:"scope,omitempty"`
}

// PKCEPair holds a cryptographically secure code verifier and its S256 code challenge.
type PKCEPair struct {
	Verifier  string
	Challenge string
	Method    string // Always "S256"
}

// LogoutTokenClaims represents verified claims extracted from an OIDC Back-Channel Logout token.
type LogoutTokenClaims struct {
	Issuer    string                 `json:"iss"`
	Subject   string                 `json:"sub,omitempty"`
	Audience  []string               `json:"aud"`
	SessionID string                 `json:"sid,omitempty"`
	TokenID   string                 `json:"jti"`
	IssuedAt  time.Time              `json:"iat"`
	Events    map[string]interface{} `json:"events"`
}

// OIDCClient defines the driven outbound port for interacting with Keycloak OIDC endpoints.
type OIDCClient interface {
	// GeneratePKCE creates a cryptographic code verifier and S256 code challenge.
	GeneratePKCE() (*PKCEPair, error)

	// BuildAuthorizationURL constructs the Keycloak authorization redirect URL with PKCE parameters.
	BuildAuthorizationURL(state, nonce, redirectURI string, pkce *PKCEPair, scopes ...string) (string, error)

	// ExchangeCode exchanges an authorization code and PKCE code verifier for OAuth2/OIDC tokens.
	ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*TokenResponse, error)

	// RefreshToken exchanges an active refresh token for renewed access and refresh tokens.
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)

	// RevokeToken revokes an active access or refresh token using the RFC 7009 revocation endpoint.
	RevokeToken(ctx context.Context, token, tokenTypeHint string) error

	// VerifyLogoutToken validates the RS256 signature and claims of an incoming Backchannel Logout token.
	VerifyLogoutToken(ctx context.Context, rawLogoutToken string) (*LogoutTokenClaims, error)
}
