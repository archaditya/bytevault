package cloudinary

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type CloudinaryStorage struct {
	cld *cloudinary.Cloudinary
}

func NewCloudinaryStorage(cloudinaryURL string) (*CloudinaryStorage, error) {
	cld, err := cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		return nil, fmt.Errorf("cloudinary init failed: %w", err)
	}
	return &CloudinaryStorage{cld: cld}, nil
}

// Helper to get pointers to booleans for struct literals
func boolPtr(b bool) *bool {
	return &b
}

// Helper to convert storage_key into a safe public_id
func cleanPublicID(storageKey string) string {
	// Strip file extension because Cloudinary handles formats dynamically
	ext := filepath.Ext(storageKey)
	return strings.TrimSuffix(storageKey, ext)
}

func (c *CloudinaryStorage) Upload(ctx context.Context, storageKey string, content io.Reader, size int64, contentType string) (string, error) {
	publicID := cleanPublicID(storageKey)
	
	resp, err := c.cld.Upload.Upload(ctx, content, uploader.UploadParams{
		PublicID:       publicID,
		UniqueFilename: boolPtr(false),
		Overwrite:      boolPtr(true),
	})
	if err != nil {
		return "", fmt.Errorf("cloudinary upload failed: %w", err)
	}

	return resp.SecureURL, nil
}

func (c *CloudinaryStorage) Download(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	// Retrieve asset details using the Admin API
	resp, err := c.cld.Admin.Asset(ctx, admin.AssetParams{
		PublicID: cleanPublicID(storageKey),
	})
	var targetURL string
	if err == nil {
		targetURL = resp.SecureURL
	} else {
		// Fallback manual URL generation
		targetURL = fmt.Sprintf("https://res.cloudinary.com/fallback/image/upload/%s", storageKey)
	}

	httpResp, err := http.Get(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch asset from Cloudinary: %w", err)
	}
	return httpResp.Body, nil
}

func (c *CloudinaryStorage) Delete(ctx context.Context, storageKey string) error {
	publicID := cleanPublicID(storageKey)
	_, err := c.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
	})
	if err != nil {
		return fmt.Errorf("failed to destroy cloudinary asset: %w", err)
	}
	return nil
}

func (c *CloudinaryStorage) GeneratePresignedUploadURL(ctx context.Context, storageKey string, contentType string, expiry time.Duration) (string, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	publicID := cleanPublicID(storageKey)

	// Build parameter queries for signature validation
	params := url.Values{}
	params.Set("public_id", publicID)
	params.Set("timestamp", timestamp)
	
	signature, err := api.SignParameters(params, c.cld.Config.Cloud.APISecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign params: %w", err)
	}

	// Returns signed API endpoint url
	return fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/auto/upload?signature=%s&api_key=%s&timestamp=%s&public_id=%s",
		c.cld.Config.Cloud.CloudName, signature, c.cld.Config.Cloud.APIKey, timestamp, publicID), nil
}

func (c *CloudinaryStorage) GeneratePresignedDownloadURL(ctx context.Context, storageKey string, expiry time.Duration) (string, error) {
	url, err := c.cld.Image(cleanPublicID(storageKey))
	if err != nil {
		return "", err
	}
	signedURL, err := url.String()
	if err != nil {
		return "", err
	}
	return signedURL, nil
}
