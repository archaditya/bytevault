package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/service"
)

type ShareHandler struct {
	shareService *service.ShareService
}

func NewShareHandler(shareService *service.ShareService) *ShareHandler {
	return &ShareHandler{shareService: shareService}
}

// POST /api/v1/shares
func (h *ShareHandler) GrantShare(c echo.Context) error {
	ownerID := c.Get("user_id").(string)

	var req struct {
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
		GranteeEmail string `json:"grantee_email"`
		Permission   string `json:"permission"`
	}

	if err := c.Bind(&req); err != nil || req.ResourceID == "" || req.GranteeEmail == "" || req.ResourceType == "" {
		return SendError(c, http.StatusBadRequest, "resource_type, resource_id and grantee_email are required")
	}

	share, err := h.shareService.GrantShare(c.Request().Context(), ownerID, req.ResourceType, req.ResourceID, req.GranteeEmail, req.Permission)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusCreated, map[string]any{
		"share": share,
	}, nil)
}

// GET /api/v1/shares?resource_type=file&resource_id=...
func (h *ShareHandler) ListResourceShares(c echo.Context) error {
	ownerID := c.Get("user_id").(string)
	resourceType := c.QueryParam("resource_type")
	resourceID := c.QueryParam("resource_id")

	if resourceType == "" || resourceID == "" {
		return SendError(c, http.StatusBadRequest, "resource_type and resource_id query params are required")
	}

	shares, err := h.shareService.ListResourceShares(c.Request().Context(), ownerID, resourceType, resourceID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"shares": shares,
	}, nil)
}

// GET /api/v1/shares/shared-with-me
func (h *ShareHandler) ListSharedWithMe(c echo.Context) error {
	userID := c.Get("user_id").(string)

	shares, err := h.shareService.ListSharedWithMeByUserID(c.Request().Context(), userID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"shares": shares,
	}, nil)
}


// DELETE /api/v1/shares/:id
func (h *ShareHandler) RevokeShare(c echo.Context) error {
	ownerID := c.Get("user_id").(string)
	shareID := c.Param("id")

	if err := h.shareService.RevokeShare(c.Request().Context(), ownerID, shareID); err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message": "Share revoked successfully",
	}, nil)
}
