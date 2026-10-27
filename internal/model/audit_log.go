package model

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID           string          `json:"id"`
	TenantID     *string         `json:"tenant_id,omitempty"`
	ActorUserID  *string         `json:"actor_user_id,omitempty"`
	ActorType    string          `json:"actor_type"` // 'user', 'admin', 'system', 'api_key'
	Action       string          `json:"action"`     // e.g. 'file.create', 'file.delete', 'auth.login'
	ResourceType string          `json:"resource_type"` // e.g. 'file', 'folder', 'user', 'share'
	ResourceID   *string         `json:"resource_id,omitempty"`
	Status       string          `json:"status"`     // 'success', 'failure'
	RequestID    *string         `json:"request_id,omitempty"`
	IPAddress    *string         `json:"ip_address,omitempty"`
	UserAgent    *string         `json:"user_agent,omitempty"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedAt    time.Time       `json:"created_at"`
}

type AuditLogFilter struct {
	TenantID     *string
	ActorUserID  *string
	Action       *string
	ResourceType *string
	Status       *string
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int
	Offset       int
}
