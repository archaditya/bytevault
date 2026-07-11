package server

import (
	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/handler"
)

// registerFileRoutes configures endpoints related to file management.
func (s *Server) registerFileRoutes(g *echo.Group, fh *handler.FileHandler, authMiddleware echo.MiddlewareFunc) {
	// Public routes
	g.GET("/files/public/:id", fh.DownloadPublic)
	g.HEAD("/files/public/:id", fh.DownloadPublic)
	g.GET("/files/public/:id/metadata", fh.GetPublicMetadata)
	
	// Local storage direct uploads dev endpoints
	g.PUT("/files/upload/direct", fh.UploadLocalDirect)
	g.GET("/files/download/direct", fh.DownloadLocalDirect)

	// Protected routes
	filesGroup := g.Group("/files", authMiddleware)
	{
		filesGroup.POST("/upload-session", fh.CreateUploadSession)
		filesGroup.POST("/:id/complete", fh.CompleteUpload)
		filesGroup.POST("/multipart-session", fh.CreateMultipartSession)
		filesGroup.POST("/:id/complete-multipart", fh.CompleteMultipartSession)
		filesGroup.POST("/:id/abort-multipart", fh.AbortMultipartSession)
		filesGroup.POST("/upload", fh.Upload) // Legacy route
		filesGroup.GET("", fh.List)
		filesGroup.GET("/:id/download", fh.Download)
		filesGroup.PATCH("/:id/share", fh.ToggleShare)
		filesGroup.PUT("/:id/move", fh.Move)
		filesGroup.DELETE("/:id", fh.Delete)
		filesGroup.GET("/:id", fh.GetDetails)
	}
}
