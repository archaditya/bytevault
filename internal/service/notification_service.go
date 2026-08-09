package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/notification/queue"
	"github.com/archaditya/bytevault/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type NotificationService struct {
	queue      *queue.RedisQueue
	notifRepo  *repository.NotificationRepository
	verifyRepo *repository.EmailVerificationRepository
	deviceRepo *repository.DeviceRepository
	userRepo   *repository.UserRepository
}

func NewNotificationService(
	q *queue.RedisQueue,
	notifRepo *repository.NotificationRepository,
	verifyRepo *repository.EmailVerificationRepository,
	deviceRepo *repository.DeviceRepository,
	userRepo *repository.UserRepository,
) *NotificationService {
	return &NotificationService{
		queue:      q,
		notifRepo:  notifRepo,
		verifyRepo: verifyRepo,
		deviceRepo: deviceRepo,
		userRepo:   userRepo,
	}
}

// GenerateAndSendOTP creates a 6-digit code and queues the verification email job.
func (s *NotificationService) GenerateAndSendOTP(ctx context.Context, userID, email, firstName, purpose string) error {
	// Generate random 6-digit numeric OTP code
	otpCode := ""
	for i := 0; i < 6; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(10))
		otpCode += num.String()
	}

	ttl := 10 * time.Minute

	// Primary: Store in Redis (fast, auto-expires)
	if s.queue != nil {
		if err := s.queue.StoreOTP(ctx, email, purpose, otpCode, ttl); err != nil {
			// Redis failed — fall through to DB
			fmt.Printf("Redis OTP store failed, falling back to DB: %v\n", err)
		}
	}

	// Fallback: Always persist to DB for durability
	verification := &model.EmailVerification{
		UserID:      userID,
		Email:       email,
		OTPCode:     otpCode,
		Purpose:     purpose,
		MaxAttempts: 5,
		ExpiresAt:   time.Now().Add(ttl),
	}
	if err := s.verifyRepo.Create(ctx, verification); err != nil {
		return fmt.Errorf("failed to save OTP code: %w", err)
	}

	// Enqueue email job (high priority — OTP emails must be fast)
	job := &queue.Job{
		ID:        fmt.Sprintf("job_email_%d", time.Now().UnixNano()),
		Type:      queue.JobTypeEmail,
		UserID:    userID,
		Priority:  queue.PriorityHigh,
		CreatedAt: time.Now(),
		Payload: map[string]any{
			"to_email": email,
			"to_name":  firstName,
			"otp":      otpCode,
			"purpose":  purpose,
		},
	}

	return s.queue.Enqueue(ctx, job)
}

// VerifyOTP verifies the code, updating attempts or invalidating it on success.
func (s *NotificationService) VerifyOTP(ctx context.Context, email, otp, purpose string) error {
	// Primary: Try Redis first (fast path)
	if s.queue != nil {
		err := s.queue.VerifyOTP(ctx, email, purpose, otp, 5)
		if err == nil {
			// Redis verified — also mark the DB record as used and verify user
			go func() {
				bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				v, dbErr := s.verifyRepo.FindLatestByEmail(bgCtx, email, purpose)
				if dbErr == nil {
					s.verifyRepo.MarkUsed(bgCtx, v.ID)
				}
				if purpose == "email_verification" || purpose == "registration" {
					s.userRepo.MarkVerified(bgCtx, s.getUserIDByEmail(bgCtx, email))
				}
			}()
			return nil
		}
		// If Redis says "not found" — fall through to DB
		// If Redis says "invalid code" or "max attempts" — return that error
		if err.Error() != "OTP not found or expired" {
			return err
		}
	}

	// Fallback: Verify from DB
	v, err := s.verifyRepo.FindLatestByEmail(ctx, email, purpose)
	if err != nil {
		return err
	}

	if time.Now().After(v.ExpiresAt) {
		return fmt.Errorf("verification code has expired")
	}

	if v.Attempts >= v.MaxAttempts {
		return fmt.Errorf("maximum validation attempts exceeded")
	}

	if v.OTPCode != otp {
		s.verifyRepo.IncrementAttempts(ctx, v.ID)
		return fmt.Errorf("invalid verification code")
	}

	if err := s.verifyRepo.MarkUsed(ctx, v.ID); err != nil {
		return err
	}

	if purpose == "email_verification" || purpose == "registration" {
		return s.userRepo.MarkVerified(ctx, v.UserID)
	}
	return nil
}

// getUserIDByEmail is a helper to resolve userID from email for async operations.
func (s *NotificationService) getUserIDByEmail(ctx context.Context, email string) string {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return ""
	}
	return user.ID
}

// QueueInAppNotification pushes a job to record and broadcast an in-app message.
func (s *NotificationService) QueueInAppNotification(ctx context.Context, userID, notifType, title, body string, metadata map[string]any) error {
	job := &queue.Job{
		ID:        fmt.Sprintf("job_inapp_%d", time.Now().UnixNano()),
		Type:      queue.JobTypeInApp,
		UserID:    userID,
		Priority:  queue.PriorityNormal,
		CreatedAt: time.Now(),
		Payload: map[string]any{
			"type":     notifType,
			"title":    title,
			"body":     body,
			"metadata": metadata,
		},
	}
	return s.queue.Enqueue(ctx, job)
}

// RegisterPushToken registers/updates a Firebase FCM token in the user_devices table.
func (s *NotificationService) RegisterPushToken(ctx context.Context, userID, fcmToken, deviceType string) error {
	device := &model.UserDevice{
		UserID:     userID,
		FcmToken:   fcmToken,
		DeviceType: deviceType,
	}
	return s.deviceRepo.Upsert(ctx, device)
}

// ListInApp paginates user notifications.
func (s *NotificationService) ListInApp(ctx context.Context, userID string, limit, offset int) ([]model.Notification, int, error) {
	return s.notifRepo.ListByUser(ctx, userID, limit, offset)
}

// MarkRead marks a user notification as read.
func (s *NotificationService) MarkRead(ctx context.Context, notifID, userID string) error {
	return s.notifRepo.MarkAsRead(ctx, notifID, userID)
}

// MarkAllRead marks all notifications as read.
func (s *NotificationService) MarkAllRead(ctx context.Context, userID string) error {
	return s.notifRepo.MarkAllAsRead(ctx, userID)
}

// GetUserByEmail returns a user by email address.
func (s *NotificationService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return s.userRepo.FindByEmail(ctx, email)
}

// VerifyOTPLoginDetails fetches a verified user's details with role info for auto-login after OTP verification.
func (s *NotificationService) VerifyOTPLoginDetails(ctx context.Context, email string) (*model.User, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to find verified user: %w", err)
	}
	return user, nil
}

// GetUnreadCount returns the count of unread notifications for a user.
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	return s.notifRepo.UnreadCount(ctx, userID)
}

// ForgotPassword handles requesting a password reset OTP.
func (s *NotificationService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("user with this email does not exist")
	}

	firstName := ""
	if user.FirstName != nil {
		firstName = *user.FirstName
	}

	return s.GenerateAndSendOTP(ctx, user.ID, user.Email, firstName, "password_reset")
}

// ResetPassword verifies the OTP and updates the user's password.
func (s *NotificationService) ResetPassword(ctx context.Context, email, otp, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	// Verify OTP
	if err := s.VerifyOTP(ctx, email, otp, "password_reset"); err != nil {
		return err
	}

	// Hash password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), 14)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Find user to get ID
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return err
	}

	// Update password
	return s.userRepo.UpdatePassword(ctx, user.ID, string(hashedBytes))
}

// ListAllNotifications paginates all system notifications for administrators.
func (s *NotificationService) ListAllNotifications(ctx context.Context, limit, offset int) ([]model.Notification, int, error) {
	return s.notifRepo.ListAll(ctx, limit, offset)
}

// SendAdminNotification broadcasts or targets a notification through specified channels.
func (s *NotificationService) SendAdminNotification(
	ctx context.Context,
	targetType string,
	targetUserID string,
	targetRole string,
	title string,
	body string,
	channels []string,
	priority string,
) (int, []string, error) {
	// 1. Resolve targeted users
	var userIDs []string
	var err error

	switch targetType {
	case "single":
		if targetUserID == "" {
			return 0, nil, fmt.Errorf("target user_id is required for single target type")
		}
		userIDs = append(userIDs, targetUserID)
	case "role":
		if targetRole == "" {
			return 0, nil, fmt.Errorf("target role is required for role target type")
		}
		userIDs, err = s.userRepo.FindUserIDsByRole(ctx, targetRole)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to fetch users by role: %w", err)
		}
	case "global":
		userIDs, err = s.userRepo.ListAllActiveIDs(ctx)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to fetch all users: %w", err)
		}
	default:
		return 0, nil, fmt.Errorf("invalid target type: %s", targetType)
	}

	if len(userIDs) == 0 {
		return 0, nil, nil
	}

	// 2. Queue notifications for each user & channel
	var notificationIDs []string
	sentCount := 0

	for _, uID := range userIDs {
		u, err := s.userRepo.FindByID(ctx, uID)
		if err != nil {
			continue // skip if user not found or deleted
		}

		firstName := ""
		if u.FirstName != nil {
			firstName = *u.FirstName
		}

		for _, ch := range channels {
			jobID := fmt.Sprintf("job_admin_%s_%d", ch, time.Now().UnixNano())
			notificationIDs = append(notificationIDs, jobID)

			switch ch {
			case "in_app":
				job := &queue.Job{
					ID:        jobID,
					Type:      queue.JobTypeInApp,
					UserID:    uID,
					Priority:  priority,
					CreatedAt: time.Now(),
					Payload: map[string]any{
						"type":  "admin.notification",
						"title": title,
						"body":  body,
						"metadata": map[string]any{
							"priority": priority,
							"sender":   "admin",
						},
					},
				}
				if s.queue != nil {
					_ = s.queue.Enqueue(ctx, job)
				}
			case "push":
				job := &queue.Job{
					ID:        jobID,
					Type:      queue.JobTypePush,
					UserID:    uID,
					Priority:  priority,
					CreatedAt: time.Now(),
					Payload: map[string]any{
						"title": title,
						"body":  body,
					},
				}
				if s.queue != nil {
					_ = s.queue.Enqueue(ctx, job)
				}
			case "email":
				job := &queue.Job{
					ID:        jobID,
					Type:      queue.JobTypeEmail,
					UserID:    uID,
					Priority:  priority,
					CreatedAt: time.Now(),
					Payload: map[string]any{
						"to_email":  u.Email,
						"to_name":   firstName,
						"subject":   title,
						"html_body": fmt.Sprintf("<p>%s</p>", body),
					},
				}
				if s.queue != nil {
					_ = s.queue.Enqueue(ctx, job)
				}
			}
		}
		sentCount++
	}

	return sentCount, notificationIDs, nil
}
