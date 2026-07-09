package model

import "time"

type ContactQuery struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	Subject      string     `json:"subject"`
	Message      string     `json:"message"`
	Reply        *string    `json:"reply,omitempty"`
	RepliedAt    *time.Time `json:"replied_at,omitempty"`
	RepliedBy    *string    `json:"replied_by,omitempty"`
	Status       string     `json:"status"` // pending, replied
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ReplierName  string     `json:"replier_name,omitempty"`
	ReplierEmail string     `json:"replier_email,omitempty"`
}
