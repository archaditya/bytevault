package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
)

type EphemeralHandler struct {
	service *service.EphemeralService
}

func NewEphemeralHandler(service *service.EphemeralService) *EphemeralHandler {
	return &EphemeralHandler{service: service}
}

// GET /api/v1/ephemeral/config (Public & Admin - reads ephemeral_settings table)
func (h *EphemeralHandler) GetConfig(c echo.Context) error {
	settings, err := h.service.GetSettings(c.Request().Context())
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to load ephemeral settings")
	}
	return SendSuccess(c, http.StatusOK, settings, nil)
}

// POST /api/v1/admin/ephemeral-config (Admin Only - updates ephemeral_settings table)
func (h *EphemeralHandler) UpdateAdminConfig(c echo.Context) error {
	var req repository.EphemeralSettings
	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "invalid settings payload")
	}

	if err := h.service.UpdateSettings(c.Request().Context(), &req); err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to update settings in DB")
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message": "Ephemeral settings saved to database",
	}, nil)
}

// POST /api/v1/ephemeral/upload-session (Public Guest)
func (h *EphemeralHandler) CreateUploadSession(c echo.Context) error {
	var req struct {
		Filename    string  `json:"filename"`
		FileSize    int64   `json:"file_size"`
		ContentType string  `json:"content_type"`
		Password    *string `json:"password"`
	}

	if err := c.Bind(&req); err != nil || req.Filename == "" || req.FileSize <= 0 {
		return SendError(c, http.StatusBadRequest, "filename and file_size are required")
	}

	ip := c.RealIP()
	ua := c.Request().UserAgent()

	share, uploadURL, err := h.service.CreateUploadSession(
		c.Request().Context(),
		req.Filename,
		req.FileSize,
		req.ContentType,
		req.Password,
		&ip,
		&ua,
	)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusCreated, map[string]any{
		"token":      share.Token,
		"upload_url": uploadURL,
		"expires_at": share.ExpiresAt,
	}, nil)
}

// POST /api/v1/ephemeral/multipart-session (Public Guest)
func (h *EphemeralHandler) CreateMultipartSession(c echo.Context) error {
	var req struct {
		Filename    string  `json:"filename"`
		FileSize    int64   `json:"file_size"`
		ContentType string  `json:"content_type"`
		Password    *string `json:"password"`
		PartCount   int     `json:"part_count"`
	}

	if err := c.Bind(&req); err != nil || req.Filename == "" || req.FileSize <= 0 || req.PartCount <= 0 {
		return SendError(c, http.StatusBadRequest, "filename, file_size, and part_count are required")
	}

	ip := c.RealIP()
	ua := c.Request().UserAgent()

	share, uploadID, partURLs, err := h.service.CreateMultipartUploadSession(
		c.Request().Context(),
		req.Filename,
		req.FileSize,
		req.ContentType,
		req.Password,
		&ip,
		&ua,
		req.PartCount,
	)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusCreated, map[string]any{
		"token":      share.Token,
		"upload_id":  uploadID,
		"part_urls":  partURLs,
		"expires_at": share.ExpiresAt,
	}, nil)
}

// POST /api/v1/ephemeral/complete-multipart/:token (Public Guest)
func (h *EphemeralHandler) CompleteMultipartSession(c echo.Context) error {
	token := c.Param("token")

	var req struct {
		UploadID string             `json:"upload_id"`
		Parts    []model.UploadPart `json:"parts"`
	}

	if err := c.Bind(&req); err != nil || req.UploadID == "" || len(req.Parts) == 0 {
		return SendError(c, http.StatusBadRequest, "upload_id and parts are required")
	}

	err := h.service.CompleteMultipartUpload(c.Request().Context(), token, req.UploadID, req.Parts)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{
		"message": "Multipart upload completed successfully",
	}, nil)
}

// POST /api/v1/ephemeral/abort-multipart/:token (Public Guest)
func (h *EphemeralHandler) AbortMultipartSession(c echo.Context) error {
	token := c.Param("token")

	var req struct {
		UploadID string `json:"upload_id"`
	}

	if err := c.Bind(&req); err != nil || req.UploadID == "" {
		return SendError(c, http.StatusBadRequest, "upload_id is required")
	}

	err := h.service.AbortMultipartUpload(c.Request().Context(), token, req.UploadID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{
		"message": "Multipart upload aborted",
	}, nil)
}

// GET /api/v1/ephemeral/metadata/:token
func (h *EphemeralHandler) GetMetadata(c echo.Context) error {
	token := c.Param("token")
	share, err := h.service.GetMetadata(c.Request().Context(), token)
	if err != nil {
		return SendError(c, http.StatusNotFound, err.Error())
	}
	return SendSuccess(c, http.StatusOK, map[string]any{"share": share}, nil)
}

// POST /api/v1/ephemeral/download/:token
func (h *EphemeralHandler) Download(c echo.Context) error {
	token := c.Param("token")
	var req struct {
		Password *string `json:"password"`
	}
	_ = c.Bind(&req)

	downloadURL, err := h.service.RequestDownload(c.Request().Context(), token, req.Password)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}
	return SendSuccess(c, http.StatusOK, map[string]any{"download_url": downloadURL}, nil)
}

// GET /api/v1/admin/ephemeral-logs
func (h *EphemeralHandler) ListAdminLogs(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	shares, total, err := h.service.ListAllForAdmin(c.Request().Context(), limit, offset)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}
	return SendSuccess(c, http.StatusOK, map[string]any{"shares": shares, "total": total}, nil)
}
