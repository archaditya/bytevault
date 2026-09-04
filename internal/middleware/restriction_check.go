package middleware

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/repository"
)

// RestrictionCheck blocks restricted users from performing upload/share/create operations.
// Restricted users can still view and download their own files, and submit appeals.
// This middleware should be applied ONLY to write routes (upload, share, create folder), not read routes.
func RestrictionCheck(userRepo *repository.UserRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID, ok := c.Get("user_id").(string)
			if !ok || userID == "" {
				return next(c)
			}

			restricted, until, reason, err := userRepo.IsUserRestricted(c.Request().Context(), userID)
			if err != nil {
				// Don't block the request on DB errors — fail open
				return next(c)
			}

			if !restricted {
				return next(c)
			}

			reasonStr := "Content policy violation"
			if reason != nil && *reason != "" {
				reasonStr = *reason
			}

			untilStr := "indefinitely"
			if until != nil {
				untilStr = until.Format("Jan 2, 2006 3:04 PM")
			}

			return c.JSON(http.StatusForbidden, map[string]any{
				"error":            "Account restricted",
				"detail":           fmt.Sprintf("Your account is restricted until %s due to: %s. You can submit an appeal from your account settings.", untilStr, reasonStr),
				"restricted_until": until,
				"reason":           reasonStr,
				"can_appeal":       true,
			})
		}
	}
}
