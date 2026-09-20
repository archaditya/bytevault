// Package ai provides AI-powered content moderation for uploaded images.
// Uses a two-layer detection strategy:
//   - Primary: HuggingFace Inference API (Falconsai/nsfw_image_detection) — deep learning, high accuracy
//   - Fallback: go-nude library — skin-color heuristic analysis, zero external deps, runs locally
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/archaditya/bytevault/internal/logger"
	"github.com/koyachi/go-nude"
)

// NSFWResult holds the detection outcome.
type NSFWResult struct {
	Score  float64 // 0.0 (safe) to 1.0 (explicit)
	Label  string  // "nsfw" or "normal"
	Method string  // "huggingface" or "heuristic" or "skipped"
}

// NSFWDetector provides NSFW content detection with AI primary + heuristic fallback.
type NSFWDetector struct {
	hfAPIToken string
	httpClient *http.Client
}

// NewNSFWDetector creates a detector. Returns nil if no HF token is provided
// (heuristic fallback will still work via DetectNSFW).
func NewNSFWDetector(hfAPIToken string) *NSFWDetector {
	return &NSFWDetector{
		hfAPIToken: hfAPIToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// IsEnabled returns true if either HuggingFace API or heuristic fallback is available.
// The heuristic fallback (go-nude) is always available, so this always returns true.
func (d *NSFWDetector) IsEnabled() bool {
	return d != nil
}

// DetectNSFW analyzes image bytes for NSFW content using a two-layer strategy.
// Returns a result with score, label, and which method was used.
// This function is designed to NEVER return an error that blocks the pipeline.
func (d *NSFWDetector) DetectNSFW(ctx context.Context, imageBytes []byte) NSFWResult {
	if d == nil || len(imageBytes) == 0 {
		return NSFWResult{Score: 0, Label: "normal", Method: "skipped"}
	}

	// Layer 1: HuggingFace Inference API (primary, high accuracy)
	if d.hfAPIToken != "" {
		result, err := d.detectWithHuggingFace(ctx, imageBytes)
		if err == nil {
			logger.Log.Info().
				Float64("score", result.Score).
				Str("label", result.Label).
				Msg("🔍 NSFW scan completed via HuggingFace AI")
			return result
		}
		logger.Log.Warn().Err(err).Msg("⚠️ HuggingFace NSFW API failed, falling back to heuristic scan")
	}

	// Layer 2: go-nude heuristic fallback (local, zero external deps)
	result := d.detectWithHeuristic(imageBytes)
	logger.Log.Info().
		Float64("score", result.Score).
		Str("label", result.Label).
		Msg("🔍 NSFW scan completed via heuristic (go-nude)")
	return result
}

// --- HuggingFace Inference API ---

type hfClassification struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

func (d *NSFWDetector) detectWithHuggingFace(ctx context.Context, imageBytes []byte) (NSFWResult, error) {
	url := "https://api-inference.huggingface.co/models/Falconsai/nsfw_image_detection"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(imageBytes))
	if err != nil {
		return NSFWResult{}, fmt.Errorf("failed to create HF request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+d.hfAPIToken)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return NSFWResult{}, fmt.Errorf("HF API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return NSFWResult{}, fmt.Errorf("HF API returned status %d: %s", resp.StatusCode, string(body))
	}

	var classifications []hfClassification
	if err := json.NewDecoder(resp.Body).Decode(&classifications); err != nil {
		return NSFWResult{}, fmt.Errorf("failed to decode HF response: %w", err)
	}

	// Find the NSFW score from classifications
	// Falconsai returns: [{"label": "nsfw", "score": 0.97}, {"label": "normal", "score": 0.03}]
	var nsfwScore float64
	var topLabel string

	for _, c := range classifications {
		if c.Label == "nsfw" {
			nsfwScore = c.Score
		}
	}

	if nsfwScore >= 0.5 {
		topLabel = "nsfw"
	} else {
		topLabel = "normal"
	}

	return NSFWResult{
		Score:  nsfwScore,
		Label:  topLabel,
		Method: "huggingface",
	}, nil
}

// detectWithHeuristic runs the go-nude local heuristic scan.
//
// DESIGN DECISIONS & ARCHITECTURAL TRADEOFFS:
// 1. JPEG Fast-Path: If input bytes already have the JPEG SOI marker (0xFF, 0xD8, 0xFF),
//    we write raw bytes directly to temp file, bypassing image.Decode and jpeg.Encode overhead.
// 2. Format Normalization: For non-JPEG formats (PNG, GIF), the image is decoded and re-encoded
//    as a temporary JPEG so go-nude's internal jpeg.Decode can evaluate it.
// 3. Fail-Open on Decoder / I/O Errors: If an image format is unsupported, corrupted, or temp
//    file I/O fails, we return a safe 'normal' score (0.05). Under no circumstance should a
//    decoding or file I/O error cause user documents (e.g. certificates, badges) to be falsely
//    marked as NSFW (0.55) and hidden from the user's dashboard.
func (d *NSFWDetector) detectWithHeuristic(imageBytes []byte) NSFWResult {
	if len(imageBytes) == 0 {
		return NSFWResult{Score: 0.05, Label: "normal", Method: "skipped"}
	}

	tmpFile, err := os.CreateTemp("", "nsfw-scan-*.jpg")
	if err != nil {
		logger.Log.Warn().Err(err).Msg("⚠️ go-nude: failed to create temp file")
		return NSFWResult{Score: 0.05, Label: "normal", Method: "skipped"}
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Optimization: If image is already JPEG (SOI marker 0xFF 0xD8 0xFF), write directly
	isJPEG := len(imageBytes) >= 3 && imageBytes[0] == 0xFF && imageBytes[1] == 0xD8 && imageBytes[2] == 0xFF
	if isJPEG {
		if _, err := tmpFile.Write(imageBytes); err != nil {
			tmpFile.Close()
			logger.Log.Warn().Err(err).Msg("⚠️ go-nude: failed to write raw jpeg to temp file")
			return NSFWResult{Score: 0.05, Label: "normal", Method: "skipped"}
		}
		tmpFile.Close()
	} else {
		// Non-JPEG image (PNG, GIF, etc.): decode and re-encode to JPEG for go-nude compatibility
		img, _, err := image.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			tmpFile.Close()
			logger.Log.Debug().Err(err).Msg("go-nude: unable to decode image for heuristic scan, passing as safe")
			return NSFWResult{Score: 0.05, Label: "normal", Method: "skipped"}
		}

		if err := jpeg.Encode(tmpFile, img, &jpeg.Options{Quality: 80}); err != nil {
			tmpFile.Close()
			logger.Log.Warn().Err(err).Msg("⚠️ go-nude: failed to encode to jpeg")
			return NSFWResult{Score: 0.05, Label: "normal", Method: "skipped"}
		}
		tmpFile.Close()
	}

	// Run heuristic scan on the standardized JPEG file
	isNude, err := nude.IsNude(tmpPath)
	if err != nil {
		logger.Log.Debug().Err(err).Msg("go-nude: analysis failed, passing as safe")
		return NSFWResult{Score: 0.05, Label: "normal", Method: "skipped"}
	}

	if isNude {
		// Heuristic detection: Skin-tone pixel detector produces frequent false positives on
		// portraits, faces, and photos with warm color palettes. Cap score at 0.35 so it
		// is stored in the audit trail as an advisory indicator, but never auto-hides the file.
		return NSFWResult{Score: 0.35, Label: "heuristic_skin_detected", Method: "heuristic"}
	}
	return NSFWResult{Score: 0.05, Label: "normal", Method: "heuristic"}
}
