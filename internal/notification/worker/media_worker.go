package worker

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
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
	"github.com/archaditya/bytevault/internal/security"
	"github.com/archaditya/bytevault/internal/storage"
)

type MediaWorker struct {
	fileRepo       *repository.FileRepository
	storage        storage.StorageProvider
	queue          *queue.RedisQueue
	malwareScanner security.MalwareScanner
	stopChan       chan struct{}
}

func NewMediaWorker(fileRepo *repository.FileRepository, storage storage.StorageProvider, queue *queue.RedisQueue) *MediaWorker {
	return &MediaWorker{
		fileRepo:       fileRepo,
		storage:        storage,
		queue:          queue,
		malwareScanner: security.NewDefaultMalwareScanner(),
		stopChan:       make(chan struct{}),
	}
}

// Start spawns background consumer goroutines listening to Redis QueueMediaProcessing
func (w *MediaWorker) Start(concurrency int) {
	for i := 0; i < concurrency; i++ {
		workerID := i + 1
		go w.runWorker(workerID)
	}
	logger.Log.Info().Int("workers", concurrency).Msg("🚀 Media & Security Workers started")
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
				logger.Log.Error().Err(err).Int("worker_id", id).Str("file_id", fileID).Msg("Failed to process file security/thumbnail")
			}
		}
	}
}

// ProcessFile executes 1. Malware Scan -> 2. Thumbnail Generation -> 3. Sets Status READY
func (w *MediaWorker) ProcessFile(ctx context.Context, fileID string) error {
	file, err := w.fileRepo.FindByID(ctx, fileID)
	if err != nil || file == nil {
		return fmt.Errorf("file not found: %s", fileID)
	}

	// 1. Malware & Virus Scanning Stage
	if w.malwareScanner != nil {
		scanStream, err := w.storage.Download(ctx, file.StorageKey)
		if err == nil {
			scanResult, scanErr := w.malwareScanner.ScanStream(ctx, scanStream)
			_ = scanStream.Close()
			if scanErr == nil && scanResult != nil && !scanResult.IsClean {
				_ = w.storage.Delete(ctx, file.StorageKey)
				_ = w.fileRepo.UpdateStatus(ctx, file.ID, "BLOCKED_MALWARE")
				logger.Log.Warn().Str("file_id", file.ID).Str("threat", scanResult.Threat).Msg("🛡️ File blocked by background malware scan")
				return fmt.Errorf("file infected: %s", scanResult.Threat)
			}
		}
	}

	// 2. Thumbnail Generation Stage
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
		// Documents, spreadsheets, text files — generate branded poster
		thumbnailBytes = generateDocumentPoster(ext)
	}

	if err == nil && len(thumbnailBytes) > 0 {
		thumbnailKey := fmt.Sprintf("user/%s/thumbnails/%s.jpg", file.UserID, file.ID)
		_, uploadErr := w.storage.Upload(ctx, thumbnailKey, bytes.NewReader(thumbnailBytes), int64(len(thumbnailBytes)), "image/jpeg")
		if uploadErr == nil {
			_ = w.fileRepo.UpdateThumbnailKey(ctx, file.ID, thumbnailKey)
		}
	}

	// 3. Mark File Status = 'READY' (Unlocks Access Gate for Users)
	if err := w.fileRepo.UpdateStatus(ctx, file.ID, "READY"); err != nil {
		return fmt.Errorf("failed to update status to READY: %w", err)
	}

	logger.Log.Info().Str("file_id", file.ID).Msg("✅ File passed security scan & thumbnailing -> Marked READY")
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
		logger.Log.Warn().Str("file_id", file.ID).Msg("FFmpeg not found. Using fallback poster for video thumbnail.")
		return generateDocumentPoster(filepath.Ext(file.Filename)), nil
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
		"-vf", "scale=300:-1",
		tmpOutput,
	)

	if err := cmd.Run(); err != nil {
		logger.Log.Warn().Err(err).Str("file_id", file.ID).Msg("FFmpeg frame extraction failed. Using fallback poster.")
		return generateDocumentPoster(filepath.Ext(file.Filename)), nil
	}

	data, err := os.ReadFile(tmpOutput)
	if err != nil || len(data) == 0 {
		return generateDocumentPoster(filepath.Ext(file.Filename)), nil
	}
	return data, nil
}

// 3. PDF Handler (pdftoppm Poppler subprocess rendering)
func (w *MediaWorker) processPDF(ctx context.Context, file *model.File) ([]byte, error) {
	pdftoppmPath, err := exec.LookPath("pdftoppm")
	if err != nil {
		logger.Log.Warn().Str("file_id", file.ID).Msg("pdftoppm not found. Using fallback poster for PDF thumbnail.")
		return generateDocumentPoster(".pdf"), nil
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
		logger.Log.Warn().Err(err).Str("file_id", file.ID).Msg("pdftoppm rendering failed. Using fallback poster.")
		return generateDocumentPoster(".pdf"), nil
	}

	renderedFile := tmpOutPrefix + "-1.jpg"
	data, err := os.ReadFile(renderedFile)
	if err != nil || len(data) == 0 {
		return generateDocumentPoster(".pdf"), nil
	}
	return data, nil
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

// generateDocumentPoster creates a branded 300x300 placeholder thumbnail for non-image files.
// Uses a color-coded background based on file extension with the extension label centered.
func generateDocumentPoster(ext string) []byte {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	if ext == "" {
		ext = "FILE"
	} else {
		ext = strings.ToUpper(ext)
	}

	// Color palette based on file category
	var bgR, bgG, bgB uint8
	switch ext {
	case "MP4", "MOV", "MKV", "WEBM", "AVI":
		bgR, bgG, bgB = 139, 92, 246 // Purple for video
	case "PDF":
		bgR, bgG, bgB = 239, 68, 68 // Red for PDF
	case "DOC", "DOCX":
		bgR, bgG, bgB = 59, 130, 246 // Blue for Word
	case "XLS", "XLSX", "CSV":
		bgR, bgG, bgB = 34, 197, 94 // Green for spreadsheets
	case "PPT", "PPTX":
		bgR, bgG, bgB = 249, 115, 22 // Orange for presentations
	case "TXT", "MD":
		bgR, bgG, bgB = 107, 114, 128 // Gray for text
	case "ZIP", "RAR", "7Z", "TAR":
		bgR, bgG, bgB = 168, 85, 247 // Violet for archives
	default:
		bgR, bgG, bgB = 75, 85, 99 // Neutral gray
	}

	width, height := 300, 300
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, colorRGBA(bgR, bgG, bgB, 255))
		}
	}

	// Draw a centered lighter rectangle as a "document card" effect (80x100 centered)
	cardLeft, cardTop := 110, 80
	cardRight, cardBottom := 190, 220
	for y := cardTop; y < cardBottom; y++ {
		for x := cardLeft; x < cardRight; x++ {
			img.SetRGBA(x, y, colorRGBA(255, 255, 255, 40))
		}
	}

	// Draw extension text as pixel blocks (simple 5x7 bitmap font for up to 4 chars)
	drawExtLabel(img, ext, width, height)

	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

func colorRGBA(r, g, b, a uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: a}
}

// drawExtLabel renders a simple pixel-font extension label centered on the image
func drawExtLabel(img *image.RGBA, label string, imgW, imgH int) {
	// Simple 5x7 bitmap font for A-Z, 0-9
	glyphs := map[byte][7]uint8{
		'A': {0x0E, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
		'B': {0x1E, 0x11, 0x11, 0x1E, 0x11, 0x11, 0x1E},
		'C': {0x0E, 0x11, 0x10, 0x10, 0x10, 0x11, 0x0E},
		'D': {0x1E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x1E},
		'E': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F},
		'F': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x10},
		'G': {0x0E, 0x11, 0x10, 0x17, 0x11, 0x11, 0x0E},
		'H': {0x11, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
		'I': {0x0E, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E},
		'J': {0x07, 0x02, 0x02, 0x02, 0x02, 0x12, 0x0C},
		'K': {0x11, 0x12, 0x14, 0x18, 0x14, 0x12, 0x11},
		'L': {0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1F},
		'M': {0x11, 0x1B, 0x15, 0x11, 0x11, 0x11, 0x11},
		'N': {0x11, 0x19, 0x15, 0x13, 0x11, 0x11, 0x11},
		'O': {0x0E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
		'P': {0x1E, 0x11, 0x11, 0x1E, 0x10, 0x10, 0x10},
		'Q': {0x0E, 0x11, 0x11, 0x11, 0x15, 0x12, 0x0D},
		'R': {0x1E, 0x11, 0x11, 0x1E, 0x14, 0x12, 0x11},
		'S': {0x0E, 0x11, 0x10, 0x0E, 0x01, 0x11, 0x0E},
		'T': {0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
		'U': {0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
		'V': {0x11, 0x11, 0x11, 0x11, 0x0A, 0x0A, 0x04},
		'W': {0x11, 0x11, 0x11, 0x11, 0x15, 0x1B, 0x11},
		'X': {0x11, 0x11, 0x0A, 0x04, 0x0A, 0x11, 0x11},
		'Y': {0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04},
		'Z': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x10, 0x1F},
		'0': {0x0E, 0x11, 0x13, 0x15, 0x19, 0x11, 0x0E},
		'1': {0x04, 0x0C, 0x04, 0x04, 0x04, 0x04, 0x0E},
		'2': {0x0E, 0x11, 0x01, 0x06, 0x08, 0x10, 0x1F},
		'3': {0x0E, 0x11, 0x01, 0x06, 0x01, 0x11, 0x0E},
		'4': {0x02, 0x06, 0x0A, 0x12, 0x1F, 0x02, 0x02},
		'5': {0x1F, 0x10, 0x1E, 0x01, 0x01, 0x11, 0x0E},
		'6': {0x06, 0x08, 0x10, 0x1E, 0x11, 0x11, 0x0E},
		'7': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x08, 0x08},
		'8': {0x0E, 0x11, 0x11, 0x0E, 0x11, 0x11, 0x0E},
		'9': {0x0E, 0x11, 0x11, 0x0F, 0x01, 0x02, 0x0C},
	}

	scale := 4
	charW := 5 * scale
	charH := 7 * scale
	gap := 2 * scale

	if len(label) > 4 {
		label = label[:4]
	}

	totalW := len(label)*charW + (len(label)-1)*gap
	startX := (imgW - totalW) / 2
	startY := (imgH - charH) / 2

	for ci, ch := range label {
		glyph, ok := glyphs[byte(ch)]
		if !ok {
			continue
		}
		ox := startX + ci*(charW+gap)
		for row := 0; row < 7; row++ {
			for col := 0; col < 5; col++ {
				if glyph[row]&(1<<uint(4-col)) != 0 {
					for dy := 0; dy < scale; dy++ {
						for dx := 0; dx < scale; dx++ {
							px := ox + col*scale + dx
							py := startY + row*scale + dy
							if px >= 0 && px < imgW && py >= 0 && py < imgH {
								img.SetRGBA(px, py, colorRGBA(255, 255, 255, 220))
							}
						}
					}
				}
			}
		}
	}
}
