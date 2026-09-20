package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/service"
)

type UploadInviteHandler struct {
	svc *service.UploadInviteService
}

func NewUploadInviteHandler(svc *service.UploadInviteService) *UploadInviteHandler {
	return &UploadInviteHandler{svc: svc}
}

// ──────────────────────────────────────────────
// Protected endpoints (owner, JWT required)
// ──────────────────────────────────────────────

// CreateInvite handles POST /api/v1/upload-invites
func (h *UploadInviteHandler) CreateInvite(c echo.Context) error {
	ownerID, ok := c.Get("user_id").(string)
	if !ok || ownerID == "" {
		return SendError(c, http.StatusUnauthorized, "authentication required")
	}

	var req struct {
		Label         string `json:"label"`
		MaxTotalBytes int64  `json:"max_total_bytes"`
		MaxFiles      int    `json:"max_files"`
		ExpiryHours   int    `json:"expiry_hours"`
		Passcode      string `json:"passcode"`
	}
	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "invalid request body")
	}

	invite, err := h.svc.CreateInvite(c.Request().Context(), ownerID, req.Label, req.MaxTotalBytes, req.MaxFiles, req.ExpiryHours, req.Passcode)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusCreated, map[string]any{"invite": invite}, nil)
}

// ListInvites handles GET /api/v1/upload-invites
func (h *UploadInviteHandler) ListInvites(c echo.Context) error {
	ownerID, ok := c.Get("user_id").(string)
	if !ok || ownerID == "" {
		return SendError(c, http.StatusUnauthorized, "authentication required")
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	invites, total, err := h.svc.ListInvites(c.Request().Context(), ownerID, limit, offset)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "failed to list invites")
	}

	return SendSuccess(c, http.StatusOK, map[string]any{"invites": invites}, &PaginationMetadata{Total: total, Limit: limit})
}

// RevokeInvite handles DELETE /api/v1/upload-invites/:id
func (h *UploadInviteHandler) RevokeInvite(c echo.Context) error {
	ownerID, ok := c.Get("user_id").(string)
	if !ok || ownerID == "" {
		return SendError(c, http.StatusUnauthorized, "authentication required")
	}

	inviteID := c.Param("id")
	if inviteID == "" {
		return SendError(c, http.StatusBadRequest, "invite ID required")
	}

	if err := h.svc.RevokeInvite(c.Request().Context(), inviteID, ownerID); err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{"message": "invite revoked"}, nil)
}

// ──────────────────────────────────────────────
// Public endpoints (guest, no auth)
// ──────────────────────────────────────────────

// GetInviteInfo handles GET /api/v1/public/upload-invite/:token
func (h *UploadInviteHandler) GetInviteInfo(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return SendError(c, http.StatusBadRequest, "token required")
	}

	invite, err := h.svc.GetInviteInfo(c.Request().Context(), token)
	if err != nil {
		return SendError(c, http.StatusNotFound, "invite not found")
	}

	// Resolve owner's per-file limit so the guest page can enforce it client-side
	maxFileBytes := h.svc.GetOwnerMaxFileSize(c.Request().Context(), invite.OwnerID)

	// Return only public-safe fields
	return SendSuccess(c, http.StatusOK, map[string]any{
		"label":           invite.Label,
		"owner_name":      invite.OwnerName,
		"has_passcode":    invite.HasPasscode,
		"status":          invite.Status,
		"max_total_bytes": invite.MaxTotalBytes,
		"max_files":       invite.MaxFiles,
		"max_file_bytes":  maxFileBytes,
		"used_bytes":      invite.UsedBytes,
		"used_files":      invite.UsedFiles,
		"expires_at":      invite.ExpiresAt,
	}, nil)
}

// VerifyPasscode handles POST /api/v1/public/upload-invite/:token/verify-passcode
func (h *UploadInviteHandler) VerifyPasscode(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return SendError(c, http.StatusBadRequest, "token required")
	}

	var req struct {
		Passcode string `json:"passcode"`
	}
	if err := c.Bind(&req); err != nil || req.Passcode == "" {
		return SendError(c, http.StatusBadRequest, "passcode required")
	}

	ip := c.RealIP()
	sessionToken, err := h.svc.VerifyPasscode(c.Request().Context(), token, req.Passcode, ip)
	if err != nil {
		return SendError(c, http.StatusForbidden, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{"session_token": sessionToken}, nil)
}

// GuestCreateUploadSession handles POST /api/v1/public/upload-invite/:token/upload-session
func (h *UploadInviteHandler) GuestCreateUploadSession(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return SendError(c, http.StatusBadRequest, "token required")
	}

	var req struct {
		Filename     string `json:"filename"`
		FileSize     int64  `json:"file_size"`
		ContentType  string `json:"content_type"`
		SessionToken string `json:"session_token"`
	}
	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "invalid request body")
	}
	if req.Filename == "" || req.FileSize <= 0 || req.ContentType == "" {
		return SendError(c, http.StatusBadRequest, "filename, file_size, and content_type are required")
	}

	ip := c.RealIP()
	userAgent := c.Request().UserAgent()

	file, uploadURL, err := h.svc.GuestCreateUploadSession(
		c.Request().Context(), token, req.Filename, req.FileSize, req.ContentType,
		req.SessionToken, ip, userAgent,
	)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"file_id":    file.ID,
		"upload_url": uploadURL,
	}, nil)
}

// GuestCompleteUpload handles POST /api/v1/public/upload-invite/:token/complete/:fileId
func (h *UploadInviteHandler) GuestCompleteUpload(c echo.Context) error {
	token := c.Param("token")
	fileID := c.Param("fileId")
	if token == "" || fileID == "" {
		return SendError(c, http.StatusBadRequest, "token and file ID required")
	}

	ip := c.RealIP()
	userAgent := c.Request().UserAgent()

	if err := h.svc.GuestCompleteUpload(c.Request().Context(), token, fileID, ip, userAgent); err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{"message": "upload completed"}, nil)
}
