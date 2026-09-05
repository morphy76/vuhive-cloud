package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// BarrierClient defines the client interface for participating in distributed start barrier rendezvous.
type BarrierClient interface {
	PreflightCheck(cfg WrapperConfig) error
	Rendezvous(ctx context.Context, cfg WrapperConfig) error
	SignalAbort(ctx context.Context, cfg WrapperConfig, reason string) error
}

// HTTPBarrierClient coordinates with the control plane barrier endpoints via HTTP.
type HTTPBarrierClient struct {
	httpClient HTTPClient
}

// NewHTTPBarrierClient creates a new HTTPBarrierClient with the provided HTTP client.
func NewHTTPBarrierClient(httpClient HTTPClient) *HTTPBarrierClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 90 * time.Second}
	}
	return &HTTPBarrierClient{
		httpClient: httpClient,
	}
}

// PreflightCheck verifies that the runner binary exists and is executable.
func (c *HTTPBarrierClient) PreflightCheck(cfg WrapperConfig) error {
	info, err := os.Stat(cfg.RunnerPath)
	if err != nil {
		return fmt.Errorf("runner binary not found at %q: %w", cfg.RunnerPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("runner path %q is a directory, expected executable file", cfg.RunnerPath)
	}
	// Check execute permission
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("runner binary at %q is not executable (mode: %v)", cfg.RunnerPath, info.Mode())
	}
	return nil
}

// SignalAbort notifies the coordinator that this worker failed initialization and the run must abort.
func (c *HTTPBarrierClient) SignalAbort(ctx context.Context, cfg WrapperConfig, reason string) error {
	log := zerolog.Ctx(ctx).With().
		Str("op", "HTTPBarrierClient.SignalAbort").
		Str("worker_id", cfg.WorkerID).
		Str("reason", reason).
		Logger()

	if cfg.CoordinatorURL == "" {
		log.Warn().Msg("coordinator URL empty; cannot signal abort")
		return nil
	}

	abortURL := strings.TrimRight(cfg.CoordinatorURL, "/") + "/abort"
	payload := map[string]string{
		"worker_id": cfg.WorkerID,
		"reason":    reason,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, abortURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Warn().Err(err).Msg("failed executing abort request to coordinator")
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	log.Info().Int("status_code", resp.StatusCode).Msg("successfully sent abort signal to barrier coordinator")
	return nil
}

// Rendezvous runs the pre-flight check, registers at the barrier, awaits peer readiness,
// and synchronizes startup by sleeping until the coordinated target start time.
func (c *HTTPBarrierClient) Rendezvous(ctx context.Context, cfg WrapperConfig) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "HTTPBarrierClient.Rendezvous").
		Str("worker_id", cfg.WorkerID).
		Int("worker_count", cfg.WorkerCount).
		Logger()
	log.Debug().Msg("starting barrier rendezvous")

	// 1. Preflight check: Ensure runner binary is present and executable
	if err := c.PreflightCheck(cfg); err != nil {
		errMsg := fmt.Sprintf("preflight check failed: %v", err)
		log.Error().Err(err).Msg("preflight check failed; aborting rendezvous")
		_ = c.SignalAbort(ctx, cfg, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// If only 1 worker and barrier not explicitly enabled with peers, skip await
	if cfg.WorkerCount <= 1 && !cfg.BarrierEnabled {
		log.Info().Msg("single worker execution; barrier skipped")
		return nil
	}

	if cfg.CoordinatorURL == "" {
		log.Warn().Msg("coordinator URL not configured; skipping distributed barrier await")
		return nil
	}

	// 2. Send Await request to coordinator
	awaitURL := strings.TrimRight(cfg.CoordinatorURL, "/") + "/await"
	timeoutMs := int(cfg.BarrierTimeout.Milliseconds())
	releaseDelayMs := int(cfg.ReleaseDelay.Milliseconds())

	payload := map[string]interface{}{
		"worker_id":        cfg.WorkerID,
		"total_workers":    cfg.WorkerCount,
		"timeout_ms":       timeoutMs,
		"release_delay_ms": releaseDelayMs,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed marshaling barrier await request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, awaitURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed creating barrier await request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed connecting to barrier coordinator")
		return fmt.Errorf("failed connecting to barrier coordinator: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		if errResp.Error == "" {
			errResp.Error = fmt.Sprintf("status code %d", resp.StatusCode)
		}
		log.Error().Int("status_code", resp.StatusCode).Str("error", errResp.Error).Msg("barrier coordinator rejected rendezvous")
		return fmt.Errorf("barrier rendezvous rejected (%d): %s", resp.StatusCode, errResp.Error)
	}

	var barrierResp struct {
		Status          string  `json:"status"`
		TargetStartTime *string `json:"target_start_time"`
		StartInMs       int64   `json:"start_in_ms"`
	}

	if err := json.Unmarshal(respBody, &barrierResp); err != nil {
		return fmt.Errorf("failed parsing barrier response JSON: %w", err)
	}

	// 3. Sub-millisecond synchronized wait
	if barrierResp.TargetStartTime != nil && strings.TrimSpace(*barrierResp.TargetStartTime) != "" {
		targetTime, err := time.Parse(time.RFC3339Nano, *barrierResp.TargetStartTime)
		if err == nil {
			waitDuration := time.Until(targetTime)
			if waitDuration > 0 {
				log.Info().
					Time("target_start_time", targetTime).
					Dur("wait_duration", waitDuration).
					Msg("barrier released; sleeping to synchronize start boundary across all pods")
				time.Sleep(waitDuration)
			}
		}
	}

	log.Info().
		Dur("duration_ms", time.Since(start)).
		Msg("completed barrier rendezvous; initiating traffic generation")

	return nil
}

// Compile-time static interface verification
var _ BarrierClient = (*HTTPBarrierClient)(nil)
