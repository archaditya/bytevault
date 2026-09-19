package server

import (
	"github.com/archaditya/bytevault/internal/handler"
)

// registerHealthRoutes adds system/infra routes.
// These are public endpoints for orchestrators and uptime monitors.
func (s *Server) registerHealthRoutes(v1 *Group) {
	healthHandler := handler.NewHealthHandler(s.db)

	v1.GET("/health", healthHandler.Health)
	v1.GET("/health/deep", healthHandler.DeepHealth)
}
