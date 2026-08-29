// Package ai provides AI-powered image labeling with a fallback provider chain.
// Supports Google Cloud Vision API (primary) and Cloudflare Workers AI (fallback).
// Both providers are optional — when credentials are not configured, labeling is silently skipped.
package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/archaditya/bytevault/internal/config"
	"github.com/archaditya/bytevault/internal/logger"
)

// MinConfidenceVision is the minimum confidence score for Google Vision labels.
const MinConfidenceVision = 0.60

// MinConfidenceCF is the minimum confidence score for Cloudflare AI labels.
// ResNet-50 distributes probability across 1000 classes, so top predictions often have 0.10-0.45 score.
const MinConfidenceCF = 0.10

// MaxLabels is the maximum number of labels to return per image.
const MaxLabels = 15

// ImageLabeler provides AI-powered image classification using a fallback chain.
// Primary: Google Cloud Vision API → Fallback: Cloudflare Workers AI.
type ImageLabeler struct {
	visionAPIKey string
	cfAccountID  string
	cfAPIToken   string
	httpClient   *http.Client
}

// NewImageLabeler creates a labeler from AI config. Returns nil if no providers are configured.
func NewImageLabeler(cfg config.AIConfig) *ImageLabeler {
	if cfg.VisionAPIKey == "" && (cfg.CFAccountID == "" || cfg.CFAPIToken == "") {
		logger.Log.Info().Msg("🏷️ No AI labeling providers configured. Image labeling will be skipped.")
		return nil
	}

	return &ImageLabeler{
		visionAPIKey: cfg.VisionAPIKey,
		cfAccountID:  cfg.CFAccountID,
		cfAPIToken:   cfg.CFAPIToken,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// IsEnabled returns true if at least one AI provider is configured.
func (l *ImageLabeler) IsEnabled() bool {
	return l != nil && (l.visionAPIKey != "" || (l.cfAccountID != "" && l.cfAPIToken != ""))
}

// LabelImage attempts to classify an image using the configured provider chain.
// Primary: Cloudflare Workers AI (100% Free) → Fallback: Google Cloud Vision API.
// Returns a list of descriptive labels (e.g., ["banana", "fruit", "food"]).
func (l *ImageLabeler) LabelImage(ctx context.Context, imageBytes []byte) []string {
	if l == nil || len(imageBytes) == 0 {
		return nil
	}

	// 1. Primary: Cloudflare Workers AI (100% Free & Fast)
	if l.cfAccountID != "" && l.cfAPIToken != "" {
		labels, err := l.labelWithCloudflareAI(ctx, imageBytes)
		if err != nil {
			logger.Log.Warn().Err(err).Msg("⚠️ Cloudflare AI labeling failed, trying Google fallback")
		} else if len(labels) > 0 {
			logger.Log.Info().Strs("labels", labels).Msg("🏷️ Cloudflare AI generated labels successfully")
			return labels
		}
	}

	// 2. Fallback / Secondary: Google Cloud Vision API
	if l.visionAPIKey != "" {
		labels, err := l.labelWithVisionAPI(ctx, imageBytes)
		if err != nil {
			logger.Log.Warn().Err(err).Msg("⚠️ Google Vision API labeling failed (all providers exhausted)")
		} else if len(labels) > 0 {
			logger.Log.Info().Strs("labels", labels).Msg("🏷️ Google Vision API generated labels successfully")
			return labels
		}
	}

	return nil
}

// --- Google Cloud Vision API ---

type visionRequest struct {
	Requests []visionAnnotateRequest `json:"requests"`
}

type visionAnnotateRequest struct {
	Image    visionImage    `json:"image"`
	Features []visionFeature `json:"features"`
}

type visionImage struct {
	Content string `json:"content"` // base64 encoded
}

type visionFeature struct {
	Type       string `json:"type"`
	MaxResults int    `json:"maxResults"`
}

type visionResponse struct {
	Responses []struct {
		LabelAnnotations []struct {
			Description string  `json:"description"`
			Score       float64 `json:"score"`
		} `json:"labelAnnotations"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"responses"`
}

func (l *ImageLabeler) labelWithVisionAPI(ctx context.Context, imageBytes []byte) ([]string, error) {
	encoded := base64.StdEncoding.EncodeToString(imageBytes)

	reqBody := visionRequest{
		Requests: []visionAnnotateRequest{
			{
				Image: visionImage{Content: encoded},
				Features: []visionFeature{
					{Type: "LABEL_DETECTION", MaxResults: MaxLabels},
				},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vision request: %w", err)
	}

	url := fmt.Sprintf("https://vision.googleapis.com/v1/images:annotate?key=%s", l.visionAPIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create vision request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vision API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vision API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var vResp visionResponse
	if err := json.NewDecoder(resp.Body).Decode(&vResp); err != nil {
		return nil, fmt.Errorf("failed to decode vision response: %w", err)
	}

	if len(vResp.Responses) == 0 {
		return nil, fmt.Errorf("vision API returned empty response")
	}

	if vResp.Responses[0].Error != nil {
		return nil, fmt.Errorf("vision API error: %s", vResp.Responses[0].Error.Message)
	}

	var labels []string
	for _, annotation := range vResp.Responses[0].LabelAnnotations {
		if annotation.Score >= MinConfidenceVision {
			labels = append(labels, strings.ToLower(annotation.Description))
		}
	}

	return labels, nil
}

// --- Cloudflare Workers AI ---

type cfAIResponse struct {
	Result []struct {
		Label string  `json:"label"`
		Score float64 `json:"score"`
	} `json:"result"`
	Success bool `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (l *ImageLabeler) labelWithCloudflareAI(ctx context.Context, imageBytes []byte) ([]string, error) {
	url := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/accounts/%s/ai/run/@cf/microsoft/resnet-50",
		l.cfAccountID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(imageBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create CF AI request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+l.cfAPIToken)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CF AI call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CF AI returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var cfResp cfAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return nil, fmt.Errorf("failed to decode CF AI response: %w", err)
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			return nil, fmt.Errorf("CF AI error: %s", cfResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("CF AI returned unsuccessful response")
	}

	var labels []string
	existingMap := make(map[string]bool)

	for _, result := range cfResp.Result {
		// Include predictions with score >= MinConfidenceCF or always include top 3 predictions
		if (result.Score >= MinConfidenceCF || len(labels) < 3) && len(labels) < MaxLabels {
			rawLabel := strings.ToLower(strings.TrimSpace(result.Label))
			rawLabel = strings.ReplaceAll(rawLabel, "_", " ")
			// Split comma separated terms: e.g. "espresso, coffee" -> ["espresso", "coffee"]
			parts := strings.Split(rawLabel, ",")
			for _, p := range parts {
				clean := strings.TrimSpace(p)
				if clean != "" && !existingMap[clean] && len(labels) < MaxLabels {
					existingMap[clean] = true
					labels = append(labels, clean)
				}
			}
		}
	}

	logger.Log.Info().
		Int("result_count", len(cfResp.Result)).
		Strs("parsed_labels", labels).
		Msg("🤖 Cloudflare Workers AI classification response parsed")

	return labels, nil
}
