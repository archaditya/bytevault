package server

import (
	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/handler"
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
)

func (s *Server) registerEphemeralRoutes(v1 *echo.Group, protected *echo.Group, eh *handler.EphemeralHandler) {
	ephemeral := v1.Group("/ephemeral")
	ephemeral.GET("/config", eh.GetConfig)
	ephemeral.POST("/upload-session", eh.CreateUploadSession)
	ephemeral.POST("/multipart-session", eh.CreateMultipartSession)
	ephemeral.POST("/refresh-part-urls/:token", eh.RefreshPartURLs)
	ephemeral.POST("/complete-multipart/:token", eh.CompleteMultipartSession)
	ephemeral.POST("/abort-multipart/:token", eh.AbortMultipartSession)
	ephemeral.POST("/progress/:token", eh.UpdateProgress)
	ephemeral.GET("/metadata/:token", eh.GetMetadata)
	ephemeral.POST("/download/:token", eh.Download)

	// Admin telemetry section & live rules control (Requires admin permission)
	protected.GET("/admin/ephemeral-logs", eh.ListAdminLogs, appMiddleware.RequirePermission("admin:users"))
	protected.POST("/admin/ephemeral-config", eh.UpdateAdminConfig, appMiddleware.RequirePermission("admin:users"))
}
