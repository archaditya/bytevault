package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/adityakkpk/bytevault/internal/repository"
)

// registerUserRoutes adds all user-related protected endpoints.
// The "protected" group already has the Auth middleware applied,
// so every route here automatically requires a valid JWT.
func (s *Server) registerUserRoutes(protected *Group, userRepo *repository.UserRepository) {
	// GET /api/v1/me — returns the currently logged-in user
	// c.Get("user_id") works because Auth middleware sets it after validating the token
	protected.GET("/me", func(c echo.Context) error {
		userID := c.Get("user_id").(string)

		user, err := userRepo.FindByID(c.Request().Context(), userID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]any{"error": "User not found"})
		}

		return c.JSON(http.StatusOK, map[string]any{"user": user})
	})

	// Future user routes go here:
	// protected.PUT("/me", userHandler.UpdateProfile)
	// protected.DELETE("/me", userHandler.DeleteAccount)
}
