package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authservice *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authservice,
	}
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c echo.Context) error {
	var req struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Email and password are required"})
	}
	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Password must be at least 8 characters"})
	}

	ip := c.RealIP()
	ua := c.Request().UserAgent()

	user, tokens, err := h.authService.Register(c.Request().Context(), req.Email, req.Password, req.FirstName, req.LastName, &ip, &ua)
	if err != nil {
		if errors.Is(err, service.ErrEmailExists) {
			return c.JSON(http.StatusConflict, map[string]any{"error": "Email already registered"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Registration failed"})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"user":   user,
		"tokens": tokens,
	})
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(c echo.Context) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Email and password are required"})
	}

	// Pass user-agent and IP for session tracking
	userAgent := c.Request().UserAgent()
	ip := c.RealIP()

	user, tokens, err := h.authService.Login(c.Request().Context(), req.Email, req.Password, &userAgent, &ip)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return c.JSON(http.StatusUnauthorized, map[string]any{"error": "Invalid email or password"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Login failed"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"user":   user,
		"tokens": tokens,
	})
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&req); err != nil || req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Refresh token is required"})
	}

	tokens, err := h.authService.RefreshTokens(c.Request().Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			return c.JSON(http.StatusUnauthorized, map[string]any{"error": "Invalid or expired refresh token"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Token refresh failed"})
	}

	return c.JSON(http.StatusOK, map[string]any{"tokens": tokens})
}

// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&req); err != nil || req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Refresh token is required"})
	}

	if err := h.authService.Logout(c.Request().Context(), req.RefreshToken); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Logout failed"})
	}

	return c.JSON(http.StatusOK, map[string]any{"message": "Logged out successfully"})
}