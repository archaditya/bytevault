package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/service"
)

// Auth validates the JWT token and sets user_id, role, and permissions in context.
// All three values are available to handlers via c.Get("user_id"), c.Get("role"), c.Get("permissions")
func Auth(authService *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]any{"error": "Authorization header required"})
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]any{"error": "Invalid authorization format"})
			}

			// ValidateAccessToken now returns userID, role, and permissions
			claims, err := authService.ValidateAccessToken(parts[1])
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