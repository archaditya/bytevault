package server

import (
	"github.com/archaditya/bytevault/internal/handler"
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
)

func (s *Server) registerAdminRoutes(
	protected *Group,
	adminHandler *handler.AdminHandler,
	moderationHandler *handler.ModerationHandler,
) {
	admin := protected.Group("/admin")

	// Stats
	admin.GET("/stats", adminHandler.GetStats, appMiddleware.RequirePermission("admin:users"))

	// Users Management
	admin.GET("/users", adminHandler.ListUsers, appMiddleware.RequirePermission("admin:users"))
	admin.GET("/users/:id", adminHandler.GetUserDetail, appMiddleware.RequirePermission("admin:users"))
	admin.GET("/users/:id/activity", adminHandler.GetUserActivityLogs, appMiddleware.RequirePermission("admin:users"))
	admin.PUT("/users/:id", adminHandler.UpdateUser, appMiddleware.RequirePermission("admin:users"))
	admin.DELETE("/users/:id", adminHandler.DeleteUser, appMiddleware.RequirePermission("admin:users"))

	// Roles & Activity logs
	admin.GET("/roles", adminHandler.ListRoles, appMiddleware.RequirePermission("admin:roles"))
	admin.GET("/activity", adminHandler.ListActivity, appMiddleware.RequirePermission("admin:activity"))

	// File Inspection
	admin.GET("/files", adminHandler.ListAllFiles, appMiddleware.RequirePermission("admin:users"))
	admin.GET("/files/shared", adminHandler.ListSharedFiles, appMiddleware.RequirePermission("admin:users"))

	// Content Moderation (NSFW detection & user restriction management)
	moderation := admin.Group("/moderation", appMiddleware.RequirePermission("admin:users"))
	{
		moderation.GET("/stats", moderationHandler.GetModerationStats)
		moderation.GET("/flagged", moderationHandler.ListFlaggedFiles)
		moderation.GET("/blocked", moderationHandler.ListBlockedFiles)
		moderation.POST("/files/:id/approve", moderationHandler.ApproveFile)
		moderation.POST("/files/:id/reject", moderationHandler.RejectFile)
		moderation.GET("/users", moderationHandler.ListRestrictedUsers)
		moderation.POST("/users/:id/restrict", moderationHandler.RestrictUser)
		moderation.POST("/users/:id/unrestrict", moderationHandler.UnrestrictUser)
		moderation.GET("/appeals", moderationHandler.ListAppeals)
		moderation.POST("/appeals/:id/approve", moderationHandler.ApproveAppeal)
		moderation.POST("/appeals/:id/reject", moderationHandler.RejectAppeal)
	}
}
