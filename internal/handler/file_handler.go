package handler

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/service"
)

type FileHandler struct {
	service *service.FileService
}

func NewFileHandler(service *service.FileService) *FileHandler {
	return &FileHandler{service: service}
}

func (h *FileHandler) Upload(c echo.Context) error {
	userID := c.Get("user_id").(string)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing file form field"})
	}

	src, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to open upload source"})
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
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "File uploaded successfully",
		"file":    fileMeta,
	})
}

func (h *FileHandler) Download(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	stream, fileMeta, err := h.service.Download(c.Request().Context(), fileID, userID)
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
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
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
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
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"files": files})
}

func (h *FileHandler) ToggleShare(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	var req struct {
		IsPublic bool `json:"is_public"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	err := h.service.ToggleShareStatus(c.Request().Context(), fileID, userID, req.IsPublic)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Share status updated successfully"})
}

func (h *FileHandler) Delete(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	err := h.service.Delete(c.Request().Context(), fileID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "File deleted successfully"})
}