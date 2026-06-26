package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/service"
)

type FolderHandler struct {
	service *service.FolderService
}

func NewFolderHandler(service *service.FolderService) *FolderHandler {
	return &FolderHandler{service: service}
}

// POST /api/v1/folders
func (h *FolderHandler) Create(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		Name     string  `json:"name"`
		ParentID *string `json:"parent_id,omitempty"`
	}

	if err := c.Bind(&req); err != nil || req.Name == "" {
		return SendError(c, http.StatusBadRequest, "Folder name is required")
	}

	folder, err := h.service.CreateFolder(c.Request().Context(), userID, req.Name, req.ParentID)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusCreated, map[string]interface{}{
		"folder": folder,
	}, nil)
}

// GET /api/v1/folders
func (h *FolderHandler) List(c echo.Context) error {
	userID := c.Get("user_id").(string)
	
	parentIDStr := c.QueryParam("parent_id")
	var parentID *string
	if parentIDStr != "" {
		parentID = &parentIDStr
	}

	flat := c.QueryParam("flat") == "true"

	folders, err := h.service.ListFolders(c.Request().Context(), userID, parentID, flat)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"folders": folders,
	}, nil)
}

// PUT /api/v1/folders/:id/move
func (h *FolderHandler) Move(c echo.Context) error {
	id := c.Param("id")
	userID := c.Get("user_id").(string)

	var req struct {
		ParentID *string `json:"parent_id"`
	}

	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	err := h.service.MoveFolder(c.Request().Context(), id, userID, req.ParentID)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{
		"message": "Folder moved successfully",
	}, nil)
}

// PUT /api/v1/folders/:id/rename
func (h *FolderHandler) Rename(c echo.Context) error {
	id := c.Param("id")
	userID := c.Get("user_id").(string)

	var req struct {
		Name string `json:"name"`
	}

	if err := c.Bind(&req); err != nil || req.Name == "" {
		return SendError(c, http.StatusBadRequest, "Folder name is required")
	}

	err := h.service.RenameFolder(c.Request().Context(), id, userID, req.Name)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{
		"message": "Folder renamed successfully",
	}, nil)
}

// DELETE /api/v1/folders/:id
func (h *FolderHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	userID := c.Get("user_id").(string)

	err := h.service.DeleteFolder(c.Request().Context(), id, userID)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{
		"message": "Folder deleted successfully",
	}, nil)
}
