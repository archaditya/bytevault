package model

import "time"

type EphemeralShare struct {
	ID           string     `json:"id"`
	Token        string     `json:"token"`
	StorageKey   string     `json:"-"`
	Filename     string     `json:"filename"`
	FileSize     int64      `json:"file_size"`
	ContentType  string     `json:"content_type"`
	Status       string     `json:"status"` // 'UPLOADING', 'READY', 'BURNED', 'EXPIRED'
	MaxDownloads int        `json:"max_downloads"`
	DownloadCount int       `json:"download_count"`
	IPAddress    *string    `json:"ip_address,omitempty"`
	UserAgent    *string    `json:"user_agent,omitempty"`
	HasPassword  bool       `json:"has_password"`
	PasswordHash *string    `json:"-"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	BurnedAt     *time.Time `json:"burned_at,omitempty"`
}
