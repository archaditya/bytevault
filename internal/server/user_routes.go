package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/archaditya/bytevault/internal/handler"
	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
	"github.com/archaditya/bytevault/internal/storage"
)

func (s *Server) registerUserRoutes(
	protected *Group,
	userRepo *repository.UserRepository,
	deviceRepo *repository.DeviceRepository,
	sessionRepo *repository.SessionRepository,
	fileRepo *repository.FileRepository,
	store storage.StorageProvider,
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

	// PATCH /api/v1/me — update profile (first_name, last_name)
	protected.PATCH("/me", func(c echo.Context) error {
		userID := c.Get("user_id").(string)

		var req struct {
			FirstName *string `json:"first_name"`
			LastName  *string `json:"last_name"`
		}
		if err := c.Bind(&req); err != nil {
			return handler.SendError(c, http.StatusBadRequest, "Invalid request body")
		}

		if err := userRepo.UpdateDetails(c.Request().Context(), userID, req.FirstName, req.LastName, nil, nil); err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to update profile")
		}

		user, err := userRepo.FindByID(c.Request().Context(), userID)
		if err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Profile updated but failed to reload user")
		}

		return handler.SendSuccess(c, http.StatusOK, map[string]any{
			"message": "Profile updated successfully",
			"user":    user,
		}, nil)
	})

	// POST /api/v1/me/avatar — upload user profile avatar
	protected.POST("/me/avatar", func(c echo.Context) error {
		userID := c.Get("user_id").(string)

		file, err := c.FormFile("avatar")
		if err != nil {
			return handler.SendError(c, http.StatusBadRequest, "No avatar file provided in form field 'avatar'")
		}

		src, err := file.Open()
		if err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to read avatar file")
		}
		defer src.Close()

		// Basic content type validation
		contentType := file.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "image/png"
		}

		// Save the file to the storage provider
		storageKey := "avatars/" + userID + "_" + file.Filename
		_, err = store.Upload(c.Request().Context(), storageKey, src, file.Size, contentType)
		if err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to upload avatar to storage: "+err.Error())
		}

				// Update database with the storage key
		if err := userRepo.UpdateAvatarURL(c.Request().Context(), userID, storageKey); err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to save avatar path to user profile")
		}

		// Return proxy URL
		proxyURL := "/api/v1/users/" + userID + "/avatar"

		return handler.SendSuccess(c, http.StatusOK, map[string]any{
			"message":    "Avatar uploaded successfully",
			"avatar_url": proxyURL,
		}, nil)
	})

	// POST /api/v1/me/change-password
	protected.POST("/me/change-password", func(c echo.Context) error {
		userID := c.Get("user_id").(string)

		var req struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if err := c.Bind(&req); err != nil || req.CurrentPassword == "" || req.NewPassword == "" {
			return handler.SendError(c, http.StatusBadRequest, "Current password and new password are required")
		}

		if len(req.NewPassword) < 6 {
			return handler.SendError(c, http.StatusBadRequest, "New password must be at least 6 characters")
		}

		user, err := userRepo.FindByID(c.Request().Context(), userID)
		if err != nil {
			return handler.SendError(c, http.StatusNotFound, "User not found")
		}

		if user.Password == nil {
			return handler.SendError(c, http.StatusBadRequest, "No password set for this account")
		}

		if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(req.CurrentPassword)); err != nil {
			return handler.SendError(c, http.StatusUnauthorized, "Current password is incorrect")
		}

		hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 14)
		if err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to hash new password")
		}
		hashedPassword := string(hashedBytes)

		if err := userRepo.UpdatePassword(c.Request().Context(), userID, hashedPassword); err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to update password")
		}

		return handler.SendSuccess(c, http.StatusOK, map[string]any{
			"message": "Password changed successfully",
		}, nil)
	})

	// GET /api/v1/me/quota
	protected.GET("/me/quota", func(c echo.Context) error {
		userID := c.Get("user_id").(string)
		used, err := fileRepo.GetUserStorageUsed(c.Request().Context(), userID)
		if err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to fetch storage usage")
		}

		remaining := int64(service.DefaultQuotaBytes) - used
		if remaining < 0 {
			remaining = 0
		}

		return handler.SendSuccess(c, http.StatusOK, map[string]any{
			"used_bytes":      used,
			"total_bytes":     int64(service.DefaultQuotaBytes),
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
