package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
)

// SecretHandler exposes HTTP endpoints for managing suite-scoped Secrets.
type SecretHandler struct {
	secretsUC inbound.SecretsUseCase
}

// NewSecretHandler constructs a new SecretHandler.
func NewSecretHandler(secretsUC inbound.SecretsUseCase) *SecretHandler {
	return &SecretHandler{
		secretsUC: secretsUC,
	}
}

// CreateSecret handles POST /api/v1/suites/:id/secrets to create a new suite-scoped secret.
func (h *SecretHandler) CreateSecret(c *gin.Context) {
	suiteID := c.Param("id")
	var req CreateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	cmd := inbound.CreateSecretCommand{
		SuiteID: suiteID,
		Key:     req.Key,
		Value:   req.Value,
	}

	secret, err := h.secretsUC.CreateSecret(c.Request.Context(), cmd)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, ToSecretResponse(secret))
}

// ListSecrets handles GET /api/v1/suites/:id/secrets to list all secrets for a suite.
func (h *SecretHandler) ListSecrets(c *gin.Context) {
	suiteID := c.Param("id")

	secrets, err := h.secretsUC.ListSecrets(c.Request.Context(), suiteID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ToSecretListResponse(secrets))
}

// UpdateSecret handles PUT /api/v1/suites/:id/secrets/:secretId to update a secret's value.
func (h *SecretHandler) UpdateSecret(c *gin.Context) {
	suiteID := c.Param("id")
	secretID := c.Param("secretId")

	var req UpdateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	cmd := inbound.UpdateSecretCommand{
		SuiteID:  suiteID,
		SecretID: secretID,
		Value:    req.Value,
	}

	secret, err := h.secretsUC.UpdateSecret(c.Request.Context(), cmd)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ToSecretResponse(secret))
}

// DeleteSecret handles DELETE /api/v1/suites/:id/secrets/:secretId to delete a suite-scoped secret.
func (h *SecretHandler) DeleteSecret(c *gin.Context) {
	suiteID := c.Param("id")
	secretID := c.Param("secretId")

	if err := h.secretsUC.DeleteSecret(c.Request.Context(), suiteID, secretID); err != nil {
		HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
