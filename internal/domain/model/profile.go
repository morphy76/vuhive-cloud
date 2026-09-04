package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Toleration represents a Kubernetes pod toleration for dedicated load generator nodes.
type Toleration struct {
	Key               string
	Operator          string
	Value             string
	Effect            string
	TolerationSeconds *int64
}

// Validate checks that the toleration adheres to Kubernetes specifications.
func (t Toleration) Validate() error {
	op := strings.TrimSpace(t.Operator)
	if op == "" {
		op = "Equal"
	}
	switch op {
	case "Exists":
		if strings.TrimSpace(t.Value) != "" {
			return fmt.Errorf("%w: operator Exists cannot have a value", ErrInvalidToleration)
		}
	case "Equal":
		if strings.TrimSpace(t.Key) == "" {
			return fmt.Errorf("%w: operator Equal requires a key", ErrInvalidToleration)
		}
	default:
		return fmt.Errorf("%w: unsupported operator %q (must be Exists or Equal)", ErrInvalidToleration, t.Operator)
	}

	effect := strings.TrimSpace(t.Effect)
	if effect != "" {
		switch effect {
		case "NoSchedule", "PreferNoSchedule", "NoExecute":
		default:
			return fmt.Errorf("%w: unsupported effect %q", ErrInvalidToleration, t.Effect)
		}
	}

	if t.TolerationSeconds != nil {
		if effect != "NoExecute" {
			return fmt.Errorf("%w: tolerationSeconds only allowed with NoExecute effect", ErrInvalidToleration)
		}
		if *t.TolerationSeconds < 0 {
			return fmt.Errorf("%w: tolerationSeconds cannot be negative", ErrInvalidToleration)
		}
	}

	return nil
}

// NodeAffinityTerm represents a single matching condition for node affinity scheduling.
type NodeAffinityTerm struct {
	Key      string
	Operator string
	Values   []string
}

// Validate checks that the node affinity term conforms to Kubernetes NodeSelectorRequirement specifications.
func (t NodeAffinityTerm) Validate() error {
	key := strings.TrimSpace(t.Key)
	if key == "" {
		return fmt.Errorf("%w: affinity term key cannot be empty", ErrInvalidAffinity)
	}

	switch t.Operator {
	case "In", "NotIn":
		if len(t.Values) == 0 {
			return fmt.Errorf("%w: operator %q requires at least one value", ErrInvalidAffinity, t.Operator)
		}
	case "Exists", "DoesNotExist":
		if len(t.Values) > 0 {
			return fmt.Errorf("%w: operator %q must not have values", ErrInvalidAffinity, t.Operator)
		}
	case "Gt", "Lt":
		if len(t.Values) != 1 {
			return fmt.Errorf("%w: operator %q requires exactly one value", ErrInvalidAffinity, t.Operator)
		}
		if _, err := strconv.ParseInt(t.Values[0], 10, 64); err != nil {
			return fmt.Errorf("%w: operator %q requires an integer value, got %q", ErrInvalidAffinity, t.Operator, t.Values[0])
		}
	default:
		return fmt.Errorf("%w: unsupported affinity operator %q", ErrInvalidAffinity, t.Operator)
	}
	return nil
}

// Affinity represents node affinity specifications for runner pods.
type Affinity struct {
	NodeSelectorTerms []NodeAffinityTerm
}

// Validate checks that all node selector terms in the affinity are valid.
func (a Affinity) Validate() error {
	for _, term := range a.NodeSelectorTerms {
		if err := term.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ResourceRequirements represents CPU and memory requests and limits for test runners.
type ResourceRequirements struct {
	cpuRequest    string
	cpuLimit      string
	memoryRequest string
	memoryLimit   string
}

// NewResourceRequirements validates and creates a new ResourceRequirements value object.
func NewResourceRequirements(cpuRequest, cpuLimit, memoryRequest, memoryLimit string) (ResourceRequirements, error) {
	reqMilli, err := parseCPUMillicores(cpuRequest)
	if err != nil {
		return ResourceRequirements{}, fmt.Errorf("%w: invalid cpu_request: %v", ErrInvalidResourceQuantity, err)
	}
	limMilli, err := parseCPUMillicores(cpuLimit)
	if err != nil {
		return ResourceRequirements{}, fmt.Errorf("%w: invalid cpu_limit: %v", ErrInvalidResourceQuantity, err)
	}
	if reqMilli > limMilli {
		return ResourceRequirements{}, fmt.Errorf("%w: cpu_request (%s) cannot exceed cpu_limit (%s)", ErrInvalidResourceQuantity, cpuRequest, cpuLimit)
	}

	reqBytes, err := parseMemoryBytes(memoryRequest)
	if err != nil {
		return ResourceRequirements{}, fmt.Errorf("%w: invalid memory_request: %v", ErrInvalidResourceQuantity, err)
	}
	limBytes, err := parseMemoryBytes(memoryLimit)
	if err != nil {
		return ResourceRequirements{}, fmt.Errorf("%w: invalid memory_limit: %v", ErrInvalidResourceQuantity, err)
	}
	if reqBytes > limBytes {
		return ResourceRequirements{}, fmt.Errorf("%w: memory_request (%s) cannot exceed memory_limit (%s)", ErrInvalidResourceQuantity, memoryRequest, memoryLimit)
	}

	return ResourceRequirements{
		cpuRequest:    strings.TrimSpace(cpuRequest),
		cpuLimit:      strings.TrimSpace(cpuLimit),
		memoryRequest: strings.TrimSpace(memoryRequest),
		memoryLimit:   strings.TrimSpace(memoryLimit),
	}, nil
}

// CPURequest returns the CPU request string.
func (r ResourceRequirements) CPURequest() string {
	return r.cpuRequest
}

// CPULimit returns the CPU limit string.
func (r ResourceRequirements) CPULimit() string {
	return r.cpuLimit
}

// MemoryRequest returns the Memory request string.
func (r ResourceRequirements) MemoryRequest() string {
	return r.memoryRequest
}

// MemoryLimit returns the Memory limit string.
func (r ResourceRequirements) MemoryLimit() string {
	return r.memoryLimit
}

// RunnerProfile represents a reusable compute configuration for executing test runners on Kubernetes.
type RunnerProfile struct {
	id           string
	name         string
	description  string
	runnerImage  string
	resources    ResourceRequirements
	nodeSelector map[string]string
	affinity     Affinity
	tolerations  []Toleration
	createdAt    time.Time
	updatedAt    time.Time
}

const DefaultRunnerImage = "alpine:3.20"

// NewRunnerProfile creates a new RunnerProfile.
func NewRunnerProfile(
	name, description, runnerImage string,
	resources ResourceRequirements,
	nodeSelector map[string]string,
	affinity Affinity,
	tolerations []Toleration,
) (*RunnerProfile, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, ErrEmptyName
	}

	if err := affinity.Validate(); err != nil {
		return nil, err
	}
	for _, tol := range tolerations {
		if err := tol.Validate(); err != nil {
			return nil, err
		}
	}

	img := strings.TrimSpace(runnerImage)
	if img == "" {
		img = DefaultRunnerImage
	}

	now := time.Now().UTC()
	return &RunnerProfile{
		id:           uuid.NewString(),
		name:         trimmedName,
		description:  strings.TrimSpace(description),
		runnerImage:  img,
		resources:    resources,
		nodeSelector: nodeSelector,
		affinity:     affinity,
		tolerations:  tolerations,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// NewRunnerProfileWithID reconstructs a RunnerProfile from persistence.
func NewRunnerProfileWithID(
	id, name, description, runnerImage string,
	resources ResourceRequirements,
	nodeSelector map[string]string,
	affinity Affinity,
	tolerations []Toleration,
	createdAt, updatedAt time.Time,
) (*RunnerProfile, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, ErrEmptyName
	}

	if err := affinity.Validate(); err != nil {
		return nil, err
	}
	for _, tol := range tolerations {
		if err := tol.Validate(); err != nil {
			return nil, err
		}
	}

	img := strings.TrimSpace(runnerImage)
	if img == "" {
		img = DefaultRunnerImage
	}

	return &RunnerProfile{
		id:           id,
		name:         trimmedName,
		description:  description,
		runnerImage:  img,
		resources:    resources,
		nodeSelector: nodeSelector,
		affinity:     affinity,
		tolerations:  tolerations,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}, nil
}

// ID returns the unique identifier.
func (p *RunnerProfile) ID() string {
	return p.id
}

// EntityID implements the Entity interface.
func (p *RunnerProfile) EntityID() string {
	return p.id
}

// Name returns the profile name.
func (p *RunnerProfile) Name() string {
	return p.name
}

// Description returns the profile description.
func (p *RunnerProfile) Description() string {
	return p.description
}

// RunnerImage returns the container image to run.
func (p *RunnerProfile) RunnerImage() string {
	return p.runnerImage
}

// Resources returns the compute resource requirements.
func (p *RunnerProfile) Resources() ResourceRequirements {
	return p.resources
}

// NodeSelector returns the node selector map.
func (p *RunnerProfile) NodeSelector() map[string]string {
	return p.nodeSelector
}

// Affinity returns the pod affinity rules.
func (p *RunnerProfile) Affinity() Affinity {
	return p.affinity
}

// Tolerations returns the list of pod tolerations.
func (p *RunnerProfile) Tolerations() []Toleration {
	return p.tolerations
}

// CreatedAt returns when the profile was created.
func (p *RunnerProfile) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns when the profile was last updated.
func (p *RunnerProfile) UpdatedAt() time.Time {
	return p.updatedAt
}

// UpdateResources updates the compute resource specifications.
func (p *RunnerProfile) UpdateResources(resources ResourceRequirements) error {
	p.resources = resources
	p.updatedAt = time.Now().UTC()
	return nil
}

// UpdateDetails updates the runner profile configuration and attributes.
func (p *RunnerProfile) UpdateDetails(
	name, description, runnerImage string,
	resources ResourceRequirements,
	nodeSelector map[string]string,
	affinity Affinity,
	tolerations []Toleration,
) error {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return ErrEmptyName
	}

	if err := affinity.Validate(); err != nil {
		return err
	}
	for _, tol := range tolerations {
		if err := tol.Validate(); err != nil {
			return err
		}
	}

	img := strings.TrimSpace(runnerImage)
	if img == "" {
		img = DefaultRunnerImage
	}

	p.name = trimmedName
	p.description = strings.TrimSpace(description)
	p.runnerImage = img
	p.resources = resources
	p.nodeSelector = nodeSelector
	p.affinity = affinity
	p.tolerations = tolerations
	p.updatedAt = time.Now().UTC()

	return nil
}

func parseCPUMillicores(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty cpu string")
	}
	if strings.HasSuffix(s, "m") {
		numStr := strings.TrimSuffix(s, "m")
		val, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil || val <= 0 {
			return 0, fmt.Errorf("invalid millicores: %s", s)
		}
		return val, nil
	}
	cores, err := strconv.ParseFloat(s, 64)
	if err != nil || cores <= 0 {
		return 0, fmt.Errorf("invalid cpu cores: %s", s)
	}
	return int64(cores * 1000), nil
}

func parseMemoryBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty memory string")
	}

	suffixes := []struct {
		suffix string
		mult   int64
	}{
		{"Pi", 1024 * 1024 * 1024 * 1024 * 1024},
		{"Ti", 1024 * 1024 * 1024 * 1024},
		{"Gi", 1024 * 1024 * 1024},
		{"Mi", 1024 * 1024},
		{"Ki", 1024},
		{"P", 1000 * 1000 * 1000 * 1000 * 1000},
		{"T", 1000 * 1000 * 1000 * 1000},
		{"G", 1000 * 1000 * 1000},
		{"M", 1000 * 1000},
		{"K", 1000},
	}

	for _, suff := range suffixes {
		if strings.HasSuffix(s, suff.suffix) {
			numStr := strings.TrimSuffix(s, suff.suffix)
			val, err := strconv.ParseFloat(numStr, 64)
			if err != nil || val <= 0 {
				return 0, fmt.Errorf("invalid memory number: %s", s)
			}
			return int64(val * float64(suff.mult)), nil
		}
	}

	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil || val <= 0 {
		return 0, fmt.Errorf("invalid memory bytes: %s", s)
	}
	return val, nil
}

// Compile-time interface assertion
var _ Entity = (*RunnerProfile)(nil)
