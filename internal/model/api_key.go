package model

import "time"

type APIKey struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	Name            string     `json:"name"`
	KeyPrefix       string     `json:"key_prefix"`
	KeyHash         string     `json:"-"`
	RotationCount   int        `json:"rotation_count"`
	MaxRotations    int        `json:"max_rotations"`
	RateLimitPerMin int        `json:"rate_limit_per_min"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	// PlainKey is populated ONLY on initial generation or rotation
	PlainKey string `json:"key,omitempty"`
}

type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}
