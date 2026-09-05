package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/rs/zerolog"
)

var _ outbound.ControlPlaneClient = (*Client)(nil)

// Config configures the outbound HTTP client for the control plane.
type Config struct {
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// Client implements outbound.ControlPlaneClient via HTTP calls to cmd/server.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient constructs an initialized Client.
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: timeout,
		}
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// CheckHealth queries the control plane /healthz endpoint.
func (c *Client) CheckHealth(ctx context.Context) (*outbound.ControlPlaneHealth, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "ControlPlaneClient.CheckHealth").
		Str("base_url", c.baseURL).
		Logger()
	log.Debug().Msg("checking upstream control plane health")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating health request")
		return nil, model.NewDomainError(model.ErrControlPlaneUnavailable, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed executing health request")
		return nil, model.NewDomainError(model.ErrControlPlaneUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusErr := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		log.Error().Err(statusErr).Dur("duration_ms", time.Since(start)).Msg("control plane returned non-2xx status")
		return nil, model.NewDomainError(model.ErrControlPlaneUnavailable, statusErr)
	}

	health := &outbound.ControlPlaneHealth{
		Status:    "UP",
		Timestamp: time.Now().UTC(),
	}

	log.Info().
		Str("status", health.Status).
		Dur("duration_ms", time.Since(start)).
		Msg("completed control plane health check")

	return health, nil
}

// GetVersion queries the control plane /version endpoint.
func (c *Client) GetVersion(ctx context.Context) (*outbound.ControlPlaneVersion, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "ControlPlaneClient.GetVersion").
		Str("base_url", c.baseURL).
		Logger()
	log.Debug().Msg("querying upstream control plane version")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/version", nil)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating version request")
		return nil, model.NewDomainError(model.ErrControlPlaneUnavailable, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed executing version request")
		return nil, model.NewDomainError(model.ErrControlPlaneUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		log.Warn().Int("status_code", resp.StatusCode).Dur("duration_ms", time.Since(start)).Msg("version endpoint returned non-200, returning fallback")
		return &outbound.ControlPlaneVersion{
			Version:   "unknown",
			Commit:    "unknown",
			BuildTime: "unknown",
		}, nil
	}

	var versionResp struct {
		Version   string `json:"version"`
		Commit    string `json:"commit"`
		BuildTime string `json:"build_time"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&versionResp); err != nil {
		log.Warn().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed parsing version response, returning fallback")
		return &outbound.ControlPlaneVersion{
			Version:   "unknown",
			Commit:    "unknown",
			BuildTime: "unknown",
		}, nil
	}

	result := &outbound.ControlPlaneVersion{
		Version:   versionResp.Version,
		Commit:    versionResp.Commit,
		BuildTime: versionResp.BuildTime,
	}

	log.Info().
		Str("version", result.Version).
		Dur("duration_ms", time.Since(start)).
		Msg("completed control plane version query")

	return result, nil
}
