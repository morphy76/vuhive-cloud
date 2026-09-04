package k8s_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/k8s"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

type inMemoryRunRepo struct {
	mu   sync.Mutex
	runs map[string]*model.TestRun
}

func newInMemoryRunRepo() *inMemoryRunRepo {
	return &inMemoryRunRepo{
		runs: make(map[string]*model.TestRun),
	}
}

func (r *inMemoryRunRepo) Save(_ context.Context, run *model.TestRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	clone, err := model.NewTestRunWithID(
		run.ID(), run.SuiteID(), run.ArtifactID(), run.ConfigurationID(), run.RunnerProfileID(), run.ScheduleID(),
		run.Status(), run.K8sJobName(), run.K8sNamespace(),
		run.StartedAt(), run.FinishedAt(), run.ExitCode(), run.SLAPassed(),
		run.Metrics(), run.S3ReportKey(), run.S3LogsKey(), run.SummaryJSON(), run.AbortReason(), run.CreatedAt(),
	)
	if err != nil {
		return err
	}
	r.runs[clone.ID()] = clone
	return nil
}

func (r *inMemoryRunRepo) FindByID(_ context.Context, id string) (*model.TestRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, exists := r.runs[id]
	if !exists {
		return nil, model.ErrNotFound
	}
	return model.NewTestRunWithID(
		run.ID(), run.SuiteID(), run.ArtifactID(), run.ConfigurationID(), run.RunnerProfileID(), run.ScheduleID(),
		run.Status(), run.K8sJobName(), run.K8sNamespace(),
		run.StartedAt(), run.FinishedAt(), run.ExitCode(), run.SLAPassed(),
		run.Metrics(), run.S3ReportKey(), run.S3LogsKey(), run.SummaryJSON(), run.AbortReason(), run.CreatedAt(),
	)
}

func (r *inMemoryRunRepo) FindByK8sJobName(_ context.Context, jobName string) (*model.TestRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, run := range r.runs {
		if run.K8sJobName() == jobName {
			return model.NewTestRunWithID(
				run.ID(), run.SuiteID(), run.ArtifactID(), run.ConfigurationID(), run.RunnerProfileID(), run.ScheduleID(),
				run.Status(), run.K8sJobName(), run.K8sNamespace(),
				run.StartedAt(), run.FinishedAt(), run.ExitCode(), run.SLAPassed(),
				run.Metrics(), run.S3ReportKey(), run.S3LogsKey(), run.SummaryJSON(), run.AbortReason(), run.CreatedAt(),
			)
		}
	}
	return nil, model.ErrNotFound
}

func (r *inMemoryRunRepo) List(_ context.Context, _ string, _ model.RunStatus) ([]*model.TestRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []*model.TestRun
	for _, run := range r.runs {
		list = append(list, run)
	}
	return list, nil
}

func (r *inMemoryRunRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.runs, id)
	return nil
}

type inMemoryScheduleRepo struct {
	mu        sync.Mutex
	schedules map[string]*model.Schedule
}

func newInMemoryScheduleRepo() *inMemoryScheduleRepo {
	return &inMemoryScheduleRepo{
		schedules: make(map[string]*model.Schedule),
	}
}

func (r *inMemoryScheduleRepo) Save(_ context.Context, schedule *model.Schedule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schedules[schedule.ID()] = schedule
	return nil
}

func (r *inMemoryScheduleRepo) FindByID(_ context.Context, id string) (*model.Schedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sched, exists := r.schedules[id]
	if !exists {
		return nil, model.ErrNotFound
	}
	return sched, nil
}

func (r *inMemoryScheduleRepo) ListBySuiteID(_ context.Context, suiteID string) ([]*model.Schedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []*model.Schedule
	for _, s := range r.schedules {
		if s.SuiteID() == suiteID {
			list = append(list, s)
		}
	}
	return list, nil
}

func (r *inMemoryScheduleRepo) ListActive(_ context.Context) ([]*model.Schedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []*model.Schedule
	for _, s := range r.schedules {
		if s.IsActive() {
			list = append(list, s)
		}
	}
	return list, nil
}

func (r *inMemoryScheduleRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.schedules, id)
	return nil
}

func TestRunnerJobWatcher_SyncJob(t *testing.T) {
	ctx := context.Background()
	cfg := k8s.DefaultConfig()
	cfg.RunnerNamespace = "vuhive-runners"
	fakeClient := fake.NewSimpleClientset()

	repo := newInMemoryRunRepo()
	schedRepo := newInMemoryScheduleRepo()
	watcher := k8s.NewRunnerJobWatcher(fakeClient, repo, schedRepo, cfg)

	t.Run("transition QUEUED to RUNNING when job becomes active", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, run))

		now := metav1.Now()
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vuhive-run-job-1",
				Namespace: "vuhive-runners",
				Labels: map[string]string{
					"app.kubernetes.io/name": "vuhive-runner",
					"vuhive.io/run-id":       run.ID(),
				},
			},
			Status: batchv1.JobStatus{
				Active:    1,
				StartTime: &now,
			},
		}

		err = watcher.SyncJob(ctx, job)
		require.NoError(t, err)

		updated, err := repo.FindByID(ctx, run.ID())
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusRunning, updated.Status())
		assert.Equal(t, "vuhive-run-job-1", updated.K8sJobName())
		assert.NotNil(t, updated.StartedAt())
	})

	t.Run("transition RUNNING to COMPLETED when job succeeds", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)
		require.NoError(t, run.Start("vuhive-run-job-2", time.Now().UTC()))
		require.NoError(t, repo.Save(ctx, run))

		now := metav1.Now()
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vuhive-run-job-2",
				Namespace: "vuhive-runners",
				Labels: map[string]string{
					"app.kubernetes.io/name": "vuhive-runner",
					"vuhive.io/run-id":       run.ID(),
				},
			},
			Status: batchv1.JobStatus{
				Succeeded:      1,
				CompletionTime: &now,
			},
		}

		err = watcher.SyncJob(ctx, job)
		require.NoError(t, err)

		updated, err := repo.FindByID(ctx, run.ID())
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusCompleted, updated.Status())
		assert.NotNil(t, updated.FinishedAt())
		require.NotNil(t, updated.ExitCode())
		assert.Equal(t, 0, *updated.ExitCode())
		assert.NotEmpty(t, updated.S3ReportKey())
		assert.NotEmpty(t, updated.S3LogsKey())
	})

	t.Run("transition RUNNING to FAILED when job fails", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)
		require.NoError(t, run.Start("vuhive-run-job-3", time.Now().UTC()))
		require.NoError(t, repo.Save(ctx, run))

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vuhive-run-job-3",
				Namespace: "vuhive-runners",
				Labels: map[string]string{
					"app.kubernetes.io/name": "vuhive-runner",
					"vuhive.io/run-id":       run.ID(),
				},
			},
			Status: batchv1.JobStatus{
				Failed: 1,
				Conditions: []batchv1.JobCondition{
					{
						Type:   batchv1.JobFailed,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}

		err = watcher.SyncJob(ctx, job)
		require.NoError(t, err)

		updated, err := repo.FindByID(ctx, run.ID())
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusFailed, updated.Status())
		assert.NotNil(t, updated.FinishedAt())
		require.NotNil(t, updated.ExitCode())
		assert.Equal(t, 1, *updated.ExitCode())
	})

	t.Run("ignore job without vuhive run-id label", func(t *testing.T) {
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-job",
				Namespace: "vuhive-runners",
			},
			Status: batchv1.JobStatus{
				Active: 1,
			},
		}
		err := watcher.SyncJob(ctx, job)
		assert.NoError(t, err)
	})

	t.Run("do nothing if run is already in terminal state", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)
		require.NoError(t, run.Start("job-term", time.Now().UTC()))
		require.NoError(t, run.Fail(1, "log", time.Now().UTC()))
		require.NoError(t, repo.Save(ctx, run))

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "job-term",
				Namespace: "vuhive-runners",
				Labels: map[string]string{
					"app.kubernetes.io/name": "vuhive-runner",
					"vuhive.io/run-id":       run.ID(),
				},
			},
			Status: batchv1.JobStatus{
				Succeeded: 1,
			},
		}

		err = watcher.SyncJob(ctx, job)
		require.NoError(t, err)

		updated, err := repo.FindByID(ctx, run.ID())
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusFailed, updated.Status())
	})

	t.Run("auto-creates and tracks linked TestRun when CronJob-spawned Job is detected", func(t *testing.T) {
		schedule, err := model.NewSchedule(
			"suite-sched-1",
			"art-sched-1",
			nil,
			"prof-sched-1",
			"nightly-cron",
			"0 0 * * *",
		)
		require.NoError(t, err)
		require.NoError(t, schedRepo.Save(ctx, schedule))

		now := metav1.Now()
		spawnedJob := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vuhive-sched-12345678-28492020",
				Namespace: "vuhive-runners",
				Labels: map[string]string{
					"app.kubernetes.io/name": "vuhive-runner",
					"vuhive.io/schedule-id":  schedule.ID(),
				},
			},
			Status: batchv1.JobStatus{
				Active:    1,
				StartTime: &now,
			},
		}

		// Initial sync when Job appears
		err = watcher.SyncJob(ctx, spawnedJob)
		require.NoError(t, err)

		// TestRun must have been auto-created and linked to schedule
		foundRun, err := repo.FindByK8sJobName(ctx, spawnedJob.Name)
		require.NoError(t, err)
		require.NotNil(t, foundRun)
		assert.Equal(t, schedule.SuiteID(), foundRun.SuiteID())
		assert.Equal(t, schedule.ArtifactID(), foundRun.ArtifactID())
		assert.Equal(t, schedule.RunnerProfileID(), foundRun.RunnerProfileID())
		require.NotNil(t, foundRun.ScheduleID())
		assert.Equal(t, schedule.ID(), *foundRun.ScheduleID())
		assert.Equal(t, spawnedJob.Name, foundRun.K8sJobName())
		assert.Equal(t, model.RunStatusRunning, foundRun.Status())

		// Second sync when Job completes
		spawnedJob.Status.Active = 0
		spawnedJob.Status.Succeeded = 1
		completionTime := metav1.Now()
		spawnedJob.Status.CompletionTime = &completionTime

		err = watcher.SyncJob(ctx, spawnedJob)
		require.NoError(t, err)

		completedRun, err := repo.FindByID(ctx, foundRun.ID())
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusCompleted, completedRun.Status())
		assert.NotEmpty(t, completedRun.S3ReportKey())
		assert.NotEmpty(t, completedRun.S3LogsKey())
	})
}

func TestRunnerJobWatcher_InformerLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := k8s.DefaultConfig()
	cfg.RunnerNamespace = "vuhive-runners"
	fakeClient := fake.NewSimpleClientset()

	repo := newInMemoryRunRepo()
	schedRepo := newInMemoryScheduleRepo()
	watcher := k8s.NewRunnerJobWatcher(fakeClient, repo, schedRepo, cfg)

	run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, run))

	// Start watcher in background
	watchErrCh := make(chan error, 1)
	go func() {
		watchErrCh <- watcher.Start(ctx)
	}()

	// Give informer a moment to start and sync
	time.Sleep(100 * time.Millisecond)

	// Create job in fake clientset
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vuhive-run-live",
			Namespace: "vuhive-runners",
			Labels: map[string]string{
				"app.kubernetes.io/name": "vuhive-runner",
				"vuhive.io/run-id":       run.ID(),
			},
		},
		Status: batchv1.JobStatus{
			Active: 1,
		},
	}
	_, err = fakeClient.BatchV1().Jobs("vuhive-runners").Create(ctx, job, metav1.CreateOptions{})
	require.NoError(t, err)

	// Verify run transitioned to RUNNING via informer event
	require.Eventually(t, func() bool {
		r, err := repo.FindByID(ctx, run.ID())
		return err == nil && r.Status() == model.RunStatusRunning
	}, 2*time.Second, 50*time.Millisecond)

	// Cancel context and verify graceful shutdown
	cancel()
	select {
	case err := <-watchErrCh:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not shut down gracefully")
	}
}
