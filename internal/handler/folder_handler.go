package handler

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/service"
	"github.com/labstack/echo/v4"
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

// PATCH /api/v1/folders/:id/share
func (h *FolderHandler) ToggleShare(c echo.Context) error {
	id := c.Param("id")
	userID := c.Get("user_id").(string)

	var req struct {
		IsPublic bool `json:"is_public"`
	}

	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	err := h.service.ToggleShareStatus(c.Request().Context(), id, userID, req.IsPublic)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"message":   "Folder share status updated successfully",
		"is_public": req.IsPublic,
	}, nil)
}

// GET /api/v1/folders/public/:id
func (h *FolderHandler) GetPublicFolder(c echo.Context) error {
	id := c.Param("id")

	folder, subfolders, files, breadcrumbs, err := h.service.GetPublicFolderContents(c.Request().Context(), id)
	if err != nil {
		return SendError(c, http.StatusNotFound, err.Error())
	}

	scheme := c.Scheme()
	host := c.Request().Host

	type EnrichedPublicFile struct {
		*model.File
		DirectURL   string `json:"direct_url"`
		DownloadURL string `json:"download_url"`
	}

	var enrichedFiles []EnrichedPublicFile
	for _, f := range files {
		enrichedFiles = append(enrichedFiles, EnrichedPublicFile{
			File:        f,
			DirectURL:   fmt.Sprintf("%s://%s/api/v1/files/raw/%s", scheme, host, f.ID),
			DownloadURL: fmt.Sprintf("%s://%s/api/v1/files/public/%s?download=true", scheme, host, f.ID),
		})
	}

	type PublicBreadcrumb struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var crumbs []PublicBreadcrumb
	for _, b := range breadcrumbs {
		crumbs = append(crumbs, PublicBreadcrumb{ID: b.ID, Name: b.Name})
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"folder":      folder,
		"subfolders":  subfolders,
		"files":       enrichedFiles,
		"breadcrumbs": crumbs,
		"zip_url":     fmt.Sprintf("%s://%s/api/v1/folders/public/%s/download", scheme, host, folder.ID),
	}, nil)
}

// POST /api/v1/folders/public/:id/save-to-vault
func (h *FolderHandler) SavePublicFolderToVault(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return SendError(c, http.StatusUnauthorized, "Authentication required to save folder to vault")
	}
	id := c.Param("id")

	destFolder, count, err := h.service.SavePublicFolderToVault(c.Request().Context(), userID, id)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"folder":       destFolder,
		"copied_count": count,
		"message":      fmt.Sprintf("Saved folder and %d file(s) to your Vault!", count),
	}, nil)
}

// GET /api/v1/folders/public/:id/download
func (h *FolderHandler) DownloadPublicFolderZip(c echo.Context) error {
	id := c.Param("id")

	folder, files, storageProvider, err := h.service.GetPublicFolderForZip(c.Request().Context(), id)
	if err != nil {
		return SendError(c, http.StatusNotFound, err.Error())
	}

	safeName := strings.ReplaceAll(folder.Name, "\"", "_")
	c.Response().Header().Set("Content-Type", "application/zip")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", safeName))
	c.Response().WriteHeader(http.StatusOK)

	zipWriter := zip.NewWriter(c.Response().Writer)
	defer zipWriter.Close()

	for _, file := range files {
		rc, err := storageProvider.Download(c.Request().Context(), file.StorageKey)
		if err != nil {
			continue
		}
		w, err := zipWriter.Create(file.Filename)
		if err != nil {
			_ = rc.Close()
			continue
		}
		_, _ = io.Copy(w, rc)
		_ = rc.Close()
	}

	return nil
}

