package model

import "time"

const (
	TransactionStatusPending  = "pending"
	TransactionStatusCaptured = "captured"
	TransactionStatusFailed   = "failed"
	TransactionStatusRefunded = "refunded"

	TransactionTypeCharge  = "subscription_charge"
	TransactionTypeUpgrade = "upgrade"
	TransactionTypeRefund  = "refund"
)

// Transaction represents a financial charge, upgrade payment, or refund.
type Transaction struct {
	ID                 string                 `json:"id"`
	UserID             string                 `json:"user_id"`
	SubscriptionID     *string                `json:"subscription_id,omitempty"`
	PackageID          *string                `json:"package_id,omitempty"`
	RazorpayPaymentID  *string                `json:"razorpay_payment_id,omitempty"`
	RazorpayOrderID    *string                `json:"razorpay_order_id,omitempty"`
	RazorpaySignature  *string                `json:"-"`
	RazorpayInvoiceID  *string                `json:"razorpay_invoice_id,omitempty"`
	AmountPaise        int                    `json:"amount_paise"`
	TaxPaise           int                    `json:"tax_paise"`
	TotalPaise         int                    `json:"total_paise"`
	Currency           string                 `json:"currency"`
	Type               string                 `json:"type"`
	Status             string                 `json:"status"`
	Description        *string                `json:"description,omitempty"`
	FailureReason      *string                `json:"failure_reason,omitempty"`
	InvoiceNumber      *string                `json:"invoice_number,omitempty"`
	InvoiceGenerated   bool                   `json:"invoice_generated"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`

	// Enriched fields for displays/receipts
	UserEmail          string                 `json:"user_email,omitempty"`
	UserName           string                 `json:"user_name,omitempty"`
	PackageName        string                 `json:"package_name,omitempty"`
}
