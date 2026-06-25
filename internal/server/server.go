package server

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/archaditya/bytevault/internal/config"
	"github.com/archaditya/bytevault/internal/logger"
)

// Server holds the Echo instance and all dependencies.
type Server struct {
	echo   *echo.Echo
	config *config.Config
	db     *pgxpool.Pool
}

// New functon creates a server. This is a dependency injection:
// pass the config in, rather than reading it from a global
func New(cfg *config.Config, db *pgxpool.Pool) *Server {
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
			var event *zerolog.Event
			if v.Status >= 500 {
				event = logger.Log.Error()
				if v.Error != nil {
					event = event.Err(v.Error)
				}
			} else if v.Status >= 400 {
				event = logger.Log.Warn()
				if v.Error != nil {
					event = event.Err(v.Error)
				}
			} else {
				event = logger.Log.Info()
			}

			event.
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
		echo:   e,
		config: cfg,
		db:     db,
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
