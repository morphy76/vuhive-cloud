package rest

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
)

// ScheduleHandler exposes HTTP REST endpoints for recurring test schedule management.
type ScheduleHandler struct {
	schedulesUC inbound.SchedulesUseCase
}

// NewScheduleHandler constructs a new ScheduleHandler.
func NewScheduleHandler(schedulesUC inbound.SchedulesUseCase) *ScheduleHandler {
	return &ScheduleHandler{
		schedulesUC: schedulesUC,
	}
}

// CreateSchedule handles POST /api/v1/schedules
func (h *ScheduleHandler) CreateSchedule(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleHandler.CreateSchedule").
		Logger()
	log.Debug().Msg("handling create schedule request")

	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("invalid create schedule request payload")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	schedule, err := h.schedulesUC.CreateSchedule(
		ctx,
		req.SuiteID,
		req.ArtifactID,
		req.ConfigurationID,
		req.RunnerProfileID,
		req.Name,
		req.CronExpression,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating schedule")
		HandleError(c, err)
		return
	}

	log.Info().
		Str("schedule_id", schedule.ID()).
		Dur("duration_ms", time.Since(start)).
		Msg("completed create schedule request")

	c.JSON(http.StatusCreated, ToScheduleResponse(schedule))
}

// GetSchedule handles GET /api/v1/schedules/:id
func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	id := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleHandler.GetSchedule").
		Str("schedule_id", id).
		Logger()
	log.Debug().Msg("handling get schedule request")

	schedule, err := h.schedulesUC.GetSchedule(ctx, id)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching schedule")
		HandleError(c, err)
		return
	}

	log.Info().
		Dur("duration_ms", time.Since(start)).
		Msg("completed get schedule request")

	c.JSON(http.StatusOK, ToScheduleResponse(schedule))
}

// ListSchedules handles GET /api/v1/schedules
func (h *ScheduleHandler) ListSchedules(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleHandler.ListSchedules").
		Logger()
	log.Debug().Msg("handling list schedules request")

	schedules, err := h.schedulesUC.ListSchedules(ctx)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing schedules")
		HandleError(c, err)
		return
	}

	log.Info().
		Int("count", len(schedules)).
		Dur("duration_ms", time.Since(start)).
		Msg("completed list schedules request")

	c.JSON(http.StatusOK, ToScheduleListResponse(schedules))
}

// UpdateSchedule handles PUT /api/v1/schedules/:id
func (h *ScheduleHandler) UpdateSchedule(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	id := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleHandler.UpdateSchedule").
		Str("schedule_id", id).
		Logger()
	log.Debug().Msg("handling update schedule request")

	var req UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("invalid update schedule request payload")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	schedule, err := h.schedulesUC.UpdateSchedule(ctx, id, req.CronExpression)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed updating schedule")
		HandleError(c, err)
		return
	}

	log.Info().
		Dur("duration_ms", time.Since(start)).
		Msg("completed update schedule request")

	c.JSON(http.StatusOK, ToScheduleResponse(schedule))
}

// DeleteSchedule handles DELETE /api/v1/schedules/:id
func (h *ScheduleHandler) DeleteSchedule(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	id := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleHandler.DeleteSchedule").
		Str("schedule_id", id).
		Logger()
	log.Debug().Msg("handling delete schedule request")

	err := h.schedulesUC.DeleteSchedule(ctx, id)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting schedule")
		HandleError(c, err)
		return
	}

	log.Info().
		Dur("duration_ms", time.Since(start)).
		Msg("completed delete schedule request")

	c.Status(http.StatusNoContent)
}
