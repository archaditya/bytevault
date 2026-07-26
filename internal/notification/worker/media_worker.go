package worker

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/archaditya/bytevault/internal/logger"
	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/notification/queue"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/storage"
)

type MediaWorker struct {
	fileRepo *repository.FileRepository
	storage  storage.StorageProvider
	queue    *queue.RedisQueue
	stopChan chan struct{}
}

func NewMediaWorker(fileRepo *repository.FileRepository, storage storage.StorageProvider, queue *queue.RedisQueue) *MediaWorker {
	return &MediaWorker{
		fileRepo: fileRepo,
		storage:  storage,
		queue:    queue,
		stopChan: make(chan struct{}),
	}
}

// Start spawns background consumer goroutines listening to Redis QueueMediaProcessing
func (w *MediaWorker) Start(concurrency int) {
	for i := 0; i < concurrency; i++ {
		workerID := i + 1
		go w.runWorker(workerID)
	}
	logger.Log.Info().Int("workers", concurrency).Msg("🚀 Media Processing Workers started")
}

func (w *MediaWorker) Stop() {
	close(w.stopChan)
}

func (w *MediaWorker) runWorker(id int) {
	ctx := context.Background()
	for {
		select {
		case <-w.stopChan:
			return
		default:
			job, err := w.queue.DequeueMediaJob(ctx, 2*time.Second)
			if err != nil || job == nil {
				continue
			}

			fileID, _ := job.Payload["file_id"].(string)
			if fileID == "" {
				continue
			}

			if err := w.ProcessFile(ctx, fileID); err != nil {
				logger.Log.Error().Err(err).Int("worker_id", id).Str("file_id", fileID).Msg("Failed to process media thumbnail")
			}
		}
	}
}

// ProcessFile determines the appropriate handler (Image, Video, PDF) based on Content-Type
func (w *MediaWorker) ProcessFile(ctx context.Context, fileID string) error {
	file, err := w.fileRepo.FindByID(ctx, fileID)
	if err != nil || file == nil {
		return fmt.Errorf("file not found: %s", fileID)
	}

	contentType := strings.ToLower(file.ContentType)
	ext := strings.ToLower(filepath.Ext(file.Filename))

	var thumbnailBytes []byte

	switch {
	case strings.HasPrefix(contentType, "image/") && contentType != "image/svg+xml":
		thumbnailBytes, err = w.processImage(ctx, file)
	case strings.HasPrefix(contentType, "video/") || ext == ".mp4" || ext == ".mov" || ext == ".mkv":
		thumbnailBytes, err = w.processVideo(ctx, file)
	case contentType == "application/pdf" || ext == ".pdf":
		thumbnailBytes, err = w.processPDF(ctx, file)
	default:
		return nil // Unsupported format for thumbnails, skip cleanly
	}

	if err != nil {
		return err
	}

	if len(thumbnailBytes) == 0 {
		return nil
	}

	thumbnailKey := fmt.Sprintf("user/%s/thumbnails/%s.jpg", file.UserID, file.ID)
	_, err = w.storage.Upload(ctx, thumbnailKey, bytes.NewReader(thumbnailBytes), int64(len(thumbnailBytes)), "image/jpeg")
	if err != nil {
		return fmt.Errorf("failed to upload thumbnail to storage: %w", err)
	}

	if err := w.fileRepo.UpdateThumbnailKey(ctx, file.ID, thumbnailKey); err != nil {
		return fmt.Errorf("failed to update thumbnail_key in DB: %w", err)
	}

	logger.Log.Info().Str("file_id", file.ID).Str("thumbnail_key", thumbnailKey).Msg("Media thumbnail generated & persisted")
	return nil
}

// 1. Image Handler (Go native image packages)
func (w *MediaWorker) processImage(ctx context.Context, file *model.File) ([]byte, error) {
	stream, err := w.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer stream.Close()

	var srcImg image.Image
	switch file.ContentType {
	case "image/jpeg", "image/jpg":
		srcImg, err = jpeg.Decode(stream)
	case "image/png":
		srcImg, err = png.Decode(stream)
	default:
		srcImg, _, err = image.Decode(stream)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return resizeImageToJPEG(srcImg, 300)
}

// 2. Video Handler (FFmpeg subprocess extraction)
func (w *MediaWorker) processVideo(ctx context.Context, file *model.File) ([]byte, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		logger.Log.Warn().Msg("FFmpeg executable not found on host. Video thumbnail skipped.")
		return nil, nil
	}

	stream, err := w.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to download video stream: %w", err)
	}
	defer stream.Close()

	tmpDir := os.TempDir()
	tmpInput := filepath.Join(tmpDir, fmt.Sprintf("vid_in_%s%s", file.ID, filepath.Ext(file.Filename)))
	tmpOutput := filepath.Join(tmpDir, fmt.Sprintf("vid_out_%s.jpg", file.ID))

	outFile, err := os.Create(tmpInput)
	if err != nil {
		return nil, err
	}
	_, _ = io.Copy(outFile, stream)
	_ = outFile.Close()

	defer os.Remove(tmpInput)
	defer os.Remove(tmpOutput)

	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-y",
		"-ss", "00:00:01",
		"-i", tmpInput,
		"-vframes", "1",
		"-vf", "scale=300:300:force_original_aspect_ratio=decrease",
		tmpOutput,
	)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg extraction failed: %w", err)
	}

	return os.ReadFile(tmpOutput)
}

// 3. PDF Handler (pdftoppm Poppler subprocess rendering)
func (w *MediaWorker) processPDF(ctx context.Context, file *model.File) ([]byte, error) {
	pdftoppmPath, err := exec.LookPath("pdftoppm")
	if err != nil {
		logger.Log.Warn().Msg("pdftoppm executable not found on host. PDF thumbnail skipped.")
		return nil, nil
	}

	stream, err := w.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to download PDF stream: %w", err)
	}
	defer stream.Close()

	tmpDir := os.TempDir()
	tmpInput := filepath.Join(tmpDir, fmt.Sprintf("pdf_in_%s.pdf", file.ID))
	tmpOutPrefix := filepath.Join(tmpDir, fmt.Sprintf("pdf_out_%s", file.ID))

	outFile, err := os.Create(tmpInput)
	if err != nil {
		return nil, err
	}
	_, _ = io.Copy(outFile, stream)
	_ = outFile.Close()

	defer os.Remove(tmpInput)
	defer os.Remove(tmpOutPrefix + "-1.jpg")

	cmd := exec.CommandContext(ctx, pdftoppmPath,
		"-jpeg",
		"-r", "150",
		"-f", "1",
		"-l", "1",
		tmpInput,
		tmpOutPrefix,
	)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm rendering failed: %w", err)
	}

	renderedFile := tmpOutPrefix + "-1.jpg"
	return os.ReadFile(renderedFile)
}

func resizeImageToJPEG(srcImg image.Image, maxDim int) ([]byte, error) {
	bounds := srcImg.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	targetW, targetH := width, height
	if width > maxDim || height > maxDim {
		if width > height {
			targetW = maxDim
			targetH = (height * maxDim) / width
		} else {
			targetH = maxDim
			targetW = (width * maxDim) / height
		}
	}

	dstImg := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			srcX := (x * width) / targetW
			srcY := (y * height) / targetH
			dstImg.Set(x, y, srcImg.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dstImg, &jpeg.Options{Quality: 75}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
