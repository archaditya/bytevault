// Package server sets up the Echo HTTP server and wires everything together.
//
// GO CONCEPTS IN THIS FILE:
// - struct embedding:     Server struct holds an Echo instance and config
// - constructor pattern:  New() function creates and configures the server
// - method:               Start() is a method on the Server struct
// - dependency injection: We pass config INTO the server (not global state)
package server

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/adityakkpk/bytevault/internal/config"
	"github.com/adityakkpk/bytevault/internal/logger"
)

// Server wraps the Echo instance and holds all dependencies.
// This is the heart of our application — it owns the HTTP server
// and all the handlers/services that plug into it.
type Server struct {
	echo   *echo.Echo     // The Echo framework instance
	config *config.Config // Application configuration
}

// New creates and configures a new Server instance.
// This is where we:
// 1. Create the Echo instance
// 2. Add middleware (logging, recovery, CORS)
// 3. Register routes
// 4. Return the configured server
func New(cfg *config.Config) *Server {
	// Create a new Echo instance
	e := echo.New()

	// Hide Echo's default banner (the ASCII art logo)
	e.HideBanner = true

	// ---- MIDDLEWARE ----
	// Middleware are functions that run BEFORE your handler.
	// They form a chain: Request → Middleware1 → Middleware2 → Handler → Response

	// Recover middleware catches panics and returns a 500 error
	// instead of crashing the entire server.
	e.Use(middleware.Recover())

	// CORS middleware allows cross-origin requests (needed for frontend).
	// In production, you'd restrict AllowOrigins to your frontend URL.
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
	}))

	// Request ID middleware adds a unique ID to each request.
	// Useful for tracing requests across logs.
	e.Use(middleware.RequestID())

	// Create our server instance
	s := &Server{
		echo:   e,
		config: cfg,
	}

	// Register all routes
	s.registerRoutes()

	return s
}

// Start begins listening for HTTP requests on the configured port.
// This method BLOCKS — it runs until the server is shut down.
func (s *Server) Start() error {
	port := s.config.Server.Port
	if port == "" {
		port = "8080" // Default port
	}

	logger.Log.Info().
		Str("port", port).
		Str("env", s.config.App.Env).
		Msg("🚀 ByteVault server starting")

	// Start listening. The colon before port means "listen on all interfaces".
	// e.g., ":8080" = listen on 0.0.0.0:8080
	return s.echo.Start(":" + port)
}
