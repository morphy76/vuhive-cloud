//go:build integration

package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/postgres"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPostgresContainer(t *testing.T) (*sql.DB, *pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("vuhivedb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "failed to start postgres container")

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	sqlDB, err := sql.Open("pgx", connStr)
	require.NoError(t, err, "failed to open sql.DB")

	poolConfig, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err, "failed to parse pool config")

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err, "failed to create pgxpool")

	cleanup := func() {
		pool.Close()
		_ = sqlDB.Close()
		_ = pgContainer.Terminate(ctx)
	}

	return sqlDB, pool, cleanup
}

func TestPostgresMigrations_Idempotency(t *testing.T) {
	ctx := context.Background()
	sqlDB, pool, cleanup := setupPostgresContainer(t)
	defer cleanup()

	// 1. Initial MigrateUp
	err := postgres.MigrateUp(ctx, sqlDB)
	require.NoError(t, err, "first MigrateUp should succeed")

	// 2. Second MigrateUp (Idempotency)
	err = postgres.MigrateUp(ctx, sqlDB)
	require.NoError(t, err, "subsequent MigrateUp should succeed idempotently")

	// Verify tables exist
	var count int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_name = 'test_suites'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// 3. MigrateReset (Down all)
	err = postgres.MigrateReset(ctx, sqlDB)
	require.NoError(t, err, "MigrateReset should roll back cleanly")

	err = pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_name = 'test_suites'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// 4. Re-apply MigrateUp
	err = postgres.MigrateUp(ctx, sqlDB)
	require.NoError(t, err, "MigrateUp after reset should succeed")
}

func TestPostgresMigrations_MigrateUpURL(t *testing.T) {
	ctx := context.Background()
	pgContainer, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("vuhivedb_url"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "failed to start postgres container")
	defer func() { _ = pgContainer.Terminate(ctx) }()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	err = postgres.MigrateUpURL(ctx, connStr)
	require.NoError(t, err, "MigrateUpURL should succeed on fresh database")
}

func TestRepositories_FullCRUD(t *testing.T) {
	ctx := context.Background()
	sqlDB, pool, cleanup := setupPostgresContainer(t)
	defer cleanup()

	require.NoError(t, postgres.MigrateUp(ctx, sqlDB))

	suiteRepo := postgres.NewTestSuiteRepository(pool)
	artifactRepo := postgres.NewArtifactRepository(pool)
	configRepo := postgres.NewConfigurationRepository(pool)
	profileRepo := postgres.NewRunnerProfileRepository(pool)
	scheduleRepo := postgres.NewScheduleRepository(pool)
	runRepo := postgres.NewTestRunRepository(pool)

	t.Run("TestSuiteRepository CRUD & Constraints", func(t *testing.T) {
		suite, err := model.NewTestSuite("checkout-load-test", "E-commerce checkout stress test")
		require.NoError(t, err)

		// 1. Save (Insert)
		err = suiteRepo.Save(ctx, suite)
		require.NoError(t, err)

		// 2. FindByID
		found, err := suiteRepo.FindByID(ctx, suite.ID())
		require.NoError(t, err)
		assert.Equal(t, suite.ID(), found.ID())
		assert.Equal(t, suite.Name(), found.Name())
		assert.Equal(t, suite.Description(), found.Description())
		assert.Equal(t, model.TestSuiteStateDraft, found.State())

		// 3. FindByName
		foundByName, err := suiteRepo.FindByName(ctx, "checkout-load-test")
		require.NoError(t, err)
		assert.Equal(t, suite.ID(), foundByName.ID())

		// 4. Update
		require.NoError(t, suite.Activate())
		require.NoError(t, suite.UpdateDetails("checkout-load-test-v2", "Updated description"))
		err = suiteRepo.Save(ctx, suite)
		require.NoError(t, err)

		updated, err := suiteRepo.FindByID(ctx, suite.ID())
		require.NoError(t, err)
		assert.Equal(t, "checkout-load-test-v2", updated.Name())
		assert.Equal(t, model.TestSuiteStateActive, updated.State())

		// 5. Unique Constraint on Name -> ErrConflict
		duplicate, err := model.NewTestSuite("checkout-load-test-v2", "Duplicate")
		require.NoError(t, err)
		err = suiteRepo.Save(ctx, duplicate)
		assert.ErrorIs(t, err, model.ErrConflict)

		// 6. List
		suites, err := suiteRepo.List(ctx)
		require.NoError(t, err)
		assert.Len(t, suites, 1)

		// 7. Delete non-existent -> ErrNotFound
		err = suiteRepo.Delete(ctx, "00000000-0000-0000-0000-000000000000")
		assert.ErrorIs(t, err, model.ErrNotFound)

		err = suiteRepo.Delete(ctx, "non-existent-id")
		assert.ErrorIs(t, err, model.ErrNotFound)

		// 8. Delete
		err = suiteRepo.Delete(ctx, suite.ID())
		require.NoError(t, err)

		_, err = suiteRepo.FindByID(ctx, suite.ID())
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("ArtifactRepository CRUD", func(t *testing.T) {
		suite, err := model.NewTestSuite("artifact-suite", "desc")
		require.NoError(t, err)
		require.NoError(t, suiteRepo.Save(ctx, suite))

		artifact, err := model.NewArtifact(suite.ID(), model.PlatformLinuxAmd64)
		require.NoError(t, err)

		// 1. Save PENDING artifact
		err = artifactRepo.Save(ctx, artifact)
		require.NoError(t, err)

		// 2. FindByID
		found, err := artifactRepo.FindByID(ctx, artifact.ID())
		require.NoError(t, err)
		assert.Equal(t, artifact.ID(), found.ID())
		assert.Equal(t, model.ArtifactStatusPending, found.Status())
		assert.Equal(t, model.PlatformLinuxAmd64, found.Platform())

		// 3. MarkBuilding & Save
		require.NoError(t, artifact.MarkBuilding())
		require.NoError(t, artifactRepo.Save(ctx, artifact))

		foundBuilding, err := artifactRepo.FindByID(ctx, artifact.ID())
		require.NoError(t, err)
		assert.Equal(t, model.ArtifactStatusBuilding, foundBuilding.Status())

		// 4. MarkReady & Save
		checksum := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		require.NoError(t, artifact.MarkReady("s3://bucket/binaries/suite.bin", checksum))
		require.NoError(t, artifactRepo.Save(ctx, artifact))

		foundReady, err := artifactRepo.FindByID(ctx, artifact.ID())
		require.NoError(t, err)
		assert.Equal(t, model.ArtifactStatusReady, foundReady.Status())
		assert.Equal(t, "s3://bucket/binaries/suite.bin", foundReady.S3BinaryKey())
		assert.Equal(t, checksum, foundReady.SHA256Checksum())

		// 5. ListBySuiteID
		list, err := artifactRepo.ListBySuiteID(ctx, suite.ID())
		require.NoError(t, err)
		assert.Len(t, list, 1)

		// 6. Delete
		err = artifactRepo.Delete(ctx, artifact.ID())
		require.NoError(t, err)

		_, err = artifactRepo.FindByID(ctx, artifact.ID())
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("ConfigurationRepository CRUD & Constraints", func(t *testing.T) {
		suite, err := model.NewTestSuite("config-suite", "desc")
		require.NoError(t, err)
		require.NoError(t, suiteRepo.Save(ctx, suite))

		config, err := model.NewConfiguration(suite.ID(), "default-config", "vus: 100\nduration: 5m\n", "s3://bucket/configs/default.yaml", true)
		require.NoError(t, err)

		// 1. Save
		err = configRepo.Save(ctx, config)
		require.NoError(t, err)

		// 2. FindByID
		found, err := configRepo.FindByID(ctx, config.ID())
		require.NoError(t, err)
		assert.Equal(t, config.ID(), found.ID())
		assert.Equal(t, "default-config", found.Name())
		assert.True(t, found.IsDefault())

		// 3. Unique (suite_id, name) collision -> ErrConflict
		dupConfig, err := model.NewConfiguration(suite.ID(), "default-config", "vus: 50\n", "s3://bucket/configs/dup.yaml", false)
		require.NoError(t, err)
		err = configRepo.Save(ctx, dupConfig)
		assert.ErrorIs(t, err, model.ErrConflict)

		// 4. Update
		require.NoError(t, config.UpdateContent("vus: 200\nduration: 10m\n", "s3://bucket/configs/updated.yaml"))
		config.SetDefault(false)
		require.NoError(t, configRepo.Save(ctx, config))

		updated, err := configRepo.FindByID(ctx, config.ID())
		require.NoError(t, err)
		assert.Equal(t, "vus: 200\nduration: 10m\n", updated.ContentYAML())
		assert.False(t, updated.IsDefault())

		// 5. ListBySuiteID
		list, err := configRepo.ListBySuiteID(ctx, suite.ID())
		require.NoError(t, err)
		assert.Len(t, list, 1)

		// 6. Delete
		require.NoError(t, configRepo.Delete(ctx, config.ID()))
		_, err = configRepo.FindByID(ctx, config.ID())
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("RunnerProfileRepository CRUD & JSONB", func(t *testing.T) {
		res, err := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
		require.NoError(t, err)

		nodeSelector := map[string]string{"node.kubernetes.io/instance-type": "c6i.xlarge"}
		affinity := model.Affinity{
			NodeSelectorTerms: []model.NodeAffinityTerm{
				{
					Key:      "topology.kubernetes.io/zone",
					Operator: "In",
					Values:   []string{"eu-west-1a", "eu-west-1b"},
				},
			},
		}
		tolerationSecs := int64(300)
		tolerations := []model.Toleration{
			{
				Key:               "dedicated",
				Operator:          "Equal",
				Value:             "vuhive-runners",
				Effect:            "NoSchedule",
				TolerationSeconds: &tolerationSecs,
			},
		}

		profile, err := model.NewRunnerProfile("perf-compute-c6i", "High performance profile", "alpine:3.20", res, nodeSelector, affinity, tolerations)
		require.NoError(t, err)

		// 1. Save
		err = profileRepo.Save(ctx, profile)
		require.NoError(t, err)

		// 2. FindByID
		found, err := profileRepo.FindByID(ctx, profile.ID())
		require.NoError(t, err)
		assert.Equal(t, profile.ID(), found.ID())
		assert.Equal(t, "perf-compute-c6i", found.Name())
		assert.Equal(t, "1000m", found.Resources().CPURequest())
		assert.Equal(t, "2000m", found.Resources().CPULimit())
		assert.Equal(t, "1Gi", found.Resources().MemoryRequest())
		assert.Equal(t, "2Gi", found.Resources().MemoryLimit())
		assert.Equal(t, nodeSelector, found.NodeSelector())
		assert.Equal(t, affinity, found.Affinity())
		assert.Equal(t, tolerations, found.Tolerations())

		// 3. FindByName
		foundByName, err := profileRepo.FindByName(ctx, "perf-compute-c6i")
		require.NoError(t, err)
		assert.Equal(t, profile.ID(), foundByName.ID())

		// 4. Duplicate Name -> ErrConflict
		dupProfile, err := model.NewRunnerProfile("perf-compute-c6i", "duplicate", "alpine:3.20", res, nil, model.Affinity{}, nil)
		require.NoError(t, err)
		err = profileRepo.Save(ctx, dupProfile)
		assert.ErrorIs(t, err, model.ErrConflict)

		// 5. List
		profiles, err := profileRepo.List(ctx)
		require.NoError(t, err)
		assert.Len(t, profiles, 1)

		// 6. Delete
		require.NoError(t, profileRepo.Delete(ctx, profile.ID()))
		_, err = profileRepo.FindByID(ctx, profile.ID())
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("ScheduleRepository CRUD & Active Filtering", func(t *testing.T) {
		suite, err := model.NewTestSuite("schedule-suite", "desc")
		require.NoError(t, err)
		require.NoError(t, suiteRepo.Save(ctx, suite))

		artifact, err := model.NewArtifact(suite.ID(), model.PlatformLinuxAmd64)
		require.NoError(t, err)
		require.NoError(t, artifactRepo.Save(ctx, artifact))

		res, err := model.NewResourceRequirements("500m", "1000m", "512Mi", "1Gi")
		require.NoError(t, err)
		profile, err := model.NewRunnerProfile("sched-profile", "desc", "alpine:3.20", res, nil, model.Affinity{}, nil)
		require.NoError(t, err)
		require.NoError(t, profileRepo.Save(ctx, profile))

		schedule, err := model.NewSchedule(suite.ID(), artifact.ID(), nil, profile.ID(), "nightly-stress-test", "0 2 * * *")
		require.NoError(t, err)

		// 1. Save
		err = scheduleRepo.Save(ctx, schedule)
		require.NoError(t, err)

		// 2. FindByID
		found, err := scheduleRepo.FindByID(ctx, schedule.ID())
		require.NoError(t, err)
		assert.Equal(t, schedule.ID(), found.ID())
		assert.Equal(t, "nightly-stress-test", found.Name())
		assert.Equal(t, "0 2 * * *", found.CronExpression())
		assert.True(t, found.IsActive())

		// 3. ListActive
		activeList, err := scheduleRepo.ListActive(ctx)
		require.NoError(t, err)
		assert.Len(t, activeList, 1)

		// 4. Deactivate & Verify ListActive excludes it
		require.NoError(t, schedule.Deactivate())
		require.NoError(t, scheduleRepo.Save(ctx, schedule))

		activeListAfter, err := scheduleRepo.ListActive(ctx)
		require.NoError(t, err)
		assert.Empty(t, activeListAfter)

		// 5. ListBySuiteID
		suiteSchedules, err := scheduleRepo.ListBySuiteID(ctx, suite.ID())
		require.NoError(t, err)
		assert.Len(t, suiteSchedules, 1)

		// 6. Delete
		require.NoError(t, scheduleRepo.Delete(ctx, schedule.ID()))
		_, err = scheduleRepo.FindByID(ctx, schedule.ID())
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("TestRunRepository Full Lifecycle & Filtered Queries", func(t *testing.T) {
		suite, err := model.NewTestSuite("run-suite", "desc")
		require.NoError(t, err)
		require.NoError(t, suiteRepo.Save(ctx, suite))

		artifact, err := model.NewArtifact(suite.ID(), model.PlatformLinuxAmd64)
		require.NoError(t, err)
		require.NoError(t, artifactRepo.Save(ctx, artifact))

		res, err := model.NewResourceRequirements("500m", "1000m", "512Mi", "1Gi")
		require.NoError(t, err)
		profile, err := model.NewRunnerProfile("run-profile", "desc", "alpine:3.20", res, nil, model.Affinity{}, nil)
		require.NoError(t, err)
		require.NoError(t, profileRepo.Save(ctx, profile))

		run, err := model.NewTestRun(suite.ID(), artifact.ID(), nil, profile.ID(), nil)
		require.NoError(t, err)

		// 1. Save (QUEUED)
		err = runRepo.Save(ctx, run)
		require.NoError(t, err)

		foundQueued, err := runRepo.FindByID(ctx, run.ID())
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusQueued, foundQueued.Status())
		assert.Equal(t, "vuhive-runners", foundQueued.K8sNamespace())

		// 2. Start (RUNNING)
		startTm := time.Now().UTC().Truncate(time.Millisecond)
		require.NoError(t, run.Start("vuhive-job-xyz123", startTm))
		require.NoError(t, runRepo.Save(ctx, run))

		foundRunning, err := runRepo.FindByID(ctx, run.ID())
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusRunning, foundRunning.Status())
		assert.Equal(t, "vuhive-job-xyz123", foundRunning.K8sJobName())
		assert.NotNil(t, foundRunning.StartedAt())

		foundByJobName, err := runRepo.FindByK8sJobName(ctx, "vuhive-job-xyz123")
		require.NoError(t, err)
		assert.Equal(t, run.ID(), foundByJobName.ID())
		assert.Equal(t, model.RunStatusRunning, foundByJobName.Status())

		// 3. Complete (COMPLETED)
		finishTm := startTm.Add(5 * time.Minute)
		metrics := model.RunMetrics{
			TotalIterations: 50000,
			TotalRequests:   200000,
			AvgTPS:          1500.50,
			P50DurationMs:   12.34,
			P90DurationMs:   25.60,
			P95DurationMs:   45.20,
			P99DurationMs:   98.75,
			ErrorRatePct:    0.005,
		}
		summaryJSON := []byte(`{"summary":"ok","tps":1500.5}`)
		require.NoError(t, run.Complete(metrics, "s3://bucket/reports/summary.json", "s3://bucket/logs/run.log", summaryJSON, true, finishTm))
		require.NoError(t, runRepo.Save(ctx, run))

		foundCompleted, err := runRepo.FindByID(ctx, run.ID())
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusCompleted, foundCompleted.Status())
		assert.NotNil(t, foundCompleted.ExitCode())
		assert.Equal(t, 0, *foundCompleted.ExitCode())
		assert.NotNil(t, foundCompleted.SLAPassed())
		assert.True(t, *foundCompleted.SLAPassed())
		assert.Equal(t, int64(50000), foundCompleted.Metrics().TotalIterations)
		assert.Equal(t, int64(200000), foundCompleted.Metrics().TotalRequests)
		assert.InDelta(t, 1500.50, foundCompleted.Metrics().AvgTPS, 0.01)
		assert.InDelta(t, 12.34, foundCompleted.Metrics().P50DurationMs, 0.01)
		assert.InDelta(t, 98.75, foundCompleted.Metrics().P99DurationMs, 0.01)
		assert.Equal(t, "s3://bucket/reports/summary.json", foundCompleted.S3ReportKey())
		assert.Equal(t, "s3://bucket/logs/run.log", foundCompleted.S3LogsKey())
		assert.JSONEq(t, string(summaryJSON), string(foundCompleted.SummaryJSON()))

		// 4. Query List with Filters
		// A. Filter by suiteID only
		listBySuite, err := runRepo.List(ctx, suite.ID(), "")
		require.NoError(t, err)
		assert.Len(t, listBySuite, 1)

		// B. Filter by suiteID and status COMPLETED
		listCompleted, err := runRepo.List(ctx, suite.ID(), model.RunStatusCompleted)
		require.NoError(t, err)
		assert.Len(t, listCompleted, 1)

		// C. Filter by status QUEUED (should be empty)
		listQueued, err := runRepo.List(ctx, suite.ID(), model.RunStatusQueued)
		require.NoError(t, err)
		assert.Empty(t, listQueued)

		// D. ListFiltered with pagination and count
		runsFiltered, totalFiltered, err := runRepo.ListFiltered(ctx, model.RunFilter{
			SuiteID: suite.ID(),
			Status:  model.RunStatusCompleted,
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), totalFiltered)
		assert.Len(t, runsFiltered, 1)

		// 5. Delete
		require.NoError(t, runRepo.Delete(ctx, run.ID()))
		_, err = runRepo.FindByID(ctx, run.ID())
		assert.ErrorIs(t, err, model.ErrNotFound)
	})
}
