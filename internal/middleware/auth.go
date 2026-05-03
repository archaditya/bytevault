package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/adityakkpk/bytevault/internal/service"
)

func Auth(authService *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]any{"error":"Authorization header required"})
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]any{"error": "Invalid authorization format"})
			}

			userID, err := authService.ValidateAccessToken(parts[1])
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]any{"error":"Invalid or expired token"})
			}

			c.Set("user_id", userID)

			return next(c)
		}
	}
}