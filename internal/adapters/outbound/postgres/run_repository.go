package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// TestRunRepository implements outbound.TestRunRepository using pgxpool.
type TestRunRepository struct {
	pool *pgxpool.Pool
}

// NewTestRunRepository constructs a new TestRunRepository.
func NewTestRunRepository(pool *pgxpool.Pool) *TestRunRepository {
	return &TestRunRepository{pool: pool}
}

// Save inserts or updates a TestRun aggregate.
func (r *TestRunRepository) Save(ctx context.Context, run *model.TestRun) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "TestRunRepository.Save").Str("run_id", run.ID()).Logger()
	log.Debug().Msg("saving test run")

	query := `
		INSERT INTO test_runs (
			id, suite_id, artifact_id, configuration_id, runner_profile_id, schedule_id,
			status, k8s_job_name, k8s_namespace, started_at, finished_at, exit_code, sla_passed,
			total_iterations, total_requests, avg_tps,
			p50_duration_ms, p90_duration_ms, p95_duration_ms, p99_duration_ms,
			error_rate_pct, s3_report_key, s3_logs_key, summary_json, abort_reason, created_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16,
			$17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26
		)
		ON CONFLICT (id) DO UPDATE SET
			suite_id = EXCLUDED.suite_id,
			artifact_id = EXCLUDED.artifact_id,
			configuration_id = EXCLUDED.configuration_id,
			runner_profile_id = EXCLUDED.runner_profile_id,
			schedule_id = EXCLUDED.schedule_id,
			status = EXCLUDED.status,
			k8s_job_name = EXCLUDED.k8s_job_name,
			k8s_namespace = EXCLUDED.k8s_namespace,
			started_at = EXCLUDED.started_at,
			finished_at = EXCLUDED.finished_at,
			exit_code = EXCLUDED.exit_code,
			sla_passed = EXCLUDED.sla_passed,
			total_iterations = EXCLUDED.total_iterations,
			total_requests = EXCLUDED.total_requests,
			avg_tps = EXCLUDED.avg_tps,
			p50_duration_ms = EXCLUDED.p50_duration_ms,
			p90_duration_ms = EXCLUDED.p90_duration_ms,
			p95_duration_ms = EXCLUDED.p95_duration_ms,
			p99_duration_ms = EXCLUDED.p99_duration_ms,
			error_rate_pct = EXCLUDED.error_rate_pct,
			s3_report_key = EXCLUDED.s3_report_key,
			s3_logs_key = EXCLUDED.s3_logs_key,
			summary_json = EXCLUDED.summary_json,
			abort_reason = EXCLUDED.abort_reason
	`
	m := run.Metrics()
	_, err := r.pool.Exec(ctx, query,
		run.ID(),
		run.SuiteID(),
		run.ArtifactID(),
		run.ConfigurationID(),
		run.RunnerProfileID(),
		run.ScheduleID(),
		string(run.Status()),
		run.K8sJobName(),
		run.K8sNamespace(),
		run.StartedAt(),
		run.FinishedAt(),
		run.ExitCode(),
		run.SLAPassed(),
		m.TotalIterations,
		m.TotalRequests,
		m.AvgTPS,
		m.P50DurationMs,
		m.P90DurationMs,
		m.P95DurationMs,
		m.P99DurationMs,
		m.ErrorRatePct,
		run.S3ReportKey(),
		run.S3LogsKey(),
		run.SummaryJSON(),
		run.AbortReason(),
		run.CreatedAt(),
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to save test run")
		return MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully saved test run")
	return nil
}

// FindByID retrieves a TestRun aggregate by ID.
func (r *TestRunRepository) FindByID(ctx context.Context, id string) (*model.TestRun, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "TestRunRepository.FindByID").Str("run_id", id).Logger()
	log.Debug().Msg("finding test run by id")

	query := `
		SELECT
			id, suite_id, artifact_id, configuration_id, runner_profile_id, schedule_id,
			status, k8s_job_name, k8s_namespace, started_at, finished_at, exit_code, sla_passed,
			total_iterations, total_requests, avg_tps,
			p50_duration_ms, p90_duration_ms, p95_duration_ms, p99_duration_ms,
			error_rate_pct, s3_report_key, s3_logs_key, summary_json, abort_reason, created_at
		FROM test_runs
		WHERE id = $1
	`
	var (
		runID           string
		suiteID         string
		artifactID      string
		configurationID *string
		runnerProfileID string
		scheduleID      *string
		status          string
		k8sJobName      string
		k8sNamespace    string
		startedAt       *time.Time
		finishedAt      *time.Time
		exitCode        *int
		slaPassed       *bool
		totalIterations int64
		totalRequests   int64
		avgTPS          float64
		p50DurationMs   float64
		p90DurationMs   float64
		p95DurationMs   float64
		p99DurationMs   float64
		errorRatePct    float64
		s3ReportKey     string
		s3LogsKey       string
		summaryJSON     []byte
		abortReason     string
		createdAt       time.Time
	)
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&runID, &suiteID, &artifactID, &configurationID, &runnerProfileID, &scheduleID,
		&status, &k8sJobName, &k8sNamespace, &startedAt, &finishedAt, &exitCode, &slaPassed,
		&totalIterations, &totalRequests, &avgTPS,
		&p50DurationMs, &p90DurationMs, &p95DurationMs, &p99DurationMs,
		&errorRatePct, &s3ReportKey, &s3LogsKey, &summaryJSON, &abortReason, &createdAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to find test run by id")
		return nil, MapError(err)
	}

	metrics := model.RunMetrics{
		TotalIterations: totalIterations,
		TotalRequests:   totalRequests,
		AvgTPS:          avgTPS,
		P50DurationMs:   p50DurationMs,
		P90DurationMs:   p90DurationMs,
		P95DurationMs:   p95DurationMs,
		P99DurationMs:   p99DurationMs,
		ErrorRatePct:    errorRatePct,
	}

	run, err := model.NewTestRunWithID(
		runID, suiteID, artifactID, configurationID, runnerProfileID, scheduleID,
		model.RunStatus(status), k8sJobName, k8sNamespace,
		startedAt, finishedAt, exitCode, slaPassed,
		metrics, s3ReportKey, s3LogsKey, summaryJSON, abortReason, createdAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute test run")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully found test run by id")
	return run, nil
}

// FindByK8sJobName retrieves a TestRun aggregate by Kubernetes Job name.
func (r *TestRunRepository) FindByK8sJobName(ctx context.Context, jobName string) (*model.TestRun, error) {
	start := time.Now()
	trimmedJobName := strings.TrimSpace(jobName)
	if trimmedJobName == "" {
		return nil, model.ErrValidation
	}

	log := zerolog.Ctx(ctx).With().Str("op", "TestRunRepository.FindByK8sJobName").Str("job_name", trimmedJobName).Logger()
	log.Debug().Msg("finding test run by k8s job name")

	query := `
		SELECT
			id, suite_id, artifact_id, configuration_id, runner_profile_id, schedule_id,
			status, k8s_job_name, k8s_namespace, started_at, finished_at, exit_code, sla_passed,
			total_iterations, total_requests, avg_tps,
			p50_duration_ms, p90_duration_ms, p95_duration_ms, p99_duration_ms,
			error_rate_pct, s3_report_key, s3_logs_key, summary_json, abort_reason, created_at
		FROM test_runs
		WHERE k8s_job_name = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	var (
		runID           string
		suiteID         string
		artifactID      string
		configurationID *string
		runnerProfileID string
		scheduleID      *string
		status          string
		k8sJobName      string
		k8sNamespace    string
		startedAt       *time.Time
		finishedAt      *time.Time
		exitCode        *int
		slaPassed       *bool
		totalIterations int64
		totalRequests   int64
		avgTPS          float64
		p50DurationMs   float64
		p90DurationMs   float64
		p95DurationMs   float64
		p99DurationMs   float64
		errorRatePct    float64
		s3ReportKey     string
		s3LogsKey       string
		summaryJSON     []byte
		abortReason     string
		createdAt       time.Time
	)
	err := r.pool.QueryRow(ctx, query, trimmedJobName).Scan(
		&runID, &suiteID, &artifactID, &configurationID, &runnerProfileID, &scheduleID,
		&status, &k8sJobName, &k8sNamespace, &startedAt, &finishedAt, &exitCode, &slaPassed,
		&totalIterations, &totalRequests, &avgTPS,
		&p50DurationMs, &p90DurationMs, &p95DurationMs, &p99DurationMs,
		&errorRatePct, &s3ReportKey, &s3LogsKey, &summaryJSON, &abortReason, &createdAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to find test run by k8s job name")
		return nil, MapError(err)
	}

	metrics := model.RunMetrics{
		TotalIterations: totalIterations,
		TotalRequests:   totalRequests,
		AvgTPS:          avgTPS,
		P50DurationMs:   p50DurationMs,
		P90DurationMs:   p90DurationMs,
		P95DurationMs:   p95DurationMs,
		P99DurationMs:   p99DurationMs,
		ErrorRatePct:    errorRatePct,
	}

	run, err := model.NewTestRunWithID(
		runID, suiteID, artifactID, configurationID, runnerProfileID, scheduleID,
		model.RunStatus(status), k8sJobName, k8sNamespace,
		startedAt, finishedAt, exitCode, slaPassed,
		metrics, s3ReportKey, s3LogsKey, summaryJSON, abortReason, createdAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute test run")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully found test run by k8s job name")
	return run, nil
}

// List returns TestRun aggregates optionally filtered by suiteID and status.
func (r *TestRunRepository) List(ctx context.Context, suiteID string, status model.RunStatus) ([]*model.TestRun, error) {
	runs, _, err := r.ListFiltered(ctx, model.RunFilter{
		SuiteID: suiteID,
		Status:  status,
	})
	return runs, err
}

// ListFiltered returns TestRun aggregates matching filter criteria along with total count.
func (r *TestRunRepository) ListFiltered(ctx context.Context, filter model.RunFilter) ([]*model.TestRun, int64, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "TestRunRepository.ListFiltered").
		Str("suite_id", filter.SuiteID).
		Str("status", string(filter.Status)).
		Str("schedule_id", filter.ScheduleID).
		Logger()
	log.Debug().Msg("listing filtered test runs")

	var (
		whereClauses []string
		args         []interface{}
	)

	if strings.TrimSpace(filter.SuiteID) != "" {
		args = append(args, strings.TrimSpace(filter.SuiteID))
		whereClauses = append(whereClauses, fmt.Sprintf("suite_id = $%d", len(args)))
	}

	if filter.Status.IsValid() {
		args = append(args, string(filter.Status))
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", len(args)))
	}

	if strings.TrimSpace(filter.ScheduleID) != "" {
		args = append(args, strings.TrimSpace(filter.ScheduleID))
		whereClauses = append(whereClauses, fmt.Sprintf("schedule_id = $%d", len(args)))
	}

	if filter.From != nil && !filter.From.IsZero() {
		args = append(args, *filter.From)
		whereClauses = append(whereClauses, fmt.Sprintf("created_at >= $%d", len(args)))
	}

	if filter.To != nil && !filter.To.IsZero() {
		args = append(args, *filter.To)
		whereClauses = append(whereClauses, fmt.Sprintf("created_at <= $%d", len(args)))
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Compute total matching count
	countQuery := "SELECT COUNT(*) FROM test_runs" + whereSQL
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed counting test runs")
		return nil, 0, MapError(err)
	}

	// 2. Fetch paginated records
	selectQuery := `
		SELECT
			id, suite_id, artifact_id, configuration_id, runner_profile_id, schedule_id,
			status, k8s_job_name, k8s_namespace, started_at, finished_at, exit_code, sla_passed,
			total_iterations, total_requests, avg_tps,
			p50_duration_ms, p90_duration_ms, p95_duration_ms, p99_duration_ms,
			error_rate_pct, s3_report_key, s3_logs_key, summary_json, abort_reason, created_at
		FROM test_runs
	` + whereSQL + " ORDER BY created_at DESC"

	queryArgs := make([]interface{}, len(args))
	copy(queryArgs, args)

	if filter.Limit > 0 {
		queryArgs = append(queryArgs, filter.Limit)
		selectQuery += fmt.Sprintf(" LIMIT $%d", len(queryArgs))
	}
	if filter.Offset > 0 {
		queryArgs = append(queryArgs, filter.Offset)
		selectQuery += fmt.Sprintf(" OFFSET $%d", len(queryArgs))
	}

	rows, err := r.pool.Query(ctx, selectQuery, queryArgs...)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed querying test runs")
		return nil, 0, MapError(err)
	}
	defer rows.Close()

	var runs []*model.TestRun
	for rows.Next() {
		var (
			runID           string
			sID             string
			artifactID      string
			configurationID *string
			runnerProfileID string
			scheduleID      *string
			st              string
			k8sJobName      string
			k8sNamespace    string
			startedAt       *time.Time
			finishedAt      *time.Time
			exitCode        *int
			slaPassed       *bool
			totalIterations int64
			totalRequests   int64
			avgTPS          float64
			p50DurationMs   float64
			p90DurationMs   float64
			p95DurationMs   float64
			p99DurationMs   float64
			errorRatePct    float64
			s3ReportKey     string
			s3LogsKey       string
			summaryJSON     []byte
			abortReason     string
			createdAt       time.Time
		)
		if err := rows.Scan(
			&runID, &sID, &artifactID, &configurationID, &runnerProfileID, &scheduleID,
			&st, &k8sJobName, &k8sNamespace, &startedAt, &finishedAt, &exitCode, &slaPassed,
			&totalIterations, &totalRequests, &avgTPS,
			&p50DurationMs, &p90DurationMs, &p95DurationMs, &p99DurationMs,
			&errorRatePct, &s3ReportKey, &s3LogsKey, &summaryJSON, &abortReason, &createdAt,
		); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to scan test run row")
			return nil, 0, MapError(err)
		}

		metrics := model.RunMetrics{
			TotalIterations: totalIterations,
			TotalRequests:   totalRequests,
			AvgTPS:          avgTPS,
			P50DurationMs:   p50DurationMs,
			P90DurationMs:   p90DurationMs,
			P95DurationMs:   p95DurationMs,
			P99DurationMs:   p99DurationMs,
			ErrorRatePct:    errorRatePct,
		}

		run, err := model.NewTestRunWithID(
			runID, sID, artifactID, configurationID, runnerProfileID, scheduleID,
			model.RunStatus(st), k8sJobName, k8sNamespace,
			startedAt, finishedAt, exitCode, slaPassed,
			metrics, s3ReportKey, s3LogsKey, summaryJSON, abortReason, createdAt,
		)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute test run from row")
			return nil, 0, err
		}
		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("error iterating test run rows")
		return nil, 0, MapError(err)
	}

	log.Info().Int("count", len(runs)).Int64("total", total).Dur("duration_ms", time.Since(start)).Msg("successfully listed test runs")
	return runs, total, nil
}

// Delete removes a TestRun by ID.
func (r *TestRunRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "TestRunRepository.Delete").Str("run_id", id).Logger()
	log.Debug().Msg("deleting test run")

	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM test_runs WHERE id = $1`, id)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to delete test run")
		return MapError(err)
	}

	if cmdTag.RowsAffected() == 0 {
		log.Error().Dur("duration_ms", time.Since(start)).Msg("test run not found for deletion")
		return model.ErrNotFound
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully deleted test run")
	return nil
}

// Static compile-time interface assertion
var _ outbound.TestRunRepository = (*TestRunRepository)(nil)
