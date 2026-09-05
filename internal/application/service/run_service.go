package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	domainservice "github.com/morphy76/vuhive-cloud/internal/domain/service"
)

// RunService implements inbound.RunsUseCase to orchestrate test run creation, execution dispatch, and lifecycle control.
type RunService struct {
	suiteRepo    outbound.TestSuiteRepository
	artifactRepo outbound.ArtifactRepository
	configRepo   outbound.ConfigurationRepository
	profileRepo  outbound.RunnerProfileRepository
	runRepo      outbound.TestRunRepository
	orchestrator outbound.RunnerOrchestratorPort
	storage      outbound.StoragePort
}

// NewRunService constructs a new RunService.
func NewRunService(
	suiteRepo outbound.TestSuiteRepository,
	artifactRepo outbound.ArtifactRepository,
	configRepo outbound.ConfigurationRepository,
	profileRepo outbound.RunnerProfileRepository,
	runRepo outbound.TestRunRepository,
	orchestrator outbound.RunnerOrchestratorPort,
	storage outbound.StoragePort,
) *RunService {
	return &RunService{
		suiteRepo:    suiteRepo,
		artifactRepo: artifactRepo,
		configRepo:   configRepo,
		profileRepo:  profileRepo,
		runRepo:      runRepo,
		orchestrator: orchestrator,
		storage:      storage,
	}
}

// TriggerRun validates inputs, creates a new TestRun aggregate, persists it in QUEUED status, and manifests a K8s Job.
func (s *RunService) TriggerRun(ctx context.Context, cmd inbound.TriggerRunCommand) (*model.TestRun, error) {
	start := time.Now()
	suiteID := strings.TrimSpace(cmd.SuiteID)
	artifactID := strings.TrimSpace(cmd.ArtifactID)
	profileID := strings.TrimSpace(cmd.RunnerProfileID)

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunService.TriggerRun").
		Str("suite_id", suiteID).
		Str("artifact_id", artifactID).
		Str("profile_id", profileID).
		Logger()
	log.Debug().Msg("starting test run trigger")

	if suiteID == "" || artifactID == "" || profileID == "" {
		err := fmt.Errorf("%w: suite_id, artifact_id, and runner_profile_id are required", model.ErrValidation)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("trigger run validation failed")
		return nil, err
	}

	// 1. Verify TestSuite exists and is ACTIVE
	suite, err := s.suiteRepo.FindByID(ctx, suiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching test suite")
		return nil, err
	}
	if suite.State() != model.TestSuiteStateActive {
		err := fmt.Errorf("%w: test suite %s is not active (status: %s)", model.ErrInvalidStateTransition, suiteID, suite.State())
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("test suite is not in active state")
		return nil, err
	}

	// 2. Verify Artifact exists, belongs to suite, and is READY
	artifact, err := s.artifactRepo.FindByID(ctx, artifactID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching artifact")
		return nil, err
	}
	if artifact.SuiteID() != suite.ID() {
		err := fmt.Errorf("%w: artifact %s does not belong to suite %s", model.ErrValidation, artifactID, suiteID)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("artifact does not belong to suite")
		return nil, err
	}
	if artifact.Status() != model.ArtifactStatusReady {
		err := fmt.Errorf("%w: artifact %s is not ready (status: %s)", model.ErrInvalidStateTransition, artifactID, artifact.Status())
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("artifact is not ready for execution")
		return nil, err
	}

	// 3. Verify Configuration if provided
	var configKey string
	if cmd.ConfigurationID != nil && strings.TrimSpace(*cmd.ConfigurationID) != "" {
		cfgID := strings.TrimSpace(*cmd.ConfigurationID)
		cfg, err := s.configRepo.FindByID(ctx, cfgID)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching configuration")
			return nil, err
		}
		if cfg.SuiteID() != suite.ID() {
			err := fmt.Errorf("%w: configuration %s does not belong to suite %s", model.ErrValidation, cfgID, suiteID)
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("configuration does not belong to suite")
			return nil, err
		}
		configKey = cfg.S3ConfigKey()
	}

	// 4. Verify RunnerProfile exists
	profile, err := s.profileRepo.FindByID(ctx, profileID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching runner profile")
		return nil, err
	}

	// 5. Create TestRun aggregate in QUEUED status
	run, err := model.NewTestRun(suite.ID(), artifact.ID(), cmd.ConfigurationID, profile.ID(), nil)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating test run aggregate")
		return nil, err
	}

	// 6. Save initial QUEUED state
	if err := s.runRepo.Save(ctx, run); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed persisting queued test run")
		return nil, err
	}

	// 7. Dispatch Kubernetes Job
	jobOpts := outbound.RunnerJobOptions{
		S3BinaryKey: artifact.S3BinaryKey(),
		S3ConfigKey: configKey,
	}

	jobName, err := s.orchestrator.DispatchJob(ctx, run, profile, jobOpts)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed dispatching runner job to kubernetes")
		// Fail the run immediately if dispatch fails
		_ = run.Fail(1, "", time.Now().UTC())
		_ = s.runRepo.Save(ctx, run)
		return nil, err
	}

	// 8. Associate Job name with the run and persist
	run.SetK8sJobName(jobName)
	if err := s.runRepo.Save(ctx, run); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed updating test run with job name")
		return nil, err
	}

	log.Info().
		Str("run_id", run.ID()).
		Str("job_name", jobName).
		Dur("duration_ms", time.Since(start)).
		Msg("completed test run trigger")

	return run, nil
}

// GetRun retrieves a TestRun by its unique identifier.
func (s *RunService) GetRun(ctx context.Context, id string) (*model.TestRun, error) {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: run id cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunService.GetRun").
		Str("run_id", trimmedID).
		Logger()
	log.Debug().Msg("fetching test run")

	run, err := s.runRepo.FindByID(ctx, trimmedID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching test run by id")
		return nil, err
	}

	log.Info().
		Str("status", string(run.Status())).
		Dur("duration_ms", time.Since(start)).
		Msg("completed test run retrieval")

	return run, nil
}

// ListRuns returns all runs for a given suite, optionally filtered by status.
func (s *RunService) ListRuns(ctx context.Context, suiteID string, status model.RunStatus) ([]*model.TestRun, error) {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunService.ListRuns").
		Str("suite_id", trimmedSuiteID).
		Str("status", string(status)).
		Logger()
	log.Debug().Msg("listing test runs")

	runs, err := s.runRepo.List(ctx, trimmedSuiteID, status)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing test runs")
		return nil, err
	}

	log.Info().
		Int("count", len(runs)).
		Dur("duration_ms", time.Since(start)).
		Msg("completed test runs listing")

	return runs, nil
}

// AbortRun cancels an active or queued run, terminates its K8s Job, and marks it as ABORTED.
func (s *RunService) AbortRun(ctx context.Context, id string, reason string) error {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return fmt.Errorf("%w: run id cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunService.AbortRun").
		Str("run_id", trimmedID).
		Str("reason", reason).
		Logger()
	log.Debug().Msg("starting test run abort")

	run, err := s.runRepo.FindByID(ctx, trimmedID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding test run to abort")
		return err
	}

	if run.Status().IsTerminal() {
		err := fmt.Errorf("%w: run %s is already in terminal status %s", model.ErrTerminalState, trimmedID, run.Status())
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("cannot abort run in terminal state")
		return err
	}

	// Terminate the Kubernetes Job if it was dispatched
	if run.K8sJobName() != "" {
		if err := s.orchestrator.AbortJob(ctx, run.K8sJobName(), run.K8sNamespace()); err != nil {
			// If the job was already deleted or not found, proceed with aborting the domain entity
			log.Warn().Err(err).Msg("aborting job in kubernetes reported error; continuing")
		}
	}

	now := time.Now().UTC()
	if err := run.Abort(reason, now); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed transitioning run state to ABORTED")
		return err
	}

	if err := s.runRepo.Save(ctx, run); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed persisting aborted test run")
		return err
	}

	log.Info().
		Dur("duration_ms", time.Since(start)).
		Msg("completed test run abort")

	return nil
}

// CompleteRun processes a test run completion callback, ingesting summary.json,
// extracting performance KPIs into PostgreSQL, and updating execution lifecycle state.
func (s *RunService) CompleteRun(ctx context.Context, cmd inbound.CompleteRunCommand) (*model.TestRun, error) {
	start := time.Now()
	runID := strings.TrimSpace(cmd.RunID)
	if runID == "" {
		return nil, fmt.Errorf("%w: run id cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunService.CompleteRun").
		Str("run_id", runID).
		Logger()
	log.Debug().Msg("starting test run completion processing")

	run, err := s.runRepo.FindByID(ctx, runID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding test run for completion")
		return nil, err
	}

	if run.Status().IsTerminal() {
		err := fmt.Errorf("%w: run %s is already in terminal status %s", model.ErrTerminalState, runID, run.Status())
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("cannot complete run in terminal state")
		return nil, err
	}

	finishTime := time.Now().UTC()
	if cmd.FinishedAt != nil && !cmd.FinishedAt.IsZero() {
		finishTime = cmd.FinishedAt.UTC()
	}

	// Auto-transition from QUEUED to RUNNING if not already transitioned
	if run.Status() == model.RunStatusQueued {
		startTime := finishTime
		_ = run.Start(run.K8sJobName(), startTime)
	}

	reportKey := strings.TrimSpace(cmd.ReportKey)
	if reportKey == "" {
		reportKey = run.S3ReportKey()
	}
	if reportKey == "" {
		reportKey = fmt.Sprintf("runs/%s/summary.json", run.ID())
	}

	logsKey := strings.TrimSpace(cmd.LogsKey)
	if logsKey == "" {
		logsKey = run.S3LogsKey()
	}
	if logsKey == "" {
		logsKey = fmt.Sprintf("runs/%s/run.log", run.ID())
	}

	// Determine exit code
	exitCode := 0
	if cmd.ExitCode != nil {
		exitCode = *cmd.ExitCode
	}

	// Resolve summary report data
	summaryData := cmd.SummaryJSON
	if len(bytes.TrimSpace(summaryData)) == 0 && s.storage != nil && reportKey != "" {
		if reader, err := s.storage.Download(ctx, reportKey); err == nil && reader != nil {
			defer func() { _ = reader.Close() }()
			if data, err := io.ReadAll(reader); err == nil {
				summaryData = data
			}
		}
	}

	// If no summary data available, or runner exited with fatal failure without report
	if len(bytes.TrimSpace(summaryData)) == 0 {
		log.Warn().
			Int("exit_code", exitCode).
			Str("report_key", reportKey).
			Msg("missing summary report; marking test run as FAILED")

		if err := run.Fail(exitCode, logsKey, finishTime); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed transitioning run to FAILED")
			return nil, err
		}
		if err := s.runRepo.Save(ctx, run); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving failed test run")
			return nil, err
		}
		return run, nil
	}

	// Parse summary report
	parsed, err := domainservice.ParseSummaryReport(summaryData)
	if err != nil {
		log.Warn().
			Err(err).
			Int("exit_code", exitCode).
			Msg("failed parsing summary report JSON; marking test run as FAILED")

		if err := run.Fail(exitCode, logsKey, finishTime); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed transitioning run to FAILED")
			return nil, err
		}
		if err := s.runRepo.Save(ctx, run); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving failed test run")
			return nil, err
		}
		return run, nil
	}

	// Complete run aggregate
	if err := run.Complete(parsed.Metrics, reportKey, logsKey, parsed.RawJSON, parsed.SLAPassed, finishTime); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed completing test run")
		return nil, err
	}

	if cmd.ExitCode != nil {
		run.SetExitCode(*cmd.ExitCode)
	}

	if err := s.runRepo.Save(ctx, run); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving completed test run")
		return nil, err
	}

	log.Info().
		Str("run_id", run.ID()).
		Int("iterations", int(parsed.Metrics.TotalIterations)).
		Float64("avg_tps", parsed.Metrics.AvgTPS).
		Bool("sla_passed", parsed.SLAPassed).
		Dur("duration_ms", time.Since(start)).
		Msg("completed test run report ingestion and metric indexing")

	return run, nil
}

// Compile-time static interface verification
var _ inbound.RunsUseCase = (*RunService)(nil)
