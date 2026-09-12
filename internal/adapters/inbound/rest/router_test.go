package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/version"
)

func TestRouter_VersionEndpoint(t *testing.T) {
	router := rest.SetupRouter(nil, nil, nil, nil)

	t.Run("GET /api/version returns HTTP 200 with populated version metadata", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

		var res rest.VersionResponse
		err := json.Unmarshal(w.Body.Bytes(), &res)
		require.NoError(t, err)

		assert.Equal(t, version.Version, res.Version)
		assert.Equal(t, version.Commit, res.Commit)
		assert.Equal(t, version.BuildTime, res.BuildTime)
	})


	t.Run("GET /api/version reflects dynamic runtime version overrides", func(t *testing.T) {
		origVersion, origCommit, origBuildTime := version.Version, version.Commit, version.BuildTime
		defer func() {
			version.Version = origVersion
			version.Commit = origCommit
			version.BuildTime = origBuildTime
		}()

		version.Version = "v1.2.3-test"
		version.Commit = "fedcba9"
		version.BuildTime = "2026-09-06T12:34:56Z"

		req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var res rest.VersionResponse
		err := json.Unmarshal(w.Body.Bytes(), &res)
		require.NoError(t, err)

		assert.Equal(t, "v1.2.3-test", res.Version)
		assert.Equal(t, "fedcba9", res.Commit)
		assert.Equal(t, "2026-09-06T12:34:56Z", res.BuildTime)
	})
}

func TestRouter_HealthEndpoints(t *testing.T) {
	router := rest.SetupRouter(nil, nil, nil, nil)

	paths := []string{"/healthz", "/api/v1/health"}
	for _, path := range paths {
		t.Run("GET "+path+" returns HTTP 200 OK", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

			var res map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &res)
			require.NoError(t, err)
			assert.Equal(t, "ok", res["status"])
		})
	}
}
