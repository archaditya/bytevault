package server

import (
	appMiddleware "github.com/adityakkpk/bytevault/internal/middleware"
	"github.com/adityakkpk/bytevault/internal/repository"
	"github.com/adityakkpk/bytevault/internal/service"
	"github.com/labstack/echo/v4"
)

// registerRoutes wires all route groups together.
// As the app grows, each feature gets its own registerXxxRoutes() function.
func (s *Server) registerRoutes() {
	// Build shared dependencies once, pass them down
	userRepo := repository.NewUserRepository(s.db)
	sessionRepo := repository.NewSessionRepository(s.db)
	authService := service.NewAuthService(userRepo, sessionRepo, s.config.JWT)

	v1 := s.echo.Group("/api/v1")

	// Each feature registers its own routes
	s.registerHealthRoutes(v1)
	s.registerAuthRoutes(v1, authService)

	// Protected group — all routes here require a valid JWT
	protected := v1.Group("", appMiddleware.Auth(authService))
	s.registerUserRoutes(protected, userRepo)
}

// Helper type so we can pass Echo groups cleanly
type Group = echo.Group