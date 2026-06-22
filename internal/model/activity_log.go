package model

import "time"

// ActivityLog records every important action in the system.
// This is your audit trail — who did what, when, from where.
//
// WHAT IS map[string]any?
// A map where keys are strings and values can be ANYTHING.
// Same as `object` in TypeScript or `dict` in Python.
// We use it for the metadata JSONB column — it can hold any extra data.
type ActivityLog struct {
	ID           string         `json:"id"`
	UserID       *string        `json:"user_id"`
	Action       string         `json:"action"`        // "user.register", "user.login", "admin.view_users"
	ResourceType *string        `json:"resource_type"`  // "user", "file", "session"
	ResourceID   *string        `json:"resource_id"`
	Metadata     map[string]any `json:"metadata"`
	IPAddress    *string        `json:"ip_address"`
	UserAgent    *string        `json:"user_agent"`
	CreatedAt    time.Time      `json:"created_at"`
}