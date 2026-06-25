package server

import (
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
	"github.com/archaditya/bytevault/internal/logger"
	"github.com/archaditya/bytevault/internal/storage"
	"github.com/archaditya/bytevault/internal/storage/local"
	"github.com/archaditya/bytevault/internal/storage/cloudinary"
	"github.com/archaditya/bytevault/internal/storage/r2"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
	"github.com/archaditya/bytevault/internal/handler"
	"github.com/labstack/echo/v4"
)

// registerRoutes wires all route groups together.
// As the app grows, each feature gets its own registerXxxRoutes() function.
func (s *Server) registerRoutes() {
	// 1. Initialize Pluggable Storage Provider
	var store storage.StorageProvider
	var err error

	switch s.config.Storage.Provider {
	case "r2":
		store, err = r2.NewR2Storage(
			s.config.Storage.R2Endpoint,
			s.config.Storage.R2AccessKeyID,
			s.config.Storage.R2SecretAccessKey,
			s.config.Storage.R2Bucket,
		)
	case "cloudinary":
		store, err = cloudinary.NewCloudinaryStorage(s.config.Storage.CloudinaryURL)
	default:
		store, err = local.NewLocalStorage(s.config.Storage.LocalDir)
	}

	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to initialize storage provider")
	}

	// 2. Initialize Repositories
	userRepo := repository.NewUserRepository(s.db)
	sessionRepo := repository.NewSessionRepository(s.db)
	roleRepo := repository.NewRoleRepository(s.db)
	activityRepo := repository.NewActivityRepository(s.db)
	deviceRepo := repository.NewDeviceRepository(s.db)
	fileRepo := repository.NewFileRepository(s.db)

	// 3. Initialize Services
	authService := service.NewAuthService(userRepo, sessionRepo, roleRepo, activityRepo, s.config.JWT)
	fileService := service.NewFileService(fileRepo, store, s.config.Storage.Provider, s.config.Storage.R2Bucket)

	// 4. Initialize Handlers
	fileHandler := handler.NewFileHandler(fileService)

	// 5. Setup Route Groups
	v1 := s.echo.Group("/api/v1")

	// Public routes
	s.registerHealthRoutes(v1)
	s.registerAuthRoutes(v1, authService)

	// Protected routes (JWT required)
	authMiddleware := appMiddleware.Auth(authService)
	protected := v1.Group("", authMiddleware)
	s.registerUserRoutes(protected, userRepo, deviceRepo, sessionRepo)

	// Admin routes (JWT + admin permissions required)
	s.registerAdminRoutes(protected, userRepo, roleRepo, sessionRepo, activityRepo)

	// File endpoints (registers private /upload and public download routes)
	s.registerFileRoutes(v1, fileHandler, authMiddleware)
}

// Helper type so we can pass Echo groups cleanly
type Group = echo.Group