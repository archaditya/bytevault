package model

import "time"

// SubscriptionAuditLog records any subscription/payment event for audit and compliance.
type SubscriptionAuditLog struct {
	ID             string                 `json:"id"`
	UserID         *string                `json:"user_id,omitempty"`
	SubscriptionID *string                `json:"subscription_id,omitempty"`
	TransactionID  *string                `json:"transaction_id,omitempty"`
	EventType      string                 `json:"event_type"`
	EventSource    string                 `json:"event_source"` // 'api', 'webhook', 'scheduler', 'admin'
	Payload        map[string]interface{} `json:"payload,omitempty"`
	Status         string                 `json:"status"`       // 'success', 'error', 'skipped'
	ErrorMessage   *string                `json:"error_message,omitempty"`
	IPAddress      *string                `json:"ip_address,omitempty"`
	UserAgent      *string                `json:"user_agent,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}
