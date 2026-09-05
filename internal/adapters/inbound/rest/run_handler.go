package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
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

// AbortRun handles POST /api/v1/runs/:id/abort.
// Cancels an active or queued test run, terminates its K8s Job, and marks it ABORTED.
func (h *RunHandler) AbortRun(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	runID := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "RunHandler.AbortRun").
		Str("run_id", runID).
		Logger()
	log.Debug().Msg("handling test run abort request")

	if runID == "" {
		log.Warn().Msg("missing run id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "run id cannot be empty"})
		return
	}

	var req AbortRunRequest
	if c.Request.Body != nil && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Warn().Err(err).Msg("invalid abort run request payload")
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request payload: " + err.Error()})
			return
		}
	}

	reason := strings.TrimSpace(req.Reason)
	requestedBy := strings.TrimSpace(req.RequestedBy)
	if reason == "" {
		reason = "manual cancellation via API"
	}
	if requestedBy != "" {
		reason = fmt.Sprintf("%s (requested by: %s)", reason, requestedBy)
	}

	run, err := h.runsUC.AbortRun(ctx, runID, reason)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed aborting test run")
		HandleError(c, err)
		return
	}

	log.Info().
		Str("run_id", run.ID()).
		Str("status", string(run.Status())).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully processed test run abort")

	c.JSON(http.StatusOK, ToRunResponse(run))
}

// Compile-time assertion
var _ = (*RunHandler)(nil)
