// Package main is the entry point for ByteVault API server.
//
// THE STARTUP SEQUENCE:
// 1. Load configuration (from .env and environment variables)
// 2. Initialize the logger (structured logging with Zerolog)
// 3. Create the HTTP server (Echo framework)
// 4. Start listening for requests
//
// GO CONCEPTS:
// - Package imports are grouped by: stdlib → external → internal
// - os.Exit(1) terminates with error status
// - The if err != nil pattern is Go's way of error handling (no try/catch)
package main

import (
	"os"

	"github.com/adityakkpk/bytevault/internal/config"
	"github.com/adityakkpk/bytevault/internal/logger"
	"github.com/adityakkpk/bytevault/internal/server"
)

func main() {
	// Step 1: Load configuration
	// This reads from .env file and environment variables.
	// If it fails, we can't start — so we exit immediately.
	cfg, err := config.Load()
	if err != nil {
		// We can't use our logger yet (it's not initialized),
		// so we print to stderr and exit.
		os.Stderr.WriteString("❌ Failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Step 2: Initialize the logger
	// We pass the environment (development/production) so the logger
	// knows whether to use pretty console output or JSON.
	logger.Init(cfg.App.Env)
	logger.Log.Info().Msg("✅ Configuration loaded successfully")

	// Step 3: Create and start the server
	// server.New() creates the Echo server with middleware and routes.
	// server.Start() blocks and listens for HTTP requests.
	srv := server.New(cfg)
	if err := srv.Start(); err != nil {
		logger.Log.Fatal().Err(err).Msg("❌ Server failed to start")
	}
}
