package email

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"

	"github.com/archaditya/bytevault/internal/model"
)

//go:embed invoice_template.html
var invoiceHTML string

var invoiceTmpl = template.Must(template.New("invoice").Parse(invoiceHTML))

// InvoiceData holds fields formatted for the HTML invoice template.
type InvoiceData struct {
	TransactionID string
	InvoiceNumber string
	Date          string
	PeriodStart   string
	PeriodEnd     string
	Status        string
	PaymentID     string
	UserName      string
	UserEmail     string
	PackageName   string
	StorageCap    string
	MaxFileSize   string
	BillingPeriod string
	BaseAmount    string
	GSTRate       string
	TaxAmount     string
	TotalAmount   string
}

// RenderInvoice generates clean HTML for a transaction and its associated subscription/package.
func RenderInvoice(txn *model.Transaction, sub *model.Subscription, pkg *model.Package) (string, error) {
	var periodStart, periodEnd string
	if sub != nil {
		if sub.CurrentPeriodStart != nil {
			periodStart = sub.CurrentPeriodStart.Format("02 Jan 2006")
		}
		if sub.CurrentPeriodEnd != nil {
			periodEnd = sub.CurrentPeriodEnd.Format("02 Jan 2006")
		}
	}

	storageCap := "5 GB"
	maxFileSize := "2 GB"
	billingPeriod := "Monthly"
	packageName := "Pro"
	gstRate := "18.00"

	if pkg != nil {
		packageName = pkg.DisplayName
		gstRate = fmt.Sprintf("%.2f", pkg.GSTRate)
		storageCap = formatBytes(pkg.StorageLimitBytes)
		maxFileSize = formatBytes(pkg.MaxFileSizeBytes)
		if pkg.BillingPeriod != "" {
			billingPeriod = pkg.BillingPeriod
		}
	}

	invoiceNum := "BV-PENDING"
	if txn.InvoiceNumber != nil {
		invoiceNum = *txn.InvoiceNumber
	}

	paymentID := ""
	if txn.RazorpayPaymentID != nil {
		paymentID = *txn.RazorpayPaymentID
	}

	baseAmt := fmt.Sprintf("%.2f", float64(txn.AmountPaise)/100.0)
	taxAmt := fmt.Sprintf("%.2f", float64(txn.TaxPaise)/100.0)
	totalAmt := fmt.Sprintf("%.2f", float64(txn.TotalPaise)/100.0)

	data := InvoiceData{
		TransactionID: txn.ID,
		InvoiceNumber: invoiceNum,
		Date:          txn.CreatedAt.Format("02 Jan 2006"),
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		Status:        "PAID",
		PaymentID:     paymentID,
		UserName:      txn.UserName,
		UserEmail:     txn.UserEmail,
		PackageName:   packageName,
		StorageCap:    storageCap,
		MaxFileSize:   maxFileSize,
		BillingPeriod: billingPeriod,
		BaseAmount:    baseAmt,
		GSTRate:       gstRate,
		TaxAmount:     taxAmt,
		TotalAmount:   totalAmt,
	}

	var buf bytes.Buffer
	if err := invoiceTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute invoice template: %w", err)
	}

	return buf.String(), nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.0f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// GenerateReceiptEmailHTML renders template.html for a payment receipt notification.
func GenerateReceiptEmailHTML(userName, invoiceNum, packageName, amountStr, invoiceURL string) string {
	heading := "Payment Receipt"
	message := fmt.Sprintf("Hi %s, thank you for subscribing to PushPortVault %s. Your payment was successful.", userName, packageName)
	body := fmt.Sprintf(`<div style="background:#181b22;border:1px solid #2a2d35;border-radius:10px;padding:20px;margin-bottom:20px;"><p style="margin:0 0 8px;font-size:14px;color:#8b8f96;">Invoice Number: <strong style="color:#f0f1f3;">%s</strong></p><p style="margin:0;font-size:18px;font-weight:700;color:#f0f1f3;">Total Paid: ₹%s</p></div><p style="margin:0;"><a href="%s" style="display:inline-block;background:#6c5ce7;color:#fff;padding:10px 18px;border-radius:8px;text-decoration:none;font-weight:600;font-size:14px;">View & Download Invoice</a></p>`, invoiceNum, amountStr, invoiceURL)
	footer := "Per PushPortVault terms, subscriptions renew automatically unless cancelled. Manage anytime in Settings."
	html, err := RenderNotification(heading, message, body, footer)
	if err != nil {
		return message
	}
	return html
}
