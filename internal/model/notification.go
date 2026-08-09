package model

import "time"

// EmailVerification stores OTP codes for email verification flows.
type EmailVerification struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	OTPCode     string    `json:"-"` // Never expose in API responses
	Purpose     string    `json:"purpose"` // "registration", "password_reset"
	Attempts    int       `json:"attempts"`
	MaxAttempts int       `json:"max_attempts"`
	IsUsed      bool      `json:"is_used"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// Notification represents an in-app notification stored in PostgreSQL.
type Notification struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Type      string         `json:"type"` // "file.shared", "account.verified", "system.alert"
	Title     string         `json:"title"`
	Body      string         `json:"body,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Channel   string         `json:"channel"` // "in_app", "email", "push"
	IsRead    bool           `json:"is_read"`
	ReadAt    *time.Time     `json:"read_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}
