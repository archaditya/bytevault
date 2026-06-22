package model

import "time"

type Role struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  *string         `json:"description"`
	Permissions  map[string]bool `json:"permissions"`
	IsSystemRole bool            `json:"is_system_role"`
	CreatedBy    *string         `json:"-"`
	UpdatedBy    *string         `json:"-"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// HasPermission checks if this role allows a specific permission.
// Example: role.HasPermission("admin:users") → true/false
func (r *Role) HasPermission(perm string) bool {
	if r.Permissions == nil {
		return false
	}
	return r.Permissions[perm]
}

// UserRole represents the mapping between a user and a role.
type UserRole struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	RoleID     string    `json:"role_id"`
	AssignedBy *string   `json:"assigned_by"`
	CreatedAt  time.Time `json:"created_at"`
}