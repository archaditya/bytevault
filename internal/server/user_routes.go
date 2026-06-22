package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
)

func (s *Server) registerUserRoutes(
	protected *Group,
	userRepo *repository.UserRepository,
	deviceRepo *repository.DeviceRepository,
	sessionRepo *repository.SessionRepository,
) {
	// GET /api/v1/me
	protected.GET("/me", func(c echo.Context) error {
		userID := c.Get("user_id").(string)
		role := c.Get("role").(string)
		perms := c.Get("permissions").(map[string]bool)

		user, err := userRepo.FindByID(c.Request().Context(), userID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]any{"error": "User not found"})
		}
		user.RoleName = role
		user.Permissions = perms

		return c.JSON(http.StatusOK, map[string]any{"user": user})
	})

	// POST /api/v1/me/devices — register FCM token
	protected.POST("/me/devices", func(c echo.Context) error {
		userID := c.Get("user_id").(string)

		var req struct {
			FcmToken   string  `json:"fcm_token"`
			DeviceType string  `json:"device_type"`
			DeviceID   *string `json:"device_id"`
		}
		if err := c.Bind(&req); err != nil || req.FcmToken == "" || req.DeviceType == "" {
			return c.JSON(http.StatusBadRequest, map[string]any{"error": "fcm_token and device_type are required"})
		}

		device := &model.UserDevice{
			UserID:     userID,
			FcmToken:   req.FcmToken,
			DeviceType: req.DeviceType,
			DeviceID:   req.DeviceID,
		}
		if err := deviceRepo.Upsert(c.Request().Context(), device); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to register device"})
		}

		return c.JSON(http.StatusOK, map[string]any{"message": "Device registered"})
	})

	// GET /api/v1/me/devices
	protected.GET("/me/devices", func(c echo.Context) error {
		userID := c.Get("user_id").(string)
		devices, err := deviceRepo.FindByUserID(c.Request().Context(), userID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to get devices"})
		}
		return c.JSON(http.StatusOK, map[string]any{"devices": devices})
	})

	// DELETE /api/v1/me/devices/:id
	protected.DELETE("/me/devices/:id", func(c echo.Context) error {
		userID := c.Get("user_id").(string)
		deviceID := c.Param("id")
		deviceRepo.Deactivate(c.Request().Context(), deviceID, userID)
		return c.JSON(http.StatusOK, map[string]any{"message": "Device removed"})
	})
}