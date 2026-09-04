package server

import (
	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/handler"
)

func (s *Server) registerEphemeralRoutes(v1 *echo.Group, protected *echo.Group, eh *handler.EphemeralHandler) {
	ephemeral := v1.Group("/ephemeral")
	ephemeral.GET("/config", eh.GetConfig)
	ephemeral.POST("/upload-session", eh.CreateUploadSession)
	ephemeral.POST("/multipart-session", eh.CreateMultipartSession)
	ephemeral.POST("/complete-multipart/:token", eh.CompleteMultipartSession)
	ephemeral.POST("/abort-multipart/:token", eh.AbortMultipartSession)
	ephemeral.GET("/metadata/:token", eh.GetMetadata)
	ephemeral.POST("/download/:token", eh.Download)

	// Admin telemetry section & live rules control
	protected.GET("/admin/ephemeral-logs", eh.ListAdminLogs)
	protected.POST("/admin/ephemeral-config", eh.UpdateAdminConfig)
}
