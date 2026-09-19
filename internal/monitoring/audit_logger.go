package monitoring

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEvent represents a single line in the JSONL daily log file.
type AuditEvent struct {
	Timestamp      string                 `json:"timestamp"`
	EventType      string                 `json:"event_type"`
	EventSource    string                 `json:"event_source"`
	UserID         string                 `json:"user_id,omitempty"`
	SubscriptionID string                 `json:"subscription_id,omitempty"`
	TransactionID  string                 `json:"transaction_id,omitempty"`
	Status         string                 `json:"status"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	Payload        map[string]interface{} `json:"payload,omitempty"`
	IPAddress      string                 `json:"ip_address,omitempty"`
	UserAgent      string                 `json:"user_agent,omitempty"`
}

// FileAuditLogger writes subscription and billing audit entries to daily rotating JSONL files.
type FileAuditLogger struct {
	baseDir     string
	mu          sync.Mutex
	currentDate string
	currentFile *os.File
}

// NewFileAuditLogger initializes the audit logger rooted at baseDir (e.g., "logs/subscription").
func NewFileAuditLogger(baseDir string) (*FileAuditLogger, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create audit log dir: %w", err)
	}

	logger := &FileAuditLogger{
		baseDir: baseDir,
	}
	return logger, nil
}

func (l *FileAuditLogger) getFileForDate(dateStr string) (*os.File, error) {
	if l.currentFile != nil && l.currentDate == dateStr {
		return l.currentFile, nil
	}

	if l.currentFile != nil {
		_ = l.currentFile.Sync()
		_ = l.currentFile.Close()
		l.currentFile = nil
	}

	filePath := filepath.Join(l.baseDir, fmt.Sprintf("%s.app.log", dateStr))
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open app log file: %w", err)
	}

	l.currentDate = dateStr
	l.currentFile = f
	return f, nil
}

// Log writes an audit event to today's DD-MM-YYYY.app.log file.
func (l *FileAuditLogger) Log(event AuditEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()
	if event.Timestamp == "" {
		event.Timestamp = now.Format(time.RFC3339)
	}

	dateStr := now.Format("02-01-2006")
	f, err := l.getFileForDate(dateStr)
	if err != nil {
		return err
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	data = append(data, '\n')
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}

	return nil
}

// Write allows FileAuditLogger to satisfy io.Writer for general logging.
func (l *FileAuditLogger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()
	dateStr := now.Format("02-01-2006")
	f, err := l.getFileForDate(dateStr)
	if err != nil {
		return 0, err
	}
	return f.Write(p)
}

// Close flushes and closes the active file handle.
func (l *FileAuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentFile != nil {
		_ = l.currentFile.Sync()
		err := l.currentFile.Close()
		l.currentFile = nil
		return err
	}
	return nil
}
