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

	allowInsecure := false
	if insecureVal := strings.TrimSpace(c.Request.FormValue("allow_insecure_imports")); insecureVal != "" {
		allowInsecure = insecureVal == "true" || insecureVal == "1"
	}

	artifacts, err := h.buildsUC.TriggerBuildWithOptions(ctx, suiteID, targetPlatform, file, header.Size, inbound.BuildOptions{
		AllowInsecureImports: allowInsecure,
	})
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

// CancelBuild handles POST /api/v1/suites/:id/artifacts/:artifactId/cancel
// Cancels an in-progress or pending build and terminates its Kubernetes Job.
func (h *ArtifactHandler) CancelBuild(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	suiteID := strings.TrimSpace(c.Param("id"))
	artifactID := strings.TrimSpace(c.Param("artifactId"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "ArtifactHandler.CancelBuild").
		Str("suite_id", suiteID).
		Str("artifact_id", artifactID).
		Logger()
	log.Debug().Msg("handling cancel build request")

	if suiteID == "" || artifactID == "" {
		log.Warn().Msg("missing suite id or artifact id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "suite id and artifact id cannot be empty"})
		return
	}

	var req CancelBuildRequest
	_ = c.ShouldBindJSON(&req)

	artifact, err := h.buildsUC.CancelBuild(ctx, suiteID, artifactID, req.Reason)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed cancelling build")
		HandleError(c, err)
		return
	}

	log.Info().
		Str("status", string(artifact.Status())).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully cancelled build")

	c.JSON(http.StatusOK, ToArtifactResponse(artifact))
}

// RetryBuild handles POST /api/v1/suites/:id/artifacts/:artifactId/retry
// Resets a FAILED or CANCELLED build to PENDING and triggers asynchronous re-compilation.
func (h *ArtifactHandler) RetryBuild(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	suiteID := strings.TrimSpace(c.Param("id"))
	artifactID := strings.TrimSpace(c.Param("artifactId"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "ArtifactHandler.RetryBuild").
		Str("suite_id", suiteID).
		Str("artifact_id", artifactID).
		Logger()
	log.Debug().Msg("handling retry build request")

	if suiteID == "" || artifactID == "" {
		log.Warn().Msg("missing suite id or artifact id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "suite id and artifact id cannot be empty"})
		return
	}

	artifact, err := h.buildsUC.RetryBuild(ctx, suiteID, artifactID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed retrying build")
		HandleError(c, err)
		return
	}

	log.Info().
		Str("status", string(artifact.Status())).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully triggered build retry")

	c.JSON(http.StatusAccepted, ToArtifactResponse(artifact))
}

// DeleteArtifact handles DELETE /api/v1/suites/:id/artifacts/:artifactId
// Deletes compiled binary and log storage assets and removes the artifact record.
func (h *ArtifactHandler) DeleteArtifact(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()
	suiteID := strings.TrimSpace(c.Param("id"))
	artifactID := strings.TrimSpace(c.Param("artifactId"))

	log := zerolog.Ctx(ctx).With().
		Str("op", "ArtifactHandler.DeleteArtifact").
		Str("suite_id", suiteID).
		Str("artifact_id", artifactID).
		Logger()
	log.Debug().Msg("handling delete artifact request")

	if suiteID == "" || artifactID == "" {
		log.Warn().Msg("missing suite id or artifact id parameter")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "suite id and artifact id cannot be empty"})
		return
	}

	if err := h.buildsUC.DeleteArtifact(ctx, suiteID, artifactID); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting artifact")
		HandleError(c, err)
		return
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed artifact deletion")
	c.Status(http.StatusNoContent)
}
