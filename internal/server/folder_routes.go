package server

import (
	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/handler"
)

func (s *Server) registerFolderRoutes(protected *echo.Group, fh *handler.FolderHandler) {
	foldersGroup := protected.Group("/folders")
	{
		foldersGroup.POST("", fh.Create)
		foldersGroup.GET("", fh.List)
		foldersGroup.PUT("/:id/move", fh.Move)
		foldersGroup.PUT("/:id/rename", fh.Rename)
		foldersGroup.DELETE("/:id", fh.Delete)
	}
}
