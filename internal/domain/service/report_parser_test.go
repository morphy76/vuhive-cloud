package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/morphy76/vuhive-cloud/internal/domain/service"
)

func TestParseSummaryReport_FullVuhiveReport(t *testing.T) {
	rawJSON := []byte(`{
		"suite_name": "Ecommerce Checkout Suite",
		"scenario": "checkout_flow",
		"version": "1.0.0",
		"commit": "a1b2c3d",
		"started_at": "2026-09-05T07:00:00Z",
		"ended_at": "2026-09-05T07:01:00Z",
		"duration": 60000000000,
		"passed": true,
		"metrics": [
			{
				"name": "vuhive.vu.iterations_total",
				"type": "counter",
				"count": 1200
			},
			{
				"name": "vuhive.vu.iterations_failed",
				"type": "counter",
				"count": 6
			},
			{
				"name": "vuhive.http.reqs",
				"type": "counter",
				"count": 6000
			},
			{
				"name": "vuhive.http.req_failed",
				"type": "rate",
				"rate": 0.005
			},
			{
				"name": "vuhive.http.req_duration",
				"type": "duration",
				"count": 6000,
				"min": 2000000,
				"mean": 18000000,
				"p50": 15000000,
				"p90": 25000000,
				"p95": 35000000,
				"p99": 50000000,
				"max": 120000000
			}
		],
		"thresholds": [
			{
				"metric": "vuhive.http.req_duration",
				"stat": "p95",
				"operator": "<=",
				"target": "50ms",
				"actual": "35ms",
				"passed": true
			},
			{
				"metric": "vuhive.http.req_failed",
				"stat": "rate",
				"operator": "<=",
				"target": "0.01",
				"actual": "0.005",
				"passed": true
			}
		]
	}`)

	result, err := service.ParseSummaryReport(rawJSON)
	require.NoError(t, err)
	assert.True(t, result.SLAPassed)
	assert.Equal(t, int64(1200), result.Metrics.TotalIterations)
	assert.Equal(t, int64(6000), result.Metrics.TotalRequests)
	assert.InDelta(t, 100.0, result.Metrics.AvgTPS, 0.1) // 6000 reqs / 60s = 100 TPS
	assert.InDelta(t, 15.0, result.Metrics.P50DurationMs, 0.01)
	assert.InDelta(t, 25.0, result.Metrics.P90DurationMs, 0.01)
	assert.InDelta(t, 35.0, result.Metrics.P95DurationMs, 0.01)
	assert.InDelta(t, 50.0, result.Metrics.P99DurationMs, 0.01)
	assert.InDelta(t, 0.5, result.Metrics.ErrorRatePct, 0.01) // 0.005 * 100 = 0.5%
	assert.NotEmpty(t, result.RawJSON)
}

func TestParseSummaryReport_ThresholdFailure(t *testing.T) {
	rawJSON := []byte(`{
		"scenario": "test_scenario",
		"duration": 10000000000,
		"passed": true,
		"metrics": [
			{
				"name": "vuhive.vu.iterations_total",
				"type": "counter",
				"count": 100
			}
		],
		"thresholds": [
			{
				"metric": "vuhive.http.req_duration",
				"stat": "p99",
				"operator": "<=",
				"target": "50ms",
				"actual": "75ms",
				"passed": false
			}
		]
	}`)

	result, err := service.ParseSummaryReport(rawJSON)
	require.NoError(t, err)
	assert.False(t, result.SLAPassed, "overall SLA must fail if any threshold failed")
}

func TestParseSummaryReport_FlatFallbackReport(t *testing.T) {
	rawJSON := []byte(`{
		"status": "PASS",
		"sla_passed": true,
		"iterations": 500,
		"total_requests": 2500,
		"avg_tps": 83.33,
		"p50_duration_ms": 12.5,
		"p90_duration_ms": 22.0,
		"p95_duration_ms": 31.4,
		"p99_duration_ms": 45.8,
		"error_rate_pct": 0.2
	}`)

	result, err := service.ParseSummaryReport(rawJSON)
	require.NoError(t, err)
	assert.True(t, result.SLAPassed)
	assert.Equal(t, int64(500), result.Metrics.TotalIterations)
	assert.Equal(t, int64(2500), result.Metrics.TotalRequests)
	assert.InDelta(t, 83.33, result.Metrics.AvgTPS, 0.01)
	assert.InDelta(t, 12.5, result.Metrics.P50DurationMs, 0.01)
	assert.InDelta(t, 22.0, result.Metrics.P90DurationMs, 0.01)
	assert.InDelta(t, 31.4, result.Metrics.P95DurationMs, 0.01)
	assert.InDelta(t, 45.8, result.Metrics.P99DurationMs, 0.01)
	assert.InDelta(t, 0.2, result.Metrics.ErrorRatePct, 0.01)
}

func TestParseSummaryReport_FailedRunnerCrashReport(t *testing.T) {
	rawJSON := []byte(`{
		"status": "FAILED",
		"exit_code": 137,
		"error": "runner terminated with code 137 without generating summary",
		"timestamp": "2026-09-05T07:15:00Z",
		"sla_passed": false
	}`)

	result, err := service.ParseSummaryReport(rawJSON)
	require.NoError(t, err)
	assert.False(t, result.SLAPassed)
	assert.Equal(t, int64(0), result.Metrics.TotalIterations)
	assert.Equal(t, int64(0), result.Metrics.TotalRequests)
	assert.Equal(t, 0.0, result.Metrics.AvgTPS)
}

func TestParseSummaryReport_StringDurations(t *testing.T) {
	rawJSON := []byte(`{
		"scenario": "duration_string_test",
		"duration": "10s",
		"passed": true,
		"metrics": [
			{
				"name": "vuhive.vu.iteration_duration",
				"type": "duration",
				"count": 50,
				"p50": "10ms",
				"p90": "20ms",
				"p95": "30ms",
				"p99": "50ms"
			},
			{
				"name": "vuhive.vu.iterations_total",
				"type": "counter",
				"count": 50
			}
		]
	}`)

	result, err := service.ParseSummaryReport(rawJSON)
	require.NoError(t, err)
	assert.True(t, result.SLAPassed)
	assert.InDelta(t, 10.0, result.Metrics.P50DurationMs, 0.01)
	assert.InDelta(t, 20.0, result.Metrics.P90DurationMs, 0.01)
	assert.InDelta(t, 30.0, result.Metrics.P95DurationMs, 0.01)
	assert.InDelta(t, 50.0, result.Metrics.P99DurationMs, 0.01)
	assert.InDelta(t, 5.0, result.Metrics.AvgTPS, 0.01) // 50 iters / 10s = 5 TPS
}

func TestParseSummaryReport_EmptyOrInvalidJSON(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		_, err := service.ParseSummaryReport(nil)
		assert.ErrorIs(t, err, model.ErrValidation)

		_, err = service.ParseSummaryReport([]byte("   "))
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("malformed json", func(t *testing.T) {
		_, err := service.ParseSummaryReport([]byte(`{not valid json`))
		assert.ErrorIs(t, err, model.ErrValidation)
	})
}
