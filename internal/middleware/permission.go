package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func RequirePermission(permission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			perms, ok := c.Get("permissions").(map[string]bool)
			if !ok {
				return c.JSON(http.StatusForbidden, map[string]any{
					"error": "No permissions found",
				})
			}

			if !perms[permission] {
				return c.JSON(http.StatusForbidden, map[string]any{
					"error": "You don't have permission: " + permission,
				})
			}

			return next(c)
		}
	}
}