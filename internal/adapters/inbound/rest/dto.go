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
