package server

import (
	"github.com/archaditya/bytevault/internal/handler"
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
)

func (s *Server) registerContactRoutes(
	v1 *Group,
	protected *Group,
	contactHandler *handler.ContactHandler,
) {
	// Public contact query submission
	v1.POST("/contact", contactHandler.Submit)

	// Admin contact management
	admin := protected.Group("/admin")
	admin.GET("/contact-queries", contactHandler.List, appMiddleware.RequirePermission("admin:users"))
	admin.POST("/contact-queries/:id/reply", contactHandler.Reply, appMiddleware.RequirePermission("admin:users"))
}
