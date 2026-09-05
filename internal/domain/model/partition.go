package model

import (
	"maps"
)

// ScenarioPartition holds the partitioned execution parameters for a single scenario.
type ScenarioPartition struct {
	Type         string
	VUs          int
	TargetTPS    int
	MaxVUs       int
	BurstBuffer  int
	StageTargets []int
}

// WorkloadPartition encapsulates the partitioned configuration and runtime overrides for a worker instance.
type WorkloadPartition struct {
	workerIndex  int
	workerCount  int
	configYAML   string
	envOverrides map[string]string
	scenarios    map[string]ScenarioPartition
}

// NewWorkloadPartition constructs and validates a WorkloadPartition value.
func NewWorkloadPartition(
	workerIndex int,
	workerCount int,
	configYAML string,
	envOverrides map[string]string,
	scenarios map[string]ScenarioPartition,
) (*WorkloadPartition, error) {
	if workerCount < 1 {
		return nil, ErrInvalidWorkerCount
	}
	if workerIndex < 0 || workerIndex >= workerCount {
		return nil, ErrInvalidWorkerIndex
	}

	envsCopy := make(map[string]string, len(envOverrides))
	maps.Copy(envsCopy, envOverrides)

	scenariosCopy := make(map[string]ScenarioPartition, len(scenarios))
	for k, v := range scenarios {
		var stagesCopy []int
		if len(v.StageTargets) > 0 {
			stagesCopy = make([]int, len(v.StageTargets))
			copy(stagesCopy, v.StageTargets)
		}
		v.StageTargets = stagesCopy
		scenariosCopy[k] = v
	}

	return &WorkloadPartition{
		workerIndex:  workerIndex,
		workerCount:  workerCount,
		configYAML:   configYAML,
		envOverrides: envsCopy,
		scenarios:    scenariosCopy,
	}, nil
}

// WorkerIndex returns the 0-based worker index.
func (p *WorkloadPartition) WorkerIndex() int {
	return p.workerIndex
}

// WorkerCount returns the total number of partitioned workers.
func (p *WorkloadPartition) WorkerCount() int {
	return p.workerCount
}

// ConfigYAML returns the partitioned YAML configuration for the worker.
func (p *WorkloadPartition) ConfigYAML() string {
	return p.configYAML
}

// EnvOverrides returns a defensive copy of environment variable overrides for the worker.
func (p *WorkloadPartition) EnvOverrides() map[string]string {
	res := make(map[string]string, len(p.envOverrides))
	maps.Copy(res, p.envOverrides)
	return res
}

// Scenarios returns a defensive copy of partitioned scenarios.
func (p *WorkloadPartition) Scenarios() map[string]ScenarioPartition {
	res := make(map[string]ScenarioPartition, len(p.scenarios))
	for k, v := range p.scenarios {
		var stagesCopy []int
		if len(v.StageTargets) > 0 {
			stagesCopy = make([]int, len(v.StageTargets))
			copy(stagesCopy, v.StageTargets)
		}
		v.StageTargets = stagesCopy
		res[k] = v
	}
	return res
}
