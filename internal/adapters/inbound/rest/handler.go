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

// ArtifactHandler exposes HTTP REST endpoints for artifact compilation and inspection.
type ArtifactHandler struct {
	buildsUC inbound.BuildsUseCase
}

// NewArtifactHandler constructs a new ArtifactHandler.
func NewArtifactHandler(buildsUC inbound.BuildsUseCase) *ArtifactHandler {
	return &ArtifactHandler{
		buildsUC: buildsUC,
	}
}

// UploadAndBuild handles POST /api/v1/suites/:id/builds
// Accepts multipart form with source archive (.tar.gz) and optional target architecture.
func (h *ArtifactHandler) UploadAndBuild(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	suiteID := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "ArtifactHandler.UploadAndBuild").
		Str("suite_id", suiteID).
		Logger()
	log.Debug().Msg("handling source upload and build trigger")

	if suiteID == "" {
		log.Warn().Msg("missing suite id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "suite id cannot be empty"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		file, header, err = c.Request.FormFile("source")
	}
	if err != nil {
		log.Warn().Err(err).Msg("missing multipart file")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "multipart form must include a source archive under 'file' or 'source'"})
		return
	}
	defer func() {
		_ = file.Close()
	}()

	archStr := strings.TrimSpace(c.Request.FormValue("platform"))
	if archStr == "" {
		archStr = strings.TrimSpace(c.Request.FormValue("arch"))
	}
	if archStr == "" {
		archStr = strings.TrimSpace(c.Request.FormValue("target_arch"))
	}

	var targetPlatform *model.Platform
	if archStr != "" && strings.ToLower(archStr) != "all" {
		p, err := model.ParsePlatform(archStr)
		if err != nil {
			log.Warn().Str("arch", archStr).Err(err).Msg("invalid platform requested")
			HandleError(c, err)
			return
		}
		targetPlatform = &p
	}

	artifacts, err := h.buildsUC.TriggerBuild(ctx, suiteID, targetPlatform, file, header.Size)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed triggering build")
		HandleError(c, err)
		return
	}

	log.Info().
		Int("artifacts_count", len(artifacts)).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully initiated async build")

	c.JSON(http.StatusAccepted, BuildTriggerResponse{
		Message:   "build triggered successfully",
		Artifacts: ToArtifactListResponse(artifacts).Artifacts,
	})
}

// ListArtifacts handles GET /api/v1/suites/:id/artifacts
// Returns a list of all compiled binary artifacts and their checksums for the given suite.
func (h *ArtifactHandler) ListArtifacts(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	suiteID := strings.TrimSpace(c.Param("id"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "ArtifactHandler.ListArtifacts").
		Str("suite_id", suiteID).
		Logger()
	log.Debug().Msg("handling list artifacts request")

	if suiteID == "" {
		log.Warn().Msg("missing suite id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "suite id cannot be empty"})
		return
	}

	artifacts, err := h.buildsUC.ListArtifacts(ctx, suiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing artifacts")
		HandleError(c, err)
		return
	}

	log.Info().
		Int("count", len(artifacts)).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully listed artifacts")

	c.JSON(http.StatusOK, ToArtifactListResponse(artifacts))
}
