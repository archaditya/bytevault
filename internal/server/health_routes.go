package server

import (
	"github.com/archaditya/bytevault/internal/handler"
)

// registerHealthRoutes adds system/infra routes.
// These are always public — no auth needed.
func (s *Server) registerHealthRoutes(v1 *Group) {
	healthHandler := handler.NewHealthHandler()

	v1.GET("/health", healthHandler.Health)
}
