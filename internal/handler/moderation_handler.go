package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
	"github.com/archaditya/bytevault/internal/storage"
)

// ModerationHandler handles admin content moderation, abuse reports, and user appeal endpoints.
type ModerationHandler struct {
	fileRepo     *repository.FileRepository
	userRepo     *repository.UserRepository
	abuseService *service.AbuseService
	notifService *service.NotificationService
	store        storage.StorageProvider
}

func NewModerationHandler(fileRepo *repository.FileRepository, userRepo *repository.UserRepository) *ModerationHandler {
	return &ModerationHandler{fileRepo: fileRepo, userRepo: userRepo}
}

func (h *ModerationHandler) SetAbuseService(as *service.AbuseService) {
	h.abuseService = as
}

func (h *ModerationHandler) SetNotificationService(ns *service.NotificationService) {
	h.notifService = ns
}

func (h *ModerationHandler) SetStorage(st storage.StorageProvider) {
	h.store = st
}

// --- Admin Endpoints ---

// GetModerationStats returns aggregate counts for the admin moderation dashboard.
func (h *ModerationHandler) GetModerationStats(c echo.Context) error {
	ctx := c.Request().Context()

	totalBlocked, totalFlagged, err := h.fileRepo.GetModerationStats(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to fetch moderation stats"})
	}

	restrictedUsers, err := h.userRepo.GetRestrictedUsersCount(ctx)
	if err != nil {
		restrictedUsers = 0
	}

	pendingAppeals, err := h.userRepo.GetPendingAppealsCount(ctx)
	if err != nil {
		pendingAppeals = 0
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"total_blocked":    totalBlocked,
			"total_flagged":    totalFlagged,
			"restricted_users": restrictedUsers,
			"pending_appeals":  pendingAppeals,
		},
	})
}

// ListFlaggedFiles returns files pending admin review (status=FLAGGED_REVIEW).
func (h *ModerationHandler) ListFlaggedFiles(c echo.Context) error {
	cursor := c.QueryParam("cursor")
	files, nextCursor, err := h.fileRepo.ListFlaggedFiles(c.Request().Context(), 20, cursor)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to list flagged files"})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"data":   map[string]any{"files": files, "next_cursor": nextCursor},
	})
}

// ListBlockedFiles returns auto-blocked NSFW files (audit trail).
func (h *ModerationHandler) ListBlockedFiles(c echo.Context) error {
	cursor := c.QueryParam("cursor")
	files, nextCursor, err := h.fileRepo.ListBlockedNSFWFiles(c.Request().Context(), 20, cursor)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to list blocked files"})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"data":   map[string]any{"files": files, "next_cursor": nextCursor},
	})
}

// ApproveFile marks a flagged file as safe and sets its status to READY.
func (h *ModerationHandler) ApproveFile(c echo.Context) error {
	fileID := c.Param("id")
	if err := h.fileRepo.UpdateStatus(c.Request().Context(), fileID, "READY"); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to approve file"})
	}
	return c.JSON(http.StatusOK, map[string]any{"status": "success", "detail": "File approved and marked READY"})
}

// RejectFile confirms NSFW, deletes the file, and applies a strike to the uploader.
func (h *ModerationHandler) RejectFile(c echo.Context) error {
	fileID := c.Param("id")
	ctx := c.Request().Context()

	file, err := h.fileRepo.FindByID(ctx, fileID)
	if err != nil || file == nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "File not found"})
	}

	// Soft-delete the file record
	_ = h.fileRepo.UpdateStatus(ctx, fileID, "BLOCKED_NSFW")

	// Increment strike on the uploader
	strikes, _ := h.userRepo.IncrementNSFWStrikes(ctx, file.UserID)

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"detail": "File rejected and user strike incremented",
		"data":   map[string]any{"user_strikes": strikes},
	})
}

// ListRestrictedUsers returns all currently restricted users.
func (h *ModerationHandler) ListRestrictedUsers(c echo.Context) error {
	users, err := h.userRepo.ListRestrictedUsers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to list restricted users"})
	}
	return c.JSON(http.StatusOK, map[string]any{"status": "success", "data": map[string]any{"users": users}})
}

// RestrictUser manually restricts a user (admin action).
func (h *ModerationHandler) RestrictUser(c echo.Context) error {
	userID := c.Param("id")
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Invalid request"})
	}

	if body.Reason == "" {
		body.Reason = "Manual restriction by admin"
	}

	if err := h.userRepo.RestrictUser(c.Request().Context(), userID, nil, body.Reason); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to restrict user"})
	}
	return c.JSON(http.StatusOK, map[string]any{"status": "success", "detail": "User restricted"})
}

// UnrestrictUser removes restriction and resets strikes (admin action).
func (h *ModerationHandler) UnrestrictUser(c echo.Context) error {
	userID := c.Param("id")
	if err := h.userRepo.UnrestrictUser(c.Request().Context(), userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to unrestrict user"})
	}
	return c.JSON(http.StatusOK, map[string]any{"status": "success", "detail": "User unrestricted and strikes reset"})
}

// ListAppeals returns all pending moderation appeals.
func (h *ModerationHandler) ListAppeals(c echo.Context) error {
	appeals, err := h.userRepo.ListPendingAppeals(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to list appeals"})
	}
	return c.JSON(http.StatusOK, map[string]any{"status": "success", "data": map[string]any{"appeals": appeals}})
}

// ApproveAppeal approves a user's appeal, un-restricting them and resetting strikes.
func (h *ModerationHandler) ApproveAppeal(c echo.Context) error {
	appealID := c.Param("id")
	adminID := c.Get("user_id").(string)
	ctx := c.Request().Context()

	var body struct {
		Notes string `json:"notes"`
	}
	_ = c.Bind(&body)

	// Fetch appeal details to obtain target user_id
	targetUserID, err := h.userRepo.GetAppealUserID(ctx, appealID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "Appeal not found"})
	}

	if err := h.userRepo.ResolveAppeal(ctx, appealID, "approved", body.Notes, adminID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to approve appeal"})
	}

	// Automatically unrestrict the user and reset strikes
	if err := h.userRepo.UnrestrictUser(ctx, targetUserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Appeal approved but failed to unrestrict user"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"detail": "Appeal approved and user restriction lifted successfully",
	})
}

// SubmitAppeal allows a restricted user to submit an appeal.
func (h *ModerationHandler) SubmitAppeal(c echo.Context) error {
	userID := c.Get("user_id").(string)
	ctx := c.Request().Context()

	// Verify user is actually restricted
	isRestricted, _, _, err := h.userRepo.IsUserRestricted(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to check restriction status"})
	}
	if !isRestricted {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Your account is not currently restricted"})
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind(&body); err != nil || body.Reason == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Please provide a reason for your appeal"})
	}

	hasPending, err := h.userRepo.HasPendingAppeal(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to check appeal status"})
	}
	if hasPending {
		return c.JSON(http.StatusConflict, map[string]any{"error": "You already have a pending appeal. Please wait for admin review."})
	}

	appeal, err := h.userRepo.CreateAppeal(ctx, userID, body.Reason)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to submit appeal"})
	}

	if h.notifService != nil {
		go func() {
			_ = h.notifService.NotifyAdmins(
				context.Background(),
				"admin.appeal",
				"New Moderation Appeal Submitted",
				"A restricted user submitted an appeal for account review",
				"/admin/moderation",
			)
		}()
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"status": "success",
		"detail": "Appeal submitted successfully. An admin will review within 24-48 hours.",
		"data":   appeal,
	})
}

// RejectAppeal rejects a user's appeal.
func (h *ModerationHandler) RejectAppeal(c echo.Context) error {
	appealID := c.Param("id")
	adminID := c.Get("user_id").(string)
	ctx := c.Request().Context()

	var body struct {
		Notes string `json:"notes"`
	}
	_ = c.Bind(&body)

	if err := h.userRepo.ResolveAppeal(ctx, appealID, "rejected", body.Notes, adminID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to reject appeal"})
	}

	return c.JSON(http.StatusOK, map[string]any{"status": "success", "detail": "Appeal rejected"})
}

// --- User-facing Endpoints ---

// GetRestrictionStatus returns the current restriction status for the authenticated user.
func (h *ModerationHandler) GetRestrictionStatus(c echo.Context) error {
	userID := c.Get("user_id").(string)
	ctx := c.Request().Context()

	restricted, until, reason, err := h.userRepo.IsUserRestricted(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to check restriction status"})
	}

	hasPending, _ := h.userRepo.HasPendingAppeal(ctx, userID)

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"is_restricted":    restricted,
			"restricted_until": until,
			"reason":           reason,
			"has_pending_appeal": hasPending,
		},
	})
}

// --- Abuse Report Endpoints ---

// SubmitAbuseReport allows public users to report violating content. Rate limited to 5/hr per IP.
func (h *ModerationHandler) SubmitAbuseReport(c echo.Context) error {
	if h.abuseService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"error": "Abuse reporting service unavailable"})
	}

	if h.notifService != nil {
		ip := c.RealIP()
		rlKey := fmt.Sprintf("PushPostVault:rl:abuse:%s", ip)
		if count, err := h.notifService.CheckRateLimit(c.Request().Context(), rlKey, 1*time.Hour); err == nil && count > 5 {
			return c.JSON(http.StatusTooManyRequests, map[string]any{"error": "Too many abuse reports submitted. Please try again later."})
		}
	}

	var req model.SubmitAbuseReportRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
	}

	report, err := h.abuseService.SubmitReport(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"status": "success",
		"data":   report,
	})
}

// ListAbuseReports returns paginated abuse reports for admin moderation.
func (h *ModerationHandler) ListAbuseReports(c echo.Context) error {
	if h.abuseService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"error": "Abuse reporting service unavailable"})
	}

	status := c.QueryParam("status")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	reports, total, err := h.abuseService.ListReports(c.Request().Context(), status, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to list abuse reports"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"reports": reports,
			"pagination": map[string]any{
				"total": total,
				"page":  page,
				"limit": limit,
			},
		},
	})
}

// GetAbuseReport returns details for a single abuse report.
func (h *ModerationHandler) GetAbuseReport(c echo.Context) error {
	if h.abuseService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"error": "Abuse reporting service unavailable"})
	}

	id := c.Param("id")
	report, err := h.abuseService.GetReport(c.Request().Context(), id)
	if err != nil || report == nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "Abuse report not found"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"data":   report,
	})
}

// TakedownAbuseReport actions takedown on the reported file and notifies reporter.
func (h *ModerationHandler) TakedownAbuseReport(c echo.Context) error {
	if h.abuseService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"error": "Abuse reporting service unavailable"})
	}

	id := c.Param("id")
	adminID := c.Get("user_id").(string)

	var req struct {
		AdminNotes *string `json:"admin_notes"`
	}
	_ = c.Bind(&req)

	if err := h.abuseService.TakedownReport(c.Request().Context(), id, req.AdminNotes, adminID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"detail": "Target file taken down and marked BLOCKED_ABUSE",
	})
}

// StrikeUploaderFromReport takes down the file and applies a moderation strike to the uploader.
func (h *ModerationHandler) StrikeUploaderFromReport(c echo.Context) error {
	if h.abuseService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"error": "Abuse reporting service unavailable"})
	}

	id := c.Param("id")
	adminID := c.Get("user_id").(string)

	var req struct {
		AdminNotes *string `json:"admin_notes"`
	}
	_ = c.Bind(&req)

	strikes, err := h.abuseService.StrikeUploader(c.Request().Context(), id, req.AdminNotes, adminID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"detail": "Target file taken down and uploader strike incremented",
		"data": map[string]any{
			"uploader_strikes": strikes,
		},
	})
}

// DismissAbuseReport marks the report as dismissed with administrative notes.
func (h *ModerationHandler) DismissAbuseReport(c echo.Context) error {
	if h.abuseService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"error": "Abuse reporting service unavailable"})
	}

	id := c.Param("id")
	adminID := c.Get("user_id").(string)

	var req struct {
		AdminNotes *string `json:"admin_notes"`
	}
	_ = c.Bind(&req)

	if err := h.abuseService.DismissReport(c.Request().Context(), id, req.AdminNotes, adminID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"detail": "Abuse report dismissed",
	})
}

// PreviewFile provides file metadata and a 1-hour presigned view URL for admin moderation inspection.
func (h *ModerationHandler) PreviewFile(c echo.Context) error {
	fileID := c.Param("id")
	ctx := c.Request().Context()

	file, err := h.fileRepo.FindByID(ctx, fileID)
	if err != nil || file == nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "File not found"})
	}

	var previewURL string
	if h.store != nil {
		url, err := h.store.GeneratePresignedDownloadURL(ctx, file.StorageKey, 1*time.Hour, file.Filename, true)
		if err == nil {
			previewURL = url
		}
	}

	uploaderEmail := ""
	uploaderStrikes := 0
	var accountCreated *time.Time
	if h.userRepo != nil && file.UserID != "" {
		if u, err := h.userRepo.FindByID(ctx, file.UserID); err == nil && u != nil {
			uploaderEmail = u.Email
			uploaderStrikes = u.NSFWStrikes
			accountCreated = &u.CreatedAt
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"file_id":          file.ID,
			"filename":         file.Filename,
			"file_size":        file.FileSize,
			"content_type":     file.ContentType,
			"status":           file.Status,
			"is_public":        file.IsPublic,
			"created_at":       file.CreatedAt,
			"preview_url":      previewURL,
			"uploader_email":   uploaderEmail,
			"uploader_strikes": uploaderStrikes,
			"account_created":  accountCreated,
		},
	})
}
