package service_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/application/service"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHousekeepingStorage struct {
	deletedKeys    []string
	objects        map[string]outbound.ObjectInfo
	lifecycleRules []outbound.LifecycleRule
}

func newMockHousekeepingStorage() *mockHousekeepingStorage {
	return &mockHousekeepingStorage{
		deletedKeys: make([]string, 0),
		objects:     make(map[string]outbound.ObjectInfo),
	}
}

func (m *mockHousekeepingStorage) Upload(_ context.Context, key string, _ io.Reader, size int64, _ string) error {
	m.objects[key] = outbound.ObjectInfo{Key: key, Size: size, LastModified: time.Now()}
	return nil
}
func (m *mockHousekeepingStorage) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockHousekeepingStorage) Delete(_ context.Context, key string) error {
	m.deletedKeys = append(m.deletedKeys, key)
	delete(m.objects, key)
	return nil
}
func (m *mockHousekeepingStorage) Exists(_ context.Context, key string) (bool, error) {
	_, ok := m.objects[key]
	return ok, nil
}
func (m *mockHousekeepingStorage) PresignDownload(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}
func (m *mockHousekeepingStorage) PresignUpload(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}
func (m *mockHousekeepingStorage) EnsureBucket(_ context.Context) error {
	return nil
}
func (m *mockHousekeepingStorage) ListObjects(_ context.Context, prefix string) ([]outbound.ObjectInfo, error) {
	var res []outbound.ObjectInfo
	for k, v := range m.objects {
		if prefix == "" || len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			res = append(res, v)
		}
	}
	return res, nil
}
func (m *mockHousekeepingStorage) PutBucketLifecycleConfiguration(_ context.Context, rules []outbound.LifecycleRule) error {
	m.lifecycleRules = rules
	return nil
}

type mockHousekeepingRunRepo struct {
	runs          map[string]*model.TestRun
	archivedRuns  []string
	clearedLogs   []string
	clearedReport []string
	deletedBatch  []string
}

func newMockHousekeepingRunRepo() *mockHousekeepingRunRepo {
	return &mockHousekeepingRunRepo{
		runs: make(map[string]*model.TestRun),
	}
}

func (m *mockHousekeepingRunRepo) Save(_ context.Context, run *model.TestRun) error {
	m.runs[run.ID()] = run
	return nil
}
func (m *mockHousekeepingRunRepo) FindByID(_ context.Context, id string) (*model.TestRun, error) {
	if r, ok := m.runs[id]; ok {
		return r, nil
	}
	return nil, model.ErrNotFound
}
func (m *mockHousekeepingRunRepo) FindByK8sJobName(_ context.Context, _ string) (*model.TestRun, error) {
	return nil, model.ErrNotFound
}
func (m *mockHousekeepingRunRepo) List(_ context.Context, _ string, _ model.RunStatus) ([]*model.TestRun, error) {
	return nil, nil
}
func (m *mockHousekeepingRunRepo) ListFiltered(_ context.Context, _ model.RunFilter) ([]*model.TestRun, int64, error) {
	return nil, 0, nil
}
func (m *mockHousekeepingRunRepo) Delete(_ context.Context, id string) error {
	delete(m.runs, id)
	return nil
}
func (m *mockHousekeepingRunRepo) ListExpiredRunsForLogs(_ context.Context, before time.Time, suiteID *string, limit int) ([]*model.TestRun, error) {
	var res []*model.TestRun
	for _, r := range m.runs {
		if suiteID != nil && *suiteID != "" && r.SuiteID() != *suiteID {
			continue
		}
		refTime := r.CreatedAt()
		if r.FinishedAt() != nil {
			refTime = *r.FinishedAt()
		}
		if refTime.Before(before) && r.S3LogsKey() != "" && r.Status().IsTerminal() {
			res = append(res, r)
		}
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}
func (m *mockHousekeepingRunRepo) ListExpiredRunsForReports(_ context.Context, before time.Time, suiteID *string, limit int) ([]*model.TestRun, error) {
	var res []*model.TestRun
	for _, r := range m.runs {
		if suiteID != nil && *suiteID != "" && r.SuiteID() != *suiteID {
			continue
		}
		refTime := r.CreatedAt()
		if r.FinishedAt() != nil {
			refTime = *r.FinishedAt()
		}
		if refTime.Before(before) && (r.S3ReportKey() != "" || len(r.SummaryJSON()) > 0) && r.Status() != model.RunStatusArchived && r.Status().IsTerminal() {
			res = append(res, r)
		}
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}
func (m *mockHousekeepingRunRepo) ListExpiredRunsForPrune(_ context.Context, before time.Time, suiteID *string, limit int) ([]*model.TestRun, error) {
	var res []*model.TestRun
	for _, r := range m.runs {
		if suiteID != nil && *suiteID != "" && r.SuiteID() != *suiteID {
			continue
		}
		refTime := r.CreatedAt()
		if r.FinishedAt() != nil {
			refTime = *r.FinishedAt()
		}
		if refTime.Before(before) && r.Status().IsTerminal() {
			res = append(res, r)
		}
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}
func (m *mockHousekeepingRunRepo) ClearLogsKey(_ context.Context, id string) error {
	m.clearedLogs = append(m.clearedLogs, id)
	if r, ok := m.runs[id]; ok {
		r.ClearS3LogsKey()
	}
	return nil
}
func (m *mockHousekeepingRunRepo) ClearReportKey(_ context.Context, id string) error {
	m.clearedReport = append(m.clearedReport, id)
	if r, ok := m.runs[id]; ok {
		r.ClearS3ReportKey()
	}
	return nil
}
func (m *mockHousekeepingRunRepo) ArchiveRun(_ context.Context, id string) error {
	m.archivedRuns = append(m.archivedRuns, id)
	if r, ok := m.runs[id]; ok {
		return r.Archive(time.Now())
	}
	return nil
}
func (m *mockHousekeepingRunRepo) DeleteBatch(_ context.Context, ids []string) (int64, error) {
	m.deletedBatch = append(m.deletedBatch, ids...)
	for _, id := range ids {
		delete(m.runs, id)
	}
	return int64(len(ids)), nil
}

type mockHousekeepingArtifactRepo struct {
	artifacts    map[string]*model.Artifact
	clearedKeys  []string
	deletedBatch []string
}

func newMockHousekeepingArtifactRepo() *mockHousekeepingArtifactRepo {
	return &mockHousekeepingArtifactRepo{
		artifacts: make(map[string]*model.Artifact),
	}
}

func (m *mockHousekeepingArtifactRepo) Save(_ context.Context, a *model.Artifact) error {
	m.artifacts[a.ID()] = a
	return nil
}
func (m *mockHousekeepingArtifactRepo) FindByID(_ context.Context, id string) (*model.Artifact, error) {
	if a, ok := m.artifacts[id]; ok {
		return a, nil
	}
	return nil, model.ErrNotFound
}
func (m *mockHousekeepingArtifactRepo) ListBySuiteID(_ context.Context, _ string) ([]*model.Artifact, error) {
	return nil, nil
}
func (m *mockHousekeepingArtifactRepo) Delete(_ context.Context, id string) error {
	m.deletedBatch = append(m.deletedBatch, id)
	delete(m.artifacts, id)
	return nil
}
func (m *mockHousekeepingArtifactRepo) ListOrphanedArtifacts(_ context.Context, before time.Time, _ int) ([]*model.Artifact, error) {
	var res []*model.Artifact
	for _, a := range m.artifacts {
		if a.CreatedAt().Before(before) && (a.Status() == model.ArtifactStatusFailed || a.Status() == model.ArtifactStatusPending) {
			res = append(res, a)
		}
	}
	return res, nil
}
func (m *mockHousekeepingArtifactRepo) ListExpiredArtifacts(_ context.Context, before time.Time, _ *string, _ int) ([]*model.Artifact, error) {
	var res []*model.Artifact
	for _, a := range m.artifacts {
		if a.CreatedAt().Before(before) && a.S3BinaryKey() != "" {
			res = append(res, a)
		}
	}
	return res, nil
}
func (m *mockHousekeepingArtifactRepo) ClearBinaryKeys(_ context.Context, id string) error {
	m.clearedKeys = append(m.clearedKeys, id)
	return nil
}

func TestHousekeepingService_Execute(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	suiteRepo := newMockSuiteRepo()
	artifactRepo := newMockHousekeepingArtifactRepo()
	runRepo := newMockHousekeepingRunRepo()
	storage := newMockHousekeepingStorage()

	policy := model.RetentionPolicy{
		LogsTTLDays:     14,
		ReportsTTLDays:  90,
		SourcesTTLDays:  30,
		BinariesTTLDays: 30,
		RunsTTLDays:     180,
	}

	svc := service.NewHousekeepingService(suiteRepo, artifactRepo, runRepo, storage, policy)

	// Create test suite
	suite, err := model.NewTestSuite("test-suite", "desc")
	require.NoError(t, err)
	require.NoError(t, suiteRepo.Save(ctx, suite))

	// 1. Old completed run with logs and reports (20 days old -> logs expired, reports not expired, not pruned)
	oldRunTime := now.Add(-20 * 24 * time.Hour)
	run1, err := model.NewTestRunWithID(
		"run-1", suite.ID(), "art-1", nil, "prof-1", nil,
		model.RunStatusCompleted, "job-1", "ns",
		&oldRunTime, &oldRunTime, nil, nil,
		model.RunMetrics{TotalRequests: 100}, "runs/run-1/summary.json", "runs/run-1/run.log",
		[]byte(`{"summary":true}`), "", oldRunTime,
	)
	require.NoError(t, err)
	require.NoError(t, runRepo.Save(ctx, run1))
	storage.objects["runs/run-1/run.log"] = outbound.ObjectInfo{Key: "runs/run-1/run.log"}
	storage.objects["runs/run-1/summary.json"] = outbound.ObjectInfo{Key: "runs/run-1/summary.json"}

	// 2. Very old run (100 days old -> logs expired, reports expired & archived, not yet 180d)
	veryOldRunTime := now.Add(-100 * 24 * time.Hour)
	run2, err := model.NewTestRunWithID(
		"run-2", suite.ID(), "art-1", nil, "prof-1", nil,
		model.RunStatusCompleted, "job-2", "ns",
		&veryOldRunTime, &veryOldRunTime, nil, nil,
		model.RunMetrics{TotalRequests: 500}, "runs/run-2/summary.json", "runs/run-2/run.log",
		[]byte(`{"summary":true}`), "", veryOldRunTime,
	)
	require.NoError(t, err)
	require.NoError(t, runRepo.Save(ctx, run2))
	storage.objects["runs/run-2/run.log"] = outbound.ObjectInfo{Key: "runs/run-2/run.log"}
	storage.objects["runs/run-2/summary.json"] = outbound.ObjectInfo{Key: "runs/run-2/summary.json"}

	// 3. Ancient run (200 days old -> past 180 days -> pruned)
	ancientRunTime := now.Add(-200 * 24 * time.Hour)
	run3, err := model.NewTestRunWithID(
		"run-3", suite.ID(), "art-1", nil, "prof-1", nil,
		model.RunStatusCompleted, "job-3", "ns",
		&ancientRunTime, &ancientRunTime, nil, nil,
		model.RunMetrics{TotalRequests: 999}, "runs/run-3/summary.json", "runs/run-3/run.log",
		[]byte(`{"summary":true}`), "", ancientRunTime,
	)
	require.NoError(t, err)
	require.NoError(t, runRepo.Save(ctx, run3))

	// 4. Orphaned failed artifact (48h old, status FAILED)
	orphanTime := now.Add(-48 * time.Hour)
	orphanArt, err := model.NewArtifactWithID(
		"art-orphan", suite.ID(), model.PlatformLinuxAmd64,
		"", "", "suites/test-suite/artifacts/art-orphan/build.log",
		model.ArtifactStatusFailed, "compilation failed", orphanTime,
	)
	require.NoError(t, err)
	require.NoError(t, artifactRepo.Save(ctx, orphanArt))
	storage.objects["suites/test-suite/artifacts/art-orphan/build.log"] = outbound.ObjectInfo{Key: "suites/test-suite/artifacts/art-orphan/build.log"}

	// 5. Expired binary artifact (40 days old, status READY)
	expBinaryTime := now.Add(-40 * 24 * time.Hour)
	expArt, err := model.NewArtifactWithID(
		"art-expired", suite.ID(), model.PlatformLinuxAmd64,
		"suites/test-suite/artifacts/art-expired/linux-amd64/runner", "sha256", "build.log",
		model.ArtifactStatusReady, "", expBinaryTime,
	)
	require.NoError(t, err)
	require.NoError(t, artifactRepo.Save(ctx, expArt))
	storage.objects["suites/test-suite/artifacts/art-expired/linux-amd64/runner"] = outbound.ObjectInfo{Key: "suites/test-suite/artifacts/art-expired/linux-amd64/runner"}

	t.Run("dry run audits without mutating", func(t *testing.T) {
		res, err := svc.RunHousekeeping(ctx, inbound.HousekeepingCommand{DryRun: true})
		require.NoError(t, err)
		assert.True(t, res.PurgedLogsCount >= 2)
		assert.True(t, res.PurgedReportsCount >= 1)
		assert.True(t, res.PrunedRunsCount >= 1)
		assert.True(t, res.PurgedArtifactsCount >= 1)

		// Assert no actual deletes occurred
		assert.Empty(t, storage.deletedKeys)
		assert.Empty(t, runRepo.deletedBatch)
		assert.Empty(t, runRepo.archivedRuns)
	})

	t.Run("live execution purges and updates state", func(t *testing.T) {
		res, err := svc.RunHousekeeping(ctx, inbound.HousekeepingCommand{DryRun: false})
		require.NoError(t, err)
		assert.True(t, res.PurgedLogsCount >= 2)
		assert.True(t, res.PurgedReportsCount >= 1)
		assert.True(t, res.PrunedRunsCount >= 1)
		assert.True(t, res.PurgedArtifactsCount >= 1)

		// S3 objects purged
		assert.Contains(t, storage.deletedKeys, "runs/run-1/run.log")
		assert.Contains(t, storage.deletedKeys, "runs/run-2/run.log")
		assert.Contains(t, storage.deletedKeys, "runs/run-2/summary.json")
		assert.Contains(t, storage.deletedKeys, "suites/test-suite/artifacts/art-orphan/build.log")
		assert.Contains(t, storage.deletedKeys, "suites/test-suite/artifacts/art-expired/linux-amd64/runner")

		// DB updates
		assert.Contains(t, runRepo.archivedRuns, "run-2")
		assert.Contains(t, runRepo.deletedBatch, "run-3")
		assert.Contains(t, artifactRepo.deletedBatch, "art-orphan")
		assert.Contains(t, artifactRepo.clearedKeys, "art-expired")
	})

	t.Run("configure bucket lifecycle rules", func(t *testing.T) {
		err := svc.ConfigureBucketLifecycle(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, storage.lifecycleRules)
		assert.Equal(t, 14, storage.lifecycleRules[0].ExpirationDays)
	})
}
