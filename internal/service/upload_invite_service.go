package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/notification/queue"
	"github.com/archaditya/bytevault/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// Invite defaults and caps. All configurable via env in future paid tiers.
const (
	DefaultInviteMaxBytes   = int64(2) * 1024 * 1024 * 1024 // 2 GB
	DefaultInviteMaxFiles   = 20
	DefaultInviteExpiryH    = 168 // 7 days
	MaxInviteMaxBytes       = int64(5) * 1024 * 1024 * 1024 // 5 GB
	MaxInviteMaxFiles       = 50
	MaxInviteExpiryH        = 720 // 30 days
	MaxActiveInvitesPerUser = 10
	// IP rate limits: per-session/byte caps, not per-file
	MaxGuestUploadsPerIP24h = 50
	InviteSessionTTL        = 1 * time.Hour
	PasscodeMaxAttempts     = 5
)

// BlockedGuestExtensions are file extensions blocked for guest uploads to prevent executable abuse.
var BlockedGuestExtensions = map[string]bool{
	".exe": true, ".bat": true, ".cmd": true, ".msi": true,
	".sh": true, ".ps1": true, ".dll": true, ".scr": true,
	".com": true, ".vbs": true, ".wsf": true, ".jar": true,
	".cpl": true, ".pif": true, ".hta": true, ".reg": true,
	".apk": true, ".ipa": true,
}

// folderNameSanitizer strips characters unsafe for folder names.
var folderNameSanitizer = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

type UploadInviteService struct {
	inviteRepo   *repository.UploadInviteRepository
	folderRepo   *repository.FolderRepository
	fileService  *FileService
	notifService *NotificationService
	userRepo     *repository.UserRepository
	redisQueue   *queue.RedisQueue
}

func NewUploadInviteService(
	inviteRepo *repository.UploadInviteRepository,
	folderRepo *repository.FolderRepository,
	fileService *FileService,
	notifService *NotificationService,
	userRepo *repository.UserRepository,
	redisQueue *queue.RedisQueue,
) *UploadInviteService {
	return &UploadInviteService{
		inviteRepo:   inviteRepo,
		folderRepo:   folderRepo,
		fileService:  fileService,
		notifService: notifService,
		userRepo:     userRepo,
		redisQueue:   redisQueue,
	}
}

// sanitizeFolderName cleans a label for use as a folder name.
func sanitizeFolderName(name string) string {
	name = strings.TrimSpace(name)
	name = folderNameSanitizer.ReplaceAllString(name, "_")
	// Collapse multiple underscores/spaces
	name = regexp.MustCompile(`[_\s]+`).ReplaceAllString(name, " ")
	name = strings.TrimSpace(name)
	// Truncate by rune count, not byte count, to avoid splitting multi-byte characters
	runes := []rune(name)
	if len(runes) > 100 {
		name = string(runes[:100])
	}
	return name
}

// ensureReceivedFolder gets or creates the root "Received" folder for the owner.
// If the folder was soft-deleted, it is re-created.
func (s *UploadInviteService) ensureReceivedFolder(ctx context.Context, ownerID string) (*model.Folder, error) {
	folders, err := s.folderRepo.ListByUserID(ctx, ownerID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list root folders: %w", err)
	}
	for _, f := range folders {
		if f.Name == "Received" {
			return f, nil
		}
	}
	// Create "Received" folder
	folder := &model.Folder{
		UserID:   ownerID,
		Name:     "Received",
		ParentID: nil,
		IsPublic: false,
	}
	if err := s.folderRepo.Create(ctx, folder); err != nil {
		return nil, fmt.Errorf("failed to create Received folder: %w", err)
	}
	return folder, nil
}

// ensureSubFolder gets or creates a sub-folder under the parent with the given name.
// Handles duplicate names by appending (2), (3), etc.
func (s *UploadInviteService) ensureSubFolder(ctx context.Context, ownerID, parentID, name string) (*model.Folder, error) {
	subFolders, err := s.folderRepo.ListByUserID(ctx, ownerID, &parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sub-folders: %w", err)
	}

	// Check if exact name exists
	for _, f := range subFolders {
		if f.Name == name {
			return f, nil
		}
	}

	// Name doesn't exist — check for duplicates and create with unique suffix if needed
	existingNames := make(map[string]bool)
	for _, f := range subFolders {
		existingNames[f.Name] = true
	}

	finalName := name
	if existingNames[finalName] {
		for i := 2; i <= 100; i++ {
			candidate := fmt.Sprintf("%s (%d)", name, i)
			if !existingNames[candidate] {
				finalName = candidate
				break
			}
		}
	}

	folder := &model.Folder{
		UserID:   ownerID,
		Name:     finalName,
		ParentID: &parentID,
		IsPublic: false,
	}
	if err := s.folderRepo.Create(ctx, folder); err != nil {
		return nil, fmt.Errorf("failed to create sub-folder: %w", err)
	}
	return folder, nil
}

// CreateInvite creates a new upload invite and its target folder.
func (s *UploadInviteService) CreateInvite(ctx context.Context, ownerID, label string, maxTotalBytes int64, maxFiles int, expiryHours int, passcode string) (*model.UploadInvite, error) {
	// Verify owner is a verified user
	owner, err := s.userRepo.FindByID(ctx, ownerID)
	if err != nil || owner == nil {
		return nil, fmt.Errorf("user not found")
	}
	if !owner.IsVerified {
		return nil, fmt.Errorf("only verified users can create upload invites")
	}

	// Check active invite cap
	activeCount, err := s.inviteRepo.CountActiveByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active invites: %w", err)
	}
	if activeCount >= MaxActiveInvitesPerUser {
		return nil, fmt.Errorf("maximum of %d active invites reached", MaxActiveInvitesPerUser)
	}

	// Apply defaults and clamp to max allowed
	if maxTotalBytes <= 0 {
		maxTotalBytes = DefaultInviteMaxBytes
	}
	if maxTotalBytes > MaxInviteMaxBytes {
		maxTotalBytes = MaxInviteMaxBytes
	}
	if maxFiles <= 0 {
		maxFiles = DefaultInviteMaxFiles
	}
	if maxFiles > MaxInviteMaxFiles {
		maxFiles = MaxInviteMaxFiles
	}
	if expiryHours <= 0 {
		expiryHours = DefaultInviteExpiryH
	}
	if expiryHours > MaxInviteExpiryH {
		expiryHours = MaxInviteExpiryH
	}

	// Sanitize label
	label = sanitizeFolderName(label)

	// Generate cryptographic token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate invite token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Create folder: Received / <label>
	receivedFolder, err := s.ensureReceivedFolder(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	subFolderName := label
	if subFolderName == "" {
		subFolderName = fmt.Sprintf("Upload %s", token[:8])
	}

	subFolder, err := s.ensureSubFolder(ctx, ownerID, receivedFolder.ID, subFolderName)
	if err != nil {
		return nil, err
	}

	// Hash passcode if provided
	var passcodeHash *string
	if passcode != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(passcode), 10)
		if err != nil {
			return nil, fmt.Errorf("failed to hash passcode: %w", err)
		}
		h := string(hash)
		passcodeHash = &h
	}

	invite := &model.UploadInvite{
		Token:          token,
		OwnerID:        ownerID,
		TargetFolderID: &subFolder.ID,
		Label:          label,
		MaxTotalBytes:  maxTotalBytes,
		MaxFiles:       maxFiles,
		PasscodeHash:   passcodeHash,
		Status:         "active",
		ExpiresAt:      time.Now().Add(time.Duration(expiryHours) * time.Hour),
	}

	if err := s.inviteRepo.Create(ctx, invite); err != nil {
		return nil, fmt.Errorf("failed to create invite: %w", err)
	}

	invite.HasPasscode = passcodeHash != nil
	return invite, nil
}

// ListInvites returns paginated invites for the owner's dashboard.
func (s *UploadInviteService) ListInvites(ctx context.Context, ownerID string, limit, offset int) ([]model.UploadInvite, int, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.inviteRepo.ListByOwner(ctx, ownerID, limit, offset)
}

// RevokeInvite marks an invite as revoked. Only the owner can revoke.
func (s *UploadInviteService) RevokeInvite(ctx context.Context, inviteID, ownerID string) error {
	invite, err := s.inviteRepo.FindByID(ctx, inviteID)
	if err != nil {
		return fmt.Errorf("failed to find invite: %w", err)
	}
	if invite == nil || invite.OwnerID != ownerID {
		return fmt.Errorf("invite not found or unauthorized")
	}
	if invite.Status != "active" {
		return fmt.Errorf("invite is already %s", invite.Status)
	}
	return s.inviteRepo.UpdateStatus(ctx, inviteID, "revoked")
}

// GetInviteInfo returns public-safe invite details for the guest page.
func (s *UploadInviteService) GetInviteInfo(ctx context.Context, token string) (*model.UploadInvite, error) {
	invite, err := s.inviteRepo.FindByToken(ctx, token)
	if err != nil || invite == nil {
		return nil, fmt.Errorf("invite not found")
	}

	// Auto-expire if past due
	if invite.Status == "active" && time.Now().After(invite.ExpiresAt) {
		_ = s.inviteRepo.UpdateStatus(ctx, invite.ID, "expired")
		invite.Status = "expired"
	}

	return invite, nil
}

// VerifyPasscode checks the passcode and returns a session token on success.
func (s *UploadInviteService) VerifyPasscode(ctx context.Context, token, passcode, ip string) (string, error) {
	invite, err := s.inviteRepo.FindByToken(ctx, token)
	if err != nil || invite == nil {
		return "", fmt.Errorf("invite not found")
	}
	if invite.Status != "active" {
		return "", fmt.Errorf("invite is %s", invite.Status)
	}
	if time.Now().After(invite.ExpiresAt) {
		_ = s.inviteRepo.UpdateStatus(ctx, invite.ID, "expired")
		return "", fmt.Errorf("invite has expired")
	}
	if !invite.HasPasscode {
		return "", fmt.Errorf("this invite does not require a passcode")
	}

	// Rate limit: 5 attempts per IP per 10 minutes (fail closed on Redis errors)
	if s.redisQueue != nil && ip != "" {
		attempts, err := s.redisQueue.CheckPasscodeRateLimit(ctx, ip, token)
		if err != nil {
			return "", fmt.Errorf("unable to verify rate limit, please try again")
		}
		if attempts > int64(PasscodeMaxAttempts) {
			return "", fmt.Errorf("too many passcode attempts, please try again later")
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*invite.PasscodeHash), []byte(passcode)); err != nil {
		return "", fmt.Errorf("incorrect passcode")
	}

	// Generate session token
	sessionBytes := make([]byte, 32)
	if _, err := rand.Read(sessionBytes); err != nil {
		return "", fmt.Errorf("failed to generate session token")
	}
	sessionToken := hex.EncodeToString(sessionBytes)

	// Store in Redis with 1-hour TTL
	if s.redisQueue != nil {
		if err := s.redisQueue.StoreInviteSession(ctx, sessionToken, invite.ID, InviteSessionTTL); err != nil {
			return "", fmt.Errorf("failed to store session: %w", err)
		}
	}

	return sessionToken, nil
}

// validateGuestSession checks that a session token is valid for the given invite.
// For invites without passcode, session token is not required.
func (s *UploadInviteService) validateGuestSession(ctx context.Context, invite *model.UploadInvite, sessionToken string) error {
	if !invite.HasPasscode {
		return nil // No passcode = no session needed
	}
	if sessionToken == "" {
		return fmt.Errorf("session token required for passcode-protected invites")
	}
	if s.redisQueue == nil {
		return fmt.Errorf("session validation unavailable")
	}
	inviteID, err := s.redisQueue.GetInviteSession(ctx, sessionToken)
	if err != nil {
		return fmt.Errorf("failed to validate session: %w", err)
	}
	if inviteID == "" {
		return fmt.Errorf("session expired or invalid, please verify the passcode again")
	}
	if inviteID != invite.ID {
		return fmt.Errorf("session does not match this invite")
	}
	return nil
}

// GuestCreateUploadSession validates all constraints and creates a presigned upload session.
func (s *UploadInviteService) GuestCreateUploadSession(
	ctx context.Context,
	token, filename string,
	size int64,
	contentType, sessionToken, ip, userAgent string,
) (*model.File, string, error) {
	// 1. Resolve and validate invite
	invite, err := s.inviteRepo.FindByToken(ctx, token)
	if err != nil || invite == nil {
		return nil, "", fmt.Errorf("invite not found")
	}
	if invite.Status != "active" {
		return nil, "", fmt.Errorf("this invite is no longer active")
	}
	if time.Now().After(invite.ExpiresAt) {
		_ = s.inviteRepo.UpdateStatus(ctx, invite.ID, "expired")
		return nil, "", fmt.Errorf("this invite has expired")
	}

	// 2. Validate session (passcode invites)
	if err := s.validateGuestSession(ctx, invite, sessionToken); err != nil {
		return nil, "", err
	}

	// 3. Block dangerous file extensions
	ext := strings.ToLower(filepath.Ext(filename))
	if BlockedGuestExtensions[ext] {
		return nil, "", fmt.Errorf("file type %s is not allowed for security reasons", ext)
	}

	// 4. IP rate limit (50 uploads per IP per 24h across all invites)
	if ip != "" {
		count, err := s.inviteRepo.CountUploadsByIPInWindow(ctx, ip)
		if err == nil && count >= MaxGuestUploadsPerIP24h {
			return nil, "", fmt.Errorf("upload rate limit reached, please try again later")
		}
	}

	// 5. Ensure target folder still exists (re-create if owner deleted it)
	if invite.TargetFolderID != nil {
		folder, err := s.folderRepo.FindByID(ctx, *invite.TargetFolderID)
		if err != nil || folder == nil {
			// Folder was deleted — re-create the path
			receivedFolder, err := s.ensureReceivedFolder(ctx, invite.OwnerID)
			if err != nil {
				return nil, "", fmt.Errorf("failed to restore target folder: %w", err)
			}
			subName := invite.Label
			if subName == "" {
				subName = fmt.Sprintf("Upload %s", invite.Token[:8])
			}
			subFolder, err := s.ensureSubFolder(ctx, invite.OwnerID, receivedFolder.ID, subName)
			if err != nil {
				return nil, "", fmt.Errorf("failed to restore sub-folder: %w", err)
			}
			invite.TargetFolderID = &subFolder.ID
		}
	}

	// 6. Atomically reserve bytes + file count (prevents race condition on parallel uploads)
	reserved, err := s.inviteRepo.ReserveUsage(ctx, invite.ID, size)
	if err != nil {
		return nil, "", fmt.Errorf("failed to reserve upload capacity: %w", err)
	}
	if !reserved {
		return nil, "", fmt.Errorf("invite capacity exceeded (max %d files, %d bytes total)", invite.MaxFiles, invite.MaxTotalBytes)
	}

	// 7. Create upload session under the owner's identity and quota
	file, uploadURL, err := s.fileService.CreateUploadSession(
		ctx, invite.OwnerID, filename, size, contentType, invite.TargetFolderID, nil, "keep_both", nil,
	)
	if err != nil {
		// Release the reserved capacity on failure
		_ = s.inviteRepo.ReleaseUsage(ctx, invite.ID, size)
		return nil, "", fmt.Errorf("failed to create upload session: %w", err)
	}

	// 8. Record audit trail (needed for IP rate limiting and CompleteUpload ownership check).
	// This MUST succeed: GuestCompleteUpload relies on this row to verify the file belongs to this invite.
	if err := s.inviteRepo.RecordFile(ctx, &model.UploadInviteFile{
		InviteID:  invite.ID,
		FileID:    file.ID,
		IPAddress: strPtr(ip),
		UserAgent: strPtr(userAgent),
	}); err != nil {
		_ = s.inviteRepo.ReleaseUsage(ctx, invite.ID, size)
		_ = s.fileService.repo.UpdateStatus(ctx, file.ID, "FAILED")
		return nil, "", fmt.Errorf("failed to record upload audit: %w", err)
	}

	return file, uploadURL, nil
}

// GuestCompleteUpload completes the upload, re-checks invite status, and queues owner notification.
// Security: verifies the file was created through THIS invite and is still in UPLOADING status.
func (s *UploadInviteService) GuestCompleteUpload(ctx context.Context, token, fileID, ip, userAgent string) error {
	// 1. Re-check invite status (handles revocation after presigned URL was issued)
	invite, err := s.inviteRepo.FindByToken(ctx, token)
	if err != nil || invite == nil {
		return fmt.Errorf("invite not found")
	}

	// 2. Verify this file was created through THIS invite (prevents acting on arbitrary owner files)
	owningInvite, err := s.inviteRepo.FindByFileID(ctx, fileID)
	if err != nil || owningInvite == nil || owningInvite.ID != invite.ID {
		return fmt.Errorf("file not found for this invite")
	}

	// 3. Verify the file is still in UPLOADING status (makes retries safe / idempotent)
	file, err := s.fileService.repo.FindByID(ctx, fileID)
	if err != nil || file == nil {
		return fmt.Errorf("file not found")
	}
	if file.Status != "UPLOADING" {
		return fmt.Errorf("this upload was already finalized")
	}

	// 4. Handle revoked invite: clean up the uploaded object
	if invite.Status == "revoked" {
		_ = s.fileService.storage.Delete(ctx, file.StorageKey)
		_ = s.fileService.repo.UpdateStatus(ctx, fileID, "FAILED")
		_ = s.inviteRepo.ReleaseUsage(ctx, invite.ID, file.FileSize)
		return fmt.Errorf("this invite has been revoked by the owner")
	}

	if invite.Status != "active" || time.Now().After(invite.ExpiresAt) {
		return fmt.Errorf("this invite is no longer active")
	}

	// 5. Delegate to FileService.CompleteUpload (magic byte validation, malware scan queue)
	if err := s.fileService.CompleteUpload(ctx, fileID, invite.OwnerID, nil); err != nil {
		// Release capacity on failed validation
		_ = s.inviteRepo.ReleaseUsage(ctx, invite.ID, file.FileSize)
		return err
	}

	// 6. Increment batched notification counter (scheduler flushes every ~10 min)
	_ = s.inviteRepo.IncrementPendingNotify(ctx, invite.ID)

	return nil
}

// FlushPendingNotifications is called by the scheduler to send batched owner notifications.
func (s *UploadInviteService) FlushPendingNotifications(ctx context.Context) error {
	invites, counts, err := s.inviteRepo.FlushPendingNotifications(ctx)
	if err != nil {
		return fmt.Errorf("failed to flush pending notifications: %w", err)
	}

	for i, inv := range invites {
		count := counts[i]
		label := inv.Label
		if label == "" {
			label = fmt.Sprintf("Upload %s", inv.Token[:8])
		}

		// Escape label for email safety (guest-controlled input)
		safeLabel := html.EscapeString(label)

		title := fmt.Sprintf("%d new file(s) received", count)
		body := fmt.Sprintf("You received %d new file(s) via invite \"%s\".", count, safeLabel)

		// In-app notification
		if s.notifService != nil {
			_ = s.notifService.QueueInAppNotification(ctx, inv.OwnerID, "invite.files_received", title, body, map[string]any{
				"invite_id": inv.ID,
				"label":     label,
				"count":     count,
			})
		}

		// Email notification (reuse existing pipeline)
		if s.redisQueue != nil {
			owner, ownerErr := s.userRepo.FindByID(ctx, inv.OwnerID)
			if ownerErr == nil && owner != nil && owner.Email != "" {
				ownerName := owner.Email
				if owner.FirstName != nil && *owner.FirstName != "" {
					ownerName = *owner.FirstName
				}
				emailJob := &queue.Job{
					ID:        fmt.Sprintf("invite-notif-email-%s-%d", inv.ID, time.Now().UnixNano()),
					Type:      queue.JobTypeEmail,
					UserID:    inv.OwnerID,
					Priority:  queue.PriorityNormal,
					CreatedAt: time.Now(),
					Payload: map[string]any{
						"to_email":  owner.Email,
						"to_name":   ownerName,
						"subject":   fmt.Sprintf("PushPostVault: %s", title),
						"html_body": fmt.Sprintf("<p>Hi %s,</p><p>%s</p><p>View them in your dashboard under <strong>Received / %s</strong>.</p>", html.EscapeString(ownerName), body, safeLabel),
					},
				}
				_ = s.redisQueue.Enqueue(ctx, emailJob)
			}
		}
	}

	return nil
}

// ExpireStaleInvites marks expired invites. Called by the scheduler.
func (s *UploadInviteService) ExpireStaleInvites(ctx context.Context) (int64, error) {
	return s.inviteRepo.ExpireStaleInvites(ctx)
}

// CleanupAbandonedUploads fails stale UPLOADING files created via invites (older than 2h)
// and releases their reserved capacity. This prevents a guest who created a session but never
// finished the PUT from permanently consuming invite capacity and owner quota.
func (s *UploadInviteService) CleanupAbandonedUploads(ctx context.Context) error {
	stale, err := s.inviteRepo.ListStaleUploads(ctx)
	if err != nil {
		return fmt.Errorf("failed to list stale invite uploads: %w", err)
	}
	for _, row := range stale {
		// Atomically claim the file: only act if it's still UPLOADING (idempotent)
		claimed, err := s.fileService.repo.ClaimStaleUpload(ctx, row.FileID)
		if err != nil || !claimed {
			continue
		}
		_ = s.fileService.storage.Delete(ctx, row.StorageKey)
		_ = s.inviteRepo.ReleaseUsage(ctx, row.InviteID, row.FileSize)
	}
	return nil
}

// GetOwnerMaxFileSize resolves the owner's effective per-file size limit.
// Returns the limit so the guest page can show it upfront.
func (s *UploadInviteService) GetOwnerMaxFileSize(ctx context.Context, ownerID string) int64 {
	maxFileSize := MaxFileSizeLimit // default 2 GB from file_service.go

	user, err := s.userRepo.FindByID(ctx, ownerID)
	if err != nil || user == nil {
		return maxFileSize
	}

	// Check subscription tier
	if s.fileService.subRepo != nil {
		if sub, err := s.fileService.subRepo.FindByUserID(ctx, ownerID); err == nil && sub != nil && sub.Package != nil && sub.Status == "active" {
			maxFileSize = sub.Package.MaxFileSizeBytes
		}
	}

	// Admin override if higher
	if user.MaxFileSizeBytes != nil && *user.MaxFileSizeBytes > maxFileSize {
		maxFileSize = *user.MaxFileSizeBytes
	}

	return maxFileSize
}

