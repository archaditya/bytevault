package server

import (
	"github.com/archaditya/bytevault/internal/handler"
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
)

// registerHealthRoutes adds system/infra routes.
// /health is public for orchestrators and uptime monitors.
// /health/deep exposes DB pool and internal telemetry, requiring admin:telemetry permission.
func (s *Server) registerHealthRoutes(v1 *Group, protected *Group) {
	healthHandler := handler.NewHealthHandler(s.db)

	v1.GET("/health", healthHandler.Health)
	protected.GET("/health/deep", healthHandler.DeepHealth, appMiddleware.RequirePermission("admin:telemetry"))
}
