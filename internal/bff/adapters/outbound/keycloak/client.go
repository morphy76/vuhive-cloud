package keycloak

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/rs/zerolog"
)

var _ outbound.OIDCClient = (*KeycloakClient)(nil)

// Config configures the Keycloak OIDC client adapter.
type Config struct {
	BaseURL      string
	Realm        string
	ClientID     string
	ClientSecret string
	TokenURL     string
	RevokeURL    string
	JWKSURL      string
	AuthURL      string
	IssuerURL    string
	HTTPClient   *http.Client
	JWKSCacheTTL time.Duration
}

type jwksKey struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

type logoutTokenPayload struct {
	Iss    string                 `json:"iss"`
	Sub    string                 `json:"sub"`
	Aud    interface{}            `json:"aud"`
	Iat    int64                  `json:"iat"`
	Jti    string                 `json:"jti"`
	Sid    string                 `json:"sid"`
	Nonce  *string                `json:"nonce,omitempty"`
	Events map[string]interface{} `json:"events"`
}

// KeycloakClient implements outbound.OIDCClient for Keycloak authentication and token management.
type KeycloakClient struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.RWMutex
	keys       map[string]*rsa.PublicKey
	lastFetch  time.Time
}

// NewKeycloakClient constructs an initialized KeycloakClient instance.
func NewKeycloakClient(cfg Config) *KeycloakClient {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	realm := cfg.Realm

	if cfg.TokenURL == "" && baseURL != "" && realm != "" {
		cfg.TokenURL = fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", baseURL, realm)
	}
	if cfg.RevokeURL == "" && baseURL != "" && realm != "" {
		cfg.RevokeURL = fmt.Sprintf("%s/realms/%s/protocol/openid-connect/revoke", baseURL, realm)
	}
	if cfg.JWKSURL == "" && baseURL != "" && realm != "" {
		cfg.JWKSURL = fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", baseURL, realm)
	}
	if cfg.AuthURL == "" && baseURL != "" && realm != "" {
		cfg.AuthURL = fmt.Sprintf("%s/realms/%s/protocol/openid-connect/auth", baseURL, realm)
	}
	if cfg.IssuerURL == "" && baseURL != "" && realm != "" {
		cfg.IssuerURL = fmt.Sprintf("%s/realms/%s", baseURL, realm)
	}
	if cfg.JWKSCacheTTL <= 0 {
		cfg.JWKSCacheTTL = 1 * time.Hour
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	return &KeycloakClient{
		cfg:        cfg,
		httpClient: client,
		keys:       make(map[string]*rsa.PublicKey),
	}
}

// GeneratePKCE generates a cryptographic code verifier and S256 code challenge conforming to RFC 7636.
func (c *KeycloakClient) GeneratePKCE() (*outbound.PKCEPair, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("%w: failed generating random bytes for pkce: %v", model.ErrInternal, err)
	}

	verifier := base64.RawURLEncoding.EncodeToString(bytes)
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	return &outbound.PKCEPair{
		Verifier:  verifier,
		Challenge: challenge,
		Method:    "S256",
	}, nil
}

// BuildAuthorizationURL constructs the Keycloak OIDC login redirect URL.
func (c *KeycloakClient) BuildAuthorizationURL(state, nonce, redirectURI string, pkce *outbound.PKCEPair, scopes ...string) (string, error) {
	if c.cfg.AuthURL == "" {
		return "", fmt.Errorf("%w: auth URL not configured", model.ErrInternal)
	}
	if pkce == nil || pkce.Challenge == "" {
		return "", fmt.Errorf("%w: pkce challenge required", model.ErrInvalidParameter)
	}

	allScopes := []string{"openid"}
	for _, s := range scopes {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" && trimmed != "openid" {
			allScopes = append(allScopes, trimmed)
		}
	}

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", c.cfg.ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("state", state)
	if nonce != "" {
		params.Set("nonce", nonce)
	}
	params.Set("code_challenge", pkce.Challenge)
	params.Set("code_challenge_method", pkce.Method)
	params.Set("scope", strings.Join(allScopes, " "))

	sep := "?"
	if strings.Contains(c.cfg.AuthURL, "?") {
		sep = "&"
	}

	return c.cfg.AuthURL + sep + params.Encode(), nil
}

// ExchangeCode exchanges an authorization code and PKCE verifier for OAuth2/OIDC tokens.
func (c *KeycloakClient) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*outbound.TokenResponse, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "KeycloakClient.ExchangeCode").
		Str("client_id", c.cfg.ClientID).
		Logger()
	log.Debug().Msg("starting authorization code exchange")

	if code == "" || codeVerifier == "" || redirectURI == "" {
		err := fmt.Errorf("%w: code, codeVerifier, and redirectURI must not be empty", model.ErrInvalidParameter)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid exchange arguments")
		return nil, err
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", c.cfg.ClientID)
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	form.Set("redirect_uri", redirectURI)
	if c.cfg.ClientSecret != "" {
		form.Set("client_secret", c.cfg.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating exchange request")
		return nil, fmt.Errorf("%w: failed creating exchange request: %v", model.ErrInternal, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("token exchange HTTP request failed")
		return nil, fmt.Errorf("%w: token exchange failed: %v", model.ErrInternal, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed reading exchange response")
		return nil, fmt.Errorf("%w: failed reading exchange response: %v", model.ErrInternal, err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Warn().
			Int("status_code", resp.StatusCode).
			Dur("duration_ms", time.Since(start)).
			Msg("token exchange failed with non-200 status")

		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
			return nil, model.ErrUnauthorized
		}
		return nil, fmt.Errorf("%w: unexpected token response status %d", model.ErrInternal, resp.StatusCode)
	}

	var tokenResp outbound.TokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed unmarshaling token response")
		return nil, fmt.Errorf("%w: corrupted token response payload: %v", model.ErrInternal, err)
	}

	log.Info().
		Str("token_type", tokenResp.TokenType).
		Int("expires_in", tokenResp.ExpiresIn).
		Dur("duration_ms", time.Since(start)).
		Msg("completed authorization code exchange")

	return &tokenResp, nil
}

// RefreshToken exchanges an active refresh token for a renewed access and refresh token.
func (c *KeycloakClient) RefreshToken(ctx context.Context, refreshToken string) (*outbound.TokenResponse, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "KeycloakClient.RefreshToken").
		Str("client_id", c.cfg.ClientID).
		Logger()
	log.Debug().Msg("starting token refresh")

	if refreshToken == "" {
		err := fmt.Errorf("%w: refresh token cannot be empty", model.ErrInvalidParameter)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("empty refresh token")
		return nil, err
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", c.cfg.ClientID)
	form.Set("refresh_token", refreshToken)
	if c.cfg.ClientSecret != "" {
		form.Set("client_secret", c.cfg.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating refresh request")
		return nil, fmt.Errorf("%w: failed creating refresh request: %v", model.ErrInternal, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("token refresh HTTP request failed")
		return nil, fmt.Errorf("%w: token refresh failed: %v", model.ErrInternal, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed reading refresh response")
		return nil, fmt.Errorf("%w: failed reading refresh response: %v", model.ErrInternal, err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Warn().
			Int("status_code", resp.StatusCode).
			Dur("duration_ms", time.Since(start)).
			Msg("token refresh failed with non-200 status")

		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
			return nil, model.ErrSessionExpired
		}
		return nil, fmt.Errorf("%w: unexpected refresh response status %d", model.ErrInternal, resp.StatusCode)
	}

	var tokenResp outbound.TokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed unmarshaling refresh response")
		return nil, fmt.Errorf("%w: corrupted refresh response payload: %v", model.ErrInternal, err)
	}

	log.Info().
		Int("expires_in", tokenResp.ExpiresIn).
		Dur("duration_ms", time.Since(start)).
		Msg("completed token refresh")

	return &tokenResp, nil
}

// RevokeToken revokes an active token via Keycloak's RFC 7009 revocation endpoint.
func (c *KeycloakClient) RevokeToken(ctx context.Context, token, tokenTypeHint string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "KeycloakClient.RevokeToken").
		Str("client_id", c.cfg.ClientID).
		Str("token_type_hint", tokenTypeHint).
		Logger()
	log.Debug().Msg("starting token revocation")

	if token == "" {
		err := fmt.Errorf("%w: token cannot be empty", model.ErrInvalidParameter)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("empty token")
		return err
	}

	form := url.Values{}
	form.Set("client_id", c.cfg.ClientID)
	form.Set("token", token)
	if tokenTypeHint != "" {
		form.Set("token_type_hint", tokenTypeHint)
	}
	if c.cfg.ClientSecret != "" {
		form.Set("client_secret", c.cfg.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.RevokeURL, strings.NewReader(form.Encode()))
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating revoke request")
		return fmt.Errorf("%w: failed creating revoke request: %v", model.ErrInternal, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("token revocation HTTP request failed")
		return fmt.Errorf("%w: token revocation failed: %v", model.ErrInternal, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warn().
			Int("status_code", resp.StatusCode).
			Dur("duration_ms", time.Since(start)).
			Msg("token revocation failed with non-200 status")
		return fmt.Errorf("%w: unexpected revocation status %d", model.ErrInternal, resp.StatusCode)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed token revocation")
	return nil
}

// VerifyLogoutToken parses, validates RS256 signature, and checks OIDC Backchannel Logout claims.
func (c *KeycloakClient) VerifyLogoutToken(ctx context.Context, rawLogoutToken string) (*outbound.LogoutTokenClaims, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "KeycloakClient.VerifyLogoutToken").
		Logger()
	log.Debug().Msg("starting logout token verification")

	parts := strings.Split(rawLogoutToken, ".")
	if len(parts) != 3 {
		err := fmt.Errorf("%w: malformed jwt structure", model.ErrUnauthorized)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid jwt segments")
		return nil, err
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: failed decoding jwt header: %v", model.ErrUnauthorized, err)
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("%w: failed parsing jwt header: %v", model.ErrUnauthorized, err)
	}

	if header.Alg != "RS256" {
		return nil, fmt.Errorf("%w: unsupported jwt alg %s", model.ErrUnauthorized, header.Alg)
	}

	// Retrieve public key from cached JWKS or refresh
	pubKey, err := c.getPublicKey(ctx, header.Kid)
	if err != nil {
		log.Error().Err(err).Str("kid", header.Kid).Msg("failed obtaining public key for kid")
		return nil, fmt.Errorf("%w: failed obtaining public key: %v", model.ErrUnauthorized, err)
	}

	// Verify cryptographic RS256 signature
	sigInput := parts[0] + "." + parts[1]
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("%w: failed decoding jwt signature: %v", model.ErrUnauthorized, err)
	}

	h := sha256.Sum256([]byte(sigInput))
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, h[:], sigBytes); err != nil {
		log.Warn().Err(err).Msg("rsa signature verification failed")
		return nil, fmt.Errorf("%w: invalid signature: %v", model.ErrUnauthorized, err)
	}

	// Parse and validate logout token payload claims
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: failed decoding jwt payload: %v", model.ErrUnauthorized, err)
	}

	var payload logoutTokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("%w: failed parsing logout token payload: %v", model.ErrUnauthorized, err)
	}

	// 1. Verify issuer
	if c.cfg.IssuerURL != "" && payload.Iss != c.cfg.IssuerURL {
		log.Warn().Str("expected_iss", c.cfg.IssuerURL).Str("actual_iss", payload.Iss).Msg("issuer mismatch")
		return nil, model.ErrUnauthorized
	}

	// 2. Verify audience contains client ID
	var audList []string
	switch v := payload.Aud.(type) {
	case string:
		audList = append(audList, v)
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				audList = append(audList, str)
			}
		}
	}

	matchedAud := false
	for _, a := range audList {
		if a == c.cfg.ClientID {
			matchedAud = true
			break
		}
	}
	if c.cfg.ClientID != "" && !matchedAud {
		log.Warn().Str("expected_aud", c.cfg.ClientID).Interface("actual_aud", audList).Msg("audience mismatch")
		return nil, model.ErrUnauthorized
	}

	// 3. Verify absence of nonce (OIDC Backchannel Logout 1.0 Section 2.4)
	if payload.Nonce != nil {
		err := fmt.Errorf("%w: logout token must not contain a nonce claim", model.ErrInvalidParameter)
		log.Warn().Err(err).Msg("forbidden nonce claim present in logout token")
		return nil, err
	}

	// 4. Verify events claim contains backchannel logout event
	if payload.Events == nil {
		err := fmt.Errorf("%w: missing events claim", model.ErrInvalidParameter)
		log.Warn().Err(err).Msg("events claim absent in logout token")
		return nil, err
	}
	if _, ok := payload.Events["http://schemas.openid.net/event/backchannel-logout"]; !ok {
		err := fmt.Errorf("%w: missing backchannel-logout event in events claim", model.ErrInvalidParameter)
		log.Warn().Err(err).Msg("backchannel-logout event missing")
		return nil, err
	}

	claims := &outbound.LogoutTokenClaims{
		Issuer:    payload.Iss,
		Subject:   payload.Sub,
		Audience:  audList,
		SessionID: payload.Sid,
		TokenID:   payload.Jti,
		IssuedAt:  time.Unix(payload.Iat, 0),
		Events:    payload.Events,
	}

	log.Info().
		Str("session_id", claims.SessionID).
		Str("sub", claims.Subject).
		Str("jti", claims.TokenID).
		Dur("duration_ms", time.Since(start)).
		Msg("completed logout token verification")

	return claims, nil
}

func (c *KeycloakClient) getPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	if time.Since(c.lastFetch) < c.cfg.JWKSCacheTTL {
		if key, ok := c.keys[kid]; ok {
			c.mu.RUnlock()
			return key, nil
		}
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(c.lastFetch) < c.cfg.JWKSCacheTTL {
		if key, ok := c.keys[kid]; ok {
			return key, nil
		}
	}

	if err := c.refreshJWKS(ctx); err != nil {
		return nil, err
	}

	if key, ok := c.keys[kid]; ok {
		return key, nil
	}

	return nil, fmt.Errorf("public key not found in jwks for kid: %s", kid)
}

func (c *KeycloakClient) refreshJWKS(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.JWKSURL, nil)
	if err != nil {
		return fmt.Errorf("failed creating jwks request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed fetching jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status from jwks endpoint: %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed decoding jwks response: %w", err)
	}

	newKeys := make(map[string]*rsa.PublicKey)
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}

		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}

		n := new(big.Int).SetBytes(nBytes)
		var e int
		for _, b := range eBytes {
			e = (e << 8) | int(b)
		}

		newKeys[k.Kid] = &rsa.PublicKey{
			N: n,
			E: e,
		}
	}

	c.keys = newKeys
	c.lastFetch = time.Now()
	return nil
}
