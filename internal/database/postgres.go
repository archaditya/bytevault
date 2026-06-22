package database

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/archaditya/bytevault/internal/logger"
)

// New creates a PostgreSQL connection pool.
//
// WHAT IS context.Context?
// Context is Go's way of passing "request-scoped" data and cancellation signals.
// Think of it as a bag that carries:
//   - Deadlines: "cancel this operation if it takes more than 5 seconds"
//   - Cancellation: "the user disconnected, stop working"
// Almost every Go function that does I/O (database, HTTP, file) takes a context.
//
// WHAT IS pgxpool.Pool?
// A pool of database connections. It:
//   - Opens multiple connections to PostgreSQL
//   - Reuses them across requests (no overhead of creating new connections)
//   - Handles connection health checks automatically
func New(dsn string) (*pgxpool.Pool, error) {
	// Parse the DSN (connection string) into pool configuration
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Pool settings
	config.MaxConns = 25                      // Max open connections at once
	config.MinConns = 5                       // Keep at least 5 connections warm
	config.MaxConnLifetime = 1 * time.Hour    // Recycle connections after 1 hour
	config.MaxConnIdleTime = 30 * time.Minute // Close idle connections after 30 min

	// Create a context with a 10-second timeout for the initial connection.
	// If PostgreSQL doesn't respond in 10 seconds, we give up.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // IMPORTANT: always call cancel() to release resources

	// Connect to the database
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Ping the database to verify the connection actually works
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}

// Close gracefully shuts down the connection pool.
// Call this when the server is shutting down.
func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}

// RunMigrations executes all SQL migration files embedded in the binary.
//
// HOW IT WORKS:
// 1. Creates a schema_migrations table to track which migrations have run
// 2. Reads all .sql files from the embedded filesystem
// 3. For each file NOT already in schema_migrations, runs the "up" part
// 4. The "up" part is everything ABOVE "---- create above / drop below ----"
//
// WHAT IS fs.FS?
// fs.FS is Go's filesystem interface. It can represent real files on disk
// OR embedded files (via embed). This makes our function flexible.
func RunMigrations(pool *pgxpool.Pool, migrationsFS fs.FS) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create tracking table if it doesn't exist
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Read all .sql files from the embedded filesystem
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Collect and sort .sql files (001_, 002_, 003_...)
	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	// Run each migration that hasn't been applied yet
	for _, fileName := range sqlFiles {
		// Check if already applied
		var exists bool
		err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
			fileName,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", fileName, err)
		}

		if exists {
			continue // Already applied, skip
		}

		// Read the file content
		fullPath := "migrations/" + fileName
		content, err := fs.ReadFile(migrationsFS, fullPath)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", fileName, err)
		}

		// Extract only the "up" part (everything before the separator)
		upSQL := extractUpSQL(string(content))

		// Execute the migration
		_, err = pool.Exec(ctx, upSQL)
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", fileName, err)
		}

		// Record that this migration was applied
		_, err = pool.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)",
			fileName,
		)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", fileName, err)
		}

		logger.Log.Info().Str("migration", fileName).Msg("Migration applied")
	}

	return nil
}

// extractUpSQL gets only the "create" part of a migration file.
// Everything before "---- create above / drop below ----" is the UP migration.
// If the separator doesn't exist, the entire file is used.
func extractUpSQL(content string) string {
	separator := "---- create above / drop below ----"
	if idx := strings.Index(content, separator); idx != -1 {
		return strings.TrimSpace(content[:idx])
	}
	return strings.TrimSpace(content)
}


/*

context.Context :=	Carries deadlines and cancellation. Almost every I/O function needs it
context.Background() :=	The "root" context — used when there's no parent context
context.WithTimeout() :=	Creates a context that auto-cancels after a duration
defer cancel() :=	defer means "run this when the function exits". Ensures cleanup happens
*pgxpool.Pool :=	Pointer to a connection pool object
time.Hour :=	Go's time package has built-in duration constants


*/