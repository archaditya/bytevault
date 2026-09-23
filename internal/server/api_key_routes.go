package server

import (
	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/handler"
)

func (s *Server) registerAPIKeyRoutes(protected *echo.Group, h *handler.APIKeyHandler) {
	group := protected.Group("/api-keys")
	group.POST("", h.Create)
	group.GET("", h.List)
	group.POST("/:id/rotate", h.Rotate)
	group.DELETE("/:id", h.Delete)
}
