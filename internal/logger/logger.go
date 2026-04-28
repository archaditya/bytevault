package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Log is a package-level variable — any file that imports this package
// can use logger.Log.Info().Msg("hello")
var Log zerolog.Logger

// Init configures the logger based on environment.
// We don't use Go's special init() function because we want
// EXPLICIT control over when this runs (after config is loaded)
func Init(env string) {
	if env == "development" {
		// Pretty colored output for development
		// Instead of JSON: {"level":"info","message":"hello"}
		// You get: 10:30PM INF hello
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		Log = zerolog.New(output).
			With().      // Start adding context fields
			Timestamp(). // Add timestamp to every log
			Caller().    // Add file:line to every log
			Logger()     // Finalize and return the logger
	} else {
		// JSON output for production (for log aggregation tools)
		Log = zerolog.New(os.Stdout).
			With().
			Timestamp().
			Logger()
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
