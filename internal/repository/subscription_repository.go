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
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

type SubscriptionRepository struct {
	db *pgxpool.Pool
}

func NewSubscriptionRepository(db *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(ctx context.Context, sub *model.Subscription) (*model.Subscription, error) {
	metaJSON, _ := json.Marshal(sub.Metadata)
	if len(metaJSON) == 0 {
		metaJSON = []byte("{}")
	}

	query := `
		INSERT INTO subscriptions (
			user_id, package_id, razorpay_subscription_id, razorpay_customer_id, razorpay_payment_id,
			status, current_period_start, current_period_end, cancelled_at, cancel_at_cycle_end,
			paused_at, pending_package_id, upgraded_from_id, grace_period_end, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		sub.UserID, sub.PackageID, sub.RazorpaySubscriptionID, sub.RazorpayCustomerID, sub.RazorpayPaymentID,
		sub.Status, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.CancelledAt, sub.CancelAtCycleEnd,
		sub.PausedAt, sub.PendingPackageID, sub.UpgradedFromID, sub.GracePeriodEnd, metaJSON,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}
	return sub, nil
}

func (r *SubscriptionRepository) FindByID(ctx context.Context, id string) (*model.Subscription, error) {
	query := `
		SELECT s.id, s.user_id, s.package_id, s.razorpay_subscription_id, s.razorpay_customer_id, s.razorpay_payment_id,
		       s.status, s.current_period_start, s.current_period_end, s.cancelled_at, s.cancel_at_cycle_end,
		       s.paused_at, s.pending_package_id, s.upgraded_from_id, s.grace_period_end, s.metadata, s.created_at, s.updated_at,
		       p.id, p.name, p.display_name, p.price_paise, p.storage_limit_bytes, p.max_file_size_bytes,
		       u.email, COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), u.email) as user_name
		FROM subscriptions s
		LEFT JOIN packages p ON s.package_id = p.id
		LEFT JOIN users u ON s.user_id = u.id
		WHERE s.id = $1
	`
	return r.scanSubscription(r.db.QueryRow(ctx, query, id))
}

func (r *SubscriptionRepository) FindByUserID(ctx context.Context, userID string) (*model.Subscription, error) {
	query := `
		SELECT s.id, s.user_id, s.package_id, s.razorpay_subscription_id, s.razorpay_customer_id, s.razorpay_payment_id,
		       s.status, s.current_period_start, s.current_period_end, s.cancelled_at, s.cancel_at_cycle_end,
		       s.paused_at, s.pending_package_id, s.upgraded_from_id, s.grace_period_end, s.metadata, s.created_at, s.updated_at,
		       p.id, p.name, p.display_name, p.price_paise, p.storage_limit_bytes, p.max_file_size_bytes,
		       u.email, COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), u.email) as user_name
		FROM subscriptions s
		LEFT JOIN packages p ON s.package_id = p.id
		LEFT JOIN users u ON s.user_id = u.id
		WHERE s.user_id = $1 AND s.status IN ('active', 'authenticated', 'pending', 'created')
		ORDER BY s.created_at DESC
		LIMIT 1
	`
	sub, err := r.scanSubscription(r.db.QueryRow(ctx, query, userID))
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return nil, nil // No active subscription is a valid state (user is on free tier)
		}
		return nil, err
	}
	return sub, nil
}

func (r *SubscriptionRepository) FindByRazorpayID(ctx context.Context, rzpSubID string) (*model.Subscription, error) {
	query := `
		SELECT s.id, s.user_id, s.package_id, s.razorpay_subscription_id, s.razorpay_customer_id, s.razorpay_payment_id,
		       s.status, s.current_period_start, s.current_period_end, s.cancelled_at, s.cancel_at_cycle_end,
		       s.paused_at, s.pending_package_id, s.upgraded_from_id, s.grace_period_end, s.metadata, s.created_at, s.updated_at,
		       p.id, p.name, p.display_name, p.price_paise, p.storage_limit_bytes, p.max_file_size_bytes,
		       u.email, COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), u.email) as user_name
		FROM subscriptions s
		LEFT JOIN packages p ON s.package_id = p.id
		LEFT JOIN users u ON s.user_id = u.id
		WHERE s.razorpay_subscription_id = $1
	`
	return r.scanSubscription(r.db.QueryRow(ctx, query, rzpSubID))
}

func (r *SubscriptionRepository) scanSubscription(row pgx.Row) (*model.Subscription, error) {
	var s model.Subscription
	var metaJSON []byte
	var pkg model.Package

	err := row.Scan(
		&s.ID, &s.UserID, &s.PackageID, &s.RazorpaySubscriptionID, &s.RazorpayCustomerID, &s.RazorpayPaymentID,
		&s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelledAt, &s.CancelAtCycleEnd,
		&s.PausedAt, &s.PendingPackageID, &s.UpgradedFromID, &s.GracePeriodEnd, &metaJSON, &s.CreatedAt, &s.UpdatedAt,
		&pkg.ID, &pkg.Name, &pkg.DisplayName, &pkg.PricePaise, &pkg.StorageLimitBytes, &pkg.MaxFileSizeBytes,
		&s.UserEmail, &s.UserName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("scan subscription: %w", err)
	}

	if len(metaJSON) > 0 {
		_ = json.Unmarshal(metaJSON, &s.Metadata)
	}
	s.Package = &pkg
	return &s, nil
}

func (r *SubscriptionRepository) UpdateStatus(ctx context.Context, id, status string, start, end *time.Time, paymentID *string) error {
	query := `
		UPDATE subscriptions SET
			status = $2,
			current_period_start = COALESCE($3, current_period_start),
			current_period_end = COALESCE($4, current_period_end),
			razorpay_payment_id = COALESCE($5, razorpay_payment_id),
			updated_at = NOW()
		WHERE id = $1
	`
	res, err := r.db.Exec(ctx, query, id, status, start, end, paymentID)
	if err != nil {
		return fmt.Errorf("update subscription status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (r *SubscriptionRepository) SetCancellation(ctx context.Context, id string, cancelAtCycleEnd bool, cancelledAt *time.Time) error {
	query := `
		UPDATE subscriptions SET
			cancel_at_cycle_end = $2,
			cancelled_at = $3,
			updated_at = NOW()
		WHERE id = $1
	`
	res, err := r.db.Exec(ctx, query, id, cancelAtCycleEnd, cancelledAt)
	if err != nil {
		return fmt.Errorf("set cancellation: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (r *SubscriptionRepository) SetPendingDowngrade(ctx context.Context, id string, pendingPkgID *string) error {
	query := `
		UPDATE subscriptions SET
			pending_package_id = $2,
			updated_at = NOW()
		WHERE id = $1
	`
	res, err := r.db.Exec(ctx, query, id, pendingPkgID)
	if err != nil {
		return fmt.Errorf("set pending downgrade: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (r *SubscriptionRepository) UpdatePackageID(ctx context.Context, id, packageID string) error {
	query := `
		UPDATE subscriptions SET
			package_id = $2,
			pending_package_id = NULL,
			updated_at = NOW()
		WHERE id = $1
	`
	res, err := r.db.Exec(ctx, query, id, packageID)
	if err != nil {
		return fmt.Errorf("update subscription package: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (r *SubscriptionRepository) SetUpgradedFrom(ctx context.Context, id string, upgradedFromID *string) error {
	query := `
		UPDATE subscriptions SET
			upgraded_from_id = $2,
			updated_at = NOW()
		WHERE id = $1
	`
	res, err := r.db.Exec(ctx, query, id, upgradedFromID)
	if err != nil {
		return fmt.Errorf("set upgraded from: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (r *SubscriptionRepository) ListExpiringOrExpired(ctx context.Context, before time.Time) ([]*model.Subscription, error) {
	query := `
		SELECT s.id, s.user_id, s.package_id, s.razorpay_subscription_id, s.razorpay_customer_id, s.razorpay_payment_id,
		       s.status, s.current_period_start, s.current_period_end, s.cancelled_at, s.cancel_at_cycle_end,
		       s.paused_at, s.pending_package_id, s.upgraded_from_id, s.grace_period_end, s.metadata, s.created_at, s.updated_at,
		       p.id, p.name, p.display_name, p.price_paise, p.storage_limit_bytes, p.max_file_size_bytes,
		       u.email, COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), u.email) as user_name
		FROM subscriptions s
		LEFT JOIN packages p ON s.package_id = p.id
		LEFT JOIN users u ON s.user_id = u.id
		WHERE s.status = 'active' AND s.current_period_end <= $1
	`
	rows, err := r.db.Query(ctx, query, before)
	if err != nil {
		return nil, fmt.Errorf("query expiring subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		var s model.Subscription
		var metaJSON []byte
		var pkg model.Package

		err := rows.Scan(
			&s.ID, &s.UserID, &s.PackageID, &s.RazorpaySubscriptionID, &s.RazorpayCustomerID, &s.RazorpayPaymentID,
			&s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelledAt, &s.CancelAtCycleEnd,
			&s.PausedAt, &s.PendingPackageID, &s.UpgradedFromID, &s.GracePeriodEnd, &metaJSON, &s.CreatedAt, &s.UpdatedAt,
			&pkg.ID, &pkg.Name, &pkg.DisplayName, &pkg.PricePaise, &pkg.StorageLimitBytes, &pkg.MaxFileSizeBytes,
			&s.UserEmail, &s.UserName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan expiring subscription: %w", err)
		}
		s.Package = &pkg
		subs = append(subs, &s)
	}
	return subs, nil
}

func (r *SubscriptionRepository) ListUpcomingRenewals(ctx context.Context, from, to time.Time) ([]*model.Subscription, error) {
	query := `
		SELECT s.id, s.user_id, s.package_id, s.razorpay_subscription_id, s.razorpay_customer_id, s.razorpay_payment_id,
		       s.status, s.current_period_start, s.current_period_end, s.cancelled_at, s.cancel_at_cycle_end,
		       s.paused_at, s.pending_package_id, s.upgraded_from_id, s.grace_period_end, s.metadata, s.created_at, s.updated_at,
		       p.id, p.name, p.display_name, p.price_paise, p.storage_limit_bytes, p.max_file_size_bytes,
		       u.email, COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), u.email) as user_name
		FROM subscriptions s
		LEFT JOIN packages p ON s.package_id = p.id
		LEFT JOIN users u ON s.user_id = u.id
		WHERE s.status = 'active' AND s.cancel_at_cycle_end = false AND s.current_period_end >= $1 AND s.current_period_end <= $2
	`
	rows, err := r.db.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("query upcoming renewals: %w", err)
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		var s model.Subscription
		var metaJSON []byte
		var pkg model.Package

		err := rows.Scan(
			&s.ID, &s.UserID, &s.PackageID, &s.RazorpaySubscriptionID, &s.RazorpayCustomerID, &s.RazorpayPaymentID,
			&s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelledAt, &s.CancelAtCycleEnd,
			&s.PausedAt, &s.PendingPackageID, &s.UpgradedFromID, &s.GracePeriodEnd, &metaJSON, &s.CreatedAt, &s.UpdatedAt,
			&pkg.ID, &pkg.Name, &pkg.DisplayName, &pkg.PricePaise, &pkg.StorageLimitBytes, &pkg.MaxFileSizeBytes,
			&s.UserEmail, &s.UserName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan upcoming renewal: %w", err)
		}
		s.Package = &pkg
		subs = append(subs, &s)
	}
	return subs, nil
}

func (r *SubscriptionRepository) ListPendingDowngrades(ctx context.Context, before time.Time) ([]*model.Subscription, error) {
	query := `
		SELECT s.id, s.user_id, s.package_id, s.razorpay_subscription_id, s.razorpay_customer_id, s.razorpay_payment_id,
		       s.status, s.current_period_start, s.current_period_end, s.cancelled_at, s.cancel_at_cycle_end,
		       s.paused_at, s.pending_package_id, s.upgraded_from_id, s.grace_period_end, s.metadata, s.created_at, s.updated_at,
		       p.id, p.name, p.display_name, p.price_paise, p.storage_limit_bytes, p.max_file_size_bytes,
		       u.email, COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), u.email) as user_name
		FROM subscriptions s
		LEFT JOIN packages p ON s.package_id = p.id
		LEFT JOIN users u ON s.user_id = u.id
		WHERE s.pending_package_id IS NOT NULL AND s.current_period_end <= $1
	`
	rows, err := r.db.Query(ctx, query, before)
	if err != nil {
		return nil, fmt.Errorf("query pending downgrades: %w", err)
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		var s model.Subscription
		var metaJSON []byte
		var pkg model.Package

		err := rows.Scan(
			&s.ID, &s.UserID, &s.PackageID, &s.RazorpaySubscriptionID, &s.RazorpayCustomerID, &s.RazorpayPaymentID,
			&s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelledAt, &s.CancelAtCycleEnd,
			&s.PausedAt, &s.PendingPackageID, &s.UpgradedFromID, &s.GracePeriodEnd, &metaJSON, &s.CreatedAt, &s.UpdatedAt,
			&pkg.ID, &pkg.Name, &pkg.DisplayName, &pkg.PricePaise, &pkg.StorageLimitBytes, &pkg.MaxFileSizeBytes,
			&s.UserEmail, &s.UserName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan pending downgrade: %w", err)
		}
		s.Package = &pkg
		subs = append(subs, &s)
	}
	return subs, nil
}

func (r *SubscriptionRepository) ListAll(ctx context.Context, status string, limit, offset int) ([]*model.Subscription, int, error) {
	countQuery := `SELECT COUNT(*) FROM subscriptions WHERE ($1 = '' OR status = $1)`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count subscriptions: %w", err)
	}

	query := `
		SELECT s.id, s.user_id, s.package_id, s.razorpay_subscription_id, s.razorpay_customer_id, s.razorpay_payment_id,
		       s.status, s.current_period_start, s.current_period_end, s.cancelled_at, s.cancel_at_cycle_end,
		       s.paused_at, s.pending_package_id, s.upgraded_from_id, s.grace_period_end, s.metadata, s.created_at, s.updated_at,
		       COALESCE(p.id, '00000000-0000-0000-0000-000000000000'::uuid),
		       COALESCE(p.name, 'unknown'),
		       COALESCE(p.display_name, 'Unknown'),
		       COALESCE(p.price_paise, 0),
		       COALESCE(p.storage_limit_bytes, 0),
		       COALESCE(p.max_file_size_bytes, 0),
		       u.email, COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), u.email) as user_name
		FROM subscriptions s
		LEFT JOIN packages p ON s.package_id = p.id
		LEFT JOIN users u ON s.user_id = u.id
		WHERE ($1 = '' OR s.status = $1)
		ORDER BY s.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		var s model.Subscription
		var metaJSON []byte
		var pkg model.Package

		err := rows.Scan(
			&s.ID, &s.UserID, &s.PackageID, &s.RazorpaySubscriptionID, &s.RazorpayCustomerID, &s.RazorpayPaymentID,
			&s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelledAt, &s.CancelAtCycleEnd,
			&s.PausedAt, &s.PendingPackageID, &s.UpgradedFromID, &s.GracePeriodEnd, &metaJSON, &s.CreatedAt, &s.UpdatedAt,
			&pkg.ID, &pkg.Name, &pkg.DisplayName, &pkg.PricePaise, &pkg.StorageLimitBytes, &pkg.MaxFileSizeBytes,
			&s.UserEmail, &s.UserName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan subscription item: %w", err)
		}
		s.Package = &pkg
		subs = append(subs, &s)
	}

	return subs, total, nil
}

// UpdateMetadata updates the metadata JSONB field of a subscription.
func (r *SubscriptionRepository) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		UPDATE subscriptions SET
			metadata = $2,
			updated_at = NOW()
		WHERE id = $1
	`
	res, err := r.db.Exec(ctx, query, id, metaJSON)
	if err != nil {
		return fmt.Errorf("update subscription metadata: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}
