package server

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
	"github.com/archaditya/bytevault/internal/repository"
)

func (s *Server) registerAdminRoutes(
	protected *Group,
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	sessionRepo *repository.SessionRepository,
	activityRepo *repository.ActivityRepository,
) {
	admin := protected.Group("/admin")

	// GET /api/v1/admin/stats
	admin.GET("/stats", func(c echo.Context) error {
		ctx := c.Request().Context()
		stats, err := userRepo.GetStats(ctx)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to get stats"})
		}
		return c.JSON(http.StatusOK, stats)
	}, appMiddleware.RequirePermission("admin:users"))

	// GET /api/v1/admin/users?page=1&limit=20
	admin.GET("/users", func(c echo.Context) error {
		page, _ := strconv.Atoi(c.QueryParam("page"))
		limit, _ := strconv.Atoi(c.QueryParam("limit"))
		if page < 1 { page = 1 }
		if limit < 1 { limit = 20 }
		offset := (page - 1) * limit

		users, total, err := userRepo.ListAll(c.Request().Context(), limit, offset)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to list users"})
		}
		return c.JSON(http.StatusOK, map[string]any{"users": users, "total": total, "page": page, "limit": limit})
	}, appMiddleware.RequirePermission("admin:users"))

	// GET /api/v1/admin/roles
	admin.GET("/roles", func(c echo.Context) error {
		roles, err := roleRepo.ListAll(c.Request().Context())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to list roles"})
		}
		return c.JSON(http.StatusOK, map[string]any{"roles": roles})
	}, appMiddleware.RequirePermission("admin:roles"))

	// GET /api/v1/admin/activity?page=1&limit=50
	admin.GET("/activity", func(c echo.Context) error {
		page, _ := strconv.Atoi(c.QueryParam("page"))
		limit, _ := strconv.Atoi(c.QueryParam("limit"))
		if page < 1 { page = 1 }
		if limit < 1 { limit = 50 }
		offset := (page - 1) * limit

		logs, total, err := activityRepo.ListAll(c.Request().Context(), limit, offset)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to list activity"})
		}
		return c.JSON(http.StatusOK, map[string]any{"logs": logs, "total": total, "page": page, "limit": limit})
	}, appMiddleware.RequirePermission("admin:activity"))
}