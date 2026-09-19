package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/archaditya/bytevault/internal/model"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
)

type TransactionRepository struct {
	db *pgxpool.Pool
}

func NewTransactionRepository(db *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) NextInvoiceNumber(ctx context.Context) (string, error) {
	var seq int64
	err := r.db.QueryRow(ctx, "SELECT nextval('invoice_number_seq')").Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("next invoice sequence: %w", err)
	}
	now := time.Now()
	return fmt.Sprintf("BV-%04d%02d-%05d", now.Year(), now.Month(), seq), nil
}

func (r *TransactionRepository) Create(ctx context.Context, txn *model.Transaction) (*model.Transaction, error) {
	metaJSON, _ := json.Marshal(txn.Metadata)
	if len(metaJSON) == 0 {
		metaJSON = []byte("{}")
	}

	query := `
		INSERT INTO transactions (
			user_id, subscription_id, package_id, razorpay_payment_id, razorpay_order_id,
			razorpay_signature, razorpay_invoice_id, amount_paise, tax_paise, total_paise,
			currency, type, status, description, failure_reason, invoice_number, invoice_generated,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		txn.UserID, txn.SubscriptionID, txn.PackageID, txn.RazorpayPaymentID, txn.RazorpayOrderID,
		txn.RazorpaySignature, txn.RazorpayInvoiceID, txn.AmountPaise, txn.TaxPaise, txn.TotalPaise,
		txn.Currency, txn.Type, txn.Status, txn.Description, txn.FailureReason, txn.InvoiceNumber, txn.InvoiceGenerated,
		metaJSON,
	).Scan(&txn.ID, &txn.CreatedAt, &txn.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}
	return txn, nil
}

func (r *TransactionRepository) Update(ctx context.Context, txn *model.Transaction) error {
	metaJSON, _ := json.Marshal(txn.Metadata)
	if len(metaJSON) == 0 {
		metaJSON = []byte("{}")
	}

	query := `
		UPDATE transactions SET
			status = $2,
			razorpay_payment_id = COALESCE($3, razorpay_payment_id),
			failure_reason = $4,
			invoice_number = COALESCE($5, invoice_number),
			invoice_generated = $6,
			metadata = $7,
			updated_at = NOW()
		WHERE id = $1
	`
	res, err := r.db.Exec(ctx, query,
		txn.ID, txn.Status, txn.RazorpayPaymentID, txn.FailureReason, txn.InvoiceNumber, txn.InvoiceGenerated, metaJSON,
	)
	if err != nil {
		return fmt.Errorf("update transaction: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrTransactionNotFound
	}
	return nil
}

func (r *TransactionRepository) FindByID(ctx context.Context, id string) (*model.Transaction, error) {
	query := `
		SELECT t.id, t.user_id, t.subscription_id, t.package_id, t.razorpay_payment_id, t.razorpay_order_id,
		       t.razorpay_signature, t.razorpay_invoice_id, t.amount_paise, t.tax_paise, t.total_paise,
		       t.currency, t.type, t.status, t.description, t.failure_reason, t.invoice_number,
		       t.invoice_generated, t.metadata, t.created_at, t.updated_at,
		       u.email, COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), u.email) as user_name,
		       COALESCE(p.display_name, 'Unknown') as package_name
		FROM transactions t
		LEFT JOIN users u ON t.user_id = u.id
		LEFT JOIN packages p ON t.package_id = p.id
		WHERE t.id = $1
	`
	return r.scanTransaction(r.db.QueryRow(ctx, query, id))
}

func (r *TransactionRepository) FindByRazorpayPaymentID(ctx context.Context, paymentID string) (*model.Transaction, error) {
	query := `
		SELECT t.id, t.user_id, t.subscription_id, t.package_id, t.razorpay_payment_id, t.razorpay_order_id,
		       t.razorpay_signature, t.razorpay_invoice_id, t.amount_paise, t.tax_paise, t.total_paise,
		       t.currency, t.type, t.status, t.description, t.failure_reason, t.invoice_number,
		       t.invoice_generated, t.metadata, t.created_at, t.updated_at,
		       u.email, COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), u.email) as user_name,
		       COALESCE(p.display_name, 'Unknown') as package_name
		FROM transactions t
		LEFT JOIN users u ON t.user_id = u.id
		LEFT JOIN packages p ON t.package_id = p.id
		WHERE t.razorpay_payment_id = $1
	`
	return r.scanTransaction(r.db.QueryRow(ctx, query, paymentID))
}

func (r *TransactionRepository) scanTransaction(row pgx.Row) (*model.Transaction, error) {
	var t model.Transaction
	var metaJSON []byte

	err := row.Scan(
		&t.ID, &t.UserID, &t.SubscriptionID, &t.PackageID, &t.RazorpayPaymentID, &t.RazorpayOrderID,
		&t.RazorpaySignature, &t.RazorpayInvoiceID, &t.AmountPaise, &t.TaxPaise, &t.TotalPaise,
		&t.Currency, &t.Type, &t.Status, &t.Description, &t.FailureReason, &t.InvoiceNumber,
		&t.InvoiceGenerated, &metaJSON, &t.CreatedAt, &t.UpdatedAt,
		&t.UserEmail, &t.UserName, &t.PackageName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("scan transaction: %w", err)
	}

	if len(metaJSON) > 0 {
		_ = json.Unmarshal(metaJSON, &t.Metadata)
	}
	return &t, nil
}

func (r *TransactionRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Transaction, int, error) {
	countQuery := `SELECT COUNT(*) FROM transactions WHERE user_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user transactions: %w", err)
	}

	query := `
		SELECT t.id, t.user_id, t.subscription_id, t.package_id, t.razorpay_payment_id, t.razorpay_order_id,
		       t.razorpay_signature, t.razorpay_invoice_id, t.amount_paise, t.tax_paise, t.total_paise,
		       t.currency, t.type, t.status, t.description, t.failure_reason, t.invoice_number,
		       t.invoice_generated, t.metadata, t.created_at, t.updated_at,
		       u.email, COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), u.email) as user_name,
		       COALESCE(p.display_name, 'Unknown') as package_name
		FROM transactions t
		LEFT JOIN users u ON t.user_id = u.id
		LEFT JOIN packages p ON t.package_id = p.id
		WHERE t.user_id = $1
		ORDER BY t.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list user transactions: %w", err)
	}
	defer rows.Close()

	var txns []*model.Transaction
	for rows.Next() {
		var t model.Transaction
		var metaJSON []byte
		err := rows.Scan(
			&t.ID, &t.UserID, &t.SubscriptionID, &t.PackageID, &t.RazorpayPaymentID, &t.RazorpayOrderID,
			&t.RazorpaySignature, &t.RazorpayInvoiceID, &t.AmountPaise, &t.TaxPaise, &t.TotalPaise,
			&t.Currency, &t.Type, &t.Status, &t.Description, &t.FailureReason, &t.InvoiceNumber,
			&t.InvoiceGenerated, &metaJSON, &t.CreatedAt, &t.UpdatedAt,
			&t.UserEmail, &t.UserName, &t.PackageName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan user transaction item: %w", err)
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &t.Metadata)
		}
		txns = append(txns, &t)
	}
	return txns, total, nil
}

func (r *TransactionRepository) ListAll(ctx context.Context, limit, offset int) ([]*model.Transaction, int, error) {
	countQuery := `SELECT COUNT(*) FROM transactions`
	var total int
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count all transactions: %w", err)
	}

	query := `
		SELECT t.id, t.user_id, t.subscription_id, t.package_id, t.razorpay_payment_id, t.razorpay_order_id,
		       t.razorpay_signature, t.razorpay_invoice_id, t.amount_paise, t.tax_paise, t.total_paise,
		       t.currency, t.type, t.status, t.description, t.failure_reason, t.invoice_number,
		       t.invoice_generated, t.metadata, t.created_at, t.updated_at,
		       u.email, COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), u.email) as user_name,
		       COALESCE(p.display_name, 'Unknown') as package_name
		FROM transactions t
		LEFT JOIN users u ON t.user_id = u.id
		LEFT JOIN packages p ON t.package_id = p.id
		ORDER BY t.created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list all transactions: %w", err)
	}
	defer rows.Close()

	var txns []*model.Transaction
	for rows.Next() {
		var t model.Transaction
		var metaJSON []byte
		err := rows.Scan(
			&t.ID, &t.UserID, &t.SubscriptionID, &t.PackageID, &t.RazorpayPaymentID, &t.RazorpayOrderID,
			&t.RazorpaySignature, &t.RazorpayInvoiceID, &t.AmountPaise, &t.TaxPaise, &t.TotalPaise,
			&t.Currency, &t.Type, &t.Status, &t.Description, &t.FailureReason, &t.InvoiceNumber,
			&t.InvoiceGenerated, &metaJSON, &t.CreatedAt, &t.UpdatedAt,
			&t.UserEmail, &t.UserName, &t.PackageName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan all transaction item: %w", err)
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &t.Metadata)
		}
		txns = append(txns, &t)
	}
	return txns, total, nil
}
