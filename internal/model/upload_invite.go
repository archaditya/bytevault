package model

import "time"

// UploadInvite represents a guest upload invite link for the Client File Portal.
type UploadInvite struct {
	ID               string     `json:"id"`
	Token            string     `json:"token"`
	OwnerID          string     `json:"owner_id"`
	TargetFolderID   *string    `json:"target_folder_id,omitempty"`
	Label            string     `json:"label"`
	MaxTotalBytes    int64      `json:"max_total_bytes"`
	MaxFiles         int        `json:"max_files"`
	UsedBytes        int64      `json:"used_bytes"`
	UsedFiles        int        `json:"used_files"`
	PasscodeHash     *string    `json:"-"`
	HasPasscode      bool       `json:"has_passcode"`
	Status           string     `json:"status"` // active, revoked, expired
	PendingNotifyCount int      `json:"-"`
	LastNotifiedAt   *time.Time `json:"-"`
	ExpiresAt        time.Time  `json:"expires_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Enriched fields for guest-facing display
	OwnerName  string `json:"owner_name,omitempty"`
	OwnerEmail string `json:"-"`
}

// UploadInviteFile tracks each file uploaded via an invite for audit and rate limiting.
type UploadInviteFile struct {
	ID        string    `json:"id"`
	InviteID  string    `json:"invite_id"`
	FileID    string    `json:"file_id"`
	IPAddress *string   `json:"ip_address,omitempty"`
	UserAgent *string   `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
