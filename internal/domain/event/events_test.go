package event_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/morphy76/vuhive-cloud/internal/domain/event"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestDomainEvents(t *testing.T) {
	now := time.Now().UTC()

	t.Run("RunStarted event", func(t *testing.T) {
		evt := event.NewRunStarted("run-1", "suite-1", "k8s-job-1", now)
		assert.Equal(t, "RunStarted", evt.EventName())
		assert.Equal(t, now, evt.OccurredAt())
		assert.Equal(t, "run-1", evt.RunID)
		assert.Equal(t, "suite-1", evt.SuiteID)
		assert.Equal(t, "k8s-job-1", evt.K8sJobName)
	})

	t.Run("RunCompleted event", func(t *testing.T) {
		metrics := model.RunMetrics{TotalIterations: 100}
		evt := event.NewRunCompleted("run-1", "suite-1", true, metrics, now)
		assert.Equal(t, "RunCompleted", evt.EventName())
		assert.Equal(t, now, evt.OccurredAt())
		assert.Equal(t, "run-1", evt.RunID)
		assert.True(t, evt.SLAPassed)
		assert.Equal(t, metrics, evt.Metrics)
	})

	t.Run("RunFailed event", func(t *testing.T) {
		evt := event.NewRunFailed("run-1", "suite-1", 1, now)
		assert.Equal(t, "RunFailed", evt.EventName())
		assert.Equal(t, now, evt.OccurredAt())
		assert.Equal(t, 1, evt.ExitCode)
	})

	t.Run("RunAborted event", func(t *testing.T) {
		evt := event.NewRunAborted("run-1", "suite-1", "timeout", now)
		assert.Equal(t, "RunAborted", evt.EventName())
		assert.Equal(t, now, evt.OccurredAt())
		assert.Equal(t, "timeout", evt.Reason)
	})

	t.Run("ArtifactReady event", func(t *testing.T) {
		evt := event.NewArtifactReady("art-1", "suite-1", model.PlatformLinuxAmd64, "s3-key", "checksum", now)
		assert.Equal(t, "ArtifactReady", evt.EventName())
		assert.Equal(t, now, evt.OccurredAt())
		assert.Equal(t, model.PlatformLinuxAmd64, evt.Platform)
	})

	t.Run("BuildFailed event", func(t *testing.T) {
		evt := event.NewBuildFailed("art-1", "suite-1", "compile error", now)
		assert.Equal(t, "BuildFailed", evt.EventName())
		assert.Equal(t, now, evt.OccurredAt())
		assert.Equal(t, "compile error", evt.ErrorMessage)
	})
}
