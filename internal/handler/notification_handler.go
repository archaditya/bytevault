package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
)

type NotificationHandler struct {
	authService *service.AuthService
	notifService *service.NotificationService
}

func NewNotificationHandler(authService *service.AuthService, notifService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		authService:  authService,
		notifService: notifService,
	}
}

// POST /api/v1/auth/verify-email
func (h *NotificationHandler) VerifyEmail(c echo.Context) error {
	var req struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}
	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Email == "" || req.OTP == "" {
		return SendError(c, http.StatusBadRequest, "Email and OTP are required")
	}

	// Verify the OTP code
	err := h.notifService.VerifyOTP(c.Request().Context(), req.Email, req.OTP, "registration")
	if err != nil {
		if errors.Is(err, repository.ErrVerificationNotFound) {
			return SendError(c, http.StatusBadRequest, "Invalid email address or verification code")
		}
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	// Fetch verified user details to log them in automatically
	// Generating JWT tokens so that user can directly access dashboard upon verification
	user, err := h.notifService.VerifyOTPLoginDetails(c.Request().Context(), req.Email)
	if err != nil {
		return SendSuccess(c, http.StatusOK, map[string]any{
			"message": "Email verified successfully. Please log in.",
		}, nil)
	}

	// Create Auth Session
	ip := c.RealIP()
	ua := c.Request().UserAgent()
	tokens, err := h.authService.CreateSessionForVerifiedUser(c.Request().Context(), user.ID, user.RoleName, user.Permissions, &ua, &ip)
	if err != nil {
		return SendSuccess(c, http.StatusOK, map[string]any{
			"message": "Email verified successfully. Please log in.",
		}, nil)
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message": "Email verified successfully!",
		"user":    user,
		"tokens":  tokens,
	}, nil)
}

// POST /api/v1/auth/resend-otp
func (h *NotificationHandler) ResendOTP(c echo.Context) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	user, err := h.notifService.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		return SendError(c, http.StatusNotFound, "User not found")
	}

	if user.IsVerified {
		return SendError(c, http.StatusBadRequest, "Email is already verified")
	}

	err = h.notifService.GenerateAndSendOTP(c.Request().Context(), user.ID, user.Email, *user.FirstName, "registration")
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to send verification code")
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message": "Verification code resent successfully",
	}, nil)
}

// GET /api/v1/notifications
func (h *NotificationHandler) ListNotifications(c echo.Context) error {
	userID := c.Get("user_id").(string)

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	notifications, total, err := h.notifService.ListInApp(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to load notifications")
	}

	unread, _ := h.notifService.GetUnreadCount(c.Request().Context(), userID)

	return SendSuccess(c, http.StatusOK, map[string]any{
		"notifications": notifications,
		"total":         total,
		"unread_count":  unread,
	}, nil)
}

// POST /api/v1/notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c echo.Context) error {
	userID := c.Get("user_id").(string)
	notifID := c.Param("id")

	if err := h.notifService.MarkRead(c.Request().Context(), notifID, userID); err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to mark notification as read")
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message": "Notification marked as read",
	}, nil)
}

// POST /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c echo.Context) error {
	userID := c.Get("user_id").(string)

	if err := h.notifService.MarkAllRead(c.Request().Context(), userID); err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to mark notifications as read")
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message": "All notifications marked as read",
	}, nil)
}

// POST /api/v1/push-tokens
func (h *NotificationHandler) RegisterPushToken(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		Token      string `json:"token"`
		DeviceType string `json:"device_type"`
	}
	if err := c.Bind(&req); err != nil || req.Token == "" {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.DeviceType == "" {
		req.DeviceType = "web"
	}

	if err := h.notifService.RegisterPushToken(c.Request().Context(), userID, req.Token, req.DeviceType); err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to register push token")
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message": "Push token registered successfully",
	}, nil)
}

// ForgotPassword handles sending a password reset verification code.
// POST /api/v1/auth/forgot-password
func (h *NotificationHandler) ForgotPassword(c echo.Context) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&req); err != nil || req.Email == "" {
		return SendError(c, http.StatusBadRequest, "Email is required")
	}

	if err := h.notifService.ForgotPassword(c.Request().Context(), req.Email); err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message": "Password reset OTP sent to email",
	}, nil)
}

// ResetPassword verifies the OTP and updates the user password.
// POST /api/v1/auth/reset-password
func (h *NotificationHandler) ResetPassword(c echo.Context) error {
	var req struct {
		Email    string `json:"email"`
		OTP      string `json:"otp"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil || req.Email == "" || req.OTP == "" || req.Password == "" {
		return SendError(c, http.StatusBadRequest, "Email, OTP and password are required")
	}

	if err := h.notifService.ResetPassword(c.Request().Context(), req.Email, req.OTP, req.Password); err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message": "Password reset successfully",
	}, nil)
}

// POST /api/v1/notifications/admin/send
func (h *NotificationHandler) SendAdminNotification(c echo.Context) error {
	var req struct {
		TargetType string   `json:"target_type" query:"target_type"`
		UserID     string   `json:"user_id" query:"user_id"`
		Role       string   `json:"role" query:"role"`
		Title      string   `json:"title" query:"title"`
		Body       string   `json:"body" query:"body"`
		Channels   []string `json:"channels" query:"channels"`
		Priority   string   `json:"priority" query:"priority"`
	}

	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request payload")
	}

	// Bind query parameters fallback for POST query string inputs
	if req.TargetType == "" {
		req.TargetType = c.QueryParam("target_type")
	}
	if req.UserID == "" {
		req.UserID = c.QueryParam("user_id")
	}
	if req.Role == "" {
		req.Role = c.QueryParam("role")
	}
	if req.Title == "" {
		req.Title = c.QueryParam("title")
	}
	if req.Body == "" {
		req.Body = c.QueryParam("body")
	}
	if req.Priority == "" {
		req.Priority = c.QueryParam("priority")
	}
	if len(req.Channels) == 0 {
		req.Channels = c.QueryParams()["channels"]
	}

	if req.TargetType == "" || req.Title == "" || req.Body == "" {
		return SendError(c, http.StatusBadRequest, "target_type, title, and body are required")
	}
	if len(req.Channels) == 0 {
		return SendError(c, http.StatusBadRequest, "At least one delivery channel is required")
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}

	sentCount, notifIDs, err := h.notifService.SendAdminNotification(
		c.Request().Context(),
		req.TargetType,
		req.UserID,
		req.Role,
		req.Title,
		req.Body,
		req.Channels,
		req.Priority,
	)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message":          "Notification sent successfully",
		"sent_count":       sentCount,
		"notification_ids": notifIDs,
	}, nil)
}

// GET /api/v1/admin/notifications
func (h *NotificationHandler) ListAllNotifications(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	notifications, total, err := h.notifService.ListAllNotifications(c.Request().Context(), limit, offset)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to load notifications")
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"notifications": notifications,
		"total":         total,
	}, nil)
}
