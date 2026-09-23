package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/service"
)

// Auth validates the JWT token OR API Key and sets user_id, role, and permissions in context.
func Auth(authService *service.AuthService, apiKeyService *service.APIKeyService, apiKeyLimiter *APIKeyRateLimiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var token string

			// 0. Check custom X-API-Key header first
			xApiKey := c.Request().Header.Get("X-API-Key")
			if xApiKey != "" {
				token = strings.TrimSpace(xApiKey)
			}

			// 1. Try to get token from Authorization header if not already found
			if token == "" {
				authHeader := c.Request().Header.Get("Authorization")
				if authHeader != "" {
					parts := strings.SplitN(authHeader, " ", 2)
					if len(parts) == 2 && (parts[0] == "Bearer" || parts[0] == "ApiKey") {
						token = strings.TrimSpace(parts[1])
					}
				}
			}

			// 2. Fallback to token query parameter (useful for iframe previews, downloads, and webhooks)
			if token == "" {
				token = c.QueryParam("token")
			}

			if token == "" {
				// Public invoice endpoints (e.g. from email links or shared receipts) can be accessed by UUID
				reqPath := c.Request().URL.Path
				if strings.HasPrefix(reqPath, "/api/v1/invoices") {
					return next(c)
				}
				return c.JSON(http.StatusUnauthorized, map[string]any{"error": "Authorization header or token parameter required"})
			}

			// Check if incoming token is an API Key (starts with ppv_live_ or bv_live_)
			if (strings.HasPrefix(token, "ppv_live_") || strings.HasPrefix(token, "bv_live_")) && apiKeyService != nil {
				apiKey, err := apiKeyService.ValidateKey(c.Request().Context(), token)
				if err != nil {
					return c.JSON(http.StatusUnauthorized, map[string]any{"error": "Invalid or revoked API key"})
				}

				// Apply standard rate limiting for API keys
				if apiKeyLimiter != nil {
					allowed, remaining, retryAfter := apiKeyLimiter.Allow(apiKey.ID, apiKey.RateLimitPerMin)
					c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", apiKey.RateLimitPerMin))
					c.Response().Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
					c.Response().Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", retryAfter))

					if !allowed {
						c.Response().Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
						return c.JSON(http.StatusTooManyRequests, map[string]any{
							"error": fmt.Sprintf("API key rate limit exceeded (%d requests/minute). Please slow down.", apiKey.RateLimitPerMin),
						})
					}
				}

				c.Set("user_id", apiKey.UserID)
				c.Set("role", "user")
				c.Set("permissions", []string{"files:read", "files:write", "files:delete", "folders:read", "folders:write"})
				c.Set("auth_type", "api_key")
				c.Set("api_key_id", apiKey.ID)
				c.Set("api_key_name", apiKey.Name)

				return next(c)
			}

			// Otherwise, validate standard JWT access token
			claims, err := authService.ValidateAccessToken(token)
			if err != nil {
				reqPath := c.Request().URL.Path
				if strings.HasPrefix(reqPath, "/api/v1/invoices") {
					return next(c)
				}
				return c.JSON(http.StatusUnauthorized, map[string]any{"error": "Invalid or expired token"})
			}

			c.Set("user_id", claims.UserID)
			c.Set("role", claims.Role)
			c.Set("permissions", claims.Permissions)
			c.Set("auth_type", "jwt")

			return next(c)
		}
	}
}
