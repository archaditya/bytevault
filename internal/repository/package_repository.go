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
	ErrPackageNotFound = errors.New("package not found")
)

type PackageRepository struct {
	db *pgxpool.Pool
}

func NewPackageRepository(db *pgxpool.Pool) *PackageRepository {
	return &PackageRepository{db: db}
}

func (r *PackageRepository) FindAll(ctx context.Context, onlyActive bool) ([]*model.Package, error) {
	query := `
		SELECT id, name, display_name, description, price_paise, gst_rate, price_with_gst_paise,
		       currency, billing_period, storage_limit_bytes, max_file_size_bytes, razorpay_plan_id,
		       is_active, sort_order, features, created_at, updated_at
		FROM packages
	`
	if onlyActive {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY sort_order ASC, created_at ASC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query packages: %w", err)
	}
	defer rows.Close()

	var packages []*model.Package
	for rows.Next() {
		var p model.Package
		var featuresJSON []byte

		err := rows.Scan(
			&p.ID, &p.Name, &p.DisplayName, &p.Description, &p.PricePaise, &p.GSTRate, &p.PriceWithGSTPaise,
			&p.Currency, &p.BillingPeriod, &p.StorageLimitBytes, &p.MaxFileSizeBytes, &p.RazorpayPlanID,
			&p.IsActive, &p.SortOrder, &featuresJSON, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan package: %w", err)
		}

		if len(featuresJSON) > 0 {
			_ = json.Unmarshal(featuresJSON, &p.Features)
		}
		packages = append(packages, &p)
	}

	return packages, nil
}

func (r *PackageRepository) FindByID(ctx context.Context, id string) (*model.Package, error) {
	query := `
		SELECT id, name, display_name, description, price_paise, gst_rate, price_with_gst_paise,
		       currency, billing_period, storage_limit_bytes, max_file_size_bytes, razorpay_plan_id,
		       is_active, sort_order, features, created_at, updated_at
		FROM packages
		WHERE id = $1
	`
	var p model.Package
	var featuresJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.Description, &p.PricePaise, &p.GSTRate, &p.PriceWithGSTPaise,
		&p.Currency, &p.BillingPeriod, &p.StorageLimitBytes, &p.MaxFileSizeBytes, &p.RazorpayPlanID,
		&p.IsActive, &p.SortOrder, &featuresJSON, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPackageNotFound
		}
		return nil, fmt.Errorf("query package by id: %w", err)
	}

	if len(featuresJSON) > 0 {
		_ = json.Unmarshal(featuresJSON, &p.Features)
	}
	return &p, nil
}

func (r *PackageRepository) FindByName(ctx context.Context, name string) (*model.Package, error) {
	query := `
		SELECT id, name, display_name, description, price_paise, gst_rate, price_with_gst_paise,
		       currency, billing_period, storage_limit_bytes, max_file_size_bytes, razorpay_plan_id,
		       is_active, sort_order, features, created_at, updated_at
		FROM packages
		WHERE LOWER(name) = LOWER($1)
	`
	var p model.Package
	var featuresJSON []byte

	err := r.db.QueryRow(ctx, query, name).Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.Description, &p.PricePaise, &p.GSTRate, &p.PriceWithGSTPaise,
		&p.Currency, &p.BillingPeriod, &p.StorageLimitBytes, &p.MaxFileSizeBytes, &p.RazorpayPlanID,
		&p.IsActive, &p.SortOrder, &featuresJSON, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPackageNotFound
		}
		return nil, fmt.Errorf("query package by name: %w", err)
	}

	if len(featuresJSON) > 0 {
		_ = json.Unmarshal(featuresJSON, &p.Features)
	}
	return &p, nil
}

func (r *PackageRepository) FindByRazorpayPlanID(ctx context.Context, planID string) (*model.Package, error) {
	query := `
		SELECT id, name, display_name, description, price_paise, gst_rate, price_with_gst_paise,
		       currency, billing_period, storage_limit_bytes, max_file_size_bytes, razorpay_plan_id,
		       is_active, sort_order, features, created_at, updated_at
		FROM packages
		WHERE razorpay_plan_id = $1
	`
	var p model.Package
	var featuresJSON []byte

	err := r.db.QueryRow(ctx, query, planID).Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.Description, &p.PricePaise, &p.GSTRate, &p.PriceWithGSTPaise,
		&p.Currency, &p.BillingPeriod, &p.StorageLimitBytes, &p.MaxFileSizeBytes, &p.RazorpayPlanID,
		&p.IsActive, &p.SortOrder, &featuresJSON, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPackageNotFound
		}
		return nil, fmt.Errorf("query package by razorpay plan id: %w", err)
	}

	if len(featuresJSON) > 0 {
		_ = json.Unmarshal(featuresJSON, &p.Features)
	}
	return &p, nil
}

func (r *PackageRepository) Create(ctx context.Context, pkg *model.Package) (*model.Package, error) {
	featuresJSON, _ := json.Marshal(pkg.Features)
	if len(featuresJSON) == 0 {
		featuresJSON = []byte("{}")
	}

	query := `
		INSERT INTO packages (
			name, display_name, description, price_paise, gst_rate, price_with_gst_paise,
			currency, billing_period, storage_limit_bytes, max_file_size_bytes, razorpay_plan_id,
			is_active, sort_order, features, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		pkg.Name, pkg.DisplayName, pkg.Description, pkg.PricePaise, pkg.GSTRate, pkg.PriceWithGSTPaise,
		pkg.Currency, pkg.BillingPeriod, pkg.StorageLimitBytes, pkg.MaxFileSizeBytes, pkg.RazorpayPlanID,
		pkg.IsActive, pkg.SortOrder, featuresJSON,
	).Scan(&pkg.ID, &pkg.CreatedAt, &pkg.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("create package: %w", err)
	}
	return pkg, nil
}

func (r *PackageRepository) Update(ctx context.Context, pkg *model.Package) error {
	featuresJSON, _ := json.Marshal(pkg.Features)
	if len(featuresJSON) == 0 {
		featuresJSON = []byte("{}")
	}

	query := `
		UPDATE packages SET
			name = $2,
			display_name = $3,
			description = $4,
			price_paise = $5,
			gst_rate = $6,
			price_with_gst_paise = $7,
			currency = $8,
			billing_period = $9,
			storage_limit_bytes = $10,
			max_file_size_bytes = $11,
			razorpay_plan_id = $12,
			is_active = $13,
			sort_order = $14,
			features = $15,
			updated_at = $16
		WHERE id = $1
	`
	now := time.Now()
	res, err := r.db.Exec(ctx, query,
		pkg.ID, pkg.Name, pkg.DisplayName, pkg.Description, pkg.PricePaise, pkg.GSTRate, pkg.PriceWithGSTPaise,
		pkg.Currency, pkg.BillingPeriod, pkg.StorageLimitBytes, pkg.MaxFileSizeBytes, pkg.RazorpayPlanID,
		pkg.IsActive, pkg.SortOrder, featuresJSON, now,
	)
	if err != nil {
		return fmt.Errorf("update package: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrPackageNotFound
	}
	pkg.UpdatedAt = now
	return nil
}

func (r *PackageRepository) UpdateRazorpayPlanID(ctx context.Context, id, planID string) error {
	query := `UPDATE packages SET razorpay_plan_id = $2, updated_at = NOW() WHERE id = $1`
	res, err := r.db.Exec(ctx, query, id, planID)
	if err != nil {
		return fmt.Errorf("update razorpay plan id: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrPackageNotFound
	}
	return nil
}

func (r *PackageRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM packages WHERE id = $1`
	res, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete package: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrPackageNotFound
	}
	return nil
}
