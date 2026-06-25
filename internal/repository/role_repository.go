package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/archaditya/bytevault/internal/model"
)

var ErrRoleNotFound = errors.New("role not found")

type RoleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{db: db}
}

// FindByName finds a role by its name (e.g., "user", "super_admin")
func (r *RoleRepository) FindByName(ctx context.Context, name string) (*model.Role, error) {
	query := `SELECT id, name, description, permissions, is_system_role, created_at, updated_at
	          FROM roles WHERE name = $1`

	var role model.Role
	var permJSON []byte // JSONB comes as raw bytes from pgx

	err := r.db.QueryRow(ctx, query, name).Scan(
		&role.ID, &role.Name, &role.Description,
		&permJSON, &role.IsSystemRole,
		&role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to find role: %w", err)
	}

	// Parse JSONB bytes into Go map
	if err := json.Unmarshal(permJSON, &role.Permissions); err != nil {
		return nil, fmt.Errorf("failed to parse permissions: %w", err)
	}

	return &role, nil
}

// FindByID finds a role by UUID
func (r *RoleRepository) FindByID(ctx context.Context, id string) (*model.Role, error) {
	query := `SELECT id, name, description, permissions, is_system_role, created_at, updated_at
	          FROM roles WHERE id = $1`

	var role model.Role
	var permJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&role.ID, &role.Name, &role.Description,
		&permJSON, &role.IsSystemRole,
		&role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to find role: %w", err)
	}

	if err := json.Unmarshal(permJSON, &role.Permissions); err != nil {
		return nil, fmt.Errorf("failed to parse permissions: %w", err)
	}

	return &role, nil
}

// ListAll returns all roles
func (r *RoleRepository) ListAll(ctx context.Context) ([]model.Role, error) {
	query := `SELECT id, name, description, permissions, is_system_role, created_at, updated_at
	          FROM roles ORDER BY created_at`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close() // Always close rows when done

	var roles []model.Role
	for rows.Next() {
		var role model.Role
		var permJSON []byte

		if err := rows.Scan(
			&role.ID, &role.Name, &role.Description,
			&permJSON, &role.IsSystemRole,
			&role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}

		if err := json.Unmarshal(permJSON, &role.Permissions); err != nil {
			return nil, fmt.Errorf("failed to parse permissions: %w", err)
		}

		roles = append(roles, role)
	}

	return roles, nil
}

// AssignRoleToUser inserts a row into user_roles
func (r *RoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID string, assignedBy *string) error {
	query := `INSERT INTO user_roles (user_id, role_id, assigned_by) VALUES ($1, $2, $3)
	          ON CONFLICT (user_id, role_id) DO NOTHING`

	_, err := r.db.Exec(ctx, query, userID, roleID, assignedBy)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}
	return nil
}

// GetUserRole returns the FIRST role assigned to a user (with permissions)
// In our system, a user typically has one primary role.
func (r *RoleRepository) GetUserRole(ctx context.Context, userID string) (*model.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.permissions, r.is_system_role, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		LIMIT 1
	`

	var role model.Role
	var permJSON []byte

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&role.ID, &role.Name, &role.Description,
		&permJSON, &role.IsSystemRole,
		&role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get user role: %w", err)
	}

	if err := json.Unmarshal(permJSON, &role.Permissions); err != nil {
		return nil, fmt.Errorf("failed to parse permissions: %w", err)
	}

	return &role, nil
}