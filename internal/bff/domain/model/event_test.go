package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerSentEvent_Constructors(t *testing.T) {
	now := time.Date(2026, 9, 6, 22, 0, 0, 0, time.UTC)

	t.Run("NewServerSentEvent validation", func(t *testing.T) {
		_, err := model.NewServerSentEvent("", model.EventSystemHeartbeat, []byte(`{}`), now)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)

		_, err = model.NewServerSentEvent("ev-1", "", []byte(`{}`), now)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)

		_, err = model.NewServerSentEvent("ev-1", model.EventSystemHeartbeat, nil, now)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)

		ev, err := model.NewServerSentEvent("ev-1", model.EventSystemHeartbeat, []byte(`{"status":"UP"}`), now)
		require.NoError(t, err)
		assert.Equal(t, "ev-1", ev.ID)
		assert.Equal(t, model.EventSystemHeartbeat, ev.Event)
		assert.JSONEq(t, `{"status":"UP"}`, string(ev.Data))
		assert.Equal(t, now, ev.Timestamp)
	})

	t.Run("NewRunStatusChangedEvent", func(t *testing.T) {
		startedAt := "2026-09-06T22:00:00Z"
		durationMs := int64(1500)
		exitCode := 0
		slaPassed := true

		payload := model.RunStatusChangedPayload{
			RunID:          "run-123",
			SuiteID:        "suite-456",
			Status:         "COMPLETED",
			PreviousStatus: "RUNNING",
			K8sJobName:     "vuhive-runner-run-123",
			StartedAt:      &startedAt,
			DurationMs:     &durationMs,
			ExitCode:       &exitCode,
			SLAPassed:      &slaPassed,
			Metrics: &model.RunMetrics{
				TotalIterations: 1000,
				TotalRequests:   5000,
				AvgTPS:          500.0,
				P50DurationMs:   10.0,
				P90DurationMs:   20.0,
				P95DurationMs:   30.0,
				P99DurationMs:   50.0,
				ErrorRatePct:    0.01,
			},
			Timestamp: now,
		}

		ev, err := model.NewRunStatusChangedEvent("ev-run-1", payload)
		require.NoError(t, err)
		assert.Equal(t, "ev-run-1", ev.ID)
		assert.Equal(t, model.EventRunStatusChanged, ev.Event)

		var decoded model.RunStatusChangedPayload
		err = json.Unmarshal(ev.Data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "run-123", decoded.RunID)
		assert.Equal(t, "COMPLETED", decoded.Status)
		assert.Equal(t, "RUNNING", decoded.PreviousStatus)
		assert.NotNil(t, decoded.Metrics)
		assert.Equal(t, int64(1000), decoded.Metrics.TotalIterations)
	})

	t.Run("NewRunStatusChangedEvent with empty RunID returns error", func(t *testing.T) {
		payload := model.RunStatusChangedPayload{
			RunID:  "",
			Status: "RUNNING",
		}
		_, err := model.NewRunStatusChangedEvent("ev-2", payload)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("NewBuildStatusChangedEvent", func(t *testing.T) {
		payload := model.BuildStatusChangedPayload{
			ArtifactID:     "art-123",
			SuiteID:        "suite-789",
			Platform:       "linux/amd64",
			Status:         "READY",
			PreviousStatus: "BUILDING",
			S3BinaryKey:    "artifacts/art-123/binary",
			SHA256Checksum: "abc123sha",
			Timestamp:      now,
		}

		ev, err := model.NewBuildStatusChangedEvent("ev-build-1", payload)
		require.NoError(t, err)
		assert.Equal(t, "ev-build-1", ev.ID)
		assert.Equal(t, model.EventBuildStatusChanged, ev.Event)

		var decoded model.BuildStatusChangedPayload
		err = json.Unmarshal(ev.Data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "art-123", decoded.ArtifactID)
		assert.Equal(t, "READY", decoded.Status)
		assert.Equal(t, "linux/amd64", decoded.Platform)
	})

	t.Run("NewBuildStatusChangedEvent with empty ArtifactID returns error", func(t *testing.T) {
		payload := model.BuildStatusChangedPayload{
			ArtifactID: "",
			Status:     "READY",
		}
		_, err := model.NewBuildStatusChangedEvent("ev-3", payload)
		assert.ErrorIs(t, err, model.ErrInvalidParameter)
	})

	t.Run("NewSystemHeartbeatEvent", func(t *testing.T) {
		payload := model.SystemHeartbeatPayload{
			Status:        "UP",
			ActiveClients: 5,
			ActiveRuns:    2,
			Timestamp:     now,
		}

		ev, err := model.NewSystemHeartbeatEvent("ev-hb-1", payload)
		require.NoError(t, err)
		assert.Equal(t, "ev-hb-1", ev.ID)
		assert.Equal(t, model.EventSystemHeartbeat, ev.Event)

		var decoded model.SystemHeartbeatPayload
		err = json.Unmarshal(ev.Data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "UP", decoded.Status)
		assert.Equal(t, 5, decoded.ActiveClients)
		assert.Equal(t, int64(2), decoded.ActiveRuns)
	})
}
