package server

import (
	"github.com/archaditya/bytevault/internal/handler"
)

// registerModerationUserRoutes adds user-facing moderation endpoints (appeal, restriction status).
func (s *Server) registerModerationUserRoutes(
	protected *Group,
	moderationHandler *handler.ModerationHandler,
) {
	moderation := protected.Group("/moderation")
	{
		moderation.GET("/status", moderationHandler.GetRestrictionStatus)
		moderation.POST("/appeal", moderationHandler.SubmitAppeal)
	}
}
