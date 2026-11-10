package model

import "time"

type AbuseReport struct {
	ID            string     `json:"id"`
	TicketNumber  string     `json:"ticket_number"`
	FileID        *string    `json:"file_id,omitempty"`
	TargetURL     string     `json:"target_url"`
	Category      string     `json:"category"` // copyright, malware, csam, obscene, pii, other
	ReporterName  string     `json:"reporter_name"`
	ReporterEmail string     `json:"reporter_email"`
	Description   string     `json:"description"`
	EvidenceURL   *string    `json:"evidence_url,omitempty"`
	Status        string     `json:"status"` // pending, investigating, actioned_takedown, dismissed
	AdminNotes    *string    `json:"admin_notes,omitempty"`
	ReviewedBy    *string    `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Enriched fields for Admin view
	FileName        *string `json:"file_name,omitempty"`
	FileSize        *int64  `json:"file_size,omitempty"`
	FileContentType *string `json:"file_content_type,omitempty"`
	FileStatus      *string `json:"file_status,omitempty"`
	UploaderID      *string `json:"uploader_id,omitempty"`
	UploaderEmail   *string `json:"uploader_email,omitempty"`
	UploaderStrikes *int    `json:"uploader_strikes,omitempty"`
	ReviewerEmail   *string `json:"reviewer_email,omitempty"`
}

type SubmitAbuseReportRequest struct {
	ReporterName  string  `json:"reporter_name"`
	ReporterEmail string  `json:"reporter_email"`
	Category      string  `json:"category"`
	TargetURL     string  `json:"target_url"`
	Description   string  `json:"description"`
	EvidenceURL   *string `json:"evidence_url,omitempty"`
}
