package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
)

// ConfigHandler exposes HTTP endpoints for managing attached TestSuite scenario Configurations.
type ConfigHandler struct {
	configsUC inbound.ConfigsUseCase
}

// NewConfigHandler constructs a new ConfigHandler.
func NewConfigHandler(configsUC inbound.ConfigsUseCase) *ConfigHandler {
	return &ConfigHandler{
		configsUC: configsUC,
	}
}

// CreateConfig handles POST /api/v1/suites/:id/configs to upload and attach a configuration.
func (h *ConfigHandler) CreateConfig(c *gin.Context) {
	suiteID := c.Param("id")
	var req CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	cmd := inbound.CreateConfigCommand{
		SuiteID:     suiteID,
		Name:        req.Name,
		ContentYAML: req.ContentYAML,
		IsDefault:   req.IsDefault,
	}

	config, err := h.configsUC.CreateConfig(c.Request.Context(), cmd)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, ToConfigResponse(config))
}

// GetConfig handles GET /api/v1/suites/:id/configs/:configId to retrieve an attached configuration.
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	suiteID := c.Param("id")
	configID := c.Param("configId")

	config, err := h.configsUC.GetConfig(c.Request.Context(), suiteID, configID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ToConfigResponse(config))
}

// ListConfigs handles GET /api/v1/suites/:id/configs to list all configurations for a suite.
func (h *ConfigHandler) ListConfigs(c *gin.Context) {
	suiteID := c.Param("id")

	configs, err := h.configsUC.ListConfigs(c.Request.Context(), suiteID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ToConfigListResponse(configs))
}

// DeleteConfig handles DELETE /api/v1/suites/:id/configs/:configId to delete an attached configuration.
func (h *ConfigHandler) DeleteConfig(c *gin.Context) {
	suiteID := c.Param("id")
	configID := c.Param("configId")

	if err := h.configsUC.DeleteConfig(c.Request.Context(), suiteID, configID); err != nil {
		HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
