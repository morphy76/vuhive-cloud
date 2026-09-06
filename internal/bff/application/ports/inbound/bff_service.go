package inbound

import (
	"context"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
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

// DashboardOverview represents the composite telemetry and overview for the dashboard view.
type DashboardOverview struct {
	BFFStatus           string                    `json:"bff_status"`
	BFFVersion          string                    `json:"bff_version"`
	ControlPlaneStatus  string                    `json:"control_plane_status"`
	ControlPlaneVersion string                    `json:"control_plane_version,omitempty"`
	ActiveRunsCount     int64                     `json:"active_runs_count"`
	RecentSuites        []outbound.SuiteSummary   `json:"recent_suites"`
	ProfilesCount       int                       `json:"profiles_count"`
	ProfilesSummary     []outbound.ProfileSummary `json:"profiles_summary"`
	Timestamp           time.Time                 `json:"timestamp"`
}

// ArtifactLinks contains direct download links to run reports and logs.
type ArtifactLinks struct {
	ReportURL string `json:"report_url,omitempty"`
	LogsURL   string `json:"logs_url,omitempty"`
}

// RunDetailComposite combines run entity metadata, KPIs, and direct links to S3 artifacts.
type RunDetailComposite struct {
	outbound.RunDetail
	ArtifactLinks ArtifactLinks `json:"artifact_links"`
}

// BFFService defines the driving inbound port for BFF use cases.
type BFFService interface {
	GetStatus(ctx context.Context) (*SystemStatus, error)
	CreateSession(ctx context.Context, cmd CreateSessionCommand) (*model.ClientSession, error)
	GetSession(ctx context.Context, id model.SessionID) (*model.ClientSession, error)
	GetDashboard(ctx context.Context) (*DashboardOverview, error)
	GetRunDetail(ctx context.Context, id string) (*RunDetailComposite, error)
	SubscribeEvents(ctx context.Context) (<-chan model.ServerSentEvent, func(), error)
	BroadcastEvent(ctx context.Context, event model.ServerSentEvent) error
}
