package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// SuiteHandler exposes HTTP endpoints for managing TestSuite aggregates.
type SuiteHandler struct {
	suitesUC inbound.SuitesUseCase
}

// NewSuiteHandler constructs a new SuiteHandler.
func NewSuiteHandler(suitesUC inbound.SuitesUseCase) *SuiteHandler {
	return &SuiteHandler{
		suitesUC: suitesUC,
	}
}

// CreateSuite handles POST /api/v1/suites to register a new TestSuite.
func (h *SuiteHandler) CreateSuite(c *gin.Context) {
	var req CreateSuiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	suite, err := h.suitesUC.CreateSuite(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, ToSuiteResponse(suite))
}

// GetSuite handles GET /api/v1/suites/:id to retrieve a specific TestSuite.
func (h *SuiteHandler) GetSuite(c *gin.Context) {
	id := c.Param("id")
	suite, err := h.suitesUC.GetSuite(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ToSuiteResponse(suite))
}

// ListSuites handles GET /api/v1/suites to list all registered TestSuites.
func (h *SuiteHandler) ListSuites(c *gin.Context) {
	suites, err := h.suitesUC.ListSuites(c.Request.Context())
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ToSuiteListResponse(suites))
}

// UpdateSuite handles PUT /api/v1/suites/:id to update an existing TestSuite.
func (h *SuiteHandler) UpdateSuite(c *gin.Context) {
	id := c.Param("id")
	var req UpdateSuiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	cmd := inbound.UpdateSuiteCommand{
		Name:        req.Name,
		Description: req.Description,
	}
	if req.State != "" {
		st := model.TestSuiteState(req.State)
		if !st.IsValid() {
			HandleError(c, model.ErrInvalidStateTransition)
			return
		}
		cmd.State = &st
	}

	suite, err := h.suitesUC.UpdateSuite(c.Request.Context(), id, cmd)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ToSuiteResponse(suite))
}

// DeleteSuite handles DELETE /api/v1/suites/:id to delete a TestSuite.
func (h *SuiteHandler) DeleteSuite(c *gin.Context) {
	id := c.Param("id")
	if err := h.suitesUC.DeleteSuite(c.Request.Context(), id); err != nil {
		HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
