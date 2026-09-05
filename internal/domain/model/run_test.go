package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestNewTestRun(t *testing.T) {
	t.Run("successfully create a new test run in QUEUED status", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)
		require.NotNil(t, run)

		assert.NotEmpty(t, run.ID())
		assert.Equal(t, run.ID(), run.EntityID())
		assert.Equal(t, "TestRun", run.AggregateType())
		assert.Equal(t, "suite-1", run.SuiteID())
		assert.Equal(t, "art-1", run.ArtifactID())
		assert.Nil(t, run.ConfigurationID())
		assert.Equal(t, "prof-1", run.RunnerProfileID())
		assert.Nil(t, run.ScheduleID())
		assert.Equal(t, model.RunStatusQueued, run.Status())
		assert.Equal(t, "vuhive-runners", run.K8sNamespace())
		assert.Nil(t, run.StartedAt())
		assert.Nil(t, run.FinishedAt())
		assert.Nil(t, run.ExitCode())
		assert.Nil(t, run.SLAPassed())
		assert.False(t, run.CreatedAt().IsZero())
	})

	t.Run("successfully create with configuration and schedule", func(t *testing.T) {
		cfgID := "cfg-1"
		schedID := "sched-1"
		run, err := model.NewTestRun("suite-1", "art-1", &cfgID, "prof-1", &schedID)
		require.NoError(t, err)

		require.NotNil(t, run.ConfigurationID())
		assert.Equal(t, "cfg-1", *run.ConfigurationID())
		require.NotNil(t, run.ScheduleID())
		assert.Equal(t, "sched-1", *run.ScheduleID())
	})

	t.Run("fail validation when required references are missing", func(t *testing.T) {
		_, err := model.NewTestRun("", "art-1", nil, "prof-1", nil)
		assert.ErrorIs(t, err, model.ErrValidation)

		_, err = model.NewTestRun("suite-1", "", nil, "prof-1", nil)
		assert.ErrorIs(t, err, model.ErrValidation)

		_, err = model.NewTestRun("suite-1", "art-1", nil, "", nil)
		assert.ErrorIs(t, err, model.ErrValidation)
	})
}

func TestTestRun_StateTransitions(t *testing.T) {
	now := time.Now().UTC()

	metrics := model.RunMetrics{
		TotalIterations: 10000,
		TotalRequests:   50000,
		AvgTPS:          2500.50,
		P50DurationMs:   2.4,
		P90DurationMs:   5.1,
		P95DurationMs:   8.3,
		P99DurationMs:   14.2,
		ErrorRatePct:    0.01,
	}

	t.Run("happy path: QUEUED -> RUNNING -> COMPLETED", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)

		startTime := now.Add(time.Second)
		err = run.Start("vuhive-run-job-123", startTime)
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusRunning, run.Status())
		assert.Equal(t, "vuhive-run-job-123", run.K8sJobName())
		assert.Equal(t, &startTime, run.StartedAt())

		finishTime := startTime.Add(time.Minute)
		summary := []byte(`{"sla": true}`)
		err = run.Complete(metrics, "s3://reports/run-1/summary.json", "s3://reports/run-1/run.log", summary, true, finishTime)
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusCompleted, run.Status())
		assert.Equal(t, &finishTime, run.FinishedAt())
		require.NotNil(t, run.ExitCode())
		assert.Equal(t, 0, *run.ExitCode())
		require.NotNil(t, run.SLAPassed())
		assert.True(t, *run.SLAPassed())
		assert.Equal(t, metrics, run.Metrics())
		assert.Equal(t, "s3://reports/run-1/summary.json", run.S3ReportKey())
		assert.Equal(t, "s3://reports/run-1/run.log", run.S3LogsKey())
		assert.Equal(t, summary, run.SummaryJSON())
	})

	t.Run("fail path from RUNNING: QUEUED -> RUNNING -> FAILED", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)

		require.NoError(t, run.Start("vuhive-run-job-123", now))

		finishTime := now.Add(time.Minute)
		err = run.Fail(1, "s3://reports/run-1/run.log", finishTime)
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusFailed, run.Status())
		assert.Equal(t, &finishTime, run.FinishedAt())
		require.NotNil(t, run.ExitCode())
		assert.Equal(t, 1, *run.ExitCode())
		require.NotNil(t, run.SLAPassed())
		assert.False(t, *run.SLAPassed())
	})

	t.Run("fail path from QUEUED: dispatch error", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)

		err = run.Fail(127, "", now)
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusFailed, run.Status())
	})

	t.Run("abort path from QUEUED", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)

		err = run.Abort("user cancelled queued run", now)
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusAborted, run.Status())
		assert.Equal(t, "user cancelled queued run", run.AbortReason())
		assert.Equal(t, &now, run.FinishedAt())
	})

	t.Run("abort path from RUNNING", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)
		require.NoError(t, run.Start("job-1", now))

		abortTime := now.Add(10 * time.Second)
		err = run.Abort("manual abort via API", abortTime)
		require.NoError(t, err)
		assert.Equal(t, model.RunStatusAborted, run.Status())
		assert.Equal(t, "manual abort via API", run.AbortReason())
	})

	t.Run("illegal transition: direct QUEUED -> COMPLETED", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)

		err = run.Complete(metrics, "rep", "log", nil, true, now)
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})

	t.Run("illegal transition: Start already RUNNING run", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)
		require.NoError(t, run.Start("job-1", now))

		err = run.Start("job-2", now)
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})

	t.Run("terminal state COMPLETED blocks all transitions", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)
		require.NoError(t, run.Start("job-1", now))
		require.NoError(t, run.Complete(metrics, "rep", "log", nil, true, now))

		assert.ErrorIs(t, run.Start("job-2", now), model.ErrTerminalState)
		assert.ErrorIs(t, run.Complete(metrics, "rep", "log", nil, true, now), model.ErrTerminalState)
		assert.ErrorIs(t, run.Fail(1, "log", now), model.ErrTerminalState)
		assert.ErrorIs(t, run.Abort("reason", now), model.ErrTerminalState)
	})

	t.Run("terminal state FAILED blocks all transitions", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)
		require.NoError(t, run.Fail(1, "log", now))

		assert.ErrorIs(t, run.Start("job-2", now), model.ErrTerminalState)
		assert.ErrorIs(t, run.Complete(metrics, "rep", "log", nil, true, now), model.ErrTerminalState)
		assert.ErrorIs(t, run.Fail(1, "log", now), model.ErrTerminalState)
		assert.ErrorIs(t, run.Abort("reason", now), model.ErrTerminalState)
	})

	t.Run("terminal state ABORTED blocks all transitions", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)
		require.NoError(t, run.Abort("reason", now))

		assert.ErrorIs(t, run.Start("job-2", now), model.ErrTerminalState)
		assert.ErrorIs(t, run.Complete(metrics, "rep", "log", nil, true, now), model.ErrTerminalState)
		assert.ErrorIs(t, run.Fail(1, "log", now), model.ErrTerminalState)
		assert.ErrorIs(t, run.Abort("reason", now), model.ErrTerminalState)
	})
}

func TestNewTestRunWithID(t *testing.T) {
	now := time.Now().UTC()
	exitCode := 0
	sla := true

	run, err := model.NewTestRunWithID(
		"run-123", "suite-1", "art-1", nil, "prof-1", nil,
		model.RunStatusCompleted, "k8s-job", "vuhive-runners",
		&now, &now, &exitCode, &sla,
		model.RunMetrics{}, "rep", "log", nil, "", now,
	)
	require.NoError(t, err)
	assert.Equal(t, "run-123", run.ID())
	assert.Equal(t, model.RunStatusCompleted, run.Status())

	_, err = model.NewTestRunWithID(
		"run-123", "suite-1", "art-1", nil, "prof-1", nil,
		"INVALID_STATUS", "k8s-job", "vuhive-runners",
		&now, &now, &exitCode, &sla,
		model.RunMetrics{}, "rep", "log", nil, "", now,
	)
	assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
}

func TestTestRun_SetK8sJobName(t *testing.T) {
	run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
	require.NoError(t, err)
	assert.Empty(t, run.K8sJobName())

	run.SetK8sJobName("vuhive-run-abc")
	assert.Equal(t, "vuhive-run-abc", run.K8sJobName())

	// Start preserves or updates job name
	now := time.Now().UTC()
	err = run.Start("", now)
	require.NoError(t, err)
	assert.Equal(t, "vuhive-run-abc", run.K8sJobName())
}

func TestTestRun_SetExitCode(t *testing.T) {
	run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
	require.NoError(t, err)
	assert.Nil(t, run.ExitCode())

	run.SetExitCode(42)
	require.NotNil(t, run.ExitCode())
	assert.Equal(t, 42, *run.ExitCode())
}

func TestTestRun_FailWithMetrics(t *testing.T) {
	now := time.Now().UTC()
	metrics := model.RunMetrics{
		TotalIterations: 10,
		TotalRequests:   50,
	}

	run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
	require.NoError(t, err)
	require.NoError(t, run.Start("job-1", now))

	finishTime := now.Add(time.Minute)
	summary := []byte(`{"status":"FAILED"}`)
	err = run.FailWithMetrics(1, metrics, "s3://rep", "s3://log", summary, finishTime)
	require.NoError(t, err)

	assert.Equal(t, model.RunStatusFailed, run.Status())
	require.NotNil(t, run.ExitCode())
	assert.Equal(t, 1, *run.ExitCode())
	require.NotNil(t, run.SLAPassed())
	assert.False(t, *run.SLAPassed())
	assert.Equal(t, metrics, run.Metrics())
	assert.Equal(t, "s3://rep", run.S3ReportKey())
	assert.Equal(t, "s3://log", run.S3LogsKey())
	assert.Equal(t, summary, run.SummaryJSON())
	assert.Equal(t, &finishTime, run.FinishedAt())

	// Cannot fail already failed run
	assert.ErrorIs(t, run.FailWithMetrics(1, metrics, "s3://rep", "s3://log", summary, finishTime), model.ErrTerminalState)
}

func TestTestRun_DurationMs(t *testing.T) {
	now := time.Now().UTC()

	t.Run("queued run has nil duration", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)
		assert.Nil(t, run.DurationMs())
	})

	t.Run("completed run returns exact duration in ms", func(t *testing.T) {
		run, err := model.NewTestRun("suite-1", "art-1", nil, "prof-1", nil)
		require.NoError(t, err)
		require.NoError(t, run.Start("job-1", now))
		finish := now.Add(1250 * time.Millisecond)
		require.NoError(t, run.Complete(model.RunMetrics{}, "s3://rep", "s3://log", []byte(`{}`), true, finish))

		dur := run.DurationMs()
		require.NotNil(t, dur)
		assert.Equal(t, int64(1250), *dur)
	})
}

