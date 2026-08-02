package model

import (
	"time"
)

// User represents a row in the users table.
//
// WHAT ARE JSON TAGS?
// `json:"id"` tells Go's JSON encoder to use "id" as the key name.
// Without it, the JSON key would be "ID" (matching the Go field name).
//
// `json:"-"` means NEVER include this field in JSON output.
// We use it for password — you never want to send passwords in API responses!
//
// WHAT IS *time.Time?
// A POINTER to time.Time. Pointers can be nil (null).
// Regular time.Time can't be nil — it always has a value.
// Since deleted_at can be NULL in the database, we use *time.Time.
type User struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Password  *string    `json:"-"` // Never expose in JSON, pointer because nullable
	HasPassword bool     `json:"has_password"`

	FirstName *string    `json:"first_name"`
	LastName  *string    `json:"last_name"`
	AvatarURL *string    `json:"avatar_url"`

	IsVerified bool      `json:"is_verified"`
	Status     *string   `json:"status"`

	MfaEnabled bool    `json:"mfa_enabled"`
 	MfaSecret  *string `json:"-"` // Never expose in JSON

	RoleName    string          `json:"role"`
	Permissions map[string]bool `json:"permissions,omitempty"`

	StorageLimitBytes *int64 `json:"storage_limit_bytes"` // User customized storage limit (bytes)
	MaxFileSizeBytes  *int64 `json:"max_file_size_bytes"` // Per-user max single file upload size (bytes)

	CreatedBy  *string   `json:"-"`
	UpdatedBy  *string   `json:"-"`
	DeletedBy  *string   `json:"-"`

	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"-"` // Pointer = nullable (soft delete)
}
