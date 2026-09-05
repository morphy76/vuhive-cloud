package outbound

import (
	"context"
	"time"
)

// ControlPlaneHealth models health information reported by upstream cmd/server.
type ControlPlaneHealth struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// ControlPlaneVersion models version details returned by upstream cmd/server.
type ControlPlaneVersion struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

// ControlPlaneClient defines the driven outbound port for communicating with the control plane server.
type ControlPlaneClient interface {
	CheckHealth(ctx context.Context) (*ControlPlaneHealth, error)
	GetVersion(ctx context.Context) (*ControlPlaneVersion, error)
}
