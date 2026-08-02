package server

import (
	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/handler"
)

func (s *Server) registerFolderRoutes(v1 *echo.Group, protected *echo.Group, fh *handler.FolderHandler) {
	// Public route for folder sharing
	v1.GET("/folders/public/:id", fh.GetPublicFolder)

	// Protected routes
	foldersGroup := protected.Group("/folders")
	{
		foldersGroup.POST("", fh.Create)
		foldersGroup.GET("", fh.List)
		foldersGroup.PUT("/:id/move", fh.Move)
		foldersGroup.PUT("/:id/rename", fh.Rename)
		foldersGroup.PATCH("/:id/share", fh.ToggleShare)
		foldersGroup.DELETE("/:id", fh.Delete)
	}
}
