package scheduler

import (
	"context"
	"time"

	"github.com/archaditya/bytevault/internal/logger"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/storage"
)

type SubscriptionProcessor interface {
	ProcessScheduledDowngrades(ctx context.Context) error
}

type LogArchiver interface {
	ArchivePastLogs(ctx context.Context) error
	PurgeExpiredLogs(ctx context.Context) error
}

type UploadInviteProcessor interface {
	ExpireStaleInvites(ctx context.Context) (int64, error)
	FlushPendingNotifications(ctx context.Context) error
	CleanupAbandonedUploads(ctx context.Context) error
}

type Scheduler struct {
	verifyRepo       *repository.EmailVerificationRepository
	notifRepo        *repository.NotificationRepository
	fileRepo         *repository.FileRepository
	userRepo         *repository.UserRepository
	store            storage.StorageProvider
	subProcessor     SubscriptionProcessor
	logArchiver      LogArchiver
	webhookEventRepo *repository.WebhookEventRepository
	inviteProcessor  UploadInviteProcessor
	ticker           *time.Ticker
	inviteTicker     *time.Ticker
	ctx              context.Context
	cancel           context.CancelFunc
}

func NewScheduler(
	verifyRepo *repository.EmailVerificationRepository,
	notifRepo *repository.NotificationRepository,
	fileRepo *repository.FileRepository,
	userRepo *repository.UserRepository,
	store storage.StorageProvider,
) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		verifyRepo: verifyRepo,
		notifRepo:  notifRepo,
		fileRepo:   fileRepo,
		userRepo:   userRepo,
		store:      store,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (s *Scheduler) Start() {
	// FIX #10: Run maintenance hourly, with an initial run after 15s warmup instead of 24h delay
	s.ticker = time.NewTicker(1 * time.Hour)
	go func() {
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(15 * time.Second):
			s.cleanup()
		}

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-s.ticker.C:
				s.cleanup()
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	if s.inviteTicker != nil {
		s.inviteTicker.Stop()
	}
	s.cancel()
}

func (s *Scheduler) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	logger.Log.Info().Msg("Scheduler running periodic cleanup tasks...")

	// 1. Clean expired OTP codes (older than 2 hours)
	otpBefore := time.Now().Add(-2 * time.Hour)
	otpDeleted, err := s.verifyRepo.CleanupExpired(ctx, otpBefore)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to clean expired OTPs")
	} else if otpDeleted > 0 {
		logger.Log.Info().Int64("deleted", otpDeleted).Msg("Cleaned expired OTPs")
	}

	// 2. Clean notifications older than 30 days
	notifBefore := time.Now().Add(-30 * 24 * time.Hour)
	notifDeleted, err := s.notifRepo.CleanupOld(ctx, notifBefore)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to clean old notifications")
	} else if notifDeleted > 0 {
		logger.Log.Info().Int64("deleted", notifDeleted).Msg("Cleaned old notifications")
	}

	// 3. Purge soft-deleted files older than 30 days → remove from R2 + hard delete row
	fileCutoff := time.Now().Add(-30 * 24 * time.Hour)
	s.purgeStorageKeys("soft-deleted files", func() ([]string, error) {
		return s.fileRepo.PurgeSoftDeletedBefore(ctx, fileCutoff)
	}, ctx)

	// 4. Purge files of soft-deleted users older than 30 days → remove from R2, keep user row
	s.purgeStorageKeys("deleted-user files", func() ([]string, error) {
		return s.userRepo.PurgeDeletedUserFiles(ctx, fileCutoff)
	}, ctx)

	// 5. Purge orphan files and avatar images from storage
	s.cleanupOrphans(ctx)
}

func (s *Scheduler) purgeStorageKeys(label string, fetchKeys func() ([]string, error), ctx context.Context) {
	keys, err := fetchKeys()
	if err != nil {
		logger.Log.Error().Err(err).Str("task", label).Msg("Failed to purge from database")
		return
	}
	if len(keys) == 0 {
		return
	}
	logger.Log.Info().Int("count", len(keys)).Str("task", label).Msg("Purging storage objects")
	for _, key := range keys {
		if delErr := s.store.Delete(ctx, key); delErr != nil {
			logger.Log.Error().Err(delErr).Str("key", key).Msg("Failed to delete from storage")
		}
	}
}

func (s *Scheduler) cleanupOrphans(ctx context.Context) {
	logger.Log.Info().Msg("Scheduler running orphan file and image cleanup...")

	// 1. Cleanup orphan avatars
	activeAvatars, err := s.userRepo.GetAllAvatarURLs(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to retrieve active avatar URLs from database")
	} else {
		avatarKeys, err := s.store.List(ctx, "avatars/")
		if err != nil {
			logger.Log.Error().Err(err).Msg("Failed to list avatar objects from storage")
		} else {
			var orphans []string
			for _, key := range avatarKeys {
				if !activeAvatars[key] {
					orphans = append(orphans, key)
				}
			}
			if len(orphans) > 0 {
				logger.Log.Info().Int("count", len(orphans)).Msg("Purging orphan avatars from storage")
				for _, key := range orphans {
					logger.Log.Info().Str("key", key).Msg("Deleting orphan avatar from storage")
					if delErr := s.store.Delete(ctx, key); delErr != nil {
						logger.Log.Error().Err(delErr).Str("key", key).Msg("Failed to delete orphan avatar from storage")
					}
				}
			}
		}
	}

	// 2. Cleanup orphan files
	activeFileKeys, err := s.fileRepo.GetAllStorageKeys(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to retrieve active file keys from database")
	} else {
		fileKeys, err := s.store.List(ctx, "user/")
		if err != nil {
			logger.Log.Error().Err(err).Msg("Failed to list file objects from storage")
		} else {
			var orphans []string
			for _, key := range fileKeys {
				if !activeFileKeys[key] {
					orphans = append(orphans, key)
				}
			}
			if len(orphans) > 0 {
				logger.Log.Info().Int("count", len(orphans)).Msg("Purging orphan files from storage")
				for _, key := range orphans {
					logger.Log.Info().Str("key", key).Msg("Deleting orphan file from storage")
					if delErr := s.store.Delete(ctx, key); delErr != nil {
						logger.Log.Error().Err(delErr).Str("key", key).Msg("Failed to delete orphan file from storage")
					}
				}
			}
		}
	}

	// 3. Process subscription downgrades and cycle expirations
	if s.subProcessor != nil {
		logger.Log.Info().Msg("Processing scheduled subscription downgrades and expirations")
		if err := s.subProcessor.ProcessScheduledDowngrades(ctx); err != nil {
			logger.Log.Error().Err(err).Msg("Failed to process scheduled subscription downgrades")
		}
	}

	// 4. Archive closed daily audit logs to cloud storage and purge logs older than 6 months
	if s.logArchiver != nil {
		logger.Log.Info().Msg("Archiving past audit logs to storage")
		if err := s.logArchiver.ArchivePastLogs(ctx); err != nil {
			logger.Log.Error().Err(err).Msg("Failed to archive audit logs")
		}
		if err := s.logArchiver.PurgeExpiredLogs(ctx); err != nil {
			logger.Log.Error().Err(err).Msg("Failed to purge expired logs from R2")
		}
	}

	// 5. Cleanup processed webhook events older than 7 days
	if s.webhookEventRepo != nil {
		webhookCutoff := time.Now().Add(-7 * 24 * time.Hour)
		deletedEvents, err := s.webhookEventRepo.CleanupOlderThan(ctx, webhookCutoff)
		if err != nil {
			logger.Log.Error().Err(err).Msg("Failed to cleanup old webhook events")
		} else if deletedEvents > 0 {
			logger.Log.Info().Int64("deleted", deletedEvents).Msg("Cleaned old webhook idempotency events")
		}
	}

	// 6. Expire stale upload invites and cleanup abandoned guest uploads
	if s.inviteProcessor != nil {
		expired, err := s.inviteProcessor.ExpireStaleInvites(ctx)
		if err != nil {
			logger.Log.Error().Err(err).Msg("Failed to expire stale upload invites")
		} else if expired > 0 {
			logger.Log.Info().Int64("expired", expired).Msg("Expired stale upload invites")
		}

		if err := s.inviteProcessor.CleanupAbandonedUploads(ctx); err != nil {
			logger.Log.Error().Err(err).Msg("Failed to cleanup abandoned invite uploads")
		}
	}
}

func (s *Scheduler) SetSubscriptionProcessor(p SubscriptionProcessor) {
	s.subProcessor = p
}

func (s *Scheduler) SetLogArchiver(a LogArchiver) {
	s.logArchiver = a
}

func (s *Scheduler) SetWebhookEventRepo(r *repository.WebhookEventRepository) {
	s.webhookEventRepo = r
}

func (s *Scheduler) SetUploadInviteProcessor(p UploadInviteProcessor) {
	s.inviteProcessor = p

	// Start a dedicated 10-minute ticker for batched notification flushing.
	// The hourly cleanup handles invite expiry + abandoned upload cleanup,
	// but notifications need a tighter loop so the owner learns about uploads quickly.
	s.inviteTicker = time.NewTicker(10 * time.Minute)
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-s.inviteTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := s.inviteProcessor.FlushPendingNotifications(ctx); err != nil {
					logger.Log.Error().Err(err).Msg("Failed to flush upload invite notifications")
				}
				cancel()
			}
		}
	}()
}
