package server

import (
	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/handler"
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
	"github.com/archaditya/bytevault/internal/repository"
)

// registerFileRoutes configures endpoints related to file management.
func (s *Server) registerFileRoutes(g *echo.Group, fh *handler.FileHandler, authMiddleware echo.MiddlewareFunc, userRepo *repository.UserRepository) {
	// Public routes
	g.GET("/files/public/:id", fh.DownloadPublic)
	g.HEAD("/files/public/:id", fh.DownloadPublic)
	g.GET("/files/public/:id/metadata", fh.GetPublicMetadata)
	
	// Local storage direct uploads dev endpoints
	g.PUT("/files/upload/direct", fh.UploadLocalDirect)
	g.GET("/files/download/direct", fh.DownloadLocalDirect)

	// Protected routes (JWT required)
	filesGroup := g.Group("/files", authMiddleware)
	{
		// Read-only routes (allowed for restricted users)
		filesGroup.GET("", fh.List)
		filesGroup.GET("/:id/download", fh.Download)
		filesGroup.GET("/:id", fh.GetDetails)
		filesGroup.GET("/:id/thumbnail", fh.GetThumbnail)

		// Write routes (blocked for restricted users via RestrictionCheck middleware)
		restrictCheck := appMiddleware.RestrictionCheck(userRepo)
		filesGroup.POST("/upload-session", fh.CreateUploadSession, restrictCheck)
		filesGroup.POST("/:id/complete", fh.CompleteUpload, restrictCheck)
		filesGroup.POST("/multipart-session", fh.CreateMultipartSession, restrictCheck)
		filesGroup.POST("/:id/complete-multipart", fh.CompleteMultipartSession, restrictCheck)
		filesGroup.POST("/:id/abort-multipart", fh.AbortMultipartSession)
		filesGroup.POST("/:id/refresh-part-urls", fh.RefreshPartURLs)
		filesGroup.POST("/upload", fh.Upload, restrictCheck) // Legacy route
		filesGroup.PATCH("/:id/share", fh.ToggleShare, restrictCheck)
		filesGroup.PUT("/:id/move", fh.Move)
		filesGroup.PUT("/:id/rename", fh.Rename)
		filesGroup.DELETE("/:id", fh.Delete)
	}
}
