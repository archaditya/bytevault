package server

import (
	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/handler"
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
	"github.com/archaditya/bytevault/internal/repository"
)

func (s *Server) registerFolderRoutes(v1 *echo.Group, protected *echo.Group, fh *handler.FolderHandler, userRepo *repository.UserRepository) {
	restrictCheck := appMiddleware.RestrictionCheck(userRepo)

	// Public route for folder sharing
	v1.GET("/folders/public/:id", fh.GetPublicFolder)
	v1.GET("/folders/public/:id/download", fh.DownloadPublicFolderZip)

	// Save shared public folder directly into user's vault
	protected.POST("/folders/public/:id/save-to-vault", fh.SavePublicFolderToVault, restrictCheck)

	// Protected routes
	foldersGroup := protected.Group("/folders")
	{
		foldersGroup.POST("", fh.Create, restrictCheck)
		foldersGroup.GET("", fh.List)
		foldersGroup.PUT("/:id/move", fh.Move)
		foldersGroup.PUT("/:id/rename", fh.Rename)
		foldersGroup.PATCH("/:id/share", fh.ToggleShare, restrictCheck)
		foldersGroup.DELETE("/:id", fh.Delete)
	}
}
