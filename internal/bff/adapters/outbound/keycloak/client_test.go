package keycloak_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/keycloak"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to generate an RSA test key pair
func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return priv
}

// Helper to build JWKS response JSON from RSA public key
func buildJWKS(kid string, pub *rsa.PublicKey) string {
	nBytes := pub.N.Bytes()
	eBytes := big.NewInt(int64(pub.E)).Bytes()
	nStr := base64.RawURLEncoding.EncodeToString(nBytes)
	eStr := base64.RawURLEncoding.EncodeToString(eBytes)

	return fmt.Sprintf(`{
		"keys": [
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": "%s",
				"n": "%s",
				"e": "%s"
			}
		]
	}`, kid, nStr, eStr)
}

// Helper to craft and sign a JWT with RSA-SHA256
func signJWT(t *testing.T, priv *rsa.PrivateKey, kid string, claims map[string]interface{}) string {
	t.Helper()

	headerJSON, err := json.Marshal(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kid,
	})
	require.NoError(t, err)
	hdrB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	claimsJSON, err := json.Marshal(claims)
	require.NoError(t, err)
	payloadB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	sigInput := hdrB64 + "." + payloadB64
	hashed := sha256.Sum256([]byte(sigInput))

	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed[:])
	require.NoError(t, err)
	sigB64 := base64.RawURLEncoding.EncodeToString(sigBytes)

	return sigInput + "." + sigB64
}

func TestKeycloakClient_InterfaceCompliance(t *testing.T) {
	var _ outbound.OIDCClient = (*keycloak.KeycloakClient)(nil)
}

func TestKeycloakClient_PKCE(t *testing.T) {
	client := keycloak.NewKeycloakClient(keycloak.Config{
		BaseURL:  "https://auth.example.com",
		Realm:    "vuhive",
		ClientID: "vuhive-web",
	})

	pkce, err := client.GeneratePKCE()
	require.NoError(t, err)
	require.NotNil(t, pkce)

	assert.Equal(t, "S256", pkce.Method)
	assert.GreaterOrEqual(t, len(pkce.Verifier), 43)
	assert.LessOrEqual(t, len(pkce.Verifier), 128)

	// Verify challenge is SHA256 base64url-encoded without padding
	h := sha256.Sum256([]byte(pkce.Verifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(h[:])
	assert.Equal(t, expectedChallenge, pkce.Challenge)
}

func TestKeycloakClient_BuildAuthorizationURL(t *testing.T) {
	client := keycloak.NewKeycloakClient(keycloak.Config{
		BaseURL:  "https://auth.example.com",
		Realm:    "vuhive",
		ClientID: "vuhive-web",
	})

	pkce, err := client.GeneratePKCE()
	require.NoError(t, err)

	authURL, err := client.BuildAuthorizationURL("state-123", "nonce-456", "https://app.example.com/callback", pkce, "openid", "profile", "roles")
	require.NoError(t, err)

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)

	assert.Equal(t, "https", parsed.Scheme)
	assert.Equal(t, "auth.example.com", parsed.Host)
	assert.Equal(t, "/realms/vuhive/protocol/openid-connect/auth", parsed.Path)

	q := parsed.Query()
	assert.Equal(t, "code", q.Get("response_type"))
	assert.Equal(t, "vuhive-web", q.Get("client_id"))
	assert.Equal(t, "state-123", q.Get("state"))
	assert.Equal(t, "nonce-456", q.Get("nonce"))
	assert.Equal(t, "https://app.example.com/callback", q.Get("redirect_uri"))
	assert.Equal(t, pkce.Challenge, q.Get("code_challenge"))
	assert.Equal(t, "S256", q.Get("code_challenge_method"))
	assert.Contains(t, q.Get("scope"), "openid")
	assert.Contains(t, q.Get("scope"), "roles")
}

func TestKeycloakClient_ExchangeCode(t *testing.T) {
	ctx := context.Background()

	t.Run("successful code exchange", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/realms/vuhive/protocol/openid-connect/token" && r.Method == http.MethodPost {
				err := r.ParseForm()
				assert.NoError(t, err)
				assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
				assert.Equal(t, "auth-code-123", r.FormValue("code"))
				assert.Equal(t, "verifier-xyz", r.FormValue("code_verifier"))
				assert.Equal(t, "vuhive-web", r.FormValue("client_id"))
				assert.Equal(t, "https://app.example.com/callback", r.FormValue("redirect_uri"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"access_token": "acc-token-1",
					"refresh_token": "ref-token-1",
					"id_token": "id-token-1",
					"token_type": "Bearer",
					"expires_in": 300,
					"scope": "openid profile"
				}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		client := keycloak.NewKeycloakClient(keycloak.Config{
			BaseURL:  ts.URL,
			Realm:    "vuhive",
			ClientID: "vuhive-web",
		})

		resp, err := client.ExchangeCode(ctx, "auth-code-123", "verifier-xyz", "https://app.example.com/callback")
		require.NoError(t, err)
		assert.Equal(t, "acc-token-1", resp.AccessToken)
		assert.Equal(t, "ref-token-1", resp.RefreshToken)
		assert.Equal(t, "id-token-1", resp.IDToken)
		assert.Equal(t, "Bearer", resp.TokenType)
		assert.Equal(t, 300, resp.ExpiresIn)
	})

	t.Run("invalid code returns ErrUnauthorized", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Code not found"}`))
		}))
		defer ts.Close()

		client := keycloak.NewKeycloakClient(keycloak.Config{
			BaseURL:  ts.URL,
			Realm:    "vuhive",
			ClientID: "vuhive-web",
		})

		_, err := client.ExchangeCode(ctx, "bad-code", "ver", "https://app.example.com/callback")
		assert.ErrorIs(t, err, model.ErrUnauthorized)
	})

	t.Run("empty code returns ErrInvalidParameter", func(t *testing.T) {
		client := keycloak.NewKeycloakClient(keycloak.Config{
			BaseURL:  "https://auth.example.com",
			Realm:    "vuhive",
			ClientID: "vuhive-web",
		})

		_, err := client.ExchangeCode(ctx, "", "ver", "https://app.example.com/callback")
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})
}

func TestKeycloakClient_RefreshToken(t *testing.T) {
	ctx := context.Background()

	t.Run("successful refresh", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/realms/vuhive/protocol/openid-connect/token" && r.Method == http.MethodPost {
				err := r.ParseForm()
				assert.NoError(t, err)
				assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
				assert.Equal(t, "valid-refresh-token", r.FormValue("refresh_token"))
				assert.Equal(t, "vuhive-web", r.FormValue("client_id"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"access_token": "new-acc-tok",
					"refresh_token": "new-ref-tok",
					"id_token": "new-id-tok",
					"token_type": "Bearer",
					"expires_in": 300
				}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		client := keycloak.NewKeycloakClient(keycloak.Config{
			BaseURL:  ts.URL,
			Realm:    "vuhive",
			ClientID: "vuhive-web",
		})

		resp, err := client.RefreshToken(ctx, "valid-refresh-token")
		require.NoError(t, err)
		assert.Equal(t, "new-acc-tok", resp.AccessToken)
		assert.Equal(t, "new-ref-tok", resp.RefreshToken)
		assert.Equal(t, 300, resp.ExpiresIn)
	})

	t.Run("expired refresh token returns ErrSessionExpired", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Session not active"}`))
		}))
		defer ts.Close()

		client := keycloak.NewKeycloakClient(keycloak.Config{
			BaseURL:  ts.URL,
			Realm:    "vuhive",
			ClientID: "vuhive-web",
		})

		_, err := client.RefreshToken(ctx, "expired-tok")
		assert.ErrorIs(t, err, model.ErrSessionExpired)
	})

	t.Run("empty refresh token returns ErrInvalidParameter", func(t *testing.T) {
		client := keycloak.NewKeycloakClient(keycloak.Config{})
		_, err := client.RefreshToken(ctx, "")
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})
}

func TestKeycloakClient_RevokeToken(t *testing.T) {
	ctx := context.Background()

	t.Run("successful revocation", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/realms/vuhive/protocol/openid-connect/revoke" && r.Method == http.MethodPost {
				err := r.ParseForm()
				assert.NoError(t, err)
				assert.Equal(t, "tok-to-revoke", r.FormValue("token"))
				assert.Equal(t, "refresh_token", r.FormValue("token_type_hint"))
				assert.Equal(t, "vuhive-web", r.FormValue("client_id"))

				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		client := keycloak.NewKeycloakClient(keycloak.Config{
			BaseURL:  ts.URL,
			Realm:    "vuhive",
			ClientID: "vuhive-web",
		})

		err := client.RevokeToken(ctx, "tok-to-revoke", "refresh_token")
		assert.NoError(t, err)
	})

	t.Run("empty token returns ErrInvalidParameter", func(t *testing.T) {
		client := keycloak.NewKeycloakClient(keycloak.Config{})
		err := client.RevokeToken(ctx, "", "refresh_token")
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})
}

func TestKeycloakClient_VerifyLogoutToken(t *testing.T) {
	ctx := context.Background()
	privKey := generateRSAKey(t)
	const kid = "key-1"

	var jwksContent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/protocol/openid-connect/certs") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(jwksContent))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	issuer := ts.URL + "/realms/vuhive"
	jwksContent = buildJWKS(kid, &privKey.PublicKey)

	client := keycloak.NewKeycloakClient(keycloak.Config{
		BaseURL:   ts.URL,
		Realm:     "vuhive",
		ClientID:  "vuhive-web",
		IssuerURL: issuer,
	})

	t.Run("valid logout token passes verification", func(t *testing.T) {
		claims := map[string]interface{}{
			"iss": issuer,
			"sub": "user-uuid-1",
			"aud": []string{"vuhive-web"},
			"iat": time.Now().Unix(),
			"jti": "logout-tok-1",
			"sid": "session-kc-123",
			"events": map[string]interface{}{
				"http://schemas.openid.net/event/backchannel-logout": map[string]interface{}{},
			},
		}

		rawToken := signJWT(t, privKey, kid, claims)
		verified, err := client.VerifyLogoutToken(ctx, rawToken)
		require.NoError(t, err)
		assert.Equal(t, issuer, verified.Issuer)
		assert.Equal(t, "user-uuid-1", verified.Subject)
		assert.Equal(t, "session-kc-123", verified.SessionID)
		assert.Equal(t, "logout-tok-1", verified.TokenID)
	})

	t.Run("logout token with forbidden nonce claim fails verification", func(t *testing.T) {
		claims := map[string]interface{}{
			"iss":   issuer,
			"aud":   "vuhive-web",
			"iat":   time.Now().Unix(),
			"jti":   "logout-tok-nonce",
			"sid":   "session-kc-123",
			"nonce": "forbidden-nonce",
			"events": map[string]interface{}{
				"http://schemas.openid.net/event/backchannel-logout": map[string]interface{}{},
			},
		}

		rawToken := signJWT(t, privKey, kid, claims)
		_, err := client.VerifyLogoutToken(ctx, rawToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nonce")
	})

	t.Run("logout token missing backchannel logout event fails", func(t *testing.T) {
		claims := map[string]interface{}{
			"iss": issuer,
			"aud": "vuhive-web",
			"iat": time.Now().Unix(),
			"jti": "logout-tok-no-ev",
			"sid": "session-kc-123",
			"events": map[string]interface{}{
				"some-other-event": map[string]interface{}{},
			},
		}

		rawToken := signJWT(t, privKey, kid, claims)
		_, err := client.VerifyLogoutToken(ctx, rawToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "backchannel-logout")
	})

	t.Run("logout token with wrong issuer fails", func(t *testing.T) {
		claims := map[string]interface{}{
			"iss": "https://attacker.com/realms/fake",
			"aud": "vuhive-web",
			"iat": time.Now().Unix(),
			"jti": "logout-tok-iss",
			"sid": "session-kc-123",
			"events": map[string]interface{}{
				"http://schemas.openid.net/event/backchannel-logout": map[string]interface{}{},
			},
		}

		rawToken := signJWT(t, privKey, kid, claims)
		_, err := client.VerifyLogoutToken(ctx, rawToken)
		assert.ErrorIs(t, err, model.ErrUnauthorized)
	})

	t.Run("logout token with wrong audience fails", func(t *testing.T) {
		claims := map[string]interface{}{
			"iss": issuer,
			"aud": "wrong-client",
			"iat": time.Now().Unix(),
			"jti": "logout-tok-aud",
			"sid": "session-kc-123",
			"events": map[string]interface{}{
				"http://schemas.openid.net/event/backchannel-logout": map[string]interface{}{},
			},
		}

		rawToken := signJWT(t, privKey, kid, claims)
		_, err := client.VerifyLogoutToken(ctx, rawToken)
		assert.ErrorIs(t, err, model.ErrUnauthorized)
	})

	t.Run("automatic JWKS key rotation succeeds on unseen kid", func(t *testing.T) {
		newPrivKey := generateRSAKey(t)
		const newKid = "rotated-key-2"

		claims := map[string]interface{}{
			"iss": issuer,
			"aud": "vuhive-web",
			"iat": time.Now().Unix(),
			"jti": "logout-tok-rotated",
			"sid": "session-rotated-456",
			"events": map[string]interface{}{
				"http://schemas.openid.net/event/backchannel-logout": map[string]interface{}{},
			},
		}

		// Update JWKS endpoint response to serve the new key
		jwksContent = buildJWKS(newKid, &newPrivKey.PublicKey)

		rawToken := signJWT(t, newPrivKey, newKid, claims)
		verified, err := client.VerifyLogoutToken(ctx, rawToken)
		require.NoError(t, err)
		assert.Equal(t, "session-rotated-456", verified.SessionID)
	})
}
