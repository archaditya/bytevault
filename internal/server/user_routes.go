package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/handler"
	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
)

const DefaultQuotaBytes = 5 * 1024 * 1024 * 1024 // 5 GB

func (s *Server) registerUserRoutes(
	protected *Group,
	userRepo *repository.UserRepository,
	deviceRepo *repository.DeviceRepository,
	sessionRepo *repository.SessionRepository,
	fileRepo *repository.FileRepository,
) {
	// GET /api/v1/me
	protected.GET("/me", func(c echo.Context) error {
		userID := c.Get("user_id").(string)
		role := c.Get("role").(string)
		perms := c.Get("permissions").(map[string]bool)

		user, err := userRepo.FindByID(c.Request().Context(), userID)
		if err != nil {
			return handler.SendError(c, http.StatusNotFound, "User not found")
		}
		user.RoleName = role
		user.Permissions = perms

		return handler.SendSuccess(c, http.StatusOK, map[string]any{"user": user}, nil)
	})

	// GET /api/v1/me/quota
	protected.GET("/me/quota", func(c echo.Context) error {
		userID := c.Get("user_id").(string)
		used, err := fileRepo.GetUserStorageUsed(c.Request().Context(), userID)
		if err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to fetch storage usage")
		}

		remaining := int64(DefaultQuotaBytes) - used
		if remaining < 0 {
			remaining = 0
		}

		return handler.SendSuccess(c, http.StatusOK, map[string]any{
			"used_bytes":      used,
			"total_bytes":     int64(DefaultQuotaBytes),
			"remaining_bytes": remaining,
		}, nil)
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
			return handler.SendError(c, http.StatusBadRequest, "fcm_token and device_type are required")
		}

		device := &model.UserDevice{
			UserID:     userID,
			FcmToken:   req.FcmToken,
			DeviceType: req.DeviceType,
			DeviceID:   req.DeviceID,
		}
		if err := deviceRepo.Upsert(c.Request().Context(), device); err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to register device")
		}

		return handler.SendSuccess(c, http.StatusOK, map[string]any{"message": "Device registered"}, nil)
	})

	// GET /api/v1/me/devices
	protected.GET("/me/devices", func(c echo.Context) error {
		userID := c.Get("user_id").(string)
		devices, err := deviceRepo.FindByUserID(c.Request().Context(), userID)
		if err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to get devices")
		}
		return handler.SendSuccess(c, http.StatusOK, map[string]any{"devices": devices}, nil)
	})

	// DELETE /api/v1/me/devices/:id
	protected.DELETE("/me/devices/:id", func(c echo.Context) error {
		userID := c.Get("user_id").(string)
		deviceID := c.Param("id")
		deviceRepo.Deactivate(c.Request().Context(), deviceID, userID)
		return handler.SendSuccess(c, http.StatusOK, map[string]any{"message": "Device removed"}, nil)
	})
}
