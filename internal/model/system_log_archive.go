package model

import "time"

// SystemLogArchive tracks the storage lifecycle of daily application logs across local disk, R2, and expiration.
type SystemLogArchive struct {
	ID                  string     `json:"id"`
	LogDate             string     `json:"log_date"`
	FileName            string     `json:"file_name"`
	StorageLocation     string     `json:"storage_location"` // 'local', 'r2', 'purged'
	StorageKey          *string    `json:"storage_key,omitempty"`
	LocalPath           *string    `json:"local_path,omitempty"`
	FileSizeBytes       int64      `json:"file_size_bytes"`
	CompressedSizeBytes int64      `json:"compressed_size_bytes"`
	ArchivedToR2At      *time.Time `json:"archived_to_r2_at,omitempty"`
	PurgedAt            *time.Time `json:"purged_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}
