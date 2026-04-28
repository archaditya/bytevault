package server

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/adityakkpk/bytevault/internal/config"
	"github.com/adityakkpk/bytevault/internal/logger"
)

// Server holds the Echo instance and all dependencies.
type Server struct {
	echo	*echo.Echo
	config *config.Config
}

// New functon creates a server. This is a dependency injection:
// pass the config in, rather than reading it from a global
func New(cfg *config.Config) *Server{
	e := echo.New()
	e.HideBanner = true

	// Middleware = functions that run BEFORE handler
	// Request → Recover → CORS → RequestID → Handler → Response

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogError:    true,
		LogRemoteIP: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Log.Info().
				Int("status", v.Status).
				Str("method", v.Method).
				Str("uri", v.URI).
				Str("ip", v.RemoteIP).
				Str("latency", v.Latency.String()). 
				Msg("request")
			return nil
		},
	}))

	e.Use(middleware.Recover()) // Catches panics and return 500 error instead of crashing the server

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
	}))

	e.Use(middleware.RequestID()) // Adds unique ID to each request

	s := &Server{
		echo: e,
		config: cfg,
	}

	s.registerRoutes()

	return s
}

func (s *Server) Start() error {
	port := s.config.Server.Port
	if port == "" {
		port = "8080"
	}

	logger.Log.Info().Str("port", port).Str("env", s.config.App.Env).Msg("🚀 ByteVault server starting")

	return s.echo.Start(":" + port)
}