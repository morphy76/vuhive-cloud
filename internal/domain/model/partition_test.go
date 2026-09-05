package model_test

import (
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkloadPartition_Validation(t *testing.T) {
	tests := []struct {
		name        string
		workerIndex int
		workerCount int
		expectedErr error
	}{
		{
			name:        "invalid zero worker count",
			workerIndex: 0,
			workerCount: 0,
			expectedErr: model.ErrInvalidWorkerCount,
		},
		{
			name:        "invalid negative worker count",
			workerIndex: 0,
			workerCount: -1,
			expectedErr: model.ErrInvalidWorkerCount,
		},
		{
			name:        "negative worker index",
			workerIndex: -1,
			workerCount: 3,
			expectedErr: model.ErrInvalidWorkerIndex,
		},
		{
			name:        "worker index equals worker count",
			workerIndex: 3,
			workerCount: 3,
			expectedErr: model.ErrInvalidWorkerIndex,
		},
		{
			name:        "worker index exceeds worker count",
			workerIndex: 5,
			workerCount: 3,
			expectedErr: model.ErrInvalidWorkerIndex,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wp, err := model.NewWorkloadPartition(
				tc.workerIndex,
				tc.workerCount,
				"version: '1.0'",
				map[string]string{"VUHIVE_WORKER_INDEX": "0"},
				nil,
			)
			assert.Nil(t, wp)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func TestNewWorkloadPartition_Success(t *testing.T) {
	scenarios := map[string]model.ScenarioPartition{
		"checkout": {
			Type: "constant_vus",
			VUs:  5,
		},
		"browse": {
			Type:         "ramping_vus",
			StageTargets: []int{2, 5, 10},
		},
		"api_calls": {
			Type:        "arrival_rate",
			TargetTPS:   25,
			MaxVUs:      50,
			BurstBuffer: 10,
		},
	}
	envs := map[string]string{
		"VUHIVE_WORKER_INDEX": "1",
		"VUHIVE_WORKER_COUNT": "3",
	}
	yamlContent := "version: '1.0'\nscenarios: {}"

	wp, err := model.NewWorkloadPartition(1, 3, yamlContent, envs, scenarios)
	require.NoError(t, err)
	require.NotNil(t, wp)

	assert.Equal(t, 1, wp.WorkerIndex())
	assert.Equal(t, 3, wp.WorkerCount())
	assert.Equal(t, yamlContent, wp.ConfigYAML())
	assert.Equal(t, envs, wp.EnvOverrides())
	assert.Equal(t, scenarios, wp.Scenarios())

	// Immutability checks
	retEnvs := wp.EnvOverrides()
	retEnvs["MUTATE"] = "bad"
	assert.NotContains(t, wp.EnvOverrides(), "MUTATE")
}
