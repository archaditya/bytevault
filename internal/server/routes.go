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

	"github.com/archaditya/bytevault/internal/ai"
	"github.com/archaditya/bytevault/internal/monitoring"
	"github.com/archaditya/bytevault/internal/notification/email"
	"github.com/archaditya/bytevault/internal/notification/queue"
	"github.com/archaditya/bytevault/internal/notification/scheduler"
	"github.com/archaditya/bytevault/internal/notification/worker"
	"github.com/archaditya/bytevault/internal/razorpay"

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
	shareRepo := repository.NewShareRepository(s.db)
	ephemeralRepo := repository.NewEphemeralShareRepository(s.db)
	ephemeralSettingRepo := repository.NewEphemeralSettingRepository(s.db)
	verifyRepo := repository.NewEmailVerificationRepository(s.db)
	notifRepo := repository.NewNotificationRepository(s.db)
	contactRepo := repository.NewContactRepository(s.db)

	// Subscription Repositories
	pkgRepo := repository.NewPackageRepository(s.db)
	subRepo := repository.NewSubscriptionRepository(s.db)
	txnRepo := repository.NewTransactionRepository(s.db)
	subAuditRepo := repository.NewSubscriptionAuditRepository(s.db)
	webhookEventRepo := repository.NewWebhookEventRepository(s.db)
	systemLogRepo := repository.NewSystemLogRepository(s.db)

	// Razorpay Client & File Audit Logging
	var razorpayClient *razorpay.Client
	if s.config.Razorpay.KeyID != "" {
		razorpayClient = razorpay.NewClient(s.config.Razorpay.KeyID, s.config.Razorpay.KeySecret, s.config.Razorpay.WebhookSecret)
	}
	auditLogger, _ := monitoring.NewFileAuditLogger("logs")
	logArchiver := monitoring.NewLogArchiver("logs", store)
	logArchiver.SetTracker(systemLogRepo)

	// 4. Initialize Services
	emailClient := email.NewBrevoClient(s.config.Notification.Brevo)
	notifService := service.NewNotificationService(redisQueue, notifRepo, verifyRepo, deviceRepo, userRepo)
	authProviderRepo := repository.NewAuthProviderRepository(s.db)
	authService := service.NewAuthService(userRepo, sessionRepo, roleRepo, activityRepo, authProviderRepo, notifService, s.config.JWT)
	fileService := service.NewFileService(fileRepo, userRepo, store, s.config.Storage.Provider, s.config.Storage.R2Bucket, redisQueue, activityRepo)
	folderService := service.NewFolderService(folderRepo, fileRepo, activityRepo)
	shareService := service.NewShareService(shareRepo, userRepo, fileRepo, folderRepo, activityRepo)
	ephemeralService := service.NewEphemeralService(ephemeralRepo, ephemeralSettingRepo, store)
	contactService := service.NewContactService(contactRepo, emailClient)

	// Subscription Services
	pkgService := service.NewPackageService(pkgRepo, razorpayClient)
	txnService := service.NewTransactionService(txnRepo, subRepo, pkgRepo, emailClient)
	appURL := "https://pushpostvault.com"
	if s.config.App.Env == "development" {
		appURL = "http://localhost:3000"
	}
	emailClient.SetAppURL(appURL)
	txnService.SetAppURL(appURL)
	subService := service.NewSubscriptionService(subRepo, pkgRepo, userRepo, subAuditRepo, razorpayClient, auditLogger, notifService)
	fileService.SetSubscriptionRepo(subRepo)

	// Bandwidth & Egress Metering
	bandwidthRepo := repository.NewBandwidthRepository(s.db)
	fileService.SetBandwidthRepo(bandwidthRepo)
	ephemeralService.SetBandwidthRepo(bandwidthRepo)

	// 5. Initialize Handlers
	fileHandler := handler.NewFileHandler(fileService, s.config.Storage.LocalDir)
	folderHandler := handler.NewFolderHandler(folderService)
	notifHandler := handler.NewNotificationHandler(authService, notifService)
	contactHandler := handler.NewContactHandler(contactService)
	shareHandler := handler.NewShareHandler(shareService)
	ephemeralHandler := handler.NewEphemeralHandler(ephemeralService)
	adminHandler := handler.NewAdminHandler(userRepo, roleRepo, sessionRepo, activityRepo, fileRepo)
	moderationHandler := handler.NewModerationHandler(fileRepo, userRepo)

	// Subscription Handlers
	subHandler := handler.NewSubscriptionHandler(subService, txnService, s.config.Razorpay.KeyID)
	pkgHandler := handler.NewPackageHandler(pkgService, subService, txnService, subRepo, subAuditRepo)
	pkgHandler.SetSystemLogRepo(systemLogRepo)
	webhookHandler := handler.NewWebhookHandler(razorpayClient, subRepo, pkgRepo, userRepo, subAuditRepo, txnService, auditLogger, notifService, webhookEventRepo)
	adminHandler.SetSubscriptionDependencies(subRepo, subService)
	adminHandler.SetMonitoringDependencies(monitoring.GlobalTelemetry, bandwidthRepo, s.db)

	// Static public assets (brand logos, icons, email assets)
	s.echo.Static("/static", "public")

	// 6. Setup Route Groups
	v1 := s.echo.Group("/api/v1")

	// Public routes
	s.registerHealthRoutes(v1)

	// Protected routes (JWT required)
	authMiddleware := appMiddleware.Auth(authService)
	protected := v1.Group("", authMiddleware)

	// Auth routes (needs both v1 for public + protected for MFA)
	s.registerAuthRoutes(v1, protected, authService, notifHandler, userRepo)

	// Delegate Route Groupings
	s.registerUserRoutes(v1, protected, userRepo, deviceRepo, sessionRepo, fileRepo, store, subRepo)
	s.registerFolderRoutes(v1, protected, folderHandler)
	s.registerNotificationRoutes(protected, notifHandler)
	s.registerAdminRoutes(protected, adminHandler, moderationHandler, pkgHandler)
	s.registerContactRoutes(v1, protected, contactHandler)
	s.registerFileRoutes(v1, fileHandler, authMiddleware, userRepo)
	s.registerShareRoutes(protected, shareHandler)
	s.registerEphemeralRoutes(v1, protected, ephemeralHandler)
	s.registerModerationUserRoutes(protected, moderationHandler)
	s.registerSubscriptionRoutes(v1, protected, subHandler, pkgHandler, webhookHandler)

	// 7. Start Background Workers and Scheduler
	if redisQueue != nil {
		wp, err := worker.NewWorkerPool(s.config, redisQueue, emailClient, userRepo, notifRepo, deviceRepo)
		if err != nil {
			logger.Log.Error().Err(err).Msg("Failed to start notification workers")
		} else {
			wp.Start()
		}

		// Start 2 concurrent Media Processing Workers (Images, Videos, PDFs + AI Image Labeling + NSFW Detection)
		imageLabeler := ai.NewImageLabeler(s.config.AI)
		nsfwDetector := ai.NewNSFWDetector(s.config.AI.HFAPIToken)
		mediaWorker := worker.NewMediaWorker(fileRepo, userRepo, store, redisQueue, imageLabeler, nsfwDetector)
		mediaWorker.Start(2)
	}

	bgScheduler := scheduler.NewScheduler(verifyRepo, notifRepo, fileRepo, userRepo, store)
	bgScheduler.SetSubscriptionProcessor(subService)
	bgScheduler.SetLogArchiver(logArchiver)
	bgScheduler.SetWebhookEventRepo(webhookEventRepo)
	bgScheduler.Start()
}

type Group = echo.Group
