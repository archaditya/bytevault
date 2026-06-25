package server

import (
	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/handler"
)

// registerFileRoutes configures endpoints related to file management.
func (s *Server) registerFileRoutes(g *echo.Group, fh *handler.FileHandler, authMiddleware echo.MiddlewareFunc) {
	// Public routes
	g.GET("/files/public/:id", fh.DownloadPublic)

	// Protected routes
	filesGroup := g.Group("/files", authMiddleware)
	{
		filesGroup.POST("/upload", fh.Upload)
		filesGroup.GET("", fh.List)
		filesGroup.GET("/:id/download", fh.Download)
		filesGroup.PATCH("/:id/share", fh.ToggleShare)
		filesGroup.DELETE("/:id", fh.Delete)
	}
}
