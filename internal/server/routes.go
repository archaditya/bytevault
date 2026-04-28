// Package server — routes.go
//
// This file is responsible for registering all HTTP routes.
// We keep routes separate from server setup for clarity.
//
// ROUTING CONCEPTS:
// - e.GET("/path", handler)  → Handles GET requests to /path
// - Route groups let you prefix a set of routes (e.g., /api/v1/...)
// - Each route maps a URL pattern to a handler function
package server

import (
	"github.com/adityakkpk/bytevault/internal/handler"
)

// registerRoutes sets up all API endpoints.
// This method belongs to Server (s *Server) so it can access s.echo.
func (s *Server) registerRoutes() {
	// Create handler instances
	healthHandler := handler.NewHealthHandler()

	// ---- API v1 Routes ----
	// Group creates a route group with a common prefix.
	// All routes in this group will start with /api/v1
	v1 := s.echo.Group("/api/v1")

	// Health check endpoint
	// GET /api/v1/health → healthHandler.Health
	v1.GET("/health", healthHandler.Health)
}
