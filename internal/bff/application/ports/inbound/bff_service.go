package inbound

import (
	"context"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
)

// SystemStatus represents aggregated health and runtime telemetry of the BFF and its upstream services.
type SystemStatus struct {
	BFFStatus           string                 `json:"bff_status"`
	BFFVersion          string                 `json:"bff_version"`
	ControlPlaneStatus  string                 `json:"control_plane_status"`
	ControlPlaneVersion string                 `json:"control_plane_version,omitempty"`
	Timestamp           time.Time              `json:"timestamp"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// CreateSessionCommand encapsulates payload parameters for initiating a client session.
type CreateSessionCommand struct {
	SessionID string
	UserID    string
	TTL       time.Duration
	Metadata  map[string]string
}

// BFFService defines the driving inbound port for BFF use cases.
type BFFService interface {
	GetStatus(ctx context.Context) (*SystemStatus, error)
	CreateSession(ctx context.Context, cmd CreateSessionCommand) (*model.ClientSession, error)
	GetSession(ctx context.Context, id model.SessionID) (*model.ClientSession, error)
}
