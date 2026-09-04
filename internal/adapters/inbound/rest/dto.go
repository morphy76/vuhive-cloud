package rest

import (
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// ArtifactResponse represents the JSON payload for a compiled binary artifact.
type ArtifactResponse struct {
	ID             string `json:"id"`
	SuiteID        string `json:"suite_id"`
	Platform       string `json:"platform"`
	S3BinaryKey    string `json:"s3_binary_key,omitempty"`
	SHA256Checksum string `json:"sha256_checksum,omitempty"`
	BuildLogsS3Key string `json:"build_logs_s3_key,omitempty"`
	Status         string `json:"status"`
	ErrorMessage   string `json:"error_message,omitempty"`
	CreatedAt      string `json:"created_at"`
}

// ArtifactListResponse represents the response containing a list of artifacts.
type ArtifactListResponse struct {
	Artifacts []ArtifactResponse `json:"artifacts"`
	Count     int                `json:"count"`
}

// BuildTriggerResponse represents the response returned when an asynchronous build is initiated.
type BuildTriggerResponse struct {
	Message   string             `json:"message"`
	Artifacts []ArtifactResponse `json:"artifacts"`
}

// ErrorResponse represents a standardized API error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ToArtifactResponse converts a domain model.Artifact into an ArtifactResponse DTO.
func ToArtifactResponse(a *model.Artifact) ArtifactResponse {
	return ArtifactResponse{
		ID:             a.ID(),
		SuiteID:        a.SuiteID(),
		Platform:       string(a.Platform()),
		S3BinaryKey:    a.S3BinaryKey(),
		SHA256Checksum: a.SHA256Checksum(),
		BuildLogsS3Key: a.BuildLogsS3Key(),
		Status:         string(a.Status()),
		ErrorMessage:   a.ErrorMessage(),
		CreatedAt:      a.CreatedAt().Format(time.RFC3339),
	}
}

// ToArtifactListResponse converts a slice of domain model.Artifact into an ArtifactListResponse DTO.
func ToArtifactListResponse(artifacts []*model.Artifact) ArtifactListResponse {
	items := make([]ArtifactResponse, 0, len(artifacts))
	for _, a := range artifacts {
		items = append(items, ToArtifactResponse(a))
	}
	return ArtifactListResponse{
		Artifacts: items,
		Count:     len(items),
	}
}

// TolerationDTO represents a Kubernetes toleration in HTTP requests and responses.
type TolerationDTO struct {
	Key               string `json:"key,omitempty"`
	Operator          string `json:"operator,omitempty"`
	Value             string `json:"value,omitempty"`
	Effect            string `json:"effect,omitempty"`
	TolerationSeconds *int64 `json:"toleration_seconds,omitempty"`
}

// NodeAffinityTermDTO represents a Kubernetes node affinity term in HTTP requests and responses.
type NodeAffinityTermDTO struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// AffinityDTO represents Kubernetes node affinity in HTTP requests and responses.
type AffinityDTO struct {
	NodeSelectorTerms []NodeAffinityTermDTO `json:"node_selector_terms,omitempty"`
}

// CreateProfileRequest represents the JSON request payload to create a new RunnerProfile.
type CreateProfileRequest struct {
	Name          string            `json:"name" binding:"required"`
	Description   string            `json:"description,omitempty"`
	RunnerImage   string            `json:"runner_image,omitempty"`
	CPURequest    string            `json:"cpu_request,omitempty"`
	CPULimit      string            `json:"cpu_limit,omitempty"`
	MemoryRequest string            `json:"memory_request,omitempty"`
	MemoryLimit   string            `json:"memory_limit,omitempty"`
	NodeSelector  map[string]string `json:"node_selector,omitempty"`
	Affinity      *AffinityDTO      `json:"affinity,omitempty"`
	Tolerations   []TolerationDTO   `json:"tolerations,omitempty"`
}

// UpdateProfileRequest represents the JSON request payload to update an existing RunnerProfile.
type UpdateProfileRequest struct {
	Name          string            `json:"name" binding:"required"`
	Description   string            `json:"description,omitempty"`
	RunnerImage   string            `json:"runner_image,omitempty"`
	CPURequest    string            `json:"cpu_request,omitempty"`
	CPULimit      string            `json:"cpu_limit,omitempty"`
	MemoryRequest string            `json:"memory_request,omitempty"`
	MemoryLimit   string            `json:"memory_limit,omitempty"`
	NodeSelector  map[string]string `json:"node_selector,omitempty"`
	Affinity      *AffinityDTO      `json:"affinity,omitempty"`
	Tolerations   []TolerationDTO   `json:"tolerations,omitempty"`
}

// ProfileResponse represents the JSON response payload for a RunnerProfile entity.
type ProfileResponse struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	RunnerImage   string            `json:"runner_image"`
	CPURequest    string            `json:"cpu_request"`
	CPULimit      string            `json:"cpu_limit"`
	MemoryRequest string            `json:"memory_request"`
	MemoryLimit   string            `json:"memory_limit"`
	NodeSelector  map[string]string `json:"node_selector"`
	Affinity      AffinityDTO       `json:"affinity"`
	Tolerations   []TolerationDTO   `json:"tolerations"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
}

// ProfileListResponse represents the JSON response payload containing a list of RunnerProfiles.
type ProfileListResponse struct {
	Profiles []ProfileResponse `json:"profiles"`
	Count    int               `json:"count"`
}

// FromAffinityDTO converts an AffinityDTO into a domain model.Affinity.
func FromAffinityDTO(dto *AffinityDTO) model.Affinity {
	if dto == nil {
		return model.Affinity{}
	}
	terms := make([]model.NodeAffinityTerm, 0, len(dto.NodeSelectorTerms))
	for _, t := range dto.NodeSelectorTerms {
		terms = append(terms, model.NodeAffinityTerm{
			Key:      t.Key,
			Operator: t.Operator,
			Values:   t.Values,
		})
	}
	return model.Affinity{NodeSelectorTerms: terms}
}

// ToAffinityDTO converts a domain model.Affinity into an AffinityDTO.
func ToAffinityDTO(affinity model.Affinity) AffinityDTO {
	terms := make([]NodeAffinityTermDTO, 0, len(affinity.NodeSelectorTerms))
	for _, t := range affinity.NodeSelectorTerms {
		terms = append(terms, NodeAffinityTermDTO{
			Key:      t.Key,
			Operator: t.Operator,
			Values:   t.Values,
		})
	}
	return AffinityDTO{NodeSelectorTerms: terms}
}

// FromTolerationsDTO converts a slice of TolerationDTO into domain model.Toleration slices.
func FromTolerationsDTO(dtos []TolerationDTO) []model.Toleration {
	if len(dtos) == 0 {
		return nil
	}
	tolerations := make([]model.Toleration, 0, len(dtos))
	for _, t := range dtos {
		tolerations = append(tolerations, model.Toleration{
			Key:               t.Key,
			Operator:          t.Operator,
			Value:             t.Value,
			Effect:            t.Effect,
			TolerationSeconds: t.TolerationSeconds,
		})
	}
	return tolerations
}

// ToTolerationsDTO converts a slice of domain model.Toleration into TolerationDTO slices.
func ToTolerationsDTO(tolerations []model.Toleration) []TolerationDTO {
	if len(tolerations) == 0 {
		return []TolerationDTO{}
	}
	dtos := make([]TolerationDTO, 0, len(tolerations))
	for _, t := range tolerations {
		dtos = append(dtos, TolerationDTO{
			Key:               t.Key,
			Operator:          t.Operator,
			Value:             t.Value,
			Effect:            t.Effect,
			TolerationSeconds: t.TolerationSeconds,
		})
	}
	return dtos
}

// ToProfileResponse converts a domain model.RunnerProfile into a ProfileResponse DTO.
func ToProfileResponse(p *model.RunnerProfile) ProfileResponse {
	nodeSelector := p.NodeSelector()
	if nodeSelector == nil {
		nodeSelector = make(map[string]string)
	}

	return ProfileResponse{
		ID:            p.ID(),
		Name:          p.Name(),
		Description:   p.Description(),
		RunnerImage:   p.RunnerImage(),
		CPURequest:    p.Resources().CPURequest(),
		CPULimit:      p.Resources().CPULimit(),
		MemoryRequest: p.Resources().MemoryRequest(),
		MemoryLimit:   p.Resources().MemoryLimit(),
		NodeSelector:  nodeSelector,
		Affinity:      ToAffinityDTO(p.Affinity()),
		Tolerations:   ToTolerationsDTO(p.Tolerations()),
		CreatedAt:     p.CreatedAt().Format(time.RFC3339),
		UpdatedAt:     p.UpdatedAt().Format(time.RFC3339),
	}
}

// ToProfileListResponse converts a slice of domain model.RunnerProfile into a ProfileListResponse DTO.
func ToProfileListResponse(profiles []*model.RunnerProfile) ProfileListResponse {
	items := make([]ProfileResponse, 0, len(profiles))
	for _, p := range profiles {
		items = append(items, ToProfileResponse(p))
	}
	return ProfileListResponse{
		Profiles: items,
		Count:    len(items),
	}
}

