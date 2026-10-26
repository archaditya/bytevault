package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/service"
	"github.com/archaditya/bytevault/internal/repository"
)

type AuthHandler struct {
	authService    *service.AuthService
	userRepo       *repository.UserRepository
	consentService *service.ConsentService
}

func NewAuthHandler(authservice *service.AuthService, userRepo ...*repository.UserRepository) *AuthHandler {
	h := &AuthHandler{authService: authservice}

	if len(userRepo) > 0 {
		h.userRepo = userRepo[0]
	}
	return h
}

func (h *AuthHandler) SetConsentService(cs *service.ConsentService) {
	h.consentService = cs
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c echo.Context) error {
	var req struct {
		Email             string `json:"email"`
		Password          string `json:"password"`
		FirstName         string `json:"first_name"`
		LastName          string `json:"last_name"`
		TermsAccepted     *bool  `json:"terms_accepted"`
		IsAdult           *bool  `json:"is_adult"`
		MarketingAccepted *bool  `json:"marketing_accepted"`
	}

	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return SendError(c, http.StatusBadRequest, "Email and password are required")
	}
	if len(req.Password) < 8 {
		return SendError(c, http.StatusBadRequest, "Password must be at least 8 characters")
	}

	// Validate DPDP compliance: user must be 18+ and accept terms/privacy
	if req.TermsAccepted != nil && !*req.TermsAccepted {
		return SendError(c, http.StatusBadRequest, "You must accept the Terms of Service and Privacy Policy")
	}
	if req.IsAdult != nil && !*req.IsAdult {
		return SendError(c, http.StatusBadRequest, "You must confirm you are 18 years of age or older")
	}

	ip := c.RealIP()
	ua := c.Request().UserAgent()

	user, tokens, err := h.authService.Register(c.Request().Context(), req.Email, req.Password, req.FirstName, req.LastName, &ip, &ua)
	if err != nil {
		if errors.Is(err, service.ErrEmailExists) {
			return SendError(c, http.StatusConflict, "Email already registered")
		}
		return SendError(c, http.StatusInternalServerError, fmt.Sprintf("Registration failed: %v", err))
	}

	// Log affirmative consent if consentService is configured
	if h.consentService != nil {
		_, _ = h.consentService.RecordGrant(c.Request().Context(), user.ID, service.ConsentTypeTermsAndPrivacy, service.CurrentPolicyVersion, ip, ua)
		if req.MarketingAccepted != nil && *req.MarketingAccepted {
			_, _ = h.consentService.RecordGrant(c.Request().Context(), user.ID, service.ConsentTypeMarketing, service.CurrentPolicyVersion, ip, ua)
		}
	}

	return SendSuccess(c, http.StatusCreated, map[string]any{
		"user":   user,
		"tokens": tokens,
	}, nil)
}

// POST /api/v1/auth/consent
func (h *AuthHandler) RecordConsent(c echo.Context) error {
	if h.consentService == nil {
		return SendError(c, http.StatusInternalServerError, "Consent service not configured")
	}

	var req struct {
		UserID      string `json:"user_id"`
		ConsentType string `json:"consent_type"`
		Version     string `json:"version"`
		Status      string `json:"status"` // "granted", "revoked"
	}

	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	if authUserID, ok := c.Get("user_id").(string); ok && authUserID != "" {
		req.UserID = authUserID
	}

	if req.UserID == "" || req.ConsentType == "" {
		return SendError(c, http.StatusBadRequest, "user_id and consent_type are required")
	}

	ip := c.RealIP()
	ua := c.Request().UserAgent()

	if req.Status == "revoked" {
		if err := h.consentService.Revoke(c.Request().Context(), req.UserID, req.ConsentType); err != nil {
			return SendError(c, http.StatusInternalServerError, "Failed to revoke consent")
		}
		return SendSuccess(c, http.StatusOK, map[string]string{"message": "Consent revoked"}, nil)
	}

	consent, err := h.consentService.RecordGrant(c.Request().Context(), req.UserID, req.ConsentType, req.Version, ip, ua)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to record consent")
	}

	return SendSuccess(c, http.StatusOK, map[string]any{"consent": consent}, nil)
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(c echo.Context) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return SendError(c, http.StatusBadRequest, "Email and password are required")
	}

	// Pass user-agent and IP for session tracking
	userAgent := c.Request().UserAgent()
	ip := c.RealIP()

	user, tokens, err := h.authService.Login(c.Request().Context(), req.Email, req.Password, &userAgent, &ip)
	if err != nil {
		if errors.Is(err, service.ErrMFARequired) {
			return SendSuccess(c, http.StatusOK, map[string]any{
				"mfa_required": true,
				"mfa_token":    tokens.AccessToken,
				"user":         user,
			}, nil)
		}
		if errors.Is(err, service.ErrInvalidCredentials) {
			return SendError(c, http.StatusUnauthorized, "Invalid email or password")
		}
		return SendError(c, http.StatusInternalServerError, fmt.Sprintf("Login failed: %v", err))
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"user":   user,
		"tokens": tokens,
	}, nil)
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&req); err != nil || req.RefreshToken == "" {
		return SendError(c, http.StatusBadRequest, "Refresh token is required")
	}

	tokens, err := h.authService.RefreshTokens(c.Request().Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			return SendError(c, http.StatusUnauthorized, "Invalid or expired refresh token")
		}
		return SendError(c, http.StatusInternalServerError, fmt.Sprintf("Token refresh failed: %v", err))
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"tokens": tokens,
	}, nil)
}

// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&req); err != nil || req.RefreshToken == "" {
		return SendError(c, http.StatusBadRequest, "Refresh token is required")
	}

	if err := h.authService.Logout(c.Request().Context(), req.RefreshToken); err != nil {
		return SendError(c, http.StatusInternalServerError, fmt.Sprintf("Logout failed: %v", err))
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message": "Logged out successfully",
	}, nil)
}

// POST /api/v1/auth/google
func (h *AuthHandler) GoogleLogin(c echo.Context) error {
	var req struct {
		IDToken   string  `json:"id_token"`
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
		AvatarURL *string `json:"avatar_url"`
	}
	if err := c.Bind(&req); err != nil || req.IDToken == "" {
		return SendError(c, http.StatusBadRequest, "Google ID token is required")
	}

	ip := c.RealIP()
	ua := c.Request().UserAgent()

	// Pass frontend profile details as a fallback to the auth service
	user, tokens, err := h.authService.GoogleLogin(c.Request().Context(), req.IDToken, req.FirstName, req.LastName, req.AvatarURL, &ua, &ip)
	if err != nil {
		return SendError(c, http.StatusUnauthorized, fmt.Sprintf("Google login failed: %v", err))
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"user":   user,
		"tokens": tokens,
	}, nil)
}

// POST /api/v1/auth/mfa/setup (Protected)
func (h *AuthHandler) MFASetup(c echo.Context) error {
	userID := c.Get("user_id").(string)

	secret, qrURI, err := h.authService.SetupMFA(c.Request().Context(), userID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"secret": secret,
		"qr_uri": qrURI,
	}, nil)
}

// POST /api/v1/auth/mfa/enable (Protected)
func (h *AuthHandler) MFAEnable(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		Code string `json:"code"`
	}
	if err := c.Bind(&req); err != nil || req.Code == "" {
		return SendError(c, http.StatusBadRequest, "TOTP code is required")
	}

	if err := h.authService.EnableMFA(c.Request().Context(), userID, req.Code); err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message": "MFA enabled successfully",
	}, nil)
}

// POST /api/v1/auth/mfa/disable (Protected)
func (h *AuthHandler) MFADisable(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req struct {
		Code string `json:"code"`
	}
	if err := c.Bind(&req); err != nil || req.Code == "" {
		return SendError(c, http.StatusBadRequest, "TOTP code is required")
	}

	if err := h.authService.DisableMFA(c.Request().Context(), userID, req.Code); err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"message": "MFA disabled successfully",
	}, nil)
}

// POST /api/v1/auth/mfa/verify-login (Public)
func (h *AuthHandler) MFAVerifyLogin(c echo.Context) error {
	var req struct {
		MFAToken string `json:"mfa_token"`
		Code     string `json:"code"`
	}
	if err := c.Bind(&req); err != nil || req.MFAToken == "" || req.Code == "" {
		return SendError(c, http.StatusBadRequest, "mfa_token and code are required")
	}

	ip := c.RealIP()
	ua := c.Request().UserAgent()

	user, tokens, err := h.authService.VerifyMFALogin(c.Request().Context(), req.MFAToken, req.Code, &ua, &ip)
	if err != nil {
		return SendError(c, http.StatusUnauthorized, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]any{
		"user":   user,
		"tokens": tokens,
	}, nil)
}
