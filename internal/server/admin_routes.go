package server

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/handler"
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
			return handler.SendError(c, http.StatusInternalServerError, "Failed to get stats")
		}
		return handler.SendSuccess(c, http.StatusOK, stats, nil)
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
			return handler.SendError(c, http.StatusInternalServerError, "Failed to list users")
		}

		pagination := handler.PaginationMetadata{
			Total: total,
			Limit: limit,
			Page:  page,
		}

		return handler.SendSuccess(c, http.StatusOK, map[string]any{"users": users}, pagination)
	}, appMiddleware.RequirePermission("admin:users"))

	// GET /api/v1/admin/roles
	admin.GET("/roles", func(c echo.Context) error {
		roles, err := roleRepo.ListAll(c.Request().Context())
		if err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to list roles")
		}
		return handler.SendSuccess(c, http.StatusOK, map[string]any{"roles": roles}, nil)
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
			return handler.SendError(c, http.StatusInternalServerError, "Failed to list activity")
		}

		pagination := handler.PaginationMetadata{
			Total: total,
			Limit: limit,
			Page:  page,
		}

		return handler.SendSuccess(c, http.StatusOK, map[string]any{"logs": logs}, pagination)
	}, appMiddleware.RequirePermission("admin:activity"))
}
