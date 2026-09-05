package rest

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
)

// MapDomainError inspects a domain error and translates it into an appropriate HTTP response.
func MapDomainError(c *gin.Context, err error) {
	if errors.Is(err, model.ErrSessionNotFound) {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}
	if errors.Is(err, model.ErrInvalidParameter) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if errors.Is(err, model.ErrUnauthorized) {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}
	if errors.Is(err, model.ErrControlPlaneUnavailable) {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
}
