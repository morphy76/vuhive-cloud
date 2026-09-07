package auth_test

import (
	"bytes"
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
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/auth"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func generateTestRSAKey(t *testing.T) (*rsa.PrivateKey, string, string) {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	nBytes := privKey.N.Bytes()
	nB64 := base64.RawURLEncoding.EncodeToString(nBytes)

	eBytes := big.NewInt(int64(privKey.E)).Bytes()
	eB64 := base64.RawURLEncoding.EncodeToString(eBytes)

	return privKey, nB64, eB64
}

func signJWT(t *testing.T, privKey *rsa.PrivateKey, kid string, claims map[string]interface{}) string {
	t.Helper()

	header := map[string]interface{}{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kid,
	}

	headerJSON, err := json.Marshal(header)
	require.NoError(t, err)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	claimsJSON, err := json.Marshal(claims)
	require.NoError(t, err)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := fmt.Sprintf("%s.%s", headerB64, claimsB64)
	hash := sha256.Sum256([]byte(signingInput))

	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, hash[:])
	require.NoError(t, err)
	sigB64 := base64.RawURLEncoding.EncodeToString(sigBytes)

	return fmt.Sprintf("%s.%s", signingInput, sigB64)
}

func TestKeycloakTokenVerifier_VerifyToken(t *testing.T) {
	privKey, nB64, eB64 := generateTestRSAKey(t)
	kid := "test-key-id"

	jwksResp := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": kid,
				"n":   nB64,
				"e":   eB64,
			},
		},
	}
	jwksBytes, err := json.Marshal(jwksResp)
	require.NoError(t, err)

	inMemoryClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(jwksBytes)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	verifier := auth.NewKeycloakTokenVerifier(auth.Config{
		JWKSURL:    "http://keycloak.local/realms/vuhive/protocol/openid-connect/certs",
		ClientID:   "vuhive-cloud-api",
		HTTPClient: inMemoryClient,
	})

	ctx := context.Background()

	t.Run("valid token with roles and groups", func(t *testing.T) {
		claimsPayload := map[string]interface{}{
			"sub":                "usr-100",
			"preferred_username": "tester",
			"email":              "tester@example.com",
			"exp":                time.Now().Add(time.Hour).Unix(),
			"realm_access": map[string]interface{}{
				"roles": []string{model.RoleDeveloper},
			},
			"groups": []string{"/developers"},
		}

		tokenStr := signJWT(t, privKey, kid, claimsPayload)
		claims, err := verifier.VerifyToken(ctx, tokenStr)
		require.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, "usr-100", claims.Subject())
		assert.Equal(t, "tester", claims.Username())
		assert.Equal(t, "tester@example.com", claims.Email())
		assert.True(t, claims.HasRole(model.RoleDeveloper))
		assert.True(t, claims.InGroup("/developers"))
	})

	t.Run("expired token returns ErrTokenExpired or ErrUnauthorized", func(t *testing.T) {
		claimsPayload := map[string]interface{}{
			"sub":                "usr-101",
			"preferred_username": "expired",
			"exp":                time.Now().Add(-time.Hour).Unix(),
		}

		tokenStr := signJWT(t, privKey, kid, claimsPayload)
		claims, err := verifier.VerifyToken(ctx, tokenStr)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, model.ErrUnauthorized)
	})

	t.Run("invalid signature from untrusted key fails", func(t *testing.T) {
		untrustedKey, _, _ := generateTestRSAKey(t)
		claimsPayload := map[string]interface{}{
			"sub": "usr-102",
			"exp": time.Now().Add(time.Hour).Unix(),
		}

		tokenStr := signJWT(t, untrustedKey, kid, claimsPayload)
		claims, err := verifier.VerifyToken(ctx, tokenStr)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, model.ErrUnauthorized)
	})

	t.Run("malformed token string fails", func(t *testing.T) {
		claims, err := verifier.VerifyToken(ctx, "not-a-valid-token")
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, model.ErrUnauthorized)
	})
}
