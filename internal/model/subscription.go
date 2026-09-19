package model

import "time"

const (
	SubscriptionStatusCreated       = "created"
	SubscriptionStatusAuthenticated = "authenticated"
	SubscriptionStatusActive        = "active"
	SubscriptionStatusPending       = "pending"
	SubscriptionStatusHalted        = "halted"
	SubscriptionStatusCancelled     = "cancelled"
	SubscriptionStatusCompleted     = "completed"
	SubscriptionStatusPaused        = "paused"
	SubscriptionStatusExpired       = "expired"
)

const (
	// DefaultFreeStorageLimitBytes is the fallback free tier storage cap (5 GB).
	DefaultFreeStorageLimitBytes = int64(5368709120)
	// DefaultFreeMaxFileSizeBytes is the fallback free tier max single file size (2 GB).
	DefaultFreeMaxFileSizeBytes = int64(2147483648)
)

// Subscription tracks a user's active or past subscription.
type Subscription struct {
	ID                     string                 `json:"id"`
	UserID                 string                 `json:"user_id"`
	PackageID              string                 `json:"package_id"`
	RazorpaySubscriptionID *string                `json:"razorpay_subscription_id,omitempty"`
	RazorpayCustomerID     *string                `json:"razorpay_customer_id,omitempty"`
	RazorpayPaymentID      *string                `json:"razorpay_payment_id,omitempty"`
	Status                 string                 `json:"status"`
	CurrentPeriodStart     *time.Time             `json:"current_period_start,omitempty"`
	CurrentPeriodEnd       *time.Time             `json:"current_period_end,omitempty"`
	CancelledAt            *time.Time             `json:"cancelled_at,omitempty"`
	CancelAtCycleEnd       bool                   `json:"cancel_at_cycle_end"`
	PausedAt               *time.Time             `json:"paused_at,omitempty"`
	PendingPackageID       *string                `json:"pending_package_id,omitempty"`
	UpgradedFromID         *string                `json:"upgraded_from_id,omitempty"`
	GracePeriodEnd         *time.Time             `json:"grace_period_end,omitempty"`
	Metadata               map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt              time.Time              `json:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at"`

	// Enriched fields
	Package                *Package               `json:"package,omitempty"`
	PendingPackage         *Package               `json:"pending_package,omitempty"`
	UserEmail              string                 `json:"user_email,omitempty"`
	UserName               string                 `json:"user_name,omitempty"`
}

// IsActive returns true only when the subscription is fully active (payment confirmed).
func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive
}

// HasActiveOrPendingStatus returns true if the subscription is in any non-terminal state
// (active, created, authenticated, or pending payment retry).
func (s *Subscription) HasActiveOrPendingStatus() bool {
	switch s.Status {
	case SubscriptionStatusActive, SubscriptionStatusAuthenticated,
		SubscriptionStatusCreated, SubscriptionStatusPending:
		return true
	}
	return false
}
