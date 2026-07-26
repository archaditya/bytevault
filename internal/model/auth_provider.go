package model

import "time"

type AuthProvider struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	Provider       string     `json:"provider"`         // e.g. "google"
	ProviderUserID string     `json:"provider_user_id"` // Google sub ID
	Email          *string    `json:"email,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
	DeletedAt      *time.Time `json:"-"`
}
