package rest

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/rs/zerolog"
)

// Handler handles inbound HTTP requests for the BFF service.
type Handler struct {
	bffService inbound.BFFService
	version    string
}

// NewHandler constructs a new Handler instance.
func NewHandler(bffService inbound.BFFService, version string) *Handler {
	return &Handler{
		bffService: bffService,
		version:    version,
	}
}

// Healthz handles GET /healthz.
func (h *Handler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

// Version handles GET /version.
func (h *Handler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, VersionResponse{Version: h.version})
}

// GetStatus handles GET /api/v1/bff/status.
func (h *Handler) GetStatus(c *gin.Context) {
	ctx := c.Request.Context()
	status, err := h.bffService.GetStatus(ctx)
	if err != nil {
		MapDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, ToStatusResponse(status))
}

// CreateSession handles POST /api/v1/bff/sessions.
func (h *Handler) CreateSession(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	ttl := time.Duration(req.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 1 * time.Hour
	}

	cmd := inbound.CreateSessionCommand{
		SessionID: strings.TrimSpace(req.SessionID),
		UserID:    strings.TrimSpace(req.UserID),
		TTL:       ttl,
		Metadata:  req.Metadata,
	}

	session, err := h.bffService.CreateSession(ctx, cmd)
	if err != nil {
		MapDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, ToSessionResponse(session))
}

// GetSession handles GET /api/v1/bff/sessions/:id.
func (h *Handler) GetSession(c *gin.Context) {
	ctx := c.Request.Context()
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "session ID is required"})
		return
	}

	log := zerolog.Ctx(ctx).With().Str("session_id", id).Logger()
	log.Debug().Msg("handling session retrieval request")

	session, err := h.bffService.GetSession(ctx, model.SessionID(id))
	if err != nil {
		MapDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, ToSessionResponse(session))
}
