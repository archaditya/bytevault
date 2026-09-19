package service

import (
	"context"
	"fmt"
	"math"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/notification/email"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/rs/zerolog/log"
)

type TransactionService struct {
	txnRepo     *repository.TransactionRepository
	subRepo     *repository.SubscriptionRepository
	pkgRepo     *repository.PackageRepository
	brevoClient *email.BrevoClient
	appURL      string
}

func (s *TransactionService) SetAppURL(url string) {
	s.appURL = url
}

func NewTransactionService(
	txnRepo *repository.TransactionRepository,
	subRepo *repository.SubscriptionRepository,
	pkgRepo *repository.PackageRepository,
	brevoClient *email.BrevoClient,
) *TransactionService {
	return &TransactionService{
		txnRepo:     txnRepo,
		subRepo:     subRepo,
		pkgRepo:     pkgRepo,
		brevoClient: brevoClient,
	}
}

// RecordCharge creates a transaction record when a subscription payment is charged.
// It assigns an invoice number and sends a receipt email.
func (s *TransactionService) RecordCharge(
	ctx context.Context,
	sub *model.Subscription,
	paymentID, orderID string,
	totalPaise int,
	currency string,
) (*model.Transaction, error) {
	// Idempotency check: see if payment was already recorded
	existing, _ := s.txnRepo.FindByRazorpayPaymentID(ctx, paymentID)
	if existing != nil {
		log.Info().Str("payment_id", paymentID).Msg("Payment already recorded, skipping duplicate")
		return existing, nil
	}

	// Calculate base and tax amounts
	var pkg *model.Package
	var err error
	if sub != nil && sub.PackageID != "" {
		pkg, err = s.pkgRepo.FindByID(ctx, sub.PackageID)
		if err != nil {
			log.Warn().Err(err).Str("package_id", sub.PackageID).Msg("Failed to load package for transaction")
		}
	}

	gstRate := 18.00
	if pkg != nil && pkg.GSTRate > 0 {
		gstRate = pkg.GSTRate
	}

	// Total = Base + Tax = Base * (1 + gstRate/100)
	basePaise := int(math.Round(float64(totalPaise) / (1.0 + (gstRate / 100.0))))
	taxPaise := totalPaise - basePaise

	invoiceNumber, err := s.txnRepo.NextInvoiceNumber(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate next invoice number")
		invoiceNumber = fmt.Sprintf("BV-ERR-%d", totalPaise)
	}

	desc := "PushPort Subscription Renewal"
	if pkg != nil {
		desc = fmt.Sprintf("PushPort %s Tier - Monthly", pkg.DisplayName)
	}

	var subID *string
	var pkgID *string
	userID := ""
	if sub != nil {
		subID = &sub.ID
		pkgID = &sub.PackageID
		userID = sub.UserID
	}

	txn := &model.Transaction{
		UserID:            userID,
		SubscriptionID:    subID,
		PackageID:         pkgID,
		RazorpayPaymentID: &paymentID,
		RazorpayOrderID:   &orderID,
		AmountPaise:       basePaise,
		TaxPaise:          taxPaise,
		TotalPaise:        totalPaise,
		Currency:          currency,
		Type:              model.TransactionTypeCharge,
		Status:            model.TransactionStatusCaptured,
		Description:       &desc,
		InvoiceNumber:     &invoiceNumber,
		InvoiceGenerated:  true,
	}

	createdTxn, err := s.txnRepo.Create(ctx, txn)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	// Send receipt email if Brevo is enabled and user email is present
	if s.brevoClient != nil && sub != nil && sub.UserEmail != "" {
		totalINR := fmt.Sprintf("%.2f", float64(totalPaise)/100.0)
		pkgName := "Pro"
		if pkg != nil {
			pkgName = pkg.DisplayName
		}
		appURL := s.appURL
		if appURL == "" {
			appURL = "http://localhost:3000"
		}
		invoiceURL := fmt.Sprintf("%s/api/v1/invoices/%s", appURL, createdTxn.ID)
		body := email.GenerateReceiptEmailHTML(sub.UserName, invoiceNumber, pkgName, totalINR, invoiceURL)
		go func() {
			_ = s.brevoClient.SendGeneric(context.Background(), sub.UserEmail, sub.UserName, "Payment Receipt: "+invoiceNumber, body)
		}()
	}

	return createdTxn, nil
}

func (s *TransactionService) GetTransactionByID(ctx context.Context, id string) (*model.Transaction, error) {
	return s.txnRepo.FindByID(ctx, id)
}

func (s *TransactionService) ListUserTransactions(ctx context.Context, userID string, limit, offset int) ([]*model.Transaction, int, error) {
	return s.txnRepo.ListByUser(ctx, userID, limit, offset)
}

func (s *TransactionService) ListAllTransactions(ctx context.Context, limit, offset int) ([]*model.Transaction, int, error) {
	return s.txnRepo.ListAll(ctx, limit, offset)
}

// GetInvoiceHTML renders the complete HTML invoice for browser display or download.
func (s *TransactionService) GetInvoiceHTML(ctx context.Context, txnID string) (string, error) {
	txn, err := s.txnRepo.FindByID(ctx, txnID)
	if err != nil {
		return "", fmt.Errorf("transaction not found: %w", err)
	}

	var sub *model.Subscription
	if txn.SubscriptionID != nil {
		sub, _ = s.subRepo.FindByID(ctx, *txn.SubscriptionID)
	}

	var pkg *model.Package
	if txn.PackageID != nil {
		pkg, _ = s.pkgRepo.FindByID(ctx, *txn.PackageID)
	}

	return email.RenderInvoice(txn, sub, pkg)
}
