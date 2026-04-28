package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HealthHandler holds dependencies for health-related endpoints.
// Empty now, but later we'll add DB, services, etc.
type HealthHandler struct{}

// NewHealthHandler is a CONSTRUCTOR function.
// Go doesn't have constructors, so convention is New<TypeName>()
func NewHealthHandler() *HealthHandler {
	return  &HealthHandler{}
}

// Health handles GET /api/v1/health
// (h *HealthHandler) = this method belongs to HealthHandler
// echo.Context = gives you request data, response helpers
// Returns error (nil = success)
func (h *HealthHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status": "healthy",
		"service": "Bytevault",
	})
}