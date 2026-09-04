package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/application/service"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// Mock suites repository
type mockSuiteRepo struct {
	suites map[string]*model.TestSuite
}

func newMockSuiteRepo() *mockSuiteRepo {
	return &mockSuiteRepo{suites: make(map[string]*model.TestSuite)}
}
func (m *mockSuiteRepo) Save(_ context.Context, suite *model.TestSuite) error {
	m.suites[suite.ID()] = suite
	return nil
}
func (m *mockSuiteRepo) FindByID(_ context.Context, id string) (*model.TestSuite, error) {
	if s, ok := m.suites[id]; ok {
		return s, nil
	}
	return nil, model.ErrNotFound
}
func (m *mockSuiteRepo) FindByName(_ context.Context, _ string) (*model.TestSuite, error) {
	return nil, model.ErrNotFound
}
func (m *mockSuiteRepo) List(_ context.Context) ([]*model.TestSuite, error) {
	return nil, nil
}
func (m *mockSuiteRepo) Delete(_ context.Context, id string) error {
	delete(m.suites, id)
	return nil
}

// Mock artifacts repository
type mockArtifactRepo struct {
	artifacts map[string]*model.Artifact
}

func newMockArtifactRepo() *mockArtifactRepo {
	return &mockArtifactRepo{artifacts: make(map[string]*model.Artifact)}
}
func (m *mockArtifactRepo) Save(_ context.Context, artifact *model.Artifact) error {
	m.artifacts[artifact.ID()] = artifact
	return nil
}
func (m *mockArtifactRepo) FindByID(_ context.Context, id string) (*model.Artifact, error) {
	if a, ok := m.artifacts[id]; ok {
		return a, nil
	}
	return nil, model.ErrNotFound
}
func (m *mockArtifactRepo) ListBySuiteID(_ context.Context, _ string) ([]*model.Artifact, error) {
	return nil, nil
}
func (m *mockArtifactRepo) Delete(_ context.Context, id string) error {
	delete(m.artifacts, id)
	return nil
}

// Mock configuration repository
type mockConfigRepo struct {
	configs map[string]*model.Configuration
}

func newMockConfigRepo() *mockConfigRepo {
	return &mockConfigRepo{configs: make(map[string]*model.Configuration)}
}
func (m *mockConfigRepo) Save(_ context.Context, config *model.Configuration) error {
	m.configs[config.ID()] = config
	return nil
}
func (m *mockConfigRepo) FindByID(_ context.Context, id string) (*model.Configuration, error) {
	if c, ok := m.configs[id]; ok {
		return c, nil
	}
	return nil, model.ErrNotFound
}
func (m *mockConfigRepo) ListBySuiteID(_ context.Context, _ string) ([]*model.Configuration, error) {
	return nil, nil
}
func (m *mockConfigRepo) Delete(_ context.Context, id string) error {
	delete(m.configs, id)
	return nil
}

// Mock runner profile repository
type mockProfileRepo struct {
	profiles map[string]*model.RunnerProfile
}

func newMockProfileRepo() *mockProfileRepo {
	return &mockProfileRepo{profiles: make(map[string]*model.RunnerProfile)}
}
func (m *mockProfileRepo) Save(_ context.Context, profile *model.RunnerProfile) error {
	m.profiles[profile.ID()] = profile
	return nil
}
func (m *mockProfileRepo) FindByID(_ context.Context, id string) (*model.RunnerProfile, error) {
	if p, ok := m.profiles[id]; ok {
		return p, nil
	}
	return nil, model.ErrNotFound
}
func (m *mockProfileRepo) FindByName(_ context.Context, _ string) (*model.RunnerProfile, error) {
	return nil, model.ErrNotFound
}
func (m *mockProfileRepo) List(_ context.Context) ([]*model.RunnerProfile, error) {
	return nil, nil
}
func (m *mockProfileRepo) Delete(_ context.Context, id string) error {
	delete(m.profiles, id)
	return nil
}

// Mock test run repository
type mockRunRepo struct {
	runs map[string]*model.TestRun
}

func newMockRunRepo() *mockRunRepo {
	return &mockRunRepo{runs: make(map[string]*model.TestRun)}
}
func (m *mockRunRepo) Save(_ context.Context, run *model.TestRun) error {
	m.runs[run.ID()] = run
	return nil
}
func (m *mockRunRepo) FindByID(_ context.Context, id string) (*model.TestRun, error) {
	if r, ok := m.runs[id]; ok {
		return r, nil
	}
	return nil, model.ErrNotFound
}
func (m *mockRunRepo) List(_ context.Context, suiteID string, status model.RunStatus) ([]*model.TestRun, error) {
	var res []*model.TestRun
	for _, r := range m.runs {
		if suiteID != "" && r.SuiteID() != suiteID {
			continue
		}
		if status.IsValid() && r.Status() != status {
			continue
		}
		res = append(res, r)
	}
	return res, nil
}
func (m *mockRunRepo) Delete(_ context.Context, id string) error {
	delete(m.runs, id)
	return nil
}

// Mock runner orchestrator
type mockRunnerOrchestrator struct {
	dispatchedJobs map[string]string
	abortedJobs    map[string]string
	dispatchErr    error
	abortErr       error
}

func newMockRunnerOrchestrator() *mockRunnerOrchestrator {
	return &mockRunnerOrchestrator{
		dispatchedJobs: make(map[string]string),
		abortedJobs:    make(map[string]string),
	}
}
func (m *mockRunnerOrchestrator) DispatchJob(_ context.Context, run *model.TestRun, _ *model.RunnerProfile, _ outbound.RunnerJobOptions) (string, error) {
	if m.dispatchErr != nil {
		return "", m.dispatchErr
	}
	jobName := "vuhive-run-" + run.ID()
	m.dispatchedJobs[run.ID()] = jobName
	return jobName, nil
}
func (m *mockRunnerOrchestrator) AbortJob(_ context.Context, k8sJobName, namespace string) error {
	if m.abortErr != nil {
		return m.abortErr
	}
	m.abortedJobs[k8sJobName] = namespace
	return nil
}

func setupTestRunService(t *testing.T) (
	*service.RunService,
	*mockSuiteRepo,
	*mockArtifactRepo,
	*mockConfigRepo,
	*mockProfileRepo,
	*mockRunRepo,
	*mockRunnerOrchestrator,
	*model.TestSuite,
	*model.Artifact,
	*model.RunnerProfile,
) {
	suiteRepo := newMockSuiteRepo()
	artifactRepo := newMockArtifactRepo()
	configRepo := newMockConfigRepo()
	profileRepo := newMockProfileRepo()
	runRepo := newMockRunRepo()
	orchestrator := newMockRunnerOrchestrator()

	suite, err := model.NewTestSuite("load-tests", "Load test suite")
	require.NoError(t, err)
	require.NoError(t, suite.Activate())
	require.NoError(t, suiteRepo.Save(context.Background(), suite))

	artifact, err := model.NewArtifact(suite.ID(), model.PlatformLinuxAmd64)
	require.NoError(t, err)
	require.NoError(t, artifact.MarkBuilding())
	require.NoError(t, artifact.MarkReady("vuhive-binaries/suite/runner", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	require.NoError(t, artifactRepo.Save(context.Background(), artifact))

	resources, err := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
	require.NoError(t, err)
	profile, err := model.NewRunnerProfile("default", "Default profile", "alpine:3.20", resources, nil, model.Affinity{}, nil)
	require.NoError(t, err)
	require.NoError(t, profileRepo.Save(context.Background(), profile))

	svc := service.NewRunService(suiteRepo, artifactRepo, configRepo, profileRepo, runRepo, orchestrator)

	return svc, suiteRepo, artifactRepo, configRepo, profileRepo, runRepo, orchestrator, suite, artifact, profile
}

func TestRunService_TriggerRun(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully trigger run without configuration", func(t *testing.T) {
		svc, _, _, _, _, runRepo, orchestrator, suite, artifact, profile := setupTestRunService(t)

		cmd := inbound.TriggerRunCommand{
			SuiteID:         suite.ID(),
			ArtifactID:      artifact.ID(),
			RunnerProfileID: profile.ID(),
		}

		run, err := svc.TriggerRun(ctx, cmd)
		require.NoError(t, err)
		require.NotNil(t, run)

		assert.Equal(t, model.RunStatusQueued, run.Status())
		assert.Equal(t, "vuhive-run-"+run.ID(), run.K8sJobName())
		assert.Contains(t, orchestrator.dispatchedJobs, run.ID())

		saved, err := runRepo.FindByID(ctx, run.ID())
		require.NoError(t, err)
		assert.Equal(t, run.ID(), saved.ID())
	})

	t.Run("successfully trigger run with configuration", func(t *testing.T) {
		svc, _, _, configRepo, _, _, _, suite, artifact, profile := setupTestRunService(t)

		cfg, err := model.NewConfiguration(suite.ID(), "staging", "vus: 100", "vuhive-configs/staging.yaml", false)
		require.NoError(t, err)
		require.NoError(t, configRepo.Save(ctx, cfg))

		cfgID := cfg.ID()
		cmd := inbound.TriggerRunCommand{
			SuiteID:         suite.ID(),
			ArtifactID:      artifact.ID(),
			ConfigurationID: &cfgID,
			RunnerProfileID: profile.ID(),
		}

		run, err := svc.TriggerRun(ctx, cmd)
		require.NoError(t, err)
		require.NotNil(t, run)
		require.NotNil(t, run.ConfigurationID())
		assert.Equal(t, cfg.ID(), *run.ConfigurationID())
	})

	t.Run("fail triggering run when suite is not active", func(t *testing.T) {
		svc, suiteRepo, _, _, _, _, _, _, artifact, profile := setupTestRunService(t)

		draftSuite, err := model.NewTestSuite("draft-suite", "desc")
		require.NoError(t, err)
		require.NoError(t, suiteRepo.Save(ctx, draftSuite))

		cmd := inbound.TriggerRunCommand{
			SuiteID:         draftSuite.ID(),
			ArtifactID:      artifact.ID(),
			RunnerProfileID: profile.ID(),
		}

		_, err = svc.TriggerRun(ctx, cmd)
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})

	t.Run("fail triggering run when artifact is not ready", func(t *testing.T) {
		svc, _, artifactRepo, _, _, _, _, suite, _, profile := setupTestRunService(t)

		pendingArt, err := model.NewArtifact(suite.ID(), model.PlatformLinuxAmd64)
		require.NoError(t, err)
		require.NoError(t, artifactRepo.Save(ctx, pendingArt))

		cmd := inbound.TriggerRunCommand{
			SuiteID:         suite.ID(),
			ArtifactID:      pendingArt.ID(),
			RunnerProfileID: profile.ID(),
		}

		_, err = svc.TriggerRun(ctx, cmd)
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})

	t.Run("fail triggering run when configuration belongs to another suite", func(t *testing.T) {
		svc, _, _, configRepo, _, _, _, suite, artifact, profile := setupTestRunService(t)

		otherCfg, err := model.NewConfiguration("other-suite", "prod", "vus: 10", "key.yaml", false)
		require.NoError(t, err)
		require.NoError(t, configRepo.Save(ctx, otherCfg))

		otherCfgID := otherCfg.ID()
		cmd := inbound.TriggerRunCommand{
			SuiteID:         suite.ID(),
			ArtifactID:      artifact.ID(),
			ConfigurationID: &otherCfgID,
			RunnerProfileID: profile.ID(),
		}

		_, err = svc.TriggerRun(ctx, cmd)
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("fail triggering run when orchestrator fails to dispatch", func(t *testing.T) {
		svc, _, _, _, _, runRepo, orchestrator, suite, artifact, profile := setupTestRunService(t)

		orchestrator.dispatchErr = errors.New("k8s api error")

		cmd := inbound.TriggerRunCommand{
			SuiteID:         suite.ID(),
			ArtifactID:      artifact.ID(),
			RunnerProfileID: profile.ID(),
		}

		_, err := svc.TriggerRun(ctx, cmd)
		assert.Error(t, err)

		// Run was marked FAILED and saved
		runs, err := runRepo.List(ctx, suite.ID(), model.RunStatusFailed)
		require.NoError(t, err)
		assert.Len(t, runs, 1)
	})
}

func TestRunService_GetListAndAbort(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _, _, runRepo, orchestrator, suite, artifact, profile := setupTestRunService(t)

	run, err := svc.TriggerRun(ctx, inbound.TriggerRunCommand{
		SuiteID:         suite.ID(),
		ArtifactID:      artifact.ID(),
		RunnerProfileID: profile.ID(),
	})
	require.NoError(t, err)

	t.Run("get run by id", func(t *testing.T) {
		got, err := svc.GetRun(ctx, run.ID())
		require.NoError(t, err)
		assert.Equal(t, run.ID(), got.ID())
	})

	t.Run("get run not found", func(t *testing.T) {
		_, err := svc.GetRun(ctx, "non-existent")
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("list runs", func(t *testing.T) {
		runs, err := svc.ListRuns(ctx, suite.ID(), model.RunStatusQueued)
		require.NoError(t, err)
		assert.Len(t, runs, 1)
	})

	t.Run("abort run successfully", func(t *testing.T) {
		// Set to RUNNING first
		require.NoError(t, run.Start("vuhive-run-"+run.ID(), time.Now().UTC()))
		require.NoError(t, runRepo.Save(ctx, run))

		err := svc.AbortRun(ctx, run.ID(), "user cancellation")
		require.NoError(t, err)

		aborted, err := svc.GetRun(ctx, run.ID())
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusAborted, aborted.Status())
		assert.Equal(t, "user cancellation", aborted.AbortReason())
		assert.Contains(t, orchestrator.abortedJobs, "vuhive-run-"+run.ID())
	})

	t.Run("abort already terminal run fails", func(t *testing.T) {
		err := svc.AbortRun(ctx, run.ID(), "again")
		assert.ErrorIs(t, err, model.ErrTerminalState)
	})
}
