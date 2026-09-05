package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/rs/zerolog"
)

// HTTPClient defines the interface for executing HTTP requests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// RunnerWrapper manages test binary execution, output capture, signal trapping, and artifact upload to S3.
type RunnerWrapper struct {
	storage       outbound.StoragePort
	httpClient    HTTPClient
	barrierClient BarrierClient
	stdout        io.Writer
	stderr        io.Writer
}

// NewRunnerWrapper creates a new RunnerWrapper using the provided storage adapter and default standard output/error.
func NewRunnerWrapper(storage outbound.StoragePort) *RunnerWrapper {
	client := &http.Client{Timeout: 90 * time.Second}
	return &RunnerWrapper{
		storage:       storage,
		httpClient:    client,
		barrierClient: NewHTTPBarrierClient(client),
		stdout:        os.Stdout,
		stderr:        os.Stderr,
	}
}

// SetHTTPClient allows overriding the HTTP client (useful for unit testing without sockets).
func (w *RunnerWrapper) SetHTTPClient(client HTTPClient) {
	if client != nil {
		w.httpClient = client
		if w.barrierClient == nil {
			w.barrierClient = NewHTTPBarrierClient(client)
		}
	}
}

// SetBarrierClient allows overriding the barrier client (useful for unit testing).
func (w *RunnerWrapper) SetBarrierClient(client BarrierClient) {
	if client != nil {
		w.barrierClient = client
	}
}

// SetOutputs allows overriding stdout and stderr writers (useful for testing).
func (w *RunnerWrapper) SetOutputs(stdout, stderr io.Writer) {
	w.stdout = stdout
	w.stderr = stderr
}

// Run executes the runner binary, streams & logs output, traps signals, uploads artifacts to S3, and returns the exit code.
func (w *RunnerWrapper) Run(ctx context.Context, cfg WrapperConfig, extraArgs []string) (int, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "RunnerWrapper.Run").
		Str("runner_path", cfg.RunnerPath).
		Str("run_id", cfg.RunID).
		Str("report_key", cfg.ReportKey).
		Str("logs_key", cfg.LogsKey).
		Logger()
	log.Debug().Msg("starting runner wrapper execution")

	if err := cfg.Validate(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid wrapper configuration")
		return 1, err
	}

	// Distributed start barrier coordination (if multi-worker or explicitly enabled)
	if cfg.BarrierEnabled || cfg.WorkerCount > 1 {
		bClient := w.barrierClient
		if bClient == nil {
			bClient = NewHTTPBarrierClient(w.httpClient)
		}
		if err := bClient.Rendezvous(ctx, cfg); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("barrier rendezvous failed; aborting execution")
			w.guaranteeReportAndLogsUpload(ctx, cfg, 1, fmt.Sprintf("barrier rendezvous failed: %v", err))
			if cfg.APICallbackURL != "" {
				w.sendCallback(ctx, cfg, 1)
			}
			return 1, err
		}
	}

	// Ensure directory for logs and summary exists
	if err := os.MkdirAll(filepath.Dir(cfg.LogPath), 0755); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to create log directory")
		return 1, fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile, err := os.OpenFile(cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to open log file")
		return 1, fmt.Errorf("failed to open log file %q: %w", cfg.LogPath, err)
	}

	// Prepare arguments
	args := []string{fmt.Sprintf("--summary-export=%s", cfg.SummaryPath)}
	if _, err := os.Stat(cfg.ConfigPath); err == nil {
		args = append(args, fmt.Sprintf("--config=%s", cfg.ConfigPath))
	}
	args = append(args, extraArgs...)

	stdoutWriter := io.MultiWriter(w.stdout, logFile)
	stderrWriter := io.MultiWriter(w.stderr, logFile)

	cmd := exec.Command(cfg.RunnerPath, args...)
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	// Setup signal trapping
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to start runner process")
		// Guarantee upload even if process failed to start
		w.guaranteeReportAndLogsUpload(ctx, cfg, 1, fmt.Sprintf("failed to start runner: %v", err))
		return 1, fmt.Errorf("failed to start runner: %w", err)
	}

	// Background signal forwarding
	doneCh := make(chan struct{})
	go func() {
		select {
		case sig := <-sigCh:
			log.Warn().Str("signal", sig.String()).Msg("received termination signal; forwarding to runner")
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		case <-ctx.Done():
			log.Warn().Msg("context cancelled; terminating runner")
			if cmd.Process != nil {
				_ = cmd.Process.Signal(syscall.SIGTERM)
			}
		case <-doneCh:
			return
		}
	}()

	waitErr := cmd.Wait()
	close(doneCh)
	_ = logFile.Close()

	exitCode := 0
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
		log.Warn().Int("exit_code", exitCode).Err(waitErr).Msg("runner process exited with non-zero status")
	} else {
		log.Info().Int("exit_code", exitCode).Msg("runner process exited successfully")
	}

	// Upload report & logs to S3 (guaranteed)
	w.guaranteeReportAndLogsUpload(ctx, cfg, exitCode, "")

	// Send callback to API if configured
	if cfg.APICallbackURL != "" {
		w.sendCallback(ctx, cfg, exitCode)
	}

	log.Info().Int("exit_code", exitCode).Dur("duration_ms", time.Since(start)).Msg("completed runner wrapper execution")
	return exitCode, nil
}

func (w *RunnerWrapper) guaranteeReportAndLogsUpload(ctx context.Context, cfg WrapperConfig, exitCode int, errDetails string) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "RunnerWrapper.guaranteeReportAndLogsUpload").
		Str("report_key", cfg.ReportKey).
		Str("logs_key", cfg.LogsKey).
		Int("exit_code", exitCode).
		Logger()

	// 1. Upload summary report
	var reportData []byte
	if fileData, err := os.ReadFile(cfg.SummaryPath); err == nil && len(bytes.TrimSpace(fileData)) > 0 {
		reportData = fileData
	} else {
		// Fallback report if summary.json wasn't created or is empty
		if errDetails == "" {
			errDetails = fmt.Sprintf("runner exited with code %d without generating summary report", exitCode)
		}
		fallback := map[string]interface{}{
			"status":     "FAILED",
			"exit_code":  exitCode,
			"error":      errDetails,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
			"sla_passed": false,
		}
		reportData, _ = json.MarshalIndent(fallback, "", "  ")
		_ = os.WriteFile(cfg.SummaryPath, reportData, 0644)
	}

	if err := w.storage.Upload(ctx, cfg.ReportKey, bytes.NewReader(reportData), int64(len(reportData)), "application/json"); err != nil {
		log.Error().Err(err).Msg("failed uploading summary report to storage")
	} else {
		log.Info().Msg("summary report successfully uploaded to storage")
	}

	// 2. Upload run logs
	logData, err := os.ReadFile(cfg.LogPath)
	if err != nil {
		logData = []byte(fmt.Sprintf("log file read error: %v\n", err))
	}
	if err := w.storage.Upload(ctx, cfg.LogsKey, bytes.NewReader(logData), int64(len(logData)), "text/plain"); err != nil {
		log.Error().Err(err).Msg("failed uploading execution logs to storage")
	} else {
		log.Info().Msg("execution logs successfully uploaded to storage")
	}

	log.Debug().Dur("duration_ms", time.Since(start)).Msg("completed artifacts upload to storage")
}

func (w *RunnerWrapper) sendCallback(ctx context.Context, cfg WrapperConfig, exitCode int) {
	log := zerolog.Ctx(ctx).With().
		Str("op", "RunnerWrapper.sendCallback").
		Str("url", cfg.APICallbackURL).
		Logger()

	payload := map[string]interface{}{
		"run_id":      cfg.RunID,
		"exit_code":   exitCode,
		"report_key":  cfg.ReportKey,
		"logs_key":    cfg.LogsKey,
		"finished_at": time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal callback payload")
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.APICallbackURL, bytes.NewReader(body))
	if err != nil {
		log.Error().Err(err).Msg("failed to create callback request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("failed to execute callback request")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warn().Int("status_code", resp.StatusCode).Msg("callback responded with non-2xx status")
	} else {
		log.Info().Int("status_code", resp.StatusCode).Msg("callback sent successfully")
	}
}

// Compile-time interface assertion
var _ Wrapper = (*RunnerWrapper)(nil)
