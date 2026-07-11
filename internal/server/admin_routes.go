package server

import (
	"github.com/archaditya/bytevault/internal/handler"
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
)

func (s *Server) registerAdminRoutes(
	protected *Group,
	adminHandler *handler.AdminHandler,
) {
	admin := protected.Group("/admin")

	// Stats
	admin.GET("/stats", adminHandler.GetStats, appMiddleware.RequirePermission("admin:users"))

	// Users Management
	admin.GET("/users", adminHandler.ListUsers, appMiddleware.RequirePermission("admin:users"))
	admin.GET("/users/:id", adminHandler.GetUserDetail, appMiddleware.RequirePermission("admin:users"))
	admin.PUT("/users/:id", adminHandler.UpdateUser, appMiddleware.RequirePermission("admin:users"))
	admin.DELETE("/users/:id", adminHandler.DeleteUser, appMiddleware.RequirePermission("admin:users"))

	// Roles & Activity logs
	admin.GET("/roles", adminHandler.ListRoles, appMiddleware.RequirePermission("admin:roles"))
	admin.GET("/activity", adminHandler.ListActivity, appMiddleware.RequirePermission("admin:activity"))

	// File Inspection
	admin.GET("/files", adminHandler.ListAllFiles, appMiddleware.RequirePermission("admin:users"))
	admin.GET("/files/shared", adminHandler.ListSharedFiles, appMiddleware.RequirePermission("admin:users"))
}
