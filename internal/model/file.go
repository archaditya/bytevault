package model

import (
	"time"
)

type File struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	Filename        string     `json:"filename"`
	StorageProvider string     `json:"storage_provider"` // local, cloudinary, r2
	Bucket          *string    `json:"bucket,omitempty"` // Nullable for providers without buckets
	StorageKey      string     `json:"storage_key"`      // e.g., user/123/docs/resume.pdf
	FileSize        int64      `json:"file_size"`
	ContentType     string     `json:"content_type"`
	IsPublic        bool       `json:"is_public"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"-"`
}