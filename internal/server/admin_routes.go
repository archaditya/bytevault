package server

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/handler"
	appMiddleware "github.com/archaditya/bytevault/internal/middleware"
	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
)

func strPtr(s string) *string {
	return &s
}

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

		// Inject role to each listed user
		var enriched []map[string]any
		for _, u := range users {
			roleName := "user"
			role, err := roleRepo.GetUserRole(c.Request().Context(), u.ID)
			if err == nil {
				roleName = role.Name
			}
			enriched = append(enriched, map[string]any{
				"id":          u.ID,
				"email":       u.Email,
				"first_name":  u.FirstName,
				"last_name":   u.LastName,
				"avatar_url":  u.AvatarURL,
				"is_verified": u.IsVerified,
				"status":      u.Status,
				"created_at":  u.CreatedAt,
				"updated_at":  u.UpdatedAt,
				"role":        roleName,
			})
		}

		pagination := handler.PaginationMetadata{
			Total: total,
			Limit: limit,
			Page:  page,
		}

		return handler.SendSuccess(c, http.StatusOK, map[string]any{"users": enriched}, pagination)
	}, appMiddleware.RequirePermission("admin:users"))

	// GET /api/v1/admin/users/:id
	admin.GET("/users/:id", func(c echo.Context) error {
		id := c.Param("id")
		ctx := c.Request().Context()

		u, err := userRepo.FindByID(ctx, id)
		if err != nil {
			return handler.SendError(c, http.StatusNotFound, "User not found")
		}

		roleName := "user"
		roleID := ""
		role, err := roleRepo.GetUserRole(ctx, id)
		if err == nil {
			roleName = role.Name
			roleID = role.ID
		}

		totalFiles, totalStorage, _ := userRepo.GetUserStorageStats(ctx, id)

		return handler.SendSuccess(c, http.StatusOK, map[string]any{
			"user": map[string]any{
				"id":          u.ID,
				"email":       u.Email,
				"first_name":  u.FirstName,
				"last_name":   u.LastName,
				"avatar_url":  u.AvatarURL,
				"is_verified": u.IsVerified,
				"status":      u.Status,
				"created_at":  u.CreatedAt,
				"updated_at":  u.UpdatedAt,
				"role":        roleName,
				"role_id":     roleID,
			},
			"total_files":   totalFiles,
			"total_storage": totalStorage,
		}, nil)
	}, appMiddleware.RequirePermission("admin:users"))

	// PUT /api/v1/admin/users/:id
	admin.PUT("/users/:id", func(c echo.Context) error {
		id := c.Param("id")
		ctx := c.Request().Context()

		var req struct {
			FirstName  *string `json:"first_name"`
			LastName   *string `json:"last_name"`
			Status     *string `json:"status"`
			IsVerified *bool   `json:"is_verified"`
			RoleID     *string `json:"role_id"`
		}

		if err := c.Bind(&req); err != nil {
			return handler.SendError(c, http.StatusBadRequest, "Invalid request body")
		}

		err := userRepo.UpdateDetails(ctx, id, req.FirstName, req.LastName, req.Status, req.IsVerified)
		if err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to update user details")
		}

		if req.RoleID != nil && *req.RoleID != "" {
			err = roleRepo.UpdateUserRole(ctx, id, *req.RoleID)
			if err != nil {
				return handler.SendError(c, http.StatusInternalServerError, "Failed to update user role")
			}
		}

		actorID := c.Get("user_id").(string)
		_ = activityRepo.Log(ctx, &model.ActivityLog{
			UserID:       &actorID,
			Action:       "admin.user.update",
			ResourceType: strPtr("user"),
			ResourceID:   &id,
		})

		return handler.SendSuccess(c, http.StatusOK, map[string]string{"message": "User updated successfully"}, nil)
	}, appMiddleware.RequirePermission("admin:users"))

	// DELETE /api/v1/admin/users/:id
	admin.DELETE("/users/:id", func(c echo.Context) error {
		id := c.Param("id")
		ctx := c.Request().Context()
		actorID := c.Get("user_id").(string)

		if id == actorID {
			return handler.SendError(c, http.StatusBadRequest, "You cannot delete your own account")
		}

		err := userRepo.SoftDelete(ctx, id, actorID)
		if err != nil {
			return handler.SendError(c, http.StatusInternalServerError, "Failed to delete user")
		}

		_ = activityRepo.Log(ctx, &model.ActivityLog{
			UserID:       &actorID,
			Action:       "admin.user.delete",
			ResourceType: strPtr("user"),
			ResourceID:   &id,
		})

		return handler.SendSuccess(c, http.StatusOK, map[string]string{"message": "User deleted successfully"}, nil)
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
