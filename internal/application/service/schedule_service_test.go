package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/application/service"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

type mockScheduleRepo struct {
	schedules map[string]*model.Schedule
}

func newMockScheduleRepo() *mockScheduleRepo {
	return &mockScheduleRepo{schedules: make(map[string]*model.Schedule)}
}

func (m *mockScheduleRepo) Save(_ context.Context, s *model.Schedule) error {
	m.schedules[s.ID()] = s
	return nil
}

func (m *mockScheduleRepo) FindByID(_ context.Context, id string) (*model.Schedule, error) {
	if s, ok := m.schedules[id]; ok {
		return s, nil
	}
	return nil, model.ErrNotFound
}

func (m *mockScheduleRepo) ListBySuiteID(_ context.Context, suiteID string) ([]*model.Schedule, error) {
	var list []*model.Schedule
	for _, s := range m.schedules {
		if s.SuiteID() == suiteID {
			list = append(list, s)
		}
	}
	return list, nil
}

func (m *mockScheduleRepo) ListActive(_ context.Context) ([]*model.Schedule, error) {
	var list []*model.Schedule
	for _, s := range m.schedules {
		if s.IsActive() {
			list = append(list, s)
		}
	}
	return list, nil
}

func (m *mockScheduleRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.schedules[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.schedules, id)
	return nil
}

type mockScheduleOrchestrator struct {
	createdCronJobs map[string]*model.Schedule
	deletedCronJobs map[string]string
	updatedCronJobs map[string]*model.Schedule
	failCreate      bool
}

func newMockScheduleOrchestrator() *mockScheduleOrchestrator {
	return &mockScheduleOrchestrator{
		createdCronJobs: make(map[string]*model.Schedule),
		deletedCronJobs: make(map[string]string),
		updatedCronJobs: make(map[string]*model.Schedule),
	}
}

func (m *mockScheduleOrchestrator) CreateCronJob(_ context.Context, schedule *model.Schedule, _ *model.RunnerProfile, _ outbound.RunnerJobOptions) (string, error) {
	if m.failCreate {
		return "", errors.New("k8s api error")
	}
	m.createdCronJobs[schedule.K8sCronJobName()] = schedule
	return schedule.K8sCronJobName(), nil
}

func (m *mockScheduleOrchestrator) UpdateCronJob(_ context.Context, schedule *model.Schedule) error {
	m.updatedCronJobs[schedule.K8sCronJobName()] = schedule
	return nil
}

func (m *mockScheduleOrchestrator) DeleteCronJob(_ context.Context, k8sCronJobName, namespace string) error {
	m.deletedCronJobs[k8sCronJobName] = namespace
	return nil
}

func setupScheduleTestContext(t *testing.T) (
	context.Context,
	*service.ScheduleService,
	*mockSuiteRepo,
	*mockArtifactRepo,
	*mockConfigRepo,
	*mockProfileRepo,
	*mockScheduleRepo,
	*mockScheduleOrchestrator,
	*model.TestSuite,
	*model.Artifact,
	*model.RunnerProfile,
) {
	ctx := context.Background()
	suiteRepo := newMockSuiteRepo()
	artifactRepo := newMockArtifactRepo()
	configRepo := newMockConfigRepo()
	profileRepo := newMockProfileRepo()
	scheduleRepo := newMockScheduleRepo()
	orchestrator := newMockScheduleOrchestrator()

	svc := service.NewScheduleService(
		suiteRepo,
		artifactRepo,
		configRepo,
		profileRepo,
		scheduleRepo,
		orchestrator,
	)

	suite, err := model.NewTestSuite("test-suite", "desc")
	require.NoError(t, err)
	require.NoError(t, suite.Activate())
	require.NoError(t, suiteRepo.Save(ctx, suite))

	artifact, err := model.NewArtifact(suite.ID(), model.PlatformLinuxAmd64)
	require.NoError(t, err)
	require.NoError(t, artifact.MarkBuilding())
	require.NoError(t, artifact.MarkReady("binaries/runner", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	require.NoError(t, artifactRepo.Save(ctx, artifact))

	res, err := model.NewResourceRequirements("500m", "1000m", "256Mi", "512Mi")
	require.NoError(t, err)
	profile, err := model.NewRunnerProfile("default-profile", "desc", "alpine:3.20", res, nil, model.Affinity{}, nil)
	require.NoError(t, err)
	require.NoError(t, profileRepo.Save(ctx, profile))

	return ctx, svc, suiteRepo, artifactRepo, configRepo, profileRepo, scheduleRepo, orchestrator, suite, artifact, profile
}

func TestScheduleService_CreateSchedule(t *testing.T) {
	ctx, svc, suiteRepo, artifactRepo, configRepo, _, scheduleRepo, orchestrator, suite, artifact, profile := setupScheduleTestContext(t)

	t.Run("successfully creates schedule and triggers K8s CronJob creation", func(t *testing.T) {
		cfg, err := model.NewConfiguration(suite.ID(), "default", "concurrency: 10", "configs/vuhive.yaml", false)
		require.NoError(t, err)
		require.NoError(t, configRepo.Save(ctx, cfg))
		cfgID := cfg.ID()

		schedule, err := svc.CreateSchedule(
			ctx,
			suite.ID(),
			artifact.ID(),
			&cfgID,
			profile.ID(),
			"nightly-run",
			"0 2 * * *",
		)
		require.NoError(t, err)
		require.NotNil(t, schedule)
		assert.Equal(t, "nightly-run", schedule.Name())
		assert.Equal(t, "0 2 * * *", schedule.CronExpression())
		assert.True(t, schedule.IsActive())

		// Persisted in DB
		persisted, err := scheduleRepo.FindByID(ctx, schedule.ID())
		require.NoError(t, err)
		assert.Equal(t, schedule.ID(), persisted.ID())

		// Manifested on K8s
		assert.Contains(t, orchestrator.createdCronJobs, schedule.K8sCronJobName())
	})

	t.Run("fails when suite does not exist", func(t *testing.T) {
		_, err := svc.CreateSchedule(ctx, "non-existent-suite", artifact.ID(), nil, profile.ID(), "nightly", "0 0 * * *")
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("fails when suite is archived", func(t *testing.T) {
		archivedSuite, err := model.NewTestSuite("archived-suite", "desc")
		require.NoError(t, err)
		require.NoError(t, archivedSuite.Archive())
		require.NoError(t, suiteRepo.Save(ctx, archivedSuite))

		_, err = svc.CreateSchedule(ctx, archivedSuite.ID(), artifact.ID(), nil, profile.ID(), "nightly", "0 0 * * *")
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})

	t.Run("fails when artifact does not belong to suite", func(t *testing.T) {
		otherSuite, err := model.NewTestSuite("other-suite", "desc")
		require.NoError(t, err)
		require.NoError(t, otherSuite.Activate())
		require.NoError(t, suiteRepo.Save(ctx, otherSuite))

		_, err = svc.CreateSchedule(ctx, otherSuite.ID(), artifact.ID(), nil, profile.ID(), "nightly", "0 0 * * *")
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("fails when artifact is not ready", func(t *testing.T) {
		pendingArt, err := model.NewArtifact(suite.ID(), model.PlatformLinuxArm64)
		require.NoError(t, err)
		require.NoError(t, artifactRepo.Save(ctx, pendingArt))

		_, err = svc.CreateSchedule(ctx, suite.ID(), pendingArt.ID(), nil, profile.ID(), "nightly", "0 0 * * *")
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})

	t.Run("fails on invalid cron expression", func(t *testing.T) {
		_, err := svc.CreateSchedule(ctx, suite.ID(), artifact.ID(), nil, profile.ID(), "nightly", "invalid cron")
		assert.ErrorIs(t, err, model.ErrInvalidCronExpression)
	})

	t.Run("cleans up DB if K8s CronJob creation fails", func(t *testing.T) {
		orchestrator.failCreate = true
		defer func() { orchestrator.failCreate = false }()

		_, err := svc.CreateSchedule(ctx, suite.ID(), artifact.ID(), nil, profile.ID(), "nightly-fail", "0 3 * * *")
		assert.Error(t, err)

		// Must not remain in scheduleRepo
		schedules, err := scheduleRepo.ListBySuiteID(ctx, suite.ID())
		require.NoError(t, err)
		for _, s := range schedules {
			assert.NotEqual(t, "nightly-fail", s.Name())
		}
	})
}

func TestScheduleService_GetAndList(t *testing.T) {
	ctx, svc, _, _, _, _, _, _, suite, artifact, profile := setupScheduleTestContext(t)

	schedule, err := svc.CreateSchedule(ctx, suite.ID(), artifact.ID(), nil, profile.ID(), "sched-1", "0 4 * * *")
	require.NoError(t, err)

	t.Run("GetSchedule returns existing schedule", func(t *testing.T) {
		found, err := svc.GetSchedule(ctx, schedule.ID())
		require.NoError(t, err)
		assert.Equal(t, schedule.ID(), found.ID())
	})

	t.Run("GetSchedule fails on non-existent schedule", func(t *testing.T) {
		_, err := svc.GetSchedule(ctx, "unknown-id")
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("ListSchedules returns active schedules", func(t *testing.T) {
		list, err := svc.ListSchedules(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
	})
}

func TestScheduleService_UpdateSchedule(t *testing.T) {
	ctx, svc, _, _, _, _, _, orchestrator, suite, artifact, profile := setupScheduleTestContext(t)

	schedule, err := svc.CreateSchedule(ctx, suite.ID(), artifact.ID(), nil, profile.ID(), "sched-update", "0 4 * * *")
	require.NoError(t, err)

	t.Run("updates cron expression and syncs to K8s CronJob", func(t *testing.T) {
		updated, err := svc.UpdateSchedule(ctx, schedule.ID(), "*/15 * * * *")
		require.NoError(t, err)
		assert.Equal(t, "*/15 * * * *", updated.CronExpression())

		// Verify orchestrator received update
		assert.Contains(t, orchestrator.updatedCronJobs, schedule.K8sCronJobName())
		assert.Equal(t, "*/15 * * * *", orchestrator.updatedCronJobs[schedule.K8sCronJobName()].CronExpression())
	})

	t.Run("fails on invalid cron expression", func(t *testing.T) {
		_, err := svc.UpdateSchedule(ctx, schedule.ID(), "bad-cron")
		assert.ErrorIs(t, err, model.ErrInvalidCronExpression)
	})
}

func TestScheduleService_DeleteSchedule(t *testing.T) {
	ctx, svc, _, _, _, _, scheduleRepo, orchestrator, suite, artifact, profile := setupScheduleTestContext(t)

	schedule, err := svc.CreateSchedule(ctx, suite.ID(), artifact.ID(), nil, profile.ID(), "sched-delete", "0 4 * * *")
	require.NoError(t, err)

	t.Run("deletes schedule from DB and K8s", func(t *testing.T) {
		err := svc.DeleteSchedule(ctx, schedule.ID())
		require.NoError(t, err)

		// Must be deleted from DB
		_, err = scheduleRepo.FindByID(ctx, schedule.ID())
		assert.ErrorIs(t, err, model.ErrNotFound)

		// Must be deleted from K8s
		assert.Contains(t, orchestrator.deletedCronJobs, schedule.K8sCronJobName())
	})

	t.Run("fails when deleting non-existent schedule", func(t *testing.T) {
		err := svc.DeleteSchedule(ctx, "non-existent")
		assert.ErrorIs(t, err, model.ErrNotFound)
	})
}
