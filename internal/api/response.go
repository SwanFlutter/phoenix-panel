// Package api contains the Gin HTTP handlers, request/response DTOs and router.
package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/phoenix-panel/phoenix/internal/service"
)

// errorResponse is the uniform error envelope returned by the API.
type errorResponse struct {
	Error string `json:"error"`
}

// fail writes a JSON error with the given status code.
func fail(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, errorResponse{Error: msg})
}

// failErr maps a service-layer error to an appropriate HTTP status.
func failErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		fail(c, http.StatusNotFound, "resource not found")
	case errors.Is(err, service.ErrConflict):
		fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrValidation):
		fail(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		fail(c, http.StatusUnauthorized, "invalid credentials")
	default:
		fail(c, http.StatusInternalServerError, "internal server error")
	}
}
