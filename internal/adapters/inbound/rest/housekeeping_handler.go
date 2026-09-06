package rest

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/rs/zerolog"
)

// HousekeepingHandler exposes administrative endpoints for data retention and storage housekeeping.
type HousekeepingHandler struct {
	housekeepingUC inbound.HousekeepingUseCase
}

// NewHousekeepingHandler constructs a new HousekeepingHandler.
func NewHousekeepingHandler(housekeepingUC inbound.HousekeepingUseCase) *HousekeepingHandler {
	return &HousekeepingHandler{housekeepingUC: housekeepingUC}
}

// RunHousekeeping handles POST /api/v1/system/housekeeping.
func (h *HousekeepingHandler) RunHousekeeping(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	log := zerolog.Ctx(ctx).With().Str("op", "HousekeepingHandler.RunHousekeeping").Logger()
	log.Debug().Msg("handling housekeeping trigger request")

	var cmd inbound.HousekeepingCommand

	// Parse query params if present
	if dryRunQuery := c.Query("dry_run"); dryRunQuery == "true" {
		cmd.DryRun = true
	}
	if suiteQuery := strings.TrimSpace(c.Query("suite_id")); suiteQuery != "" {
		cmd.SuiteID = &suiteQuery
	}

	// Parse JSON body if present and Content-Type is application/json
	if c.ContentType() == "application/json" && c.Request.Body != nil && c.Request.ContentLength > 0 {
		var req HousekeepingRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			if req.DryRun {
				cmd.DryRun = true
			}
			if req.SuiteID != nil && strings.TrimSpace(*req.SuiteID) != "" {
				s := strings.TrimSpace(*req.SuiteID)
				cmd.SuiteID = &s
			}
		}
	}

	result, err := h.housekeepingUC.RunHousekeeping(ctx, cmd)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("housekeeping execution failed")
		HandleError(c, err)
		return
	}

	log.Info().
		Int64("purged_logs", result.PurgedLogsCount).
		Int64("pruned_runs", result.PrunedRunsCount).
		Dur("duration_ms", time.Since(start)).
		Msg("housekeeping trigger request completed")

	c.JSON(http.StatusOK, ToHousekeepingResponse(result))
}

// GetPolicy handles GET /api/v1/system/housekeeping/policy.
func (h *HousekeepingHandler) GetPolicy(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	log := zerolog.Ctx(ctx).With().Str("op", "HousekeepingHandler.GetPolicy").Logger()
	log.Debug().Msg("handling get retention policy request")

	policy := h.housekeepingUC.GetGlobalPolicy(ctx)
	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed get retention policy request")

	c.JSON(http.StatusOK, ToRetentionPolicyResponse(policy))
}
