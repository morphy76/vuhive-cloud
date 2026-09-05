package rest

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// HandleError inspects domain errors and renders the appropriate HTTP status code and JSON error payload.
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, model.ErrNotFound),
		errors.Is(err, model.ErrBarrierNotFound),
		errors.Is(err, model.ErrReportNotFound),
		errors.Is(err, model.ErrLogsNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
	case errors.Is(err, model.ErrValidation),
		errors.Is(err, model.ErrInvalidPlatform),
		errors.Is(err, model.ErrEmptyName),
		errors.Is(err, model.ErrInvalidResourceQuantity),
		errors.Is(err, model.ErrInvalidAffinity),
		errors.Is(err, model.ErrInvalidToleration),
		errors.Is(err, model.ErrInvalidCronExpression),
		errors.Is(err, model.ErrInvalidWorkerCount),
		errors.Is(err, model.ErrInvalidStateTransition):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	case errors.Is(err, model.ErrBarrierTimeout):
		c.JSON(http.StatusRequestTimeout, ErrorResponse{Error: err.Error()})
	case errors.Is(err, model.ErrBarrierAborted):
		c.JSON(http.StatusFailedDependency, ErrorResponse{Error: err.Error()})
	case errors.Is(err, model.ErrConflict),
		errors.Is(err, model.ErrRunInFlight),
		errors.Is(err, model.ErrWorkerAlreadyRegistered),
		errors.Is(err, model.ErrBarrierReleased),
		errors.Is(err, model.ErrTerminalState):
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}
