package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/service"
)

type FileHandler struct {
	service  *service.FileService
	localDir string // Base dir to support direct local storage uploads
}

func NewFileHandler(service *service.FileService, localDir string) *FileHandler {
	return &FileHandler{
		service:  service,
		localDir: localDir,
	}
}

// POST /api/v1/files/upload-session
func (h *FileHandler) CreateUploadSession(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		Filename    string `json:"filename"`
		FileSize    int64  `json:"file_size"`
		ContentType string `json:"content_type"`
	}

	if err := c.Bind(&req); err != nil || req.Filename == "" || req.FileSize <= 0 || req.ContentType == "" {
		return SendError(c, http.StatusBadRequest, "Invalid request parameters")
	}

	fileMeta, uploadURL, err := h.service.CreateUploadSession(c.Request().Context(), userID, req.Filename, req.FileSize, req.ContentType)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, fmt.Sprintf("failed to generate upload URL: %v", err))
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"file_id":    fileMeta.ID,
		"upload_url": uploadURL,
	}, nil)
}

// POST /api/v1/files/:id/complete
func (h *FileHandler) CompleteUpload(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	err := h.service.CompleteUpload(c.Request().Context(), fileID, userID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{
		"message": "File upload completed successfully",
	}, nil)
}

// PUT /api/v1/files/upload/direct
// Dev environment route to support direct file upload mock to Local Disk
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
// Dev environment route to support direct file download mock from Local Disk
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

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+filepath.Base(storageKey))
	c.Response().WriteHeader(http.StatusOK)
	_, err = io.Copy(c.Response().Writer, file)
	return err
}

func (h *FileHandler) Upload(c echo.Context) error {
	userID := c.Get("user_id").(string)

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
	)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusCreated, map[string]interface{}{
		"message": "File uploaded successfully",
		"file":    fileMeta,
	}, nil)
}

func (h *FileHandler) Download(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	stream, fileMeta, err := h.service.Download(c.Request().Context(), fileID, userID)
	if err != nil {
		return SendError(c, http.StatusForbidden, err.Error())
	}
	defer stream.Close()

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileMeta.Filename)
	c.Response().Header().Set(echo.HeaderContentType, fileMeta.ContentType)
	c.Response().WriteHeader(http.StatusOK)

	_, err = io.Copy(c.Response().Writer, stream)
	return err
}

func (h *FileHandler) DownloadPublic(c echo.Context) error {
	fileID := c.Param("id")

	stream, fileMeta, err := h.service.DownloadPublic(c.Request().Context(), fileID)
	if err != nil {
		return SendError(c, http.StatusNotFound, err.Error())
	}
	defer stream.Close()

	c.Response().Header().Set(echo.HeaderContentDisposition, "inline; filename="+fileMeta.Filename)
	c.Response().Header().Set(echo.HeaderContentType, fileMeta.ContentType)
	c.Response().WriteHeader(http.StatusOK)

	_, err = io.Copy(c.Response().Writer, stream)
	return err
}

func (h *FileHandler) List(c echo.Context) error {
	userID := c.Get("user_id").(string)

	files, err := h.service.ListUserFiles(c.Request().Context(), userID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{"files": files}, nil)
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

func (h *FileHandler) Delete(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	err := h.service.Delete(c.Request().Context(), fileID, userID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{"message": "File deleted successfully"}, nil)
}
