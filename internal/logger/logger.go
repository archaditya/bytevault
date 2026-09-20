package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Log is a package-level variable — any file that imports this package
// can use logger.Log.Info().Msg("hello")
var Log zerolog.Logger

// DailyFileWriter appends all service logs to DD-MM-YYYY.app.log in the logs directory.
type DailyFileWriter struct {
	baseDir     string
	mu          sync.Mutex
	currentDate string
	currentFile *os.File
}

func (w *DailyFileWriter) getFile(dateStr string) (*os.File, error) {
	filePath := filepath.Join(w.baseDir, fmt.Sprintf("%s.app.log", dateStr))

	if w.currentFile != nil && w.currentDate == dateStr {
		// Verify the file was not deleted or unlinked on disk
		if _, err := os.Stat(filePath); err == nil {
			return w.currentFile, nil
		}
		// File was deleted externally; close dangling handle and recreate
		_ = w.currentFile.Close()
		w.currentFile = nil
	}

	if w.currentFile != nil {
		_ = w.currentFile.Sync()
		_ = w.currentFile.Close()
		w.currentFile = nil
	}

	_ = os.MkdirAll(w.baseDir, 0755)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	w.currentDate = dateStr
	w.currentFile = f
	return f, nil
}

func (w *DailyFileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	dateStr := time.Now().UTC().Format("02-01-2006")
	f, err := w.getFile(dateStr)
	if err != nil {
		return 0, err
	}
	return f.Write(p)
}

func (w *DailyFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentFile != nil {
		_ = w.currentFile.Sync()
		err := w.currentFile.Close()
		w.currentFile = nil
		return err
	}
	return nil
}

// Init configures the logger based on environment and starts writing to logs/DD-MM-YYYY.app.log.
func Init(env string) {
	_ = os.MkdirAll("logs", 0755)
	fileWriter := &DailyFileWriter{baseDir: "logs"}

	// Create today's log file immediately upon service startup
	todayStr := time.Now().UTC().Format("02-01-2006")
	if initF, err := fileWriter.getFile(todayStr); err == nil && initF != nil {
		_ = initF.Sync()
	}

	var multiWriter io.Writer
	if env == "development" {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		multiWriter = io.MultiWriter(consoleWriter, fileWriter)
		Log = zerolog.New(multiWriter).
			With().
			Timestamp().
			Caller().
			Logger()
	} else {
		multiWriter = io.MultiWriter(os.Stdout, fileWriter)
		Log = zerolog.New(multiWriter).
			With().
			Timestamp().
			Logger()
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
