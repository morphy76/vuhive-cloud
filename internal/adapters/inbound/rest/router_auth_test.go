package rest_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRouter_RBACProtection(t *testing.T) {
	mockVerifier := new(MockTokenVerifier)
	mockBuilds := new(MockBuildsUseCase)
	mockProfiles := new(MockProfilesUseCase)
	mockSchedules := &mockSchedulesUseCase{}
	mockRuns := &mockRunsUseCase{}
	mockBarrier := &mockBarrierUseCase{}
	mockHK := new(MockHousekeepingUseCase)
	mockSuites := new(MockSuitesUseCase)
	mockConfigs := new(MockConfigsUseCase)

	devClaims := model.NewClaims("dev-1", "dev", "dev@example.com", []string{model.RoleDeveloper}, nil, time.Now().Add(time.Hour))
	depClaims := model.NewClaims("dep-1", "dep", "dep@example.com", []string{model.RoleDeployer}, nil, time.Now().Add(time.Hour))
	adminClaims := model.NewClaims("adm-1", "adm", "adm@example.com", []string{model.RoleAdmin}, nil, time.Now().Add(time.Hour))
	runnerClaims := model.NewClaims("run-1", "runner", "run@example.com", []string{model.RoleRunner}, nil, time.Now().Add(time.Hour))
	viewerClaims := model.NewClaims("view-1", "viewer", "v@example.com", []string{model.RoleViewer}, nil, time.Now().Add(time.Hour))

	mockVerifier.On("VerifyToken", mock.Anything, "dev-token").Return(devClaims, nil)
	mockVerifier.On("VerifyToken", mock.Anything, "dep-token").Return(depClaims, nil)
	mockVerifier.On("VerifyToken", mock.Anything, "admin-token").Return(adminClaims, nil)
	mockVerifier.On("VerifyToken", mock.Anything, "runner-token").Return(runnerClaims, nil)
	mockVerifier.On("VerifyToken", mock.Anything, "viewer-token").Return(viewerClaims, nil)

	router := rest.SetupRouterWithConfig(rest.RouterConfig{
		BuildsUC:       mockBuilds,
		ProfilesUC:     mockProfiles,
		SchedulesUC:    mockSchedules,
		RunsUC:         mockRuns,
		BarrierUC:      mockBarrier,
		HousekeepingUC: mockHK,
		SuitesUC:       mockSuites,
		ConfigsUC:      mockConfigs,
		TokenVerifier:  mockVerifier,
	})

	t.Run("public endpoints bypass authentication", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		req2, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)

		req3, _ := http.NewRequest(http.MethodGet, "/api/version", nil)
		w3 := httptest.NewRecorder()
		router.ServeHTTP(w3, req3)
		assert.Equal(t, http.StatusOK, w3.Code)

		req4, _ := http.NewRequest(http.MethodGet, "/api/openapi.yaml", nil)
		w4 := httptest.NewRecorder()
		router.ServeHTTP(w4, req4)
		assert.Equal(t, http.StatusOK, w4.Code)

		req5, _ := http.NewRequest(http.MethodGet, "/api/openapi.json", nil)
		w5 := httptest.NewRecorder()
		router.ServeHTTP(w5, req5)
		assert.Equal(t, http.StatusOK, w5.Code)
	})

	t.Run("unauthenticated request to protected endpoint returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/runs", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("developer can upload suite build but cannot trigger runs", func(t *testing.T) {
		mockBuilds.On("TriggerBuildWithOptions", mock.Anything, "suite-1", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]*model.Artifact{}, nil).Once()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("source", "source.tar.gz")
		_, _ = part.Write([]byte("fake-tar"))
		_ = writer.WriteField("platform", "linux/amd64")
		_ = writer.Close()

		// Developer uploads suite
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/suites/suite-1/builds", body)
		req.Header.Set("Authorization", "Bearer dev-token")
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusAccepted, w.Code)

		// Developer attempts to trigger a run -> 403 Forbidden
		runReq, _ := http.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader([]byte(`{"suite_id":"suite-1","profile_id":"prof-1"}`)))
		runReq.Header.Set("Authorization", "Bearer dev-token")
		runReq.Header.Set("Content-Type", "application/json")
		wRun := httptest.NewRecorder()
		router.ServeHTTP(wRun, runReq)
		assert.Equal(t, http.StatusForbidden, wRun.Code)
		assert.Contains(t, wRun.Body.String(), "forbidden")
	})

	t.Run("deployer can trigger run but cannot create profile", func(t *testing.T) {
		testRun, _ := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		mockRuns.triggerRunFunc = func(ctx context.Context, cmd inbound.TriggerRunCommand) (*model.TestRun, error) {
			return testRun, nil
		}

		runReq, _ := http.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader([]byte(`{"suite_id":"suite-1","artifact_id":"art-1","runner_profile_id":"prof-1"}`)))
		runReq.Header.Set("Authorization", "Bearer dep-token")
		runReq.Header.Set("Content-Type", "application/json")
		wRun := httptest.NewRecorder()
		router.ServeHTTP(wRun, runReq)
		assert.Equal(t, http.StatusCreated, wRun.Code)

		// Deployer attempts to create a runner profile -> 403 Forbidden
		profReq, _ := http.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader([]byte(`{"name":"large"}`)))
		profReq.Header.Set("Authorization", "Bearer dep-token")
		profReq.Header.Set("Content-Type", "application/json")
		wProf := httptest.NewRecorder()
		router.ServeHTTP(wProf, profReq)
		assert.Equal(t, http.StatusForbidden, wProf.Code)
	})

	t.Run("runner can complete run and await barrier", func(t *testing.T) {
		mockRuns.completeRunFunc = func(ctx context.Context, cmd inbound.CompleteRunCommand) (*model.TestRun, error) {
			testRun, _ := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
			return testRun, nil
		}

		completeReq, _ := http.NewRequest(http.MethodPost, "/api/v1/runs/complete", bytes.NewReader([]byte(`{"run_id":"run-1","exit_code":0}`)))
		completeReq.Header.Set("Authorization", "Bearer runner-token")
		completeReq.Header.Set("Content-Type", "application/json")
		wComp := httptest.NewRecorder()
		router.ServeHTTP(wComp, completeReq)
		assert.Equal(t, http.StatusOK, wComp.Code)

		// Runner cannot trigger runs
		runReq, _ := http.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader([]byte(`{"suite_id":"suite-1"}`)))
		runReq.Header.Set("Authorization", "Bearer runner-token")
		runReq.Header.Set("Content-Type", "application/json")
		wRun := httptest.NewRecorder()
		router.ServeHTTP(wRun, runReq)
		assert.Equal(t, http.StatusForbidden, wRun.Code)
	})

	t.Run("admin can create runner profile", func(t *testing.T) {
		res, _ := model.NewResourceRequirements("100m", "500m", "128Mi", "512Mi")
		profile, _ := model.NewRunnerProfile("large", "test", "alpine:latest", res, nil, model.Affinity{}, nil)
		mockProfiles.On("CreateProfile", mock.Anything, mock.Anything).Return(profile, nil).Once()

		profReq, _ := http.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader([]byte(`{
			"name": "large",
			"resources": {"cpu_request":"100m","cpu_limit":"500m","memory_request":"128Mi","memory_limit":"512Mi"}
		}`)))
		profReq.Header.Set("Authorization", "Bearer admin-token")
		profReq.Header.Set("Content-Type", "application/json")
		wProf := httptest.NewRecorder()
		router.ServeHTTP(wProf, profReq)
		assert.Equal(t, http.StatusCreated, wProf.Code)
	})

	t.Run("developer can create suite and configuration", func(t *testing.T) {
		suite, _ := model.NewTestSuite("dev-suite", "desc")
		mockSuites.On("CreateSuite", mock.Anything, "dev-suite", "desc").Return(suite, nil).Once()

		cfg, _ := model.NewConfiguration(suite.ID(), "default", "vus: 1", "key", true)
		mockConfigs.On("CreateConfig", mock.Anything, inbound.CreateConfigCommand{
			SuiteID:     suite.ID(),
			Name:        "default",
			ContentYAML: "vus: 1",
			IsDefault:   true,
		}).Return(cfg, nil).Once()

		// Developer creates suite
		suiteReq, _ := http.NewRequest(http.MethodPost, "/api/v1/suites", bytes.NewReader([]byte(`{"name":"dev-suite","description":"desc"}`)))
		suiteReq.Header.Set("Authorization", "Bearer dev-token")
		suiteReq.Header.Set("Content-Type", "application/json")
		wSuite := httptest.NewRecorder()
		router.ServeHTTP(wSuite, suiteReq)
		assert.Equal(t, http.StatusCreated, wSuite.Code)

		// Developer creates configuration
		cfgReq, _ := http.NewRequest(http.MethodPost, "/api/v1/suites/"+suite.ID()+"/configs", bytes.NewReader([]byte(`{"name":"default","content_yaml":"vus: 1","is_default":true}`)))
		cfgReq.Header.Set("Authorization", "Bearer dev-token")
		cfgReq.Header.Set("Content-Type", "application/json")
		wCfg := httptest.NewRecorder()
		router.ServeHTTP(wCfg, cfgReq)
		assert.Equal(t, http.StatusCreated, wCfg.Code)
	})

	t.Run("viewer cannot create suite", func(t *testing.T) {
		suiteReq, _ := http.NewRequest(http.MethodPost, "/api/v1/suites", bytes.NewReader([]byte(`{"name":"viewer-suite"}`)))
		suiteReq.Header.Set("Authorization", "Bearer viewer-token")
		suiteReq.Header.Set("Content-Type", "application/json")
		wSuite := httptest.NewRecorder()
		router.ServeHTTP(wSuite, suiteReq)
		assert.Equal(t, http.StatusForbidden, wSuite.Code)
	})
}
