package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
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
	AuthToken  string
	MaxRetries int
	HTTPClient *http.Client
}

// Client implements outbound.ControlPlaneClient via HTTP calls to cmd/server.
type Client struct {
	baseURL    string
	authToken  string
	maxRetries int
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
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   50,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		httpClient = &http.Client{
			Timeout:   timeout,
			Transport: transport,
		}
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &Client{
		baseURL:    baseURL,
		authToken:  cfg.AuthToken,
		maxRetries: cfg.MaxRetries,
		httpClient: httpClient,
	}
}

// HTTPClient returns the underlying *http.Client.
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// executeRequest executes an HTTP request with Bearer token propagation and retries for idempotent calls.
func (c *Client) executeRequest(ctx context.Context, method, targetURL string, body io.Reader) (*http.Response, error) {
	attempts := 1
	if c.maxRetries > 0 && (method == http.MethodGet || method == http.MethodHead) {
		attempts += c.maxRetries
	}

	var lastErr error
	var resp *http.Response

	for i := 0; i < attempts; i++ {
		if i > 0 {
			backoff := time.Duration(25*(1<<i)) * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
		if err != nil {
			return nil, err
		}

		if c.authToken != "" && req.Header.Get("Authorization") == "" {
			req.Header.Set("Authorization", "Bearer "+c.authToken)
		}

		resp, err = c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 500 && (method == http.MethodGet || method == http.MethodHead) && i < attempts-1 {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("upstream server returned status %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return resp, nil
}

// CheckHealth queries the control plane /healthz endpoint.
func (c *Client) CheckHealth(ctx context.Context) (*outbound.ControlPlaneHealth, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "ControlPlaneClient.CheckHealth").
		Str("base_url", c.baseURL).
		Logger()
	log.Debug().Msg("checking upstream control plane health")

	resp, err := c.executeRequest(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
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

	resp, err := c.executeRequest(ctx, http.MethodGet, c.baseURL+"/version", nil)
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

// GetActiveRunsCount queries RUNNING and QUEUED runs to return the total count of active test runs.
func (c *Client) GetActiveRunsCount(ctx context.Context) (int64, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "ControlPlaneClient.GetActiveRunsCount").
		Logger()
	log.Debug().Msg("querying active test runs count")

	statuses := []string{"RUNNING", "QUEUED"}
	var totalActive int64

	for _, st := range statuses {
		u := fmt.Sprintf("%s/api/v1/runs?status=%s&limit=1", c.baseURL, url.QueryEscape(st))
		resp, err := c.executeRequest(ctx, http.MethodGet, u, nil)
		if err != nil {
			log.Error().Err(err).Str("status", st).Dur("duration_ms", time.Since(start)).Msg("failed querying runs by status")
			return 0, model.NewDomainError(model.ErrControlPlaneUnavailable, err)
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			statusErr := fmt.Errorf("unexpected status %d while querying %s runs", resp.StatusCode, st)
			log.Error().Err(statusErr).Dur("duration_ms", time.Since(start)).Msg("control plane returned non-200")
			return 0, model.NewDomainError(model.ErrControlPlaneUnavailable, statusErr)
		}

		var runListResp struct {
			Total int64 `json:"total"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&runListResp); err != nil {
			_ = resp.Body.Close()
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed decoding runs list response")
			return 0, model.NewDomainError(model.ErrInternal, err)
		}
		_ = resp.Body.Close()

		totalActive += runListResp.Total
	}

	log.Info().
		Int64("active_runs", totalActive).
		Dur("duration_ms", time.Since(start)).
		Msg("completed active test runs count query")

	return totalActive, nil
}

// ListRecentSuites queries test suites from the control plane. If 404 is returned, an empty slice is returned gracefully.
func (c *Client) ListRecentSuites(ctx context.Context, limit int) ([]outbound.SuiteSummary, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "ControlPlaneClient.ListRecentSuites").
		Int("limit", limit).
		Logger()
	log.Debug().Msg("listing recent test suites")

	targetURL := c.baseURL + "/api/v1/suites"
	if limit > 0 {
		targetURL += "?limit=" + strconv.Itoa(limit)
	}

	resp, err := c.executeRequest(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed querying test suites")
		return nil, model.NewDomainError(model.ErrControlPlaneUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		log.Debug().Msg("control plane suites endpoint returned 404, returning empty slice")
		return []outbound.SuiteSummary{}, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusErr := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		log.Error().Err(statusErr).Dur("duration_ms", time.Since(start)).Msg("failed listing test suites")
		return nil, model.NewDomainError(model.ErrControlPlaneUnavailable, statusErr)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed reading response body")
		return nil, model.NewDomainError(model.ErrInternal, err)
	}

	var suitesWrapper struct {
		Suites []outbound.SuiteSummary `json:"suites"`
	}
	if err := json.Unmarshal(bodyBytes, &suitesWrapper); err == nil && suitesWrapper.Suites != nil {
		log.Info().Int("count", len(suitesWrapper.Suites)).Dur("duration_ms", time.Since(start)).Msg("completed listing test suites")
		return suitesWrapper.Suites, nil
	}

	var plainSuites []outbound.SuiteSummary
	if err := json.Unmarshal(bodyBytes, &plainSuites); err == nil {
		log.Info().Int("count", len(plainSuites)).Dur("duration_ms", time.Since(start)).Msg("completed listing test suites")
		return plainSuites, nil
	}

	log.Info().Int("count", 0).Dur("duration_ms", time.Since(start)).Msg("completed listing test suites with empty fallback")
	return []outbound.SuiteSummary{}, nil
}

// ListProfiles queries registered runner profiles from the control plane.
func (c *Client) ListProfiles(ctx context.Context) ([]outbound.ProfileSummary, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "ControlPlaneClient.ListProfiles").
		Logger()
	log.Debug().Msg("listing runner profiles")

	resp, err := c.executeRequest(ctx, http.MethodGet, c.baseURL+"/api/v1/profiles", nil)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed querying runner profiles")
		return nil, model.NewDomainError(model.ErrControlPlaneUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusErr := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		log.Error().Err(statusErr).Dur("duration_ms", time.Since(start)).Msg("failed listing runner profiles")
		return nil, model.NewDomainError(model.ErrControlPlaneUnavailable, statusErr)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed reading profile response body")
		return nil, model.NewDomainError(model.ErrInternal, err)
	}

	var profilesWrapper struct {
		Profiles []outbound.ProfileSummary `json:"profiles"`
	}
	if err := json.Unmarshal(bodyBytes, &profilesWrapper); err == nil && profilesWrapper.Profiles != nil {
		log.Info().Int("count", len(profilesWrapper.Profiles)).Dur("duration_ms", time.Since(start)).Msg("completed listing runner profiles")
		return profilesWrapper.Profiles, nil
	}

	var plainProfiles []outbound.ProfileSummary
	if err := json.Unmarshal(bodyBytes, &plainProfiles); err == nil {
		log.Info().Int("count", len(plainProfiles)).Dur("duration_ms", time.Since(start)).Msg("completed listing runner profiles")
		return plainProfiles, nil
	}

	return []outbound.ProfileSummary{}, nil
}

// GetRun queries execution metadata, duration, exit code, and indexed performance KPIs for a specific run.
func (c *Client) GetRun(ctx context.Context, id string) (*outbound.RunDetail, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "ControlPlaneClient.GetRun").
		Str("run_id", id).
		Logger()
	log.Debug().Msg("fetching test run details")

	if strings.TrimSpace(id) == "" {
		return nil, model.NewDomainError(model.ErrInvalidParameter, errors.New("run id cannot be empty"))
	}

	resp, err := c.executeRequest(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/runs/%s", c.baseURL, url.PathEscape(id)), nil)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching test run")
		return nil, model.NewDomainError(model.ErrControlPlaneUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		log.Info().Dur("duration_ms", time.Since(start)).Msg("test run not found in control plane")
		return nil, model.ErrRunNotFound
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusErr := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		log.Error().Err(statusErr).Dur("duration_ms", time.Since(start)).Msg("error fetching test run")
		return nil, model.NewDomainError(model.ErrControlPlaneUnavailable, statusErr)
	}

	var run outbound.RunDetail
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed decoding test run")
		return nil, model.NewDomainError(model.ErrInternal, err)
	}

	log.Info().
		Str("run_id", run.ID).
		Str("status", run.Status).
		Dur("duration_ms", time.Since(start)).
		Msg("completed test run retrieval")

	return &run, nil
}

// GetRunReportURL retrieves a presigned download URL for the summary report.
func (c *Client) GetRunReportURL(ctx context.Context, id string) (string, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "ControlPlaneClient.GetRunReportURL").
		Str("run_id", id).
		Logger()
	log.Debug().Msg("fetching presigned report download URL")

	targetURL := fmt.Sprintf("%s/api/v1/runs/%s/report?presign=true", c.baseURL, url.PathEscape(id))
	resp, err := c.executeRequest(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching report URL")
		return "", model.NewDomainError(model.ErrControlPlaneUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusErr := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return "", model.NewDomainError(model.ErrControlPlaneUnavailable, statusErr)
	}

	var urlResp struct {
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&urlResp); err != nil {
		return "", model.NewDomainError(model.ErrInternal, err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed report presigned URL fetch")
	return urlResp.DownloadURL, nil
}

// GetRunLogsURL retrieves a presigned download URL for the execution logs.
func (c *Client) GetRunLogsURL(ctx context.Context, id string) (string, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "ControlPlaneClient.GetRunLogsURL").
		Str("run_id", id).
		Logger()
	log.Debug().Msg("fetching presigned logs download URL")

	targetURL := fmt.Sprintf("%s/api/v1/runs/%s/logs?presign=true", c.baseURL, url.PathEscape(id))
	resp, err := c.executeRequest(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching logs URL")
		return "", model.NewDomainError(model.ErrControlPlaneUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusErr := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return "", model.NewDomainError(model.ErrControlPlaneUnavailable, statusErr)
	}

	var urlResp struct {
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&urlResp); err != nil {
		return "", model.NewDomainError(model.ErrInternal, err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed logs presigned URL fetch")
	return urlResp.DownloadURL, nil
}

