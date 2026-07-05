package scheduler

import (
	"context"
	"time"

	"github.com/archaditya/bytevault/internal/logger"
	"github.com/archaditya/bytevault/internal/repository"
)

type Scheduler struct {
	verifyRepo *repository.EmailVerificationRepository
	notifRepo  *repository.NotificationRepository
	ticker     *time.Ticker
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewScheduler(
	verifyRepo *repository.EmailVerificationRepository,
	notifRepo *repository.NotificationRepository,
) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		verifyRepo: verifyRepo,
		notifRepo:  notifRepo,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start initiates the background scheduler.
func (s *Scheduler) Start() {
	s.ticker = time.NewTicker(6 * time.Hour) // Run cleanup every 6 hours
	go func() {
		// Run immediate initial cleanup on start
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

// Stop shuts down the scheduler.
func (s *Scheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.cancel()
}

func (s *Scheduler) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Minute)
	defer cancel()

	logger.Log.Info().Msg("Scheduler running periodic database cleanup tasks...")

	// 1. Clean expired OTP codes (older than 2 hours)
	otpBefore := time.Now().Add(-2 * time.Hour)
	otpDeleted, err := s.verifyRepo.CleanupExpired(ctx, otpBefore)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to clean up expired OTP verification codes")
	} else if otpDeleted > 0 {
		logger.Log.Info().Int64("rows_deleted", otpDeleted).Msg("Cleaned up expired OTP verification codes")
	}

	// 2. Clean notifications older than 30 days
	notifBefore := time.Now().Add(-30 * 24 * time.Hour)
	notifDeleted, err := s.notifRepo.CleanupOld(ctx, notifBefore)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to clean up old notifications")
	} else if notifDeleted > 0 {
		logger.Log.Info().Int64("rows_deleted", notifDeleted).Msg("Cleaned up old notifications")
	}
}
