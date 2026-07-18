package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
)

type AdminHandler struct {
	userRepo     *repository.UserRepository
	roleRepo     *repository.RoleRepository
	sessionRepo  *repository.SessionRepository
	activityRepo *repository.ActivityRepository
	fileRepo     *repository.FileRepository
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	sessionRepo *repository.SessionRepository,
	activityRepo *repository.ActivityRepository,
	fileRepo *repository.FileRepository,
) *AdminHandler {
	return &AdminHandler{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		sessionRepo:  sessionRepo,
		activityRepo: activityRepo,
		fileRepo:     fileRepo,
	}
}

// Helper to get pointer to string
func strPtr(s string) *string {
	return &s
}

// GET /api/v1/admin/stats
func (h *AdminHandler) GetStats(c echo.Context) error {
	ctx := c.Request().Context()
	stats, err := h.userRepo.GetStats(ctx)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to get stats")
	}
	return SendSuccess(c, http.StatusOK, stats, nil)
}

// GET /api/v1/admin/users
func (h *AdminHandler) ListUsers(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	search := c.QueryParam("q")
	status := c.QueryParam("status")
	role := c.QueryParam("role")

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	users, total, err := h.userRepo.ListAll(c.Request().Context(), search, status, role, limit, offset)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to list users")
	}

	var enriched []map[string]any
	for _, u := range users {
		roleName := "user"
		roleInfo, err := h.roleRepo.GetUserRole(c.Request().Context(), u.ID)
		if err == nil {
			roleName = roleInfo.Name
		}
		enriched = append(enriched, map[string]any{
			"id":          u.ID,
			"email":       u.Email,
			"first_name":  u.FirstName,
			"last_name":   u.LastName,
			"avatar_url":  u.AvatarURL,
			"is_verified": u.IsVerified,
			"status":      u.Status,
			"storage_limit_bytes": u.StorageLimitBytes,
			"max_file_size_bytes": u.MaxFileSizeBytes,
			"created_at":  u.CreatedAt,
			"updated_at":  u.UpdatedAt,
			"role":        roleName,
		})
	}

	pagination := PaginationMetadata{
		Total: total,
		Limit: limit,
		Page:  page,
	}

	return SendSuccess(c, http.StatusOK, map[string]any{"users": enriched}, pagination)
}

// GET /api/v1/admin/users/:id
func (h *AdminHandler) GetUserDetail(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	u, err := h.userRepo.FindByID(ctx, id)
	if err != nil {
		return SendError(c, http.StatusNotFound, "User not found")
	}

	roleName := "user"
	roleID := ""
	role, err := h.roleRepo.GetUserRole(ctx, id)
	if err == nil {
		roleName = role.Name
		roleID = role.ID
	}

	totalFiles, totalStorage, _ := h.userRepo.GetUserStorageStats(ctx, id)

	return SendSuccess(c, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":          u.ID,
			"email":       u.Email,
			"first_name":  u.FirstName,
			"last_name":   u.LastName,
			"avatar_url":  u.AvatarURL,
			"is_verified": u.IsVerified,
			"status":      u.Status,
			"storage_limit_bytes": u.StorageLimitBytes,
			"max_file_size_bytes": u.MaxFileSizeBytes,
			"created_at":  u.CreatedAt,
			"updated_at":  u.UpdatedAt,
			"role":        roleName,
			"role_id":     roleID,
		},
		"total_files":   totalFiles,
		"total_storage": totalStorage,
	}, nil)
}

// PUT /api/v1/admin/users/:id
func (h *AdminHandler) UpdateUser(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	var req struct {
		FirstName         *string `json:"first_name"`
		LastName          *string `json:"last_name"`
		Status            *string `json:"status"`
		IsVerified        *bool   `json:"is_verified"`
		RoleID            *string `json:"role_id"`
		StorageLimitBytes *int64  `json:"storage_limit_bytes"`
		MaxFileSizeBytes  *int64  `json:"max_file_size_bytes"`
	}

	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	err := h.userRepo.UpdateDetails(ctx, id, req.FirstName, req.LastName, req.Status, req.IsVerified, req.StorageLimitBytes, req.MaxFileSizeBytes)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to update user details")
	}

	if req.RoleID != nil && *req.RoleID != "" {
		err = h.roleRepo.UpdateUserRole(ctx, id, *req.RoleID)
		if err != nil {
			return SendError(c, http.StatusInternalServerError, "Failed to update user role")
		}
	}

	actorID := c.Get("user_id").(string)
	_ = h.activityRepo.Log(ctx, &model.ActivityLog{
		UserID:       &actorID,
		Action:       "admin.user.update",
		ResourceType: strPtr("user"),
		ResourceID:   &id,
	})

	return SendSuccess(c, http.StatusOK, map[string]string{"message": "User updated successfully"}, nil)
}

// DELETE /api/v1/admin/users/:id
func (h *AdminHandler) DeleteUser(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()
	actorID := c.Get("user_id").(string)

	if id == actorID {
		return SendError(c, http.StatusBadRequest, "You cannot delete your own account")
	}

	err := h.userRepo.SoftDelete(ctx, id, actorID)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to delete user")
	}

	_ = h.activityRepo.Log(ctx, &model.ActivityLog{
		UserID:       &actorID,
		Action:       "admin.user.delete",
		ResourceType: strPtr("user"),
		ResourceID:   &id,
	})

	return SendSuccess(c, http.StatusOK, map[string]string{"message": "User deleted successfully"}, nil)
}

// GET /api/v1/admin/roles
func (h *AdminHandler) ListRoles(c echo.Context) error {
	roles, err := h.roleRepo.ListAll(c.Request().Context())
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to list roles")
	}
	return SendSuccess(c, http.StatusOK, map[string]any{"roles": roles}, nil)
}

// GET /api/v1/admin/activity
func (h *AdminHandler) ListActivity(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	offset := (page - 1) * limit

	logs, total, err := h.activityRepo.ListAll(c.Request().Context(), limit, offset)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to list activity")
	}

	pagination := PaginationMetadata{
		Total: total,
		Limit: limit,
		Page:  page,
	}

	return SendSuccess(c, http.StatusOK, map[string]any{"logs": logs}, pagination)
}

// GET /api/v1/admin/files
func (h *AdminHandler) ListAllFiles(c echo.Context) error {
	search := c.QueryParam("q")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	cursor := c.QueryParam("cursor")
	if limit < 1 {
		limit = 20
	}

	files, nextCursor, err := h.fileRepo.ListAllFiles(c.Request().Context(), search, limit, cursor)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to list files")
	}

	return SendSuccess(c, http.StatusOK, map[string]any{"files": files}, PaginationMetadata{
		Limit: limit, NextCursor: nextCursor,
	})
}

// GET /api/v1/admin/files/shared
func (h *AdminHandler) ListSharedFiles(c echo.Context) error {
	search := c.QueryParam("q")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	cursor := c.QueryParam("cursor")
	if limit < 1 {
		limit = 20
	}

	files, nextCursor, err := h.fileRepo.ListAllSharedFiles(c.Request().Context(), search, limit, cursor)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to list shared files")
	}

	return SendSuccess(c, http.StatusOK, map[string]any{"files": files}, PaginationMetadata{
		Limit: limit, NextCursor: nextCursor,
	})
}
