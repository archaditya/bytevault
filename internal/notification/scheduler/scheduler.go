package scheduler

import (
	"context"
	"time"

	"github.com/archaditya/bytevault/internal/logger"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/storage"
)

type Scheduler struct {
	verifyRepo *repository.EmailVerificationRepository
	notifRepo  *repository.NotificationRepository
	fileRepo   *repository.FileRepository
	userRepo   *repository.UserRepository
	store      storage.StorageProvider
	ticker     *time.Ticker
	ctx        context.Context
	cancel     context.CancelFunc
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
	s.ticker = time.NewTicker(6 * time.Hour)
	go func() {
		s.cleanup()
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
					if delErr := s.store.Delete(ctx, key); delErr != nil {
						logger.Log.Error().Err(delErr).Str("key", key).Msg("Failed to delete orphan file from storage")
					}
				}
			}
		}
	}
}
