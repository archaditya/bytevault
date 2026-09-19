package model

import "time"

// Package represents a subscription tier in ByteVault.
type Package struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	DisplayName       string                 `json:"display_name"`
	Description       *string                `json:"description,omitempty"`
	PricePaise        int                    `json:"price_paise"`
	GSTRate           float64                `json:"gst_rate"`
	PriceWithGSTPaise int                    `json:"price_with_gst_paise"`
	Currency          string                 `json:"currency"`
	BillingPeriod     string                 `json:"billing_period"`
	StorageLimitBytes int64                  `json:"storage_limit_bytes"`
	MaxFileSizeBytes  int64                  `json:"max_file_size_bytes"`
	RazorpayPlanID    *string                `json:"razorpay_plan_id,omitempty"`
	IsActive          bool                   `json:"is_active"`
	SortOrder         int                    `json:"sort_order"`
	Features          map[string]interface{} `json:"features,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}
