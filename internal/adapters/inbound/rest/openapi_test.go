package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/morphy76/vuhive-cloud/api"
	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/version"
)

func TestOpenAPI_Endpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := rest.SetupRouter(nil, nil, nil, nil)

	t.Run("GET /api/openapi.yaml returns HTTP 200 with application/yaml and valid OpenAPI 3.1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/yaml", w.Header().Get("Content-Type"))

		var doc map[string]interface{}
		err := yaml.Unmarshal(w.Body.Bytes(), &doc)
		require.NoError(t, err)
		assert.Equal(t, "3.1.0", doc["openapi"])
		assert.NotNil(t, doc["info"])
		assert.NotNil(t, doc["paths"])
	})

	t.Run("GET /api/openapi.json returns HTTP 200 with application/json and valid OpenAPI 3.1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var doc map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &doc)
		require.NoError(t, err)
		assert.Equal(t, "3.1.0", doc["openapi"])
		assert.NotNil(t, doc["info"])
		assert.NotNil(t, doc["paths"])
	})

	t.Run("GET /api/version returns HTTP 200 with VersionResponse", func(t *testing.T) {
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
}

func TestOpenAPI_EmbeddedFS(t *testing.T) {
	yamlBytes, err := api.SpecFS.ReadFile("openapi.yaml")
	require.NoError(t, err)
	assert.Equal(t, api.OpenAPISpecYAML, yamlBytes)

	jsonBytes, err := api.SpecFS.ReadFile("openapi.json")
	require.NoError(t, err)
	assert.Equal(t, api.OpenAPISpecJSON, jsonBytes)
}

func TestOpenAPI_RouteCoverage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Build router with all use cases mocked
	buildsUC := new(MockBuildsUseCase)
	profilesUC := new(MockProfilesUseCase)
	schedulesUC := new(mockSchedulesUseCase)
	runsUC := new(mockRunsUseCase)
	barrierUC := new(mockBarrierUseCase)
	housekeepingUC := new(MockHousekeepingUseCase)
	suitesUC := new(MockSuitesUseCase)
	configsUC := new(MockConfigsUseCase)

	router := rest.SetupRouterWithConfig(rest.RouterConfig{
		BuildsUC:       buildsUC,
		ProfilesUC:     profilesUC,
		SchedulesUC:    schedulesUC,
		RunsUC:         runsUC,
		BarrierUC:      barrierUC,
		HousekeepingUC: housekeepingUC,
		SuitesUC:       suitesUC,
		ConfigsUC:      configsUC,
	})

	// Parse OpenAPI 3.1 specification
	var doc struct {
		OpenAPI string                            `yaml:"openapi"`
		Paths   map[string]map[string]interface{} `yaml:"paths"`
	}
	err := yaml.Unmarshal(api.OpenAPISpecYAML, &doc)
	require.NoError(t, err, "OpenAPI specification YAML must be valid syntax")
	assert.Equal(t, "3.1.0", doc.OpenAPI)

	paramRegex := regexp.MustCompile(`:([a-zA-Z0-9_]+)`)

	for _, route := range router.Routes() {
		// Convert Gin route path format (e.g., /api/v1/suites/:id/builds)
		// to OpenAPI path format (e.g., /api/v1/suites/{id}/builds)
		openAPIPath := paramRegex.ReplaceAllString(route.Path, "{$1}")
		method := route.Method

		pathItem, exists := doc.Paths[openAPIPath]
		assert.Truef(t, exists, "Route path %q (method %s) must be declared in OpenAPI spec", openAPIPath, method)
		if exists {
			var opKey string
			switch method {
			case http.MethodGet:
				opKey = "get"
			case http.MethodPost:
				opKey = "post"
			case http.MethodPut:
				opKey = "put"
			case http.MethodDelete:
				opKey = "delete"
			case http.MethodPatch:
				opKey = "patch"
			}
			_, opExists := pathItem[opKey]
			assert.Truef(t, opExists, "Operation %s for path %q must be declared in OpenAPI spec", method, openAPIPath)
		}
	}
}
