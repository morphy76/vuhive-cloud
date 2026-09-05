package service

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"gopkg.in/yaml.v3"
)

// WorkloadPartitioner divides load test concurrency and arrival rates across N distributed worker pods.
type WorkloadPartitioner struct{}

// NewWorkloadPartitioner creates a new WorkloadPartitioner domain service.
func NewWorkloadPartitioner() *WorkloadPartitioner {
	return &WorkloadPartitioner{}
}

// PartitionWorkload splits scenarios defined in rawYAML evenly across workerCount workers.
func (p *WorkloadPartitioner) PartitionWorkload(rawYAML []byte, workerCount int) ([]model.WorkloadPartition, error) {
	trimmed := bytes.TrimSpace(rawYAML)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("%w: configuration YAML cannot be empty", model.ErrValidation)
	}
	if workerCount < 1 {
		return nil, model.ErrInvalidWorkerCount
	}

	var rootNode yaml.Node
	if err := yaml.Unmarshal(trimmed, &rootNode); err != nil {
		return nil, fmt.Errorf("%w: invalid YAML: %v", model.ErrValidation, err)
	}

	scenariosNode := findNodeByKey(&rootNode, "scenarios")
	if scenariosNode == nil || scenariosNode.Kind != yaml.MappingNode || len(scenariosNode.Content) == 0 {
		return nil, model.ErrNoScenariosDefined
	}

	partitions := make([]model.WorkloadPartition, workerCount)
	for i := 0; i < workerCount; i++ {
		wp, err := p.partitionForIndex(&rootNode, i, workerCount)
		if err != nil {
			return nil, err
		}
		partitions[i] = *wp
	}

	return partitions, nil
}

// PartitionForWorker splits scenarios and returns the partition for the specified workerIndex.
func (p *WorkloadPartitioner) PartitionForWorker(rawYAML []byte, workerIndex, workerCount int) (*model.WorkloadPartition, error) {
	trimmed := bytes.TrimSpace(rawYAML)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("%w: configuration YAML cannot be empty", model.ErrValidation)
	}
	if workerCount < 1 {
		return nil, model.ErrInvalidWorkerCount
	}
	if workerIndex < 0 || workerIndex >= workerCount {
		return nil, model.ErrInvalidWorkerIndex
	}

	var rootNode yaml.Node
	if err := yaml.Unmarshal(trimmed, &rootNode); err != nil {
		return nil, fmt.Errorf("%w: invalid YAML: %v", model.ErrValidation, err)
	}

	scenariosNode := findNodeByKey(&rootNode, "scenarios")
	if scenariosNode == nil || scenariosNode.Kind != yaml.MappingNode || len(scenariosNode.Content) == 0 {
		return nil, model.ErrNoScenariosDefined
	}

	return p.partitionForIndex(&rootNode, workerIndex, workerCount)
}

func (p *WorkloadPartitioner) partitionForIndex(rootNode *yaml.Node, workerIndex, workerCount int) (*model.WorkloadPartition, error) {
	clonedRoot := cloneYAMLNode(rootNode)
	scenariosNode := findNodeByKey(clonedRoot, "scenarios")
	if scenariosNode == nil || scenariosNode.Kind != yaml.MappingNode {
		return nil, model.ErrNoScenariosDefined
	}

	scenariosSummary := make(map[string]model.ScenarioPartition)
	envOverrides := map[string]string{
		"VUHIVE_WORKER_INDEX": strconv.Itoa(workerIndex),
		"VUHIVE_WORKER_COUNT": strconv.Itoa(workerCount),
		"VUHIVE_POD_INDEX":    strconv.Itoa(workerIndex),
		"VUHIVE_POD_COUNT":    strconv.Itoa(workerCount),
	}

	// Iterate through each scenario mapping (key at 2*k, value at 2*k+1)
	for i := 0; i < len(scenariosNode.Content); i += 2 {
		scenarioName := scenariosNode.Content[i].Value
		scenarioBody := scenariosNode.Content[i+1]
		if scenarioBody.Kind != yaml.MappingNode {
			continue
		}

		typeNode := findNodeInMapping(scenarioBody, "type")
		scenarioType := "constant_vus"
		if typeNode != nil {
			scenarioType = strings.TrimSpace(typeNode.Value)
		}

		normName := normalizeEnvName(scenarioName)
		sp := model.ScenarioPartition{
			Type: scenarioType,
		}

		switch scenarioType {
		case "constant_vus":
			vusNode := findNodeInMapping(scenarioBody, "vus")
			if vusNode != nil {
				origVUs, _ := strconv.Atoi(vusNode.Value)
				partVUs := partitionValue(origVUs, workerIndex, workerCount)
				vusNode.Value = strconv.Itoa(partVUs)
				sp.VUs = partVUs
				envOverrides[fmt.Sprintf("VUHIVE_SCENARIOS_%s_VUS", normName)] = strconv.Itoa(partVUs)
			}

		case "arrival_rate":
			tpsNode := findNodeInMapping(scenarioBody, "target_tps")
			if tpsNode != nil {
				origTPS, _ := strconv.Atoi(tpsNode.Value)
				partTPS := partitionValue(origTPS, workerIndex, workerCount)
				tpsNode.Value = strconv.Itoa(partTPS)
				sp.TargetTPS = partTPS
				envOverrides[fmt.Sprintf("VUHIVE_SCENARIOS_%s_TARGET_TPS", normName)] = strconv.Itoa(partTPS)
			}

			maxVUsNode := findNodeInMapping(scenarioBody, "max_vus")
			if maxVUsNode != nil {
				origMaxVUs, _ := strconv.Atoi(maxVUsNode.Value)
				partMaxVUs := partitionValue(origMaxVUs, workerIndex, workerCount)
				maxVUsNode.Value = strconv.Itoa(partMaxVUs)
				sp.MaxVUs = partMaxVUs
				envOverrides[fmt.Sprintf("VUHIVE_SCENARIOS_%s_MAX_VUS", normName)] = strconv.Itoa(partMaxVUs)
			}

			burstNode := findNodeInMapping(scenarioBody, "burst_buffer")
			if burstNode != nil {
				origBurst, _ := strconv.Atoi(burstNode.Value)
				if origBurst > 0 {
					partBurst := partitionValue(origBurst, workerIndex, workerCount)
					burstNode.Value = strconv.Itoa(partBurst)
					sp.BurstBuffer = partBurst
					envOverrides[fmt.Sprintf("VUHIVE_SCENARIOS_%s_BURST_BUFFER", normName)] = strconv.Itoa(partBurst)
				}
			}

		case "ramping_vus":
			stagesNode := findNodeInMapping(scenarioBody, "stages")
			if stagesNode != nil && stagesNode.Kind == yaml.SequenceNode {
				stageTargets := make([]int, len(stagesNode.Content))
				for sIdx, stageItem := range stagesNode.Content {
					if stageItem.Kind != yaml.MappingNode {
						continue
					}
					targetNode := findNodeInMapping(stageItem, "target")
					if targetNode != nil {
						origTarget, _ := strconv.Atoi(targetNode.Value)
						partTarget := partitionValue(origTarget, workerIndex, workerCount)
						targetNode.Value = strconv.Itoa(partTarget)
						stageTargets[sIdx] = partTarget
					}
				}
				sp.StageTargets = stageTargets
			}
		}

		scenariosSummary[scenarioName] = sp
	}

	// Also check top-level or settings concurrency
	settingsNode := findNodeByKey(clonedRoot, "settings")
	if settingsNode != nil && settingsNode.Kind == yaml.MappingNode {
		concurrencyNode := findNodeInMapping(settingsNode, "concurrency")
		if concurrencyNode != nil {
			origConc, _ := strconv.Atoi(concurrencyNode.Value)
			concurrencyNode.Value = strconv.Itoa(partitionValue(origConc, workerIndex, workerCount))
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(clonedRoot); err != nil {
		return nil, fmt.Errorf("%w: failed encoding partitioned YAML: %v", model.ErrValidation, err)
	}
	_ = enc.Close()

	return model.NewWorkloadPartition(
		workerIndex,
		workerCount,
		buf.String(),
		envOverrides,
		scenariosSummary,
	)
}

func partitionValue(total, workerIndex, workerCount int) int {
	if total <= 0 || workerCount <= 0 {
		return 0
	}
	base := total / workerCount
	rem := total % workerCount
	if workerIndex < rem {
		return base + 1
	}
	return base
}

func normalizeEnvName(name string) string {
	cleaned := strings.ReplaceAll(name, "-", "_")
	cleaned = strings.ReplaceAll(cleaned, " ", "_")
	cleaned = strings.ReplaceAll(cleaned, ".", "_")
	return strings.ToUpper(cleaned)
}

func findNodeByKey(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode {
		for _, child := range node.Content {
			if res := findNodeByKey(child, key); res != nil {
				return res
			}
		}
		return nil
	}
	if node.Kind == yaml.MappingNode {
		return findNodeInMapping(node, key)
	}
	return nil
}

func findNodeInMapping(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func cloneYAMLNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	clone := &yaml.Node{
		Kind:        node.Kind,
		Style:       node.Style,
		Tag:         node.Tag,
		Value:       node.Value,
		Anchor:      node.Anchor,
		Alias:       node.Alias,
		HeadComment: node.HeadComment,
		LineComment: node.LineComment,
		FootComment: node.FootComment,
		Line:        node.Line,
		Column:      node.Column,
	}
	if len(node.Content) > 0 {
		clone.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			clone.Content[i] = cloneYAMLNode(child)
		}
	}
	return clone
}
