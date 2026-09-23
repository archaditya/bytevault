package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/service"
)

type APIKeyHandler struct {
	service *service.APIKeyService
}

func NewAPIKeyHandler(service *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{service: service}
}

// POST /api/v1/api-keys
func (h *APIKeyHandler) Create(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req model.CreateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	key, err := h.service.CreateKey(c.Request().Context(), userID, req.Name)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusCreated, map[string]interface{}{
		"api_key": key,
		"message": "API key generated successfully. Copy it now; it cannot be shown again.",
	}, nil)
}

// GET /api/v1/api-keys
func (h *APIKeyHandler) List(c echo.Context) error {
	userID := c.Get("user_id").(string)

	keys, err := h.service.ListKeys(c.Request().Context(), userID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"api_keys": keys,
		"max_keys": service.MaxKeysPerUser,
	}, nil)
}

// POST /api/v1/api-keys/:id/rotate
func (h *APIKeyHandler) Rotate(c echo.Context) error {
	userID := c.Get("user_id").(string)
	keyID := c.Param("id")

	key, err := h.service.RotateKey(c.Request().Context(), keyID, userID)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"api_key": key,
		"message": "API key rotated successfully. Previous key is now revoked. Copy new key now.",
	}, nil)
}

// DELETE /api/v1/api-keys/:id
func (h *APIKeyHandler) Delete(c echo.Context) error {
	userID := c.Get("user_id").(string)
	keyID := c.Param("id")

	if err := h.service.DeleteKey(c.Request().Context(), keyID, userID); err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]interface{}{
		"message": "API key permanently revoked and deleted",
	}, nil)
}
