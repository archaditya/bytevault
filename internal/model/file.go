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
	ThumbnailKey    *string    `json:"thumbnail_key,omitempty"`
	ThumbnailURL    *string    `json:"thumbnail_url,omitempty"`
	FileSize        int64      `json:"file_size"`
	ContentType     string     `json:"content_type"`
	IsPublic        bool       `json:"is_public"`
	Status          string     `json:"status"`           // UPLOADING, READY, FAILED
	FolderID        *string    `json:"folder_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"-"`
	Downloads       int64      `json:"downloads"`

	// Enriched fields for user/admin views
	OwnerName       string     `json:"owner_name,omitempty"`
	OwnerEmail      string     `json:"owner_email,omitempty"`
}

type UploadPart struct {
	PartNumber int32 `json:"part_number"`
	ETag string `json:"etag"`
}
