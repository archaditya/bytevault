// Package logger provides a pre-configured Zerolog logger for the application.
//
// GO CONCEPTS IN THIS FILE:
// - Global variable:  `Log` is a package-level variable accessible by importing this package.
// - zerolog.Logger:   The logger type from zerolog library.
// - ConsoleWriter:    A human-friendly log formatter (colored, pretty-printed).
//                     In production, you'd use raw JSON (just zerolog.New(os.Stdout)).
// - Method chaining:  zerolog uses builder pattern — each method returns the logger
//                     so you can chain: New().With().Timestamp().Str().Logger()
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Log is the global logger instance.
// Other packages use it like: logger.Log.Info().Msg("hello")
var Log zerolog.Logger

// Init sets up the global logger.
// In development, we use ConsoleWriter for pretty, colored output.
// In production, you'd switch to JSON output for log aggregation.
//
// WHY NOT init()?
// We could use Go's special init() function (auto-called before main),
// but explicit Init(env) gives us control over WHEN and HOW the logger
// is configured, especially since we need the app environment setting.
func Init(env string) {
	// In development, use a pretty console writer
	if env == "development" {
		// ConsoleWriter formats logs like:
		// 10:30PM INF Server started port=8080
		// Instead of raw JSON: {"level":"info","port":"8080","message":"Server started"}
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		Log = zerolog.New(output).
			With().           // With() starts a context builder
			Timestamp().      // Adds timestamp to every log line
			Caller().         // Adds file:line to every log line (great for debugging!)
			Logger()
	} else {
		// In production, use JSON output (for tools like ELK, Datadog, etc.)
		Log = zerolog.New(os.Stdout).
			With().
			Timestamp().
			Logger()
	}

	// Set global log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
