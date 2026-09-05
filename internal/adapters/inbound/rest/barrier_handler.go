package rest

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// BarrierHandler exposes HTTP REST endpoints for distributed start barrier rendezvous coordination.
type BarrierHandler struct {
	barrierUC inbound.BarrierUseCase
}

// NewBarrierHandler constructs a new BarrierHandler.
func NewBarrierHandler(barrierUC inbound.BarrierUseCase) *BarrierHandler {
	return &BarrierHandler{
		barrierUC: barrierUC,
	}
}

// AwaitBarrier handles POST /api/v1/runs/:id/barrier/await.
// Worker pods register and block until all peers are healthy and ready, then receive synchronized target start time.
func (h *BarrierHandler) AwaitBarrier(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	runID := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "BarrierHandler.AwaitBarrier").
		Str("run_id", runID).
		Logger()
	log.Debug().Msg("handling barrier await request")

	if runID == "" {
		log.Warn().Msg("missing run id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "run id cannot be empty"})
		return
	}

	var req BarrierAwaitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("invalid barrier await request payload")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request payload: " + err.Error()})
		return
	}

	var timeout time.Duration
	if req.TimeoutMs != nil && *req.TimeoutMs > 0 {
		timeout = time.Duration(*req.TimeoutMs) * time.Millisecond
	}

	var releaseDelay time.Duration
	if req.ReleaseDelayMs != nil && *req.ReleaseDelayMs > 0 {
		releaseDelay = time.Duration(*req.ReleaseDelayMs) * time.Millisecond
	}

	cmd := inbound.AwaitRendezvousCommand{
		RunID:        runID,
		WorkerID:     strings.TrimSpace(req.WorkerID),
		TotalWorkers: req.TotalWorkers,
		Timeout:      timeout,
		ReleaseDelay: releaseDelay,
	}

	res, err := h.barrierUC.AwaitRendezvous(ctx, cmd)
	if err != nil {
		log.Warn().Err(err).Dur("duration_ms", time.Since(start)).Msg("barrier rendezvous await failed")
		HandleError(c, err)
		return
	}

	targetTimeStr := res.TargetStartTime.Format(time.RFC3339Nano)
	resp := BarrierResponse{
		RunID:           res.RunID,
		WorkerID:        res.WorkerID,
		Status:          string(res.Status),
		TotalWorkers:    res.TotalWorkers,
		ReadyWorkers:    res.TotalWorkers,
		TargetStartTime: &targetTimeStr,
		StartInMs:       res.StartIn.Milliseconds(),
	}

	log.Info().
		Str("status", string(res.Status)).
		Time("target_start_time", res.TargetStartTime).
		Dur("start_in_ms", res.StartIn).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully completed barrier rendezvous")

	c.JSON(http.StatusOK, resp)
}

// AbortBarrier handles POST /api/v1/runs/:id/barrier/abort.
// A worker signals an early failure/abort, promptly unblocking peers.
func (h *BarrierHandler) AbortBarrier(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	runID := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "BarrierHandler.AbortBarrier").
		Str("run_id", runID).
		Logger()
	log.Debug().Msg("handling barrier abort request")

	if runID == "" {
		log.Warn().Msg("missing run id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "run id cannot be empty"})
		return
	}

	var req BarrierAbortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("invalid barrier abort request payload")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request payload: " + err.Error()})
		return
	}

	cmd := inbound.SignalAbortCommand{
		RunID:    runID,
		WorkerID: strings.TrimSpace(req.WorkerID),
		Reason:   strings.TrimSpace(req.Reason),
	}

	if err := h.barrierUC.SignalAbort(ctx, cmd); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed signaling barrier abort")
		HandleError(c, err)
		return
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully aborted barrier rendezvous")
	c.JSON(http.StatusOK, gin.H{"status": string(model.BarrierStatusAborted)})
}

// GetBarrier handles GET /api/v1/runs/:id/barrier.
// Returns the current status of the rendezvous session.
func (h *BarrierHandler) GetBarrier(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	runID := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "BarrierHandler.GetBarrier").
		Str("run_id", runID).
		Logger()
	log.Debug().Msg("handling get barrier status request")

	if runID == "" {
		log.Warn().Msg("missing run id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "run id cannot be empty"})
		return
	}

	session, err := h.barrierUC.GetBarrierStatus(ctx, runID)
	if err != nil {
		log.Warn().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed retrieving barrier status")
		HandleError(c, err)
		return
	}

	log.Info().
		Str("status", string(session.Status())).
		Int("ready_workers", session.ReadyCount()).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully retrieved barrier status")

	c.JSON(http.StatusOK, ToBarrierResponse(session))
}
