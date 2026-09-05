package service_test

import (
	"fmt"
	"testing"


	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/morphy76/vuhive-cloud/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWorkloadPartitioner_Validation(t *testing.T) {
	p := service.NewWorkloadPartitioner()

	t.Run("empty YAML returns validation error", func(t *testing.T) {
		parts, err := p.PartitionWorkload([]byte(""), 3)
		assert.Nil(t, parts)
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("invalid YAML syntax returns validation error", func(t *testing.T) {
		parts, err := p.PartitionWorkload([]byte("invalid: [unclosed"), 3)
		assert.Nil(t, parts)
		assert.ErrorIs(t, err, model.ErrValidation)
	})


	t.Run("worker count less than 1 returns ErrInvalidWorkerCount", func(t *testing.T) {
		raw := []byte("version: '1.0'\nscenarios:\n  test:\n    type: constant_vus\n    vus: 10\n")
		parts, err := p.PartitionWorkload(raw, 0)
		assert.Nil(t, parts)
		assert.ErrorIs(t, err, model.ErrInvalidWorkerCount)

		parts, err = p.PartitionWorkload(raw, -2)
		assert.Nil(t, parts)
		assert.ErrorIs(t, err, model.ErrInvalidWorkerCount)
	})

	t.Run("missing scenarios returns ErrNoScenariosDefined", func(t *testing.T) {
		raw := []byte("version: '1.0'\ndefault_scenario: main\n")
		parts, err := p.PartitionWorkload(raw, 2)
		assert.Nil(t, parts)
		assert.ErrorIs(t, err, model.ErrNoScenariosDefined)
	})

	t.Run("PartitionForWorker validates worker index", func(t *testing.T) {
		raw := []byte("version: '1.0'\nscenarios:\n  test:\n    type: constant_vus\n    vus: 10\n")
		part, err := p.PartitionForWorker(raw, -1, 3)
		assert.Nil(t, part)
		assert.ErrorIs(t, err, model.ErrInvalidWorkerIndex)

		part, err = p.PartitionForWorker(raw, 3, 3)
		assert.Nil(t, part)
		assert.ErrorIs(t, err, model.ErrInvalidWorkerIndex)
	})
}

func TestWorkloadPartitioner_ConstantVUs(t *testing.T) {
	p := service.NewWorkloadPartitioner()

	t.Run("even division across workers", func(t *testing.T) {
		raw := []byte(`version: '1.0'
scenarios:
  checkout:
    type: constant_vus
    vus: 10
    run_period: 30s
`)
		parts, err := p.PartitionWorkload(raw, 2)
		require.NoError(t, err)
		require.Len(t, parts, 2)

		assert.Equal(t, 5, parts[0].Scenarios()["checkout"].VUs)
		assert.Equal(t, 5, parts[1].Scenarios()["checkout"].VUs)
		assert.Equal(t, 10, parts[0].Scenarios()["checkout"].VUs+parts[1].Scenarios()["checkout"].VUs)
	})

	t.Run("uneven division with remainder distribution", func(t *testing.T) {
		raw := []byte(`version: '1.0'
scenarios:
  checkout:
    type: constant_vus
    vus: 10
    run_period: 30s
`)
		parts, err := p.PartitionWorkload(raw, 3)
		require.NoError(t, err)
		require.Len(t, parts, 3)

		// 10 / 3 = 3 remainder 1 => [4, 3, 3]
		assert.Equal(t, 4, parts[0].Scenarios()["checkout"].VUs)
		assert.Equal(t, 3, parts[1].Scenarios()["checkout"].VUs)
		assert.Equal(t, 3, parts[2].Scenarios()["checkout"].VUs)

		sum := parts[0].Scenarios()["checkout"].VUs +
			parts[1].Scenarios()["checkout"].VUs +
			parts[2].Scenarios()["checkout"].VUs
		assert.Equal(t, 10, sum)
	})

	t.Run("VUs less than worker count", func(t *testing.T) {
		raw := []byte(`version: '1.0'
scenarios:
  light:
    type: constant_vus
    vus: 2
`)
		parts, err := p.PartitionWorkload(raw, 5)
		require.NoError(t, err)
		require.Len(t, parts, 5)

		// 2 / 5 => [1, 1, 0, 0, 0]
		assert.Equal(t, 1, parts[0].Scenarios()["light"].VUs)
		assert.Equal(t, 1, parts[1].Scenarios()["light"].VUs)
		assert.Equal(t, 0, parts[2].Scenarios()["light"].VUs)
		assert.Equal(t, 0, parts[3].Scenarios()["light"].VUs)
		assert.Equal(t, 0, parts[4].Scenarios()["light"].VUs)

		sum := 0
		for _, part := range parts {
			sum += part.Scenarios()["light"].VUs
		}
		assert.Equal(t, 2, sum)
	})

	t.Run("single worker preserves configuration", func(t *testing.T) {
		raw := []byte(`version: '1.0'
scenarios:
  single:
    type: constant_vus
    vus: 42
`)
		parts, err := p.PartitionWorkload(raw, 1)
		require.NoError(t, err)
		require.Len(t, parts, 1)
		assert.Equal(t, 42, parts[0].Scenarios()["single"].VUs)
	})
}

func TestWorkloadPartitioner_ArrivalRate(t *testing.T) {
	p := service.NewWorkloadPartitioner()

	raw := []byte(`version: '1.0'
scenarios:
  api_load:
    type: arrival_rate
    target_tps: 50
    max_vus: 60
    burst_buffer: 20
    run_period: 1m
`)

	parts, err := p.PartitionWorkload(raw, 3)
	require.NoError(t, err)
	require.Len(t, parts, 3)

	// target_tps: 50 / 3 => [17, 17, 16] (sum = 50)
	assert.Equal(t, 17, parts[0].Scenarios()["api_load"].TargetTPS)
	assert.Equal(t, 17, parts[1].Scenarios()["api_load"].TargetTPS)
	assert.Equal(t, 16, parts[2].Scenarios()["api_load"].TargetTPS)

	// max_vus: 60 / 3 => [20, 20, 20] (sum = 60)
	assert.Equal(t, 20, parts[0].Scenarios()["api_load"].MaxVUs)
	assert.Equal(t, 20, parts[1].Scenarios()["api_load"].MaxVUs)
	assert.Equal(t, 20, parts[2].Scenarios()["api_load"].MaxVUs)

	// burst_buffer: 20 / 3 => [7, 7, 6] (sum = 20)
	assert.Equal(t, 7, parts[0].Scenarios()["api_load"].BurstBuffer)
	assert.Equal(t, 7, parts[1].Scenarios()["api_load"].BurstBuffer)
	assert.Equal(t, 6, parts[2].Scenarios()["api_load"].BurstBuffer)

	// Assert sums match exactly
	sumTPS := 0
	sumMaxVUs := 0
	sumBurst := 0
	for _, part := range parts {
		sc := part.Scenarios()["api_load"]
		sumTPS += sc.TargetTPS
		sumMaxVUs += sc.MaxVUs
		sumBurst += sc.BurstBuffer
	}
	assert.Equal(t, 50, sumTPS)
	assert.Equal(t, 60, sumMaxVUs)
	assert.Equal(t, 20, sumBurst)
}

func TestWorkloadPartitioner_RampingVUs(t *testing.T) {
	p := service.NewWorkloadPartitioner()

	raw := []byte(`version: '1.0'
scenarios:
  ramp:
    type: ramping_vus
    stages:
      - duration: 30s
        target: 10
      - duration: 1m
        target: 25
      - duration: 10s
        target: 0
`)

	parts, err := p.PartitionWorkload(raw, 4)
	require.NoError(t, err)
	require.Len(t, parts, 4)

	// Stage 0: target 10 / 4 => [3, 3, 2, 2], sum = 10
	assert.Equal(t, 3, parts[0].Scenarios()["ramp"].StageTargets[0])
	assert.Equal(t, 3, parts[1].Scenarios()["ramp"].StageTargets[0])
	assert.Equal(t, 2, parts[2].Scenarios()["ramp"].StageTargets[0])
	assert.Equal(t, 2, parts[3].Scenarios()["ramp"].StageTargets[0])

	// Stage 1: target 25 / 4 => [7, 6, 6, 6], sum = 25
	assert.Equal(t, 7, parts[0].Scenarios()["ramp"].StageTargets[1])
	assert.Equal(t, 6, parts[1].Scenarios()["ramp"].StageTargets[1])
	assert.Equal(t, 6, parts[2].Scenarios()["ramp"].StageTargets[1])
	assert.Equal(t, 6, parts[3].Scenarios()["ramp"].StageTargets[1])

	// Stage 2: target 0 / 4 => [0, 0, 0, 0], sum = 0
	assert.Equal(t, 0, parts[0].Scenarios()["ramp"].StageTargets[2])
	assert.Equal(t, 0, parts[1].Scenarios()["ramp"].StageTargets[2])
	assert.Equal(t, 0, parts[2].Scenarios()["ramp"].StageTargets[2])
	assert.Equal(t, 0, parts[3].Scenarios()["ramp"].StageTargets[2])

	// Verify sum across workers for each stage
	for stageIdx, expectedTarget := range []int{10, 25, 0} {
		sumStage := 0
		for _, part := range parts {
			sumStage += part.Scenarios()["ramp"].StageTargets[stageIdx]
		}
		assert.Equal(t, expectedTarget, sumStage, "stage %d sum mismatch", stageIdx)
	}

	// Verify generated YAML retains durations and structure
	var parsedDoc map[string]any
	err = yaml.Unmarshal([]byte(parts[0].ConfigYAML()), &parsedDoc)
	require.NoError(t, err)
	scenariosMap := parsedDoc["scenarios"].(map[string]any)
	rampMap := scenariosMap["ramp"].(map[string]any)
	stagesSlice := rampMap["stages"].([]any)
	require.Len(t, stagesSlice, 3)

	s0 := stagesSlice[0].(map[string]any)
	assert.Equal(t, "30s", s0["duration"])
	assert.Equal(t, 3, s0["target"])

	s1 := stagesSlice[1].(map[string]any)
	assert.Equal(t, "1m", s1["duration"])
	assert.Equal(t, 7, s1["target"])

	s2 := stagesSlice[2].(map[string]any)
	assert.Equal(t, "10s", s2["duration"])
	assert.Equal(t, 0, s2["target"])
}

func TestWorkloadPartitioner_MultiScenarioAndPreservation(t *testing.T) {
	p := service.NewWorkloadPartitioner()

	raw := []byte(`version: "1.0"
default_scenario: checkout_flow

scenarios:
  checkout_flow:
    type: constant_vus
    vus: 10
    ramp_up: 5s
    run_period: 30s
    ramp_down: 5s
    vu_timeout: 2s
    http:
      base_url: "https://api.example.com"
      timeout: 5s
    thresholds:
      - metric: http_request_duration
        stat: p95
        operator: "<"
        target: "200ms"

  user_registration:
    type: arrival_rate
    target_tps: 50
    max_vus: 50
    ramp_up: 10s
    run_period: 1m
    vu_timeout: 1s
`)

	parts, err := p.PartitionWorkload(raw, 3)
	require.NoError(t, err)
	require.Len(t, parts, 3)

	// Verify worker 0
	w0 := parts[0]
	assert.Equal(t, 0, w0.WorkerIndex())
	assert.Equal(t, 3, w0.WorkerCount())

	// Environment overrides
	env0 := w0.EnvOverrides()
	assert.Equal(t, "0", env0["VUHIVE_WORKER_INDEX"])
	assert.Equal(t, "3", env0["VUHIVE_WORKER_COUNT"])
	assert.Equal(t, "0", env0["VUHIVE_POD_INDEX"])
	assert.Equal(t, "3", env0["VUHIVE_POD_COUNT"])
	assert.Equal(t, "4", env0["VUHIVE_SCENARIOS_CHECKOUT_FLOW_VUS"])
	assert.Equal(t, "17", env0["VUHIVE_SCENARIOS_USER_REGISTRATION_TARGET_TPS"])
	assert.Equal(t, "17", env0["VUHIVE_SCENARIOS_USER_REGISTRATION_MAX_VUS"])

	// Parse generated YAML for worker 0 to ensure other fields are preserved
	var parsedDoc map[string]any
	err = yaml.Unmarshal([]byte(w0.ConfigYAML()), &parsedDoc)
	require.NoError(t, err)

	assert.Equal(t, "1.0", parsedDoc["version"])
	assert.Equal(t, "checkout_flow", parsedDoc["default_scenario"])

	scenariosMap := parsedDoc["scenarios"].(map[string]any)
	checkoutMap := scenariosMap["checkout_flow"].(map[string]any)
	assert.Equal(t, "constant_vus", checkoutMap["type"])
	assert.Equal(t, 4, checkoutMap["vus"])
	assert.Equal(t, "5s", checkoutMap["ramp_up"])
	assert.Equal(t, "30s", checkoutMap["run_period"])
	assert.Equal(t, "2s", checkoutMap["vu_timeout"])

	httpMap := checkoutMap["http"].(map[string]any)
	assert.Equal(t, "https://api.example.com", httpMap["base_url"])

	thresholdsSlice := checkoutMap["thresholds"].([]any)
	require.Len(t, thresholdsSlice, 1)
	th0 := thresholdsSlice[0].(map[string]any)
	assert.Equal(t, "http_request_duration", th0["metric"])
	assert.Equal(t, "p95", th0["stat"])
	assert.Equal(t, "<", th0["operator"])
	assert.Equal(t, "200ms", th0["target"])

	// Verify worker 2
	w2 := parts[2]
	env2 := w2.EnvOverrides()
	assert.Equal(t, "2", env2["VUHIVE_WORKER_INDEX"])
	assert.Equal(t, "3", env2["VUHIVE_SCENARIOS_CHECKOUT_FLOW_VUS"])
	assert.Equal(t, "16", env2["VUHIVE_SCENARIOS_USER_REGISTRATION_TARGET_TPS"])
	assert.Equal(t, "16", env2["VUHIVE_SCENARIOS_USER_REGISTRATION_MAX_VUS"])

	// Verify PartitionForWorker matches PartitionWorkload for worker 2
	w2Single, err := p.PartitionForWorker(raw, 2, 3)
	require.NoError(t, err)
	assert.Equal(t, w2.ConfigYAML(), w2Single.ConfigYAML())
	assert.Equal(t, w2.EnvOverrides(), w2Single.EnvOverrides())
	assert.Equal(t, w2.Scenarios(), w2Single.Scenarios())
}

func TestWorkloadPartitioner_MathematicalInvariant(t *testing.T) {
	p := service.NewWorkloadPartitioner()

	// Invariant: sum of partitions across N workers must equal original value for any V and N
	testValues := []int{1, 2, 3, 5, 7, 10, 16, 33, 50, 99, 100, 250, 1000, 7777}
	workerCounts := []int{1, 2, 3, 4, 5, 7, 10, 12, 16, 25, 32, 50}

	for _, v := range testValues {
		for _, n := range workerCounts {
			name := fmt.Sprintf("V=%d_N=%d", v, n)
			t.Run(name, func(t *testing.T) {
				rawYAML := fmt.Sprintf(`version: '1.0'
scenarios:
  test_scenario:
    type: constant_vus
    vus: %d
`, v)
				parts, err := p.PartitionWorkload([]byte(rawYAML), n)
				require.NoError(t, err)
				require.Len(t, parts, n)

				sum := 0
				for _, part := range parts {
					sum += part.Scenarios()["test_scenario"].VUs
				}
				assert.Equal(t, v, sum, "Sum of partitioned VUs must match exactly")
			})
		}
	}
}

func TestWorkloadPartitioner_SettingsConcurrencyFallback(t *testing.T) {
	p := service.NewWorkloadPartitioner()

	// Top-level or settings concurrency fallback
	raw := []byte(`version: '1'
settings:
  concurrency: 50
scenarios:
  default:
    type: constant_vus
    vus: 50
`)

	parts, err := p.PartitionWorkload(raw, 3)
	require.NoError(t, err)
	require.Len(t, parts, 3)

	var doc map[string]any
	err = yaml.Unmarshal([]byte(parts[0].ConfigYAML()), &doc)
	require.NoError(t, err)
	settings := doc["settings"].(map[string]any)
	assert.Equal(t, 17, settings["concurrency"])
}
