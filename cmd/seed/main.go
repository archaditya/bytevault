package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"

	"github.com/archaditya/bytevault/internal/config"
	"github.com/archaditya/bytevault/internal/database"
	"github.com/archaditya/bytevault/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.App.Env)

	dbPool, err := database.New(cfg.Database.DSN())
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close(dbPool)

	ctx := context.Background()

	// 1. Check if super_admin role exists
	var superAdminRoleID string
	err = dbPool.QueryRow(ctx, "SELECT id FROM roles WHERE name = $1", "super_admin").Scan(&superAdminRoleID)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Role 'super_admin' not found in database. Make sure migrations have run.")
	}

	// 2. Define admin user credentials from environment with secure fallback
	adminEmail := os.Getenv("SEED_ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@pushport.com"
	}

	adminPassword := os.Getenv("SEED_ADMIN_PASSWORD")
	isGenerated := false
	if adminPassword == "" {
		randBytes := make([]byte, 16)
		if _, err := rand.Read(randBytes); err != nil {
			logger.Log.Fatal().Err(err).Msg("Failed to generate secure random password")
		}
		adminPassword = hex.EncodeToString(randBytes) + "A1!"
		isGenerated = true
	}
	adminFirstName := "System"
	adminLastName := "Admin"

	// Check if admin user already exists
	var existingID string
	err = dbPool.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", adminEmail).Scan(&existingID)
	if err == nil {
		logger.Log.Info().Str("email", adminEmail).Msg("Admin user already exists")
		os.Exit(0)
	}

	// 3. Hash password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(adminPassword), 14)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to hash password")
	}
	hashedPassword := string(hashedBytes)

	// 4. Create admin user record
	var adminUserID string
	insertUserQuery := `
		INSERT INTO users (email, password, first_name, last_name, is_verified, status, updated_at)
		VALUES ($1, $2, $3, $4, true, 'active', NOW())
		RETURNING id
	`
	err = dbPool.QueryRow(ctx, insertUserQuery, adminEmail, hashedPassword, adminFirstName, adminLastName).Scan(&adminUserID)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to create admin user")
	}

	// 5. Assign super_admin role to admin user
	insertUserRoleQuery := `
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`
	_, err = dbPool.Exec(ctx, insertUserRoleQuery, adminUserID, superAdminRoleID)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to assign role to admin user")
	}

	logEvent := logger.Log.Info().
		Str("email", adminEmail).
		Str("role", "super_admin")

	if isGenerated {
		logEvent.Msg("Successfully seeded admin user with secure random password")
	} else {
		logEvent.Msg("Successfully seeded admin user from environment credentials")
	}
}
