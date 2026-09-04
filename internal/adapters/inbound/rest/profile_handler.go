package rest

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/rs/zerolog"
)

// ProfileHandler exposes HTTP REST endpoints for Runner Profile management.
type ProfileHandler struct {
	profilesUC inbound.ProfilesUseCase
}

// NewProfileHandler constructs a new ProfileHandler.
func NewProfileHandler(profilesUC inbound.ProfilesUseCase) *ProfileHandler {
	return &ProfileHandler{
		profilesUC: profilesUC,
	}
}

// CreateProfile handles POST /api/v1/profiles
// Creates a new reusable Runner Profile specifying compute limits, affinity, and tolerations.
func (h *ProfileHandler) CreateProfile(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	log := zerolog.Ctx(ctx).With().
		Str("op", "ProfileHandler.CreateProfile").
		Logger()
	log.Debug().Msg("handling create runner profile request")

	var req CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("invalid create runner profile request payload")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	cmd := inbound.CreateProfileCommand{
		Name:          req.Name,
		Description:   req.Description,
		RunnerImage:   req.RunnerImage,
		CPURequest:    req.CPURequest,
		CPULimit:      req.CPULimit,
		MemoryRequest: req.MemoryRequest,
		MemoryLimit:   req.MemoryLimit,
		NodeSelector:  req.NodeSelector,
		Affinity:      FromAffinityDTO(req.Affinity),
		Tolerations:   FromTolerationsDTO(req.Tolerations),
	}

	profile, err := h.profilesUC.CreateProfile(ctx, cmd)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating runner profile")
		HandleError(c, err)
		return
	}

	log.Info().
		Str("profile_id", profile.ID()).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully created runner profile")

	c.JSON(http.StatusCreated, ToProfileResponse(profile))
}

// ListProfiles handles GET /api/v1/profiles
// Returns a list of all defined runner profiles.
func (h *ProfileHandler) ListProfiles(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	log := zerolog.Ctx(ctx).With().
		Str("op", "ProfileHandler.ListProfiles").
		Logger()
	log.Debug().Msg("handling list runner profiles request")

	profiles, err := h.profilesUC.ListProfiles(ctx)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing runner profiles")
		HandleError(c, err)
		return
	}

	log.Info().
		Int("count", len(profiles)).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully listed runner profiles")

	c.JSON(http.StatusOK, ToProfileListResponse(profiles))
}

// GetProfile handles GET /api/v1/profiles/:id
// Retrieves detail of a single runner profile by ID.
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	profileID := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "ProfileHandler.GetProfile").
		Str("profile_id", profileID).
		Logger()
	log.Debug().Msg("handling get runner profile request")

	if profileID == "" {
		log.Warn().Msg("missing profile id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "profile id cannot be empty"})
		return
	}

	profile, err := h.profilesUC.GetProfile(ctx, profileID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed retrieving runner profile")
		HandleError(c, err)
		return
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully retrieved runner profile")
	c.JSON(http.StatusOK, ToProfileResponse(profile))
}

// UpdateProfile handles PUT /api/v1/profiles/:id
// Updates the configuration and Kubernetes resource specifications of an existing runner profile.
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	profileID := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "ProfileHandler.UpdateProfile").
		Str("profile_id", profileID).
		Logger()
	log.Debug().Msg("handling update runner profile request")

	if profileID == "" {
		log.Warn().Msg("missing profile id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "profile id cannot be empty"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("invalid update runner profile request payload")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	cmd := inbound.UpdateProfileCommand{
		Name:          req.Name,
		Description:   req.Description,
		RunnerImage:   req.RunnerImage,
		CPURequest:    req.CPURequest,
		CPULimit:      req.CPULimit,
		MemoryRequest: req.MemoryRequest,
		MemoryLimit:   req.MemoryLimit,
		NodeSelector:  req.NodeSelector,
		Affinity:      FromAffinityDTO(req.Affinity),
		Tolerations:   FromTolerationsDTO(req.Tolerations),
	}

	profile, err := h.profilesUC.UpdateProfile(ctx, profileID, cmd)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed updating runner profile")
		HandleError(c, err)
		return
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully updated runner profile")
	c.JSON(http.StatusOK, ToProfileResponse(profile))
}

// DeleteProfile handles DELETE /api/v1/profiles/:id
// Deletes a runner profile by ID.
func (h *ProfileHandler) DeleteProfile(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	profileID := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "ProfileHandler.DeleteProfile").
		Str("profile_id", profileID).
		Logger()
	log.Debug().Msg("handling delete runner profile request")

	if profileID == "" {
		log.Warn().Msg("missing profile id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "profile id cannot be empty"})
		return
	}

	if err := h.profilesUC.DeleteProfile(ctx, profileID); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting runner profile")
		HandleError(c, err)
		return
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully deleted runner profile")
	c.Status(http.StatusNoContent)
}
