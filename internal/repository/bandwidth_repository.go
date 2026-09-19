package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TopBandwidthConsumer struct {
	UserID     *string `json:"user_id"`
	UserEmail  *string `json:"user_email"`
	TotalBytes int64   `json:"total_bytes"`
	Count      int64   `json:"transfer_count"`
}

type EgressSummary struct {
	TotalBytesTransferred int64                  `json:"total_bytes_transferred"`
	TotalTransferCount    int64                  `json:"total_transfer_count"`
	Breakdown             map[string]int64       `json:"breakdown"`
	TopConsumers          []TopBandwidthConsumer `json:"top_consumers"`
}

type BandwidthRepository struct {
	db *pgxpool.Pool
}

func NewBandwidthRepository(db *pgxpool.Pool) *BandwidthRepository {
	return &BandwidthRepository{db: db}
}

func (r *BandwidthRepository) RecordEgress(ctx context.Context, userID *string, fileID *string, bytesTransferred int64, transferType, ip string) error {
	query := `
		INSERT INTO bandwidth_logs (user_id, file_id, bytes_transferred, transfer_type, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
	_, err := r.db.Exec(ctx, query, userID, fileID, bytesTransferred, transferType, ip)
	if err != nil {
		return fmt.Errorf("record bandwidth log: %w", err)
	}
	return nil
}

func (r *BandwidthRepository) GetEgressSummary(ctx context.Context, since time.Time) (*EgressSummary, error) {
	summary := &EgressSummary{
		Breakdown:    make(map[string]int64),
		TopConsumers: make([]TopBandwidthConsumer, 0),
	}

	// 1. Total and transfer count
	totQuery := `
		SELECT COALESCE(SUM(bytes_transferred), 0), COUNT(*)
		FROM bandwidth_logs
		WHERE created_at >= $1
	`
	err := r.db.QueryRow(ctx, totQuery, since).Scan(&summary.TotalBytesTransferred, &summary.TotalTransferCount)
	if err != nil {
		return nil, fmt.Errorf("query total bandwidth: %w", err)
	}

	// 2. Type breakdown
	typeQuery := `
		SELECT transfer_type, COALESCE(SUM(bytes_transferred), 0)
		FROM bandwidth_logs
		WHERE created_at >= $1
		GROUP BY transfer_type
	`
	rows, err := r.db.Query(ctx, typeQuery, since)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tType string
			var bSum int64
			if err := rows.Scan(&tType, &bSum); err == nil {
				summary.Breakdown[tType] = bSum
			}
		}
	}

	// 3. Top 5 consumers
	topQuery := `
		SELECT b.user_id, u.email, SUM(b.bytes_transferred) as total, COUNT(*) as cnt
		FROM bandwidth_logs b
		LEFT JOIN users u ON b.user_id = u.id
		WHERE b.created_at >= $1 AND b.user_id IS NOT NULL
		GROUP BY b.user_id, u.email
		ORDER BY total DESC
		LIMIT 5
	`
	topRows, err := r.db.Query(ctx, topQuery, since)
	if err == nil {
		defer topRows.Close()
		for topRows.Next() {
			var c TopBandwidthConsumer
			if err := topRows.Scan(&c.UserID, &c.UserEmail, &c.TotalBytes, &c.Count); err == nil {
				summary.TopConsumers = append(summary.TopConsumers, c)
			}
		}
	}

	return summary, nil
}
