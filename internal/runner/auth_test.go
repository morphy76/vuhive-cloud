package runner_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestResolveBearerToken(t *testing.T) {
	ctx := context.Background()

	t.Run("returns direct auth token when provided", func(t *testing.T) {
		cfg := runner.WrapperConfig{
			AuthToken: "static-token-123",
		}
		token, err := runner.ResolveBearerToken(ctx, cfg, http.DefaultClient)
		require.NoError(t, err)
		assert.Equal(t, "static-token-123", token)
	})

	t.Run("fetches token via client credentials grant", func(t *testing.T) {
		mockClient := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPost, r.Method)
				bodyBytes, _ := io.ReadAll(r.Body)
				bodyStr := string(bodyBytes)
				assert.Contains(t, bodyStr, "grant_type=client_credentials")
				assert.Contains(t, bodyStr, "client_id=vuhive-runner")
				assert.Contains(t, bodyStr, "client_secret=secret-123")

				respBytes, _ := json.Marshal(map[string]interface{}{
					"access_token": "m2m-jwt-token-456",
					"token_type":   "Bearer",
					"expires_in":   300,
				})

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(respBytes)),
					Header:     make(http.Header),
				}, nil
			}),
		}

		cfg := runner.WrapperConfig{
			ClientID:     "vuhive-runner",
			ClientSecret: "secret-123",
			TokenURL:     "http://keycloak.local/token",
		}

		token, err := runner.ResolveBearerToken(ctx, cfg, mockClient)
		require.NoError(t, err)
		assert.Equal(t, "m2m-jwt-token-456", token)
	})

	t.Run("returns empty string when no auth is configured", func(t *testing.T) {
		cfg := runner.WrapperConfig{}
		token, err := runner.ResolveBearerToken(ctx, cfg, http.DefaultClient)
		require.NoError(t, err)
		assert.Equal(t, "", token)
	})
}
