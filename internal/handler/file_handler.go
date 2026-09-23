package handler

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/notification/worker"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
	"github.com/labstack/echo/v4"
)

type FileHandler struct {
	service       *service.FileService
	folderService *service.FolderService
	mediaWorker   *worker.MediaWorker
	localDir      string // Base dir to support direct local storage uploads
}

func NewFileHandler(service *service.FileService, localDir string) *FileHandler {
	return &FileHandler{
		service:  service,
		localDir: localDir,
	}
}

func (h *FileHandler) SetFolderService(fs *service.FolderService) {
	h.folderService = fs
}

func (h *FileHandler) SetMediaWorker(mw *worker.MediaWorker) {
	h.mediaWorker = mw
}

// POST /api/v1/files/upload-session
func (h *FileHandler) CreateUploadSession(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		Filename       string   `json:"filename"`
		FileSize       int64    `json:"file_size"`
		ContentType    string   `json:"content_type"`
		FolderID       *string  `json:"folder_id,omitempty"`
		Tags           []string `json:"tags,omitempty"`
		ConflictAction string   `json:"conflict_action,omitempty"` // "replace" | "keep_both"
		ContentHash    *string  `json:"content_hash,omitempty"`
	}

	if err := c.Bind(&req); err != nil || req.Filename == "" || req.FileSize <= 0 || req.ContentType == "" {
		return SendError(c, http.StatusBadRequest, "Invalid request parameters")
	}

	fileMeta, uploadURL, err := h.service.CreateUploadSession(
		c.Request().Context(),
		userID,
		req.Filename,
		req.FileSize,
		req.ContentType,
		req.FolderID,
		req.Tags,
		req.ConflictAction,
		req.ContentHash,
	)
	if err != nil {
		var conflictErr *service.FileConflictError
		if errors.As(err, &conflictErr) {
			return SendConflict(c, conflictErr.Error(), map[string]interface{}{
				"filename":      conflictErr.Filename,
				"existing_file": conflictErr.ExistingFile,
			})
		}
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"file_id":    fileMeta.ID,
		"filename":   fileMeta.Filename,
		"upload_url": uploadURL,
	}, nil)
}

// POST /api/v1/files/check-conflicts
func (h *FileHandler) CheckConflicts(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		FolderID  *string  `json:"folder_id,omitempty"`
		Filenames []string `json:"filenames"`
	}

	if err := c.Bind(&req); err != nil || len(req.Filenames) == 0 {
		return SendError(c, http.StatusBadRequest, "filenames array is required")
	}

	conflicts, err := h.service.CheckConflicts(c.Request().Context(), userID, req.Filenames, req.FolderID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"conflicts": conflicts,
	}, nil)
}

// POST /api/v1/files/:id/complete
func (h *FileHandler) CompleteUpload(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	var req struct {
		ContentHash *string `json:"content_hash,omitempty"`
	}
	_ = c.Bind(&req)

	err := h.service.CompleteUpload(c.Request().Context(), fileID, userID, req.ContentHash)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{
		"message": "File upload completed successfully",
	}, nil)
}

// PUT /api/v1/files/upload/direct
func (h *FileHandler) UploadLocalDirect(c echo.Context) error {
	storageKey := c.QueryParam("key")
	if storageKey == "" {
		return SendError(c, http.StatusBadRequest, "Missing key parameter")
	}

	fullPath := filepath.Join(h.localDir, storageKey)
	if err := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to create directory structure")
	}

	dest, err := os.Create(fullPath)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to create local file")
	}
	defer dest.Close()

	if _, err := io.Copy(dest, c.Request().Body); err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to save local file bytes")
	}

	return c.NoContent(http.StatusOK)
}

// GET /api/v1/files/download/direct
func (h *FileHandler) DownloadLocalDirect(c echo.Context) error {
	storageKey := c.QueryParam("key")
	if storageKey == "" {
		return SendError(c, http.StatusBadRequest, "Missing key parameter")
	}

	fullPath := filepath.Join(h.localDir, storageKey)
	file, err := os.Open(fullPath)
	if err != nil {
		return SendError(c, http.StatusNotFound, "Local file not found")
	}
	defer file.Close()

	// 1. Detect Content-Type dynamically from file extension
	contentType := mime.TypeByExtension(filepath.Ext(storageKey))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Response().Header().Set(echo.HeaderContentType, contentType)

	// 2. Serve images inline so they display in <img>, otherwise attachment download
	disposition := "attachment"
	if strings.HasPrefix(contentType, "image/") || c.QueryParam("inline") == "true" {
		disposition = "inline"
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, disposition+"; filename="+filepath.Base(storageKey))
	c.Response().WriteHeader(http.StatusOK)
	_, err = io.Copy(c.Response().Writer, file)
	return err
}

func (h *FileHandler) Upload(c echo.Context) error {
	userID := c.Get("user_id").(string)
	folderIDStr := c.FormValue("folder_id")
	
	var folderID *string
	if folderIDStr != "" {
		folderID = &folderIDStr
	}

	// 0. Auto-organize API uploads into a folder named after the API Key (e.g. "Dev Key" or "API Uploads")
	if c.Get("auth_type") == "api_key" && folderID == nil && h.folderService != nil {
		folderName := "API Uploads"
		if keyName, ok := c.Get("api_key_name").(string); ok && strings.TrimSpace(keyName) != "" {
			folderName = strings.TrimSpace(keyName)
		}
		if apiKeyFolder, err := h.folderService.GetOrCreateFolder(c.Request().Context(), userID, folderName, nil); err == nil && apiKeyFolder != nil {
			folderID = &apiKeyFolder.ID
		}
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return SendError(c, http.StatusBadRequest, "Missing file form field")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to open upload source")
	}
	defer src.Close()

	fileMeta, err := h.service.Upload(
		c.Request().Context(),
		userID,
		fileHeader.Filename,
		fileHeader.Size,
		fileHeader.Header.Get("Content-Type"),
		src,
		folderID,
	)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	// 1. For API Key uploads (developer mode), default to public so direct raw URLs work in <img> and curl
	isPublic := false
	if c.Get("auth_type") == "api_key" {
		isPublic = c.FormValue("is_public") != "false" // default true for API key uploads
	} else {
		isPublic = c.FormValue("is_public") == "true"
	}

	if isPublic {
		_ = h.service.ToggleShareStatus(c.Request().Context(), fileMeta.ID, userID, true)
		fileMeta.IsPublic = true
	}

	contentType := strings.ToLower(fileMeta.ContentType)
	ext := strings.ToLower(filepath.Ext(fileMeta.Filename))
	isVisual := strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "video/") || ext == ".mp4" || ext == ".mov" || ext == ".mkv" || ext == ".pdf" || contentType == "application/pdf"

	// 2. Synchronously run AI labeling, NSFW scoring, and thumbnailing if media worker is available
	if h.mediaWorker != nil && isVisual {
		if err := h.mediaWorker.ProcessFile(c.Request().Context(), fileMeta.ID); err == nil {
			if updated, err := h.service.GetFileDetails(c.Request().Context(), fileMeta.ID, userID); err == nil && updated != nil {
				fileMeta = updated
			}
		}
	}

	scheme := c.Scheme()
	host := c.Request().Host
	directURL := fmt.Sprintf("%s://%s/api/v1/files/raw/%s", scheme, host, fileMeta.ID)

	var thumbnailURL *string
	if fileMeta.ThumbnailKey != nil && *fileMeta.ThumbnailKey != "" {
		tURL := fmt.Sprintf("%s://%s/api/v1/files/public/%s/thumbnail", scheme, host, fileMeta.ID)
		thumbnailURL = &tURL
		fileMeta.ThumbnailURL = thumbnailURL
	} else if isVisual {
		tURL := fmt.Sprintf("%s://%s/api/v1/files/public/%s/thumbnail", scheme, host, fileMeta.ID)
		thumbnailURL = &tURL
	}

	return SendSuccess(c, http.StatusCreated, map[string]interface{}{
		"message":       "File uploaded successfully",
		"file":          fileMeta,
		"direct_url":    directURL,
		"thumbnail_url": thumbnailURL,
		"tags":          fileMeta.Tags,
		"nsfw_score":    fileMeta.NSFWScore,
	}, nil)
}

func (h *FileHandler) Download(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	inline := c.QueryParam("inline") == "true"

	url, _, err := h.service.Download(c.Request().Context(), fileID, userID, inline)
	if err != nil {
		return SendError(c, http.StatusForbidden, err.Error())
	}

	return c.Redirect(http.StatusFound, url)
}

// GET /api/v1/files/public/:id/metadata
func (h *FileHandler) GetPublicMetadata(c echo.Context) error {
	fileID := c.Param("id")

	fileMeta, err := h.service.GetPublicMetadata(c.Request().Context(), fileID)
	if err != nil {
		return SendError(c, http.StatusNotFound, err.Error())
	}

	scheme := c.Scheme()
	host := c.Request().Host
	directURL := fmt.Sprintf("%s://%s/api/v1/files/raw/%s", scheme, host, fileMeta.ID)

	data := map[string]interface{}{
		"id":            fileMeta.ID,
		"filename":      fileMeta.Filename,
		"file_size":     fileMeta.FileSize,
		"content_type":  fileMeta.ContentType,
		"created_at":    fileMeta.CreatedAt,
		"has_thumbnail": fileMeta.ThumbnailKey != nil && *fileMeta.ThumbnailKey != "",
		"thumbnail_url": fileMeta.ThumbnailURL,
		"direct_url":    directURL,
		"tags":          fileMeta.Tags,
		"nsfw_score":    fileMeta.NSFWScore,
		"status":        fileMeta.Status,
	}

	return SendSuccess(c, http.StatusOK, data, nil)
}

// GET /api/v1/files/public/:id/thumbnail
func (h *FileHandler) GetPublicThumbnail(c echo.Context) error {
	fileID := c.Param("id")

	url, err := h.service.GetPublicThumbnailURL(c.Request().Context(), fileID)
	if err != nil {
		return SendError(c, http.StatusNotFound, "thumbnail not available")
	}

	return c.Redirect(http.StatusFound, url)
}

func (h *FileHandler) DownloadPublic(c echo.Context) error {
	fileID := c.Param("id")

	inline := c.QueryParam("inline") == "true"

	url, _, err := h.service.DownloadPublic(c.Request().Context(), fileID, inline)
	if err != nil {
		return SendError(c, http.StatusNotFound, err.Error())
	}

	return c.Redirect(http.StatusFound, url)
}

// ServeRaw streams or redirects directly to the raw file payload (inline by default, or attachment if download=true).
// Perfect for developer integration in <img>, <video>, curl, or direct downloads without web UI wrappers.
func (h *FileHandler) ServeRaw(c echo.Context) error {
	fileID := c.Param("id")
	inline := c.QueryParam("download") != "true"

	url, file, err := h.service.DownloadPublic(c.Request().Context(), fileID, inline)
	if err != nil {
		// If not public, check if authorized via Bearer or API Key
		userID, ok := c.Get("user_id").(string)
		if !ok || userID == "" {
			return SendError(c, http.StatusUnauthorized, "File is private or requires authorization")
		}
		url, file, err = h.service.Download(c.Request().Context(), fileID, userID, inline)
		if err != nil {
			return SendError(c, http.StatusForbidden, err.Error())
		}
	}

	if file != nil && file.ContentType != "" {
		c.Response().Header().Set("Content-Type", file.ContentType)
	}

	return c.Redirect(http.StatusFound, url)
}

func (h *FileHandler) List(c echo.Context) error {
	userID := c.Get("user_id").(string)

	filterFolder := false
	var folderID *string

	if c.QueryParams().Has("folder_id") {
		folderIDStr := c.QueryParam("folder_id")
		if folderIDStr != "all" {
			filterFolder = true
			if folderIDStr != "" && folderIDStr != "null" && folderIDStr != "root" {
				folderID = &folderIDStr
			}
		}
	}

	search := c.QueryParam("q")
	sortBy := c.QueryParam("sort_by")
	sortDir := c.QueryParam("sort_dir")
	cursor := c.QueryParam("cursor")

	var isPublic *bool
	if isPublicStr := c.QueryParam("is_public"); isPublicStr != "" {
		ip := isPublicStr == "true"
		isPublic = &ip
	}

	limit := 20
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	params := repository.ListFilesParams{
		UserID:       userID,
		FilterFolder: filterFolder,
		FolderID:     folderID,
		Search:       search,
		SortBy:       sortBy,
		SortDir:      sortDir,
		Limit:        limit,
		Cursor:       cursor,
		IsPublic:     isPublic,
	}

	files, nextCursor, err := h.service.ListUserFiles(c.Request().Context(), params)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	var pagination map[string]any
	if nextCursor != "" {
		pagination = map[string]any{
			"next_cursor": nextCursor,
			"has_more":    true,
		}
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{"files": files}, pagination)
}

func (h *FileHandler) ToggleShare(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	var req struct {
		IsPublic bool `json:"is_public"`
	}
	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	err := h.service.ToggleShareStatus(c.Request().Context(), fileID, userID, req.IsPublic)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{"message": "Share status updated successfully"}, nil)
}

// PUT /api/v1/files/:id/move
func (h *FileHandler) Move(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	var req struct {
		FolderID *string `json:"folder_id"`
	}
	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	err := h.service.MoveFile(c.Request().Context(), fileID, userID, req.FolderID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{"message": "File moved successfully"}, nil)
}

// PUT /api/v1/files/:id/rename
func (h *FileHandler) Rename(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	var req struct {
		Filename string `json:"filename"`
	}
	if err := c.Bind(&req); err != nil || req.Filename == "" {
		return SendError(c, http.StatusBadRequest, "Invalid filename")
	}

	err := h.service.RenameFile(c.Request().Context(), fileID, userID, req.Filename)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{"message": "File renamed successfully"}, nil)
}

func (h *FileHandler) Delete(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	err := h.service.Delete(c.Request().Context(), fileID, userID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{"message": "File deleted successfully"}, nil)
}

// GET /api/v1/files/:id
func (h *FileHandler) GetDetails(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	file, err := h.service.GetFileDetails(c.Request().Context(), fileID, userID)
	if err != nil {
		return SendError(c, http.StatusNotFound, err.Error())
	}

	scheme := c.Scheme()
	host := c.Request().Host
	directURL := fmt.Sprintf("%s://%s/api/v1/files/raw/%s", scheme, host, file.ID)

	var thumbnailURL *string
	if file.ThumbnailKey != nil && *file.ThumbnailKey != "" {
		tURL := fmt.Sprintf("%s://%s/api/v1/files/public/%s/thumbnail", scheme, host, file.ID)
		thumbnailURL = &tURL
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"file":          file,
		"direct_url":    directURL,
		"thumbnail_url": thumbnailURL,
		"tags":          file.Tags,
		"nsfw_score":    file.NSFWScore,
	}, nil)
}


// POST /api/v1/files/multipart-session
func (h *FileHandler) CreateMultipartSession(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		Filename       string  `json:"filename"`
		FileSize       int64   `json:"file_size"`
		ContentType    string  `json:"content_type"`
		FolderID       *string `json:"folder_id,omitempty"`
		PartCount      int     `json:"part_count"`
		ConflictAction string  `json:"conflict_action,omitempty"`
		ContentHash    *string `json:"content_hash,omitempty"`
	}

	if err := c.Bind(&req); err != nil || req.Filename == "" || req.FileSize <= 0 || req.ContentType == "" || req.PartCount <= 0 {
		return SendError(c, http.StatusBadRequest, "Invalid request parameters")
	}

	fileMeta, uploadID, partURLs, err := h.service.CreateMultipartUploadSession(
		c.Request().Context(),
		userID,
		req.Filename,
		req.FileSize,
		req.ContentType,
		req.FolderID,
		req.PartCount,
		req.ConflictAction,
		req.ContentHash,
	)
	if err != nil {
		var conflictErr *service.FileConflictError
		if errors.As(err, &conflictErr) {
			return SendConflict(c, conflictErr.Error(), map[string]interface{}{
				"filename":      conflictErr.Filename,
				"existing_file": conflictErr.ExistingFile,
			})
		}
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"file_id":   fileMeta.ID,
		"filename":  fileMeta.Filename,
		"upload_id": uploadID,
		"part_urls": partURLs,
	}, nil)
}

// POST /api/v1/files/:id/complete-multipart
func (h *FileHandler) CompleteMultipartSession(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	var req struct {
		UploadID    string             `json:"upload_id"`
		Parts       []model.UploadPart `json:"parts"`
		ContentHash *string            `json:"content_hash,omitempty"`
	}

	if err := c.Bind(&req); err != nil || req.UploadID == "" || len(req.Parts) == 0 {
		return SendError(c, http.StatusBadRequest, "Invalid completion parameters")
	}

	err := h.service.CompleteMultipartUpload(c.Request().Context(), fileID, userID, req.UploadID, req.Parts, req.ContentHash)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{
		"message": "Multipart upload completed successfully",
	}, nil)
}

// POST /api/v1/files/:id/abort-multipart
func (h *FileHandler) AbortMultipartSession(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	var req struct {
		UploadID string `json:"upload_id"`
	}

	if err := c.Bind(&req); err != nil || req.UploadID == "" {
		return SendError(c, http.StatusBadRequest, "Invalid abort parameters")
	}

	err := h.service.AbortMultipartUpload(c.Request().Context(), fileID, userID, req.UploadID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{
		"message": "Multipart upload aborted successfully",
	}, nil)
}

// POST /api/v1/files/:id/refresh-part-urls
func (h *FileHandler) RefreshPartURLs(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	var req struct {
		UploadID    string  `json:"upload_id"`
		PartNumbers []int32 `json:"part_numbers"`
	}

	if err := c.Bind(&req); err != nil || req.UploadID == "" || len(req.PartNumbers) == 0 {
		return SendError(c, http.StatusBadRequest, "Invalid request: upload_id and part_numbers are required")
	}

	refreshedURLs, err := h.service.RefreshMultipartPartURLs(c.Request().Context(), fileID, userID, req.UploadID, req.PartNumbers)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"part_urls": refreshedURLs,
	}, nil)
}

// GET /api/v1/files/:id/thumbnail
func (h *FileHandler) GetThumbnail(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	url, _, err := h.service.GetThumbnail(c.Request().Context(), fileID, userID)
	if err != nil {
		// Return 404 if thumbnail is not available instead of redirecting <img> tag to raw video/pdf stream
		return SendError(c, http.StatusNotFound, "thumbnail not available for this file")
	}

	return c.Redirect(http.StatusFound, url)
}

