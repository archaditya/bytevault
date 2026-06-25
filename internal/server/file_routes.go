package server

import (
	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/handler"
)

// registerFileRoutes configures endpoints related to file management.
func (s *Server) registerFileRoutes(g *echo.Group, fh *handler.FileHandler, authMiddleware echo.MiddlewareFunc) {
	// Public routes
	g.GET("/files/public/:id", fh.DownloadPublic)
	
	// Local storage direct uploads dev endpoints
	g.PUT("/files/upload/direct", fh.UploadLocalDirect)
	g.GET("/files/download/direct", fh.DownloadLocalDirect)

	// Protected routes
	filesGroup := g.Group("/files", authMiddleware)
	{
		filesGroup.POST("/upload-session", fh.CreateUploadSession)
		filesGroup.POST("/:id/complete", fh.CompleteUpload)
		filesGroup.POST("/upload", fh.Upload) // Keep as fallback legacy route
		filesGroup.GET("", fh.List)
		filesGroup.GET("/:id/download", fh.Download)
		filesGroup.PATCH("/:id/share", fh.ToggleShare)
		filesGroup.DELETE("/:id", fh.Delete)
	}
}
