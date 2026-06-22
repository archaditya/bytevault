package server

import (
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
	"github.com/labstack/echo/v4"
)

// registerRoutes wires all route groups together.
// As the app grows, each feature gets its own registerXxxRoutes() function.
func (s *Server) registerRoutes() {
	// Repositories
	userRepo := repository.NewUserRepository(s.db)
	sessionRepo := repository.NewSessionRepository(s.db)
	roleRepo := repository.NewRoleRepository(s.db)
	activityRepo := repository.NewActivityRepository(s.db)
	deviceRepo := repository.NewDeviceRepository(s.db)

	// Services
	authService := service.NewAuthService(userRepo, sessionRepo, roleRepo, activityRepo, s.config.JWT)

	v1 := s.echo.Group("/api/v1")

	// Public routes
	s.registerHealthRoutes(v1)
	s.registerAuthRoutes(v1, authService)

	// Protected routes (JWT required)
	protected := v1.Group("", appMiddleware.Auth(authService))
	s.registerUserRoutes(protected, userRepo, deviceRepo, sessionRepo)

	// Admin routes (JWT + admin permissions required)
	s.registerAdminRoutes(protected, userRepo, roleRepo, sessionRepo, activityRepo)
}

// Helper type so we can pass Echo groups cleanly
type Group = echo.Group