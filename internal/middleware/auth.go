package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/service"
)

// Auth validates the JWT token and sets user_id, role, and permissions in context.
func Auth(authService *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var token string
			
			// 1. Try to get token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && parts[0] == "Bearer" {
					token = parts[1]
				}
			}

			// 2. Fallback to token query parameter (useful for iframe previews and downloads)
			if token == "" {
				token = c.QueryParam("token")
			}

			if token == "" {
				return c.JSON(http.StatusUnauthorized, map[string]any{"error": "Authorization header or token query parameter required"})
			}

			// ValidateAccessToken now returns userID, role, and permissions
			claims, err := authService.ValidateAccessToken(token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]any{"error": "Invalid or expired token"})
			}

			c.Set("user_id", claims.UserID)
			c.Set("role", claims.Role)
			c.Set("permissions", claims.Permissions)

			return next(c)
		}
	}
}
