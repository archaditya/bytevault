package server

import (
	"github.com/adityakkpk/bytevault/internal/handler"
)

func (s *Server) registerRoutes() {
	healthHandler := handler.NewHealthHandler()

	v1 := s.echo.Group("/api/v1")

	v1.GET("/health", healthHandler.Health)
}