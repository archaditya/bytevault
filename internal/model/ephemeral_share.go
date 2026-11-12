package model

import "time"

type EphemeralState string

const (
	EphemeralStateInitiated EphemeralState = "INITIATED"
	EphemeralStateUploading EphemeralState = "UPLOADING"
	EphemeralStateActive    EphemeralState = "ACTIVE"
	EphemeralStateBurned    EphemeralState = "BURNED"
	EphemeralStateExpired   EphemeralState = "EXPIRED"
	EphemeralStateFailed    EphemeralState = "FAILED"
	EphemeralStatePurged    EphemeralState = "PURGED"
)

var validEphemeralTransitions = map[EphemeralState][]EphemeralState{
	EphemeralStateInitiated: {EphemeralStateUploading, EphemeralStateActive, EphemeralStateBurned, EphemeralStateFailed, EphemeralStatePurged},
	EphemeralStateUploading: {EphemeralStateActive, EphemeralStateBurned, EphemeralStateFailed, EphemeralStatePurged},
	EphemeralStateActive:    {EphemeralStateBurned, EphemeralStateExpired, EphemeralStatePurged},
	EphemeralStateBurned:    {EphemeralStatePurged},
	EphemeralStateExpired:   {EphemeralStatePurged},
	EphemeralStateFailed:    {EphemeralStatePurged},
	EphemeralStatePurged:    {}, // Terminal
}

// CanTransitionTo validates if an ephemeral share can legally transition from state s to state next.
func (s EphemeralState) CanTransitionTo(next EphemeralState) bool {
	if s == next {
		return true // Idempotent update
	}
	allowed, ok := validEphemeralTransitions[s]
	if !ok {
		return false
	}
	for _, target := range allowed {
		if target == next {
			return true
		}
	}
	return false
}

type EphemeralShare struct {
	ID            string     `json:"id"`
	Token         string     `json:"token"`
	StorageKey    string     `json:"-"`
	UploadID      *string    `json:"upload_id,omitempty"`
	Filename      string     `json:"filename"`
	FileSize      int64      `json:"file_size"`
	ContentType   string     `json:"content_type"`
	Status        string     `json:"status"` // Uses EphemeralState constants
	MaxDownloads  int        `json:"max_downloads"`
	DownloadCount int        `json:"download_count"`
	IPAddress     *string    `json:"ip_address,omitempty"`
	UserAgent     *string    `json:"user_agent,omitempty"`
	HasPassword   bool       `json:"has_password"`
	PasswordHash  *string    `json:"-"`
	ExpiresAt     time.Time  `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	BurnedAt      *time.Time `json:"burned_at,omitempty"`
	PreviewURL    *string    `json:"preview_url,omitempty"`
	IsLive        bool       `json:"is_live"`
	ProgressPct   int        `json:"progress_pct"`
	UploadedBytes int64      `json:"uploaded_bytes"`
}

// EphemeralMetadataDTO is the public metadata representation for guest transfers.
// It explicitly omits internal database IDs, storage keys, and sender IP / User-Agent.
type EphemeralMetadataDTO struct {
	Token         string    `json:"token"`
	Filename      string    `json:"filename"`
	FileSize      int64     `json:"file_size"`
	ContentType   string    `json:"content_type"`
	Status        string    `json:"status"`
	MaxDownloads  int       `json:"max_downloads"`
	DownloadCount int       `json:"download_count"`
	HasPassword   bool      `json:"has_password"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
	PreviewURL    *string   `json:"preview_url,omitempty"`
	IsLive        bool      `json:"is_live"`
	ProgressPct   int       `json:"progress_pct"`
	UploadedBytes int64     `json:"uploaded_bytes"`
}

// ToMetadataDTO converts the full database model to a safe public DTO.
func (s *EphemeralShare) ToMetadataDTO() *EphemeralMetadataDTO {
	if s == nil {
		return nil
	}
	return &EphemeralMetadataDTO{
		Token:         s.Token,
		Filename:      s.Filename,
		FileSize:      s.FileSize,
		ContentType:   s.ContentType,
		Status:        s.Status,
		MaxDownloads:  s.MaxDownloads,
		DownloadCount: s.DownloadCount,
		HasPassword:   s.HasPassword,
		ExpiresAt:     s.ExpiresAt,
		CreatedAt:     s.CreatedAt,
		PreviewURL:    s.PreviewURL,
		IsLive:        s.IsLive,
		ProgressPct:   s.ProgressPct,
		UploadedBytes: s.UploadedBytes,
	}
}
