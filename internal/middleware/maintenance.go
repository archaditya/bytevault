package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/repository"
)

// MaintenanceModeMiddleware intercepts non-exempt requests when maintenance_mode is enabled.
// Exempt paths: /health, /static, /api/v1/health, /api/v1/auth/login, /api/v1/admin/*
func MaintenanceModeMiddleware(repo *repository.SystemSettingRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path

			// Always allow health checks, static assets, admin console routes, and login
			if strings.HasPrefix(path, "/health") ||
				strings.HasPrefix(path, "/api/v1/health") ||
				strings.HasPrefix(path, "/static") ||
				strings.HasPrefix(path, "/api/v1/admin") ||
				path == "/api/v1/auth/login" {
				return next(c)
			}

			// If user is authenticated as super_admin or admin, allow them through
			userRole, _ := c.Get("role").(string)
			if userRole == "super_admin" || userRole == "admin" {
				return next(c)
			}

			if repo != nil && repo.IsMaintenanceMode(c.Request().Context()) {
				return c.JSON(http.StatusServiceUnavailable, map[string]any{
					"success": false,
					"message": "PushPostVault is currently under scheduled maintenance. Please check back shortly.",
					"error":   "maintenance_mode",
				})
			}

			return next(c)
		}
	}
}
