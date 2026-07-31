package server

import (
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

	// Wrap storage provider with the environment name prefix to isolate local and prod files
 	store = storage.NewPrefixedStorageProvider(s.config.App.Env, store)

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
	authProviderRepo := repository.NewAuthProviderRepository(s.db)
	authService := service.NewAuthService(userRepo, sessionRepo, roleRepo, activityRepo, authProviderRepo, notifService, s.config.JWT)
	fileService := service.NewFileService(fileRepo, userRepo, store, s.config.Storage.Provider, s.config.Storage.R2Bucket, redisQueue)
	folderService := service.NewFolderService(folderRepo, fileRepo)
	contactService := service.NewContactService(contactRepo, emailClient)

	// 5. Initialize Handlers
	fileHandler := handler.NewFileHandler(fileService, s.config.Storage.LocalDir)
	folderHandler := handler.NewFolderHandler(folderService)
	notifHandler := handler.NewNotificationHandler(authService, notifService)
	contactHandler := handler.NewContactHandler(contactService)
	adminHandler := handler.NewAdminHandler(userRepo, roleRepo, sessionRepo, activityRepo, fileRepo)

	// 6. Setup Route Groups
	v1 := s.echo.Group("/api/v1")

	// Public routes
	s.registerHealthRoutes(v1)
	s.registerAuthRoutes(v1, authService, notifHandler, userRepo)

	// Protected routes (JWT required)
	authMiddleware := appMiddleware.Auth(authService)
	protected := v1.Group("", authMiddleware)

	// Delegate Route Groupings
	s.registerUserRoutes(v1, protected, userRepo, deviceRepo, sessionRepo, fileRepo, store)
	s.registerFolderRoutes(v1,protected, folderHandler)
	s.registerNotificationRoutes(protected, notifHandler)
	s.registerAdminRoutes(protected, adminHandler)
	s.registerContactRoutes(v1, protected, contactHandler)
	s.registerFileRoutes(v1, fileHandler, authMiddleware)

	// 7. Start Background Workers and Scheduler
	if redisQueue != nil {
		wp, err := worker.NewWorkerPool(s.config, redisQueue, emailClient, userRepo, notifRepo, pushTokenRepo)
		if err != nil {
			logger.Log.Error().Err(err).Msg("Failed to start notification workers")
		} else {
			wp.Start()
		}

		// Start 2 concurrent Media Processing Workers (Images, Videos, PDFs)
		mediaWorker := worker.NewMediaWorker(fileRepo, store, redisQueue)
		mediaWorker.Start(2)
	}

	bgScheduler := scheduler.NewScheduler(verifyRepo, notifRepo, fileRepo, userRepo, store)
	bgScheduler.Start()
}

type Group = echo.Group
