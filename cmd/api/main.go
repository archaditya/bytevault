package main

import (
	"embed"
	"os"

	"github.com/archaditya/bytevault/internal/config"
	"github.com/archaditya/bytevault/internal/database"
	"github.com/archaditya/bytevault/internal/logger"
	"github.com/archaditya/bytevault/internal/server"
)

// WHAT IS //go:embed?
// This is a COMPILER DIRECTIVE — a special comment Go reads at compile time.
// It takes the "migrations" folder and bakes ALL files inside it into your binary.
// After compilation, migrationsFS contains the SQL files — no external files needed.
//
// embed.FS implements fs.FS, so we can pass it to our RunMigrations function.
//
//go:embed migrations
var migrationsFS embed.FS

func main() {
	// Step 1: Load config
	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("Failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Step 2: Init logger
	logger.Init(cfg.App.Env)
	logger.Log.Info().Msg("Configuration loaded")

	// Step 3: Connect to database
	// cfg.Database.DSN() builds the connection string from .env values
	dbPool, err := database.New(cfg.Database.DSN())
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close(dbPool) // Close pool when main() exits
	logger.Log.Info().Msg("Database connected")

	// Step 4: Run migrations automatically
	// Pass migrationsFS which contains our embedded SQL files
	if err := database.RunMigrations(dbPool, migrationsFS); err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to run migrations")
	}
	logger.Log.Info().Msg("Migrations Completed.")

	// Step 5: Create and start server
	srv := server.New(cfg, dbPool)
	if err := srv.Start(); err != nil {
		logger.Log.Fatal().Err(err).Msg("Server failed")
	}
}
