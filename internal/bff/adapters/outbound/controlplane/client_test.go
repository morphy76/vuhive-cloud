package controlplane_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/controlplane"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClient_CheckHealth(t *testing.T) {
	ctx := context.Background()

	t.Run("successful health check", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "http://controlplane/healthz", req.URL.String())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		health, err := client.CheckHealth(ctx)
		require.NoError(t, err)
		assert.Equal(t, "UP", health.Status)
	})

	t.Run("server returns 500 error", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString(`internal error`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		_, err := client.CheckHealth(ctx)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrControlPlaneUnavailable)
	})

	t.Run("server connection refused / transport error", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp: connection refused")
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		_, err := client.CheckHealth(ctx)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrControlPlaneUnavailable)
	})
}

func TestClient_GetVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("successful version check", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "http://controlplane/version", req.URL.String())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"version":"0.0.1","commit":"abc1234","build_time":"2026-09-05T10:00:00Z"}`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		ver, err := client.GetVersion(ctx)
		require.NoError(t, err)
		assert.Equal(t, "0.0.1", ver.Version)
		assert.Equal(t, "abc1234", ver.Commit)
	})

	t.Run("server returns 404 for version", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString(`not found`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		ver, err := client.GetVersion(ctx)
		require.NoError(t, err)
		assert.Equal(t, "unknown", ver.Version)
	})

	t.Run("server connection error during version fetch", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network failure")
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		_, err := client.GetVersion(ctx)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrControlPlaneUnavailable)
	})
}
