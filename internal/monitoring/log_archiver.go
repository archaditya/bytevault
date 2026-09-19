package monitoring

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/storage"
	"github.com/rs/zerolog/log"
)

// SystemLogTracker abstracts DB operations for tracking log files across local, R2, and purged states.
type SystemLogTracker interface {
	UpsertLocalLog(ctx context.Context, logDate time.Time, fileName, localPath string, fileSize int64) error
	MarkArchivedToR2(ctx context.Context, logDate time.Time, storageKey string, compressedSize int64) error
	ListExpiredR2(ctx context.Context, cutoff time.Time) ([]*model.SystemLogArchive, error)
	MarkPurged(ctx context.Context, logDate string) error
}

// LogArchiver archives daily app log files to persistent cloud storage (R2) and purges logs older than 6 months.
type LogArchiver struct {
	baseDir  string
	provider storage.StorageProvider
	tracker  SystemLogTracker
}

// NewLogArchiver creates a new log archiver.
func NewLogArchiver(baseDir string, provider storage.StorageProvider) *LogArchiver {
	return &LogArchiver{
		baseDir:  baseDir,
		provider: provider,
	}
}

// SetTracker attaches a DB repository for tracking log lifecycle records.
func (a *LogArchiver) SetTracker(tracker SystemLogTracker) {
	a.tracker = tracker
}

// ArchivePastLogs keeps 30 days of logs on the server and uploads older logs to R2.
func (a *LogArchiver) ArchivePastLogs(ctx context.Context) error {
	if a.provider == nil {
		log.Warn().Msg("LogArchiver: no storage provider configured, skipping log upload")
		return nil
	}

	// 1 month (30 days) retention on the server
	now := time.Now().UTC()
	retentionCutoff := now.AddDate(0, 0, -30)

	files, err := os.ReadDir(a.baseDir)
	if err != nil {
		return fmt.Errorf("read audit log dir: %w", err)
	}

	for _, entry := range files {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".app.log") {
			continue
		}

		dateStr := strings.TrimSuffix(entry.Name(), ".app.log")
		fileDate, err := time.Parse("02-01-2006", dateStr)
		if err != nil {
			continue
		}

		filePath := filepath.Join(a.baseDir, entry.Name())
		info, infoErr := entry.Info()
		fileSize := int64(0)
		if infoErr == nil && info != nil {
			fileSize = info.Size()
		}

		// If the file is within the 30-day window, register/update it as local
		if fileDate.After(retentionCutoff) {
			if a.tracker != nil {
				_ = a.tracker.UpsertLocalLog(ctx, fileDate, entry.Name(), filePath, fileSize)
			}
			continue
		}

		// File is older than 30 days: compress and upload to R2
		if err := a.archiveFile(ctx, filePath, fileDate, dateStr); err != nil {
			log.Error().Err(err).Str("file", entry.Name()).Msg("LogArchiver: failed to archive log file")
			continue
		}
	}

	return nil
}

func (a *LogArchiver) archiveFile(ctx context.Context, localPath string, fileDate time.Time, dateStr string) error {
	rawBytes, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("read local log file: %w", err)
	}

	if len(rawBytes) == 0 {
		_ = os.Remove(localPath)
		return nil
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(rawBytes); err != nil {
		return fmt.Errorf("gzip compress log: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}

	// Cloudflare R2 storage key format: logs/app/YYYY/MM/DD-MM-YYYY.app.log.gz
	storageKey := fmt.Sprintf("logs/app/%s/%s/%s.app.log.gz",
		fileDate.Format("2006"),
		fileDate.Format("01"),
		dateStr,
	)

	compressedData := buf.Bytes()
	_, err = a.provider.Upload(ctx, storageKey, bytes.NewReader(compressedData), int64(len(compressedData)), "application/gzip")
	if err != nil {
		return fmt.Errorf("upload log archive to storage: %w", err)
	}

	// Update DB record that log is now on Cloudflare R2
	if a.tracker != nil {
		_ = a.tracker.MarkArchivedToR2(ctx, fileDate, storageKey, int64(len(compressedData)))
	}

	log.Info().Str("key", storageKey).Msg("LogArchiver: successfully uploaded log archive to R2")
	_ = os.Remove(localPath)
	return nil
}

// PurgeExpiredLogs permanently deletes log files older than 6 months (180 days) from Cloudflare R2.
func (a *LogArchiver) PurgeExpiredLogs(ctx context.Context) error {
	if a.provider == nil || a.tracker == nil {
		return nil
	}

	// 6 months (180 days) expiration cutoff
	cutoff := time.Now().UTC().AddDate(0, 0, -180)
	expiredLogs, err := a.tracker.ListExpiredR2(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("list expired r2 logs: %w", err)
	}

	for _, item := range expiredLogs {
		if item.StorageKey != nil && *item.StorageKey != "" {
			if delErr := a.provider.Delete(ctx, *item.StorageKey); delErr != nil {
				log.Warn().Err(delErr).Str("key", *item.StorageKey).Msg("LogArchiver: failed to delete expired log from R2")
				continue
			}
			log.Info().Str("key", *item.StorageKey).Msg("LogArchiver: permanently purged log older than 6 months from R2")
		}
		_ = a.tracker.MarkPurged(ctx, item.LogDate)
	}

	return nil
}
