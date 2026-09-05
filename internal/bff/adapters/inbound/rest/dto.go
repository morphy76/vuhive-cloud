package rest

import (
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
)

// HealthResponse represents the response payload for /healthz.
type HealthResponse struct {
	Status string `json:"status"`
}

// VersionResponse represents the response payload for /version.
type VersionResponse struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
}

// StatusResponse represents the aggregated BFF and control plane status payload.
type StatusResponse struct {
	BFFStatus           string                 `json:"bff_status"`
	BFFVersion          string                 `json:"bff_version"`
	ControlPlaneStatus  string                 `json:"control_plane_status"`
	ControlPlaneVersion string                 `json:"control_plane_version,omitempty"`
	Timestamp           time.Time              `json:"timestamp"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// CreateSessionRequest encapsulates the request body for creating a BFF session.
type CreateSessionRequest struct {
	SessionID  string            `json:"session_id" binding:"required"`
	UserID     string            `json:"user_id" binding:"required"`
	TTLSeconds int64             `json:"ttl_seconds"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// SessionResponse represents the serialized client session payload.
type SessionResponse struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt time.Time         `json:"expires_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ErrorResponse represents a standardized JSON error message.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ToStatusResponse maps application SystemStatus to inbound REST DTO.
func ToStatusResponse(s *inbound.SystemStatus) StatusResponse {
	return StatusResponse{
		BFFStatus:           s.BFFStatus,
		BFFVersion:          s.BFFVersion,
		ControlPlaneStatus:  s.ControlPlaneStatus,
		ControlPlaneVersion: s.ControlPlaneVersion,
		Timestamp:           s.Timestamp,
		Metadata:            s.Metadata,
	}
}

// ToSessionResponse maps domain ClientSession to inbound REST DTO.
func ToSessionResponse(s *model.ClientSession) SessionResponse {
	return SessionResponse{
		ID:        string(s.ID),
		UserID:    s.UserID,
		CreatedAt: s.CreatedAt,
		ExpiresAt: s.ExpiresAt,
		Metadata:  s.Metadata,
	}
}
