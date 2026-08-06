package worker

import (
	"context"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"

	"github.com/archaditya/bytevault/internal/config"
	"github.com/archaditya/bytevault/internal/logger"
	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/notification/email"
	"github.com/archaditya/bytevault/internal/notification/queue"
	"github.com/archaditya/bytevault/internal/repository"
)

type WorkerPool struct {
	queue         *queue.RedisQueue
	emailClient   *email.BrevoClient
	fcmClient     *messaging.Client
	userRepo      *repository.UserRepository
	notifRepo     *repository.NotificationRepository
	pushTokenRepo *repository.PushTokenRepository
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewWorkerPool(
	cfg *config.Config,
	q *queue.RedisQueue,
	emailClient *email.BrevoClient,
	userRepo *repository.UserRepository,
	notifRepo *repository.NotificationRepository,
	pushTokenRepo *repository.PushTokenRepository,
) (*WorkerPool, error) {
	ctx, cancel := context.WithCancel(context.Background())

	var fcmClient *messaging.Client
	if cfg.Notification.Firebase.CredentialsFile != "" {
		opt := option.WithCredentialsFile(cfg.Notification.Firebase.CredentialsFile)
		app, err := firebase.NewApp(ctx, &firebase.Config{
			ProjectID: "personal-project-933",
		}, opt)
		if err == nil {
			messagingClient, err := app.Messaging(ctx)
			if err == nil {
				fcmClient = messagingClient
				logger.Log.Info().Msg("Firebase Admin initialized successfully")
			} else {
				logger.Log.Warn().Err(err).Msg("Failed to initialize Firebase Messaging client")
			}
		} else {
			logger.Log.Warn().Err(err).Msg("Failed to initialize Firebase App")
		}
	}

	return &WorkerPool{
		queue:         q,
		emailClient:   emailClient,
		fcmClient:     fcmClient,
		userRepo:      userRepo,
		notifRepo:     notifRepo,
		pushTokenRepo: pushTokenRepo,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Start spawns multiple concurrent worker goroutines.
func (wp *WorkerPool) Start() {
	// Start 2 Email workers
	for i := 1; i <= 2; i++ {
		go wp.runEmailWorker(i)
	}

	// Start 2 Push notification workers
	for i := 1; i <= 2; i++ {
		go wp.runPushWorker(i)
	}

	// Start 2 DB Writer workers for in-app notifications
	for i := 1; i <= 2; i++ {
		go wp.runDBWriterWorker(i)
	}

	logger.Log.Info().Msg("Notification workers started successfully")
}

// Stop gracefully cancels workers.
func (wp *WorkerPool) Stop() {
	wp.cancel()
}

func (wp *WorkerPool) runEmailWorker(id int) {
	logger.Log.Info().Int("worker_id", id).Msg("Email worker started")
	for {
		select {
		case <-wp.ctx.Done():
			return
		default:
			job, err := wp.queue.DequeueWithPriority(wp.ctx, queue.QueueEmail, 5*time.Second)
			if err != nil {
				logger.Log.Error().Err(err).Int("worker_id", id).Msg("Error dequeuing email job")
				time.Sleep(1 * time.Second)
				continue
			}
			if job == nil {
				continue
			}

			// Process job
			toEmail, _ := job.Payload["to_email"].(string)
			toName, _ := job.Payload["to_name"].(string)
			otp, _ := job.Payload["otp"].(string)
			purpose, _ := job.Payload["purpose"].(string)

			var sendErr error
			if purpose == "registration" {
				sendErr = wp.emailClient.SendOTP(wp.ctx, toEmail, toName, otp)
			} else if purpose == "password_reset" {
				sendErr = wp.emailClient.SendPasswordReset(wp.ctx, toEmail, toName, otp)
			} else {
				subject, _ := job.Payload["subject"].(string)
				htmlBody, _ := job.Payload["html_body"].(string)
				sendErr = wp.emailClient.SendGeneric(wp.ctx, toEmail, toName, subject, htmlBody)
			}

			if sendErr != nil {
				logger.Log.Error().Err(sendErr).Str("job_id", job.ID).Msg("Email worker failed to send email")
				wp.retryJob(job, sendErr)
			} else {
				job.DeliveryStatus = queue.DeliveryDelivered
				logger.Log.Info().Str("job_id", job.ID).Msg("Email delivered successfully")
			}

		}
	}
}

func (wp *WorkerPool) runPushWorker(id int) {
	logger.Log.Info().Int("worker_id", id).Msg("Push worker started")
	for {
		select {
		case <-wp.ctx.Done():
			return
		default:
			job, err := wp.queue.DequeueWithPriority(wp.ctx, queue.QueuePush, 5*time.Second)
			if err != nil {
				logger.Log.Error().Err(err).Int("worker_id", id).Msg("Error dequeuing push job")
				time.Sleep(1 * time.Second)
				continue
			}
			if job == nil {
				continue
			}

			if wp.fcmClient == nil {
				logger.Log.Warn().Str("job_id", job.ID).Msg("Firebase Messaging not initialized, skipping push job")
				continue
			}

			title, _ := job.Payload["title"].(string)
			body, _ := job.Payload["body"].(string)

			// Get all active push tokens for user
			tokens, err := wp.pushTokenRepo.GetActiveByUser(wp.ctx, job.UserID)
			if err != nil {
				logger.Log.Error().Err(err).Str("user_id", job.UserID).Msg("Failed to query user push tokens")
				continue
			}

			for _, t := range tokens {
				message := &messaging.Message{
					Token: t.Token,
					Notification: &messaging.Notification{
						Title: title,
						Body:  body,
					},
				}
				_, sendErr := wp.fcmClient.Send(wp.ctx, message)
				if sendErr != nil {
					logger.Log.Warn().Err(sendErr).Str("token", t.Token).Msg("Failed to send push, deactivating token")
					wp.pushTokenRepo.Deactivate(wp.ctx, t.Token)
				}
			}
		}
	}
}

func (wp *WorkerPool) runDBWriterWorker(id int) {
	logger.Log.Info().Int("worker_id", id).Msg("DB Writer worker started")
	for {
		select {
		case <-wp.ctx.Done():
			return
		default:
			job, err := wp.queue.DequeueWithPriority(wp.ctx, queue.QueueInApp, 5*time.Second)
			if err != nil {
				logger.Log.Error().Err(err).Int("worker_id", id).Msg("Error dequeuing DB write job")
				time.Sleep(1 * time.Second)
				continue
			}
			if job == nil {
				continue
			}

			title, _ := job.Payload["title"].(string)
			body, _ := job.Payload["body"].(string)
			jobType, _ := job.Payload["type"].(string)
			var metadata map[string]any
			if m, ok := job.Payload["metadata"].(map[string]any); ok {
				metadata = m
			}

			notif := &model.Notification{
				UserID:   job.UserID,
				Type:     jobType,
				Title:    title,
				Body:     body,
				Metadata: metadata,
				Channel:  "in_app",
			}

			if err := wp.notifRepo.Create(wp.ctx, notif); err != nil {
				logger.Log.Error().Err(err).Str("job_id", job.ID).Msg("Failed to save in-app notification to database")
			}
		}
	}
}

func (wp *WorkerPool) retryJob(job *queue.Job, lastErr error) {
	job.LastError = lastErr.Error()
	if job.Retries >= job.MaxRetries {
		job.DeliveryStatus = queue.DeliveryFailed
		logger.Log.Warn().
			Str("job_id", job.ID).
			Str("type", job.Type).
			Int("retries", job.Retries).
			Str("last_error", job.LastError).
			Msg("Job failed permanently after max retries")
		return
	}

	job.Retries++
	// Exponential backoff: 2s, 4s, 8s
	delay := time.Duration(1<<uint(job.Retries)) * 2 * time.Second
	logger.Log.Info().
		Str("job_id", job.ID).
		Int("retry", job.Retries).
		Dur("delay", delay).
		Msg("Retrying job")

	time.AfterFunc(delay, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		wp.queue.Enqueue(ctx, job)
	})
}
