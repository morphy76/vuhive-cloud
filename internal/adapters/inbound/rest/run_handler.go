package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// RunHandler exposes HTTP REST endpoints for managing and completing test runs.
type RunHandler struct {
	runsUC inbound.RunsUseCase
}

// NewRunHandler constructs a new RunHandler.
func NewRunHandler(runsUC inbound.RunsUseCase) *RunHandler {
	return &RunHandler{
		runsUC: runsUC,
	}
}

// CompleteRun handles POST /api/v1/runs/:id/complete and POST /api/v1/runs/complete.
// Ingests deterministic vuhive summary.json, extracts KPIs into PostgreSQL,
// and updates the run execution lifecycle state.
func (h *RunHandler) CompleteRun(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	pathID := strings.TrimSpace(c.Param("id"))
	if strings.EqualFold(pathID, "complete") {
		pathID = ""
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunHandler.CompleteRun").
		Str("path_id", pathID).
		Logger()
	log.Debug().Msg("handling test run completion callback")

	var req CompleteRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("invalid complete run request payload")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request payload: " + err.Error()})
		return
	}

	runID := pathID
	if runID == "" {
		runID = strings.TrimSpace(req.RunID)
	}
	if runID == "" {
		log.Warn().Msg("missing run id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "run id cannot be empty"})
		return
	}

	// Resolve summary JSON
	var summaryJSON []byte
	if len(req.SummaryJSON) > 0 {
		summaryJSON = req.SummaryJSON
	} else if len(req.Summary) > 0 {
		if data, err := json.Marshal(req.Summary); err == nil {
			summaryJSON = data
		}
	}

	// Parse finished_at timestamp if provided
	var finishedAt *time.Time
	if req.FinishedAt != nil && strings.TrimSpace(*req.FinishedAt) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.FinishedAt)); err == nil {
			finishedAt = &t
		}
	}

	cmd := inbound.CompleteRunCommand{
		RunID:       runID,
		ExitCode:    req.ExitCode,
		ReportKey:   req.ReportKey,
		LogsKey:     req.LogsKey,
		FinishedAt:  finishedAt,
		SummaryJSON: summaryJSON,
	}

	run, err := h.runsUC.CompleteRun(ctx, cmd)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed completing test run")
		HandleError(c, err)
		return
	}

	log.Info().
		Str("run_id", run.ID()).
		Str("status", string(run.Status())).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully processed test run completion")

	c.JSON(http.StatusOK, ToRunResponse(run))
}

// GetRun handles GET /api/v1/runs/:id.
// Returns execution status, metadata, duration, exit code, SLA status, and indexed performance KPIs.
func (h *RunHandler) GetRun(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	runID := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunHandler.GetRun").
		Str("run_id", runID).
		Logger()
	log.Debug().Msg("handling get test run request")

	if runID == "" {
		log.Warn().Msg("missing run id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "run id cannot be empty"})
		return
	}

	run, err := h.runsUC.GetRun(ctx, runID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching test run")
		HandleError(c, err)
		return
	}

	log.Info().
		Str("status", string(run.Status())).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully retrieved test run")

	c.JSON(http.StatusOK, ToRunResponse(run))
}

// ListRuns handles GET /api/v1/runs.
// Supports filtering by suite_id, status, schedule_id, from, to date range, limit and offset.
func (h *RunHandler) ListRuns(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	suiteID := strings.TrimSpace(c.Query("suite_id"))
	statusStr := strings.TrimSpace(c.Query("status"))
	scheduleID := strings.TrimSpace(c.Query("schedule_id"))
	fromStr := strings.TrimSpace(c.Query("from"))
	toStr := strings.TrimSpace(c.Query("to"))

	limit := 20
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	var fromTime *time.Time
	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromTime = &t
		}
	}

	var toTime *time.Time
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			toTime = &t
		}
	}

	filter := model.RunFilter{
		SuiteID:    suiteID,
		Status:     model.RunStatus(statusStr),
		ScheduleID: scheduleID,
		From:       fromTime,
		To:         toTime,
		Limit:      limit,
		Offset:     offset,
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunHandler.ListRuns").
		Str("suite_id", suiteID).
		Str("status", statusStr).
		Str("schedule_id", scheduleID).
		Int("limit", limit).
		Int("offset", offset).
		Logger()
	log.Debug().Msg("handling list test runs request")

	runs, total, err := h.runsUC.ListRuns(ctx, filter)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing test runs")
		HandleError(c, err)
		return
	}

	log.Info().
		Int("count", len(runs)).
		Int64("total", total).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully listed test runs")

	c.JSON(http.StatusOK, ToRunListResponse(runs, total, limit, offset))
}

// GetRunReport handles GET /api/v1/runs/:id/report.
// Streams raw summary.json or generates a presigned download URL when ?presign=true.
func (h *RunHandler) GetRunReport(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	runID := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunHandler.GetRunReport").
		Str("run_id", runID).
		Logger()
	log.Debug().Msg("handling get test run report request")

	if runID == "" {
		log.Warn().Msg("missing run id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "run id cannot be empty"})
		return
	}

	if strings.EqualFold(c.Query("presign"), "true") || strings.EqualFold(c.Query("download"), "presigned") {
		lifetime := 15 * time.Minute
		url, err := h.runsUC.GetRunReportURL(ctx, runID, lifetime)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed generating report presigned URL")
			HandleError(c, err)
			return
		}
		c.JSON(http.StatusOK, PresignedURLResponse{
			DownloadURL:      url,
			ExpiresInSeconds: int64(lifetime.Seconds()),
		})
		return
	}

	rc, err := h.runsUC.GetRunReport(ctx, runID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching test run report")
		HandleError(c, err)
		return
	}
	defer func() { _ = rc.Close() }()

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "inline; filename=\"summary.json\"")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, rc)
}

// GetRunLogs handles GET /api/v1/runs/:id/logs.
// Streams raw run.log or generates a presigned download URL when ?presign=true.
func (h *RunHandler) GetRunLogs(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	runID := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunHandler.GetRunLogs").
		Str("run_id", runID).
		Logger()
	log.Debug().Msg("handling get test run logs request")

	if runID == "" {
		log.Warn().Msg("missing run id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "run id cannot be empty"})
		return
	}

	if strings.EqualFold(c.Query("presign"), "true") || strings.EqualFold(c.Query("download"), "presigned") {
		lifetime := 15 * time.Minute
		url, err := h.runsUC.GetRunLogsURL(ctx, runID, lifetime)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed generating logs presigned URL")
			HandleError(c, err)
			return
		}
		c.JSON(http.StatusOK, PresignedURLResponse{
			DownloadURL:      url,
			ExpiresInSeconds: int64(lifetime.Seconds()),
		})
		return
	}

	rc, err := h.runsUC.GetRunLogs(ctx, runID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching test run logs")
		HandleError(c, err)
		return
	}
	defer func() { _ = rc.Close() }()

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", "inline; filename=\"run.log\"")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, rc)
}

// Compile-time assertion
var _ = (*RunHandler)(nil)

