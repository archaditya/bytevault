package server

import (
	"io"
	"net/http"
	"strings"

	"github.com/archaditya/bytevault/internal/handler"
	"github.com/archaditya/bytevault/internal/logger"
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
	"github.com/archaditya/bytevault/internal/storage"
	"github.com/archaditya/bytevault/internal/storage/cloudinary"
	"github.com/archaditya/bytevault/internal/storage/local"
	"github.com/archaditya/bytevault/internal/storage/r2"

	"github.com/archaditya/bytevault/internal/notification/email"
	"github.com/archaditya/bytevault/internal/notification/queue"
	"github.com/archaditya/bytevault/internal/notification/scheduler"
	"github.com/archaditya/bytevault/internal/notification/worker"

	"github.com/labstack/echo/v4"
)

// registerRoutes wires all route groups together.
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

	// 2. Initialize Redis Queue (must happen before services that depend on it)
	redisQueue, err := queue.NewRedisQueue(s.config.Redis)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to initialize Redis Queue. Notification features will be unavailable.")
	}

	// 3. Initialize Repositories
	userRepo := repository.NewUserRepository(s.db)
	sessionRepo := repository.NewSessionRepository(s.db)
	roleRepo := repository.NewRoleRepository(s.db)
	activityRepo := repository.NewActivityRepository(s.db)
	deviceRepo := repository.NewDeviceRepository(s.db)
	fileRepo := repository.NewFileRepository(s.db)
	folderRepo := repository.NewFolderRepository(s.db)
	verifyRepo := repository.NewEmailVerificationRepository(s.db)
	notifRepo := repository.NewNotificationRepository(s.db)
	pushTokenRepo := repository.NewPushTokenRepository(s.db)
	contactRepo := repository.NewContactRepository(s.db)

	// 4. Initialize Services
	emailClient := email.NewBrevoClient(s.config.Notification.Brevo)
	notifService := service.NewNotificationService(redisQueue, notifRepo, verifyRepo, pushTokenRepo, userRepo)
	authService := service.NewAuthService(userRepo, sessionRepo, roleRepo, activityRepo, notifService, s.config.JWT)
	fileService := service.NewFileService(fileRepo, store, s.config.Storage.Provider, s.config.Storage.R2Bucket)
	folderService := service.NewFolderService(folderRepo, fileRepo)
	contactService := service.NewContactService(contactRepo, emailClient)

	// 5. Initialize Handlers
	fileHandler := handler.NewFileHandler(fileService, s.config.Storage.LocalDir)
	folderHandler := handler.NewFolderHandler(folderService)
	notifHandler := handler.NewNotificationHandler(authService, notifService)
	contactHandler := handler.NewContactHandler(contactService)

	// 6. Setup Route Groups
	v1 := s.echo.Group("/api/v1")

	// Public routes
	s.registerHealthRoutes(v1)
	s.registerAuthRoutes(v1, authService, notifHandler, userRepo)
	v1.POST("/contact", contactHandler.Submit)

	// Public user avatar endpoint proxy
	v1.GET("/users/:id/avatar", func(c echo.Context) error {
		userID := c.Param("id")
		user, err := userRepo.FindByID(c.Request().Context(), userID)
		if err != nil || user.AvatarURL == nil || *user.AvatarURL == "" {
			return c.Redirect(http.StatusFound, "https://www.gravatar.com/avatar/?d=mp")
		}

		// If it's a Google OAuth avatar or another external URL, redirect directly
		if strings.HasPrefix(*user.AvatarURL, "http") {
			return c.Redirect(http.StatusFound, *user.AvatarURL)
		}

		// Otherwise download the file securely
		stream, err := store.Download(c.Request().Context(), *user.AvatarURL)
		if err != nil {
			return c.Redirect(http.StatusFound, "https://www.gravatar.com/avatar/?d=mp")
		}
		defer stream.Close()

		// Detect standard image content types
		contentType := "image/jpeg"
		lowerURL := strings.ToLower(*user.AvatarURL)
		if strings.HasSuffix(lowerURL, ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(lowerURL, ".webp") {
			contentType = "image/webp"
		} else if strings.HasSuffix(lowerURL, ".gif") {
			contentType = "image/gif"
		}

		c.Response().Header().Set(echo.HeaderContentType, contentType)
		c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=86400") // Cache locally for 1 day
		c.Response().WriteHeader(http.StatusOK)
		_, err = io.Copy(c.Response().Writer, stream)
		return err
	})

	// Protected routes (JWT required)
	authMiddleware := appMiddleware.Auth(authService)
	protected := v1.Group("", authMiddleware)
	s.registerUserRoutes(protected, userRepo, deviceRepo, sessionRepo, fileRepo, store)
	s.registerFolderRoutes(protected, folderHandler)
	s.registerNotificationRoutes(protected, notifHandler)

	// Admin routes (JWT + admin permissions required)
	s.registerAdminRoutes(protected, userRepo, roleRepo, sessionRepo, activityRepo, fileRepo)

	// Admin contact queries routes
	adminGroup := protected.Group("/admin")
	adminGroup.GET("/contact-queries", contactHandler.List, appMiddleware.RequirePermission("admin:users"))
	adminGroup.POST("/contact-queries/:id/reply", contactHandler.Reply, appMiddleware.RequirePermission("admin:users"))

	// File endpoints
	s.registerFileRoutes(v1, fileHandler, authMiddleware)

	// 7. Start Background Workers and Scheduler
	if redisQueue != nil {
		wp, err := worker.NewWorkerPool(s.config, redisQueue, emailClient, userRepo, notifRepo, pushTokenRepo)
		if err != nil {
			logger.Log.Error().Err(err).Msg("Failed to start notification workers")
		} else {
			wp.Start()
		}
	}

	bgScheduler := scheduler.NewScheduler(verifyRepo, notifRepo, fileRepo, userRepo, store)
	bgScheduler.Start()
}

type Group = echo.Group
