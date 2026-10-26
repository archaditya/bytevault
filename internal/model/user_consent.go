package model

import "time"

type UserConsent struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	ConsentType string     `json:"consent_type"` // e.g. "terms_and_privacy", "marketing"
	Version     string     `json:"version"`      // e.g. "2026.1"
	Status      string     `json:"status"`       // "granted", "revoked"
	IPAddress   *string    `json:"ip_address,omitempty"`
	UserAgent   *string    `json:"user_agent,omitempty"`
	GrantedAt   time.Time  `json:"granted_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
