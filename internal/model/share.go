package model

import "time"

type Share struct {
	ID           string    `json:"id"`
	ResourceType string    `json:"resource_type"` // 'file' or 'folder'
	ResourceID   string    `json:"resource_id"`
	OwnerID      string    `json:"owner_id"`
	GranteeEmail string    `json:"grantee_email"`
	GranteeID    *string   `json:"grantee_id,omitempty"`
	Permission   string    `json:"permission"` // 'VIEWER' or 'EDITOR'
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
