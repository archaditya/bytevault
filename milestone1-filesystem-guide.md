# Milestone 1 — Pluggable Storage & Core File System

This guide outlines the complete implementation of **Milestone 1 — Core File System** supporting pluggable storage providers: **Local Disk**, **Cloudflare R2**, and **Cloudinary**.

---

## 🛠️ Step 1: Database Migration (`006_create_files_table.sql`)

Create or update `cmd/api/migrations/006_create_files_table.sql`:

```sql
-- cmd/api/migrations/006_create_files_table.sql

-- Create files table to store file metadata with multi-provider columns
CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    storage_provider VARCHAR(50) NOT NULL, -- 'local', 'cloudinary', or 'r2'
    bucket VARCHAR(255),                  -- Cloud storage bucket name (e.g. for R2)
    storage_key TEXT NOT NULL,            -- e.g. user/123/docs/resume.pdf
    file_size BIGINT NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    is_public BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_files_user_id ON files(user_id);
CREATE INDEX IF NOT EXISTS idx_files_deleted_at ON files(deleted_at);
CREATE INDEX IF NOT EXISTS idx_files_provider_key ON files(storage_provider, storage_key);

---- create above / drop below ----

DROP TABLE IF EXISTS files;
```

---

## ⚙️ Step 2: Environment Variables (`.env`)

Add the following to your `.env` and `.env.example` configurations:

```env
# Storage Configurations
# Options: "local", "cloudinary", "r2"
STORAGE_PROVIDER=local

# Local Storage settings
STORAGE_LOCAL_DIR=./uploads

# Cloudinary settings
STORAGE_CLOUDINARY_URL=cloudinary://your_api_key:your_api_secret@your_cloud_name

# Cloudflare R2 / S3 settings
STORAGE_R2_ENDPOINT=https://your-account-id.r2.cloudflarestorage.com
STORAGE_R2_ACCESS_KEY_ID=your-access-key-id
STORAGE_R2_SECRET_ACCESS_KEY=your-secret-access-key
STORAGE_R2_BUCKET=bytevault-bucket
```

Update `internal/config/config.go` to capture these:

```go
type Config struct {
	Server   ServerConfig   `koanf:"server"`
	Database DatabaseConfig `koanf:"db"`
	App      AppConfig      `koanf:"app"`
	JWT      JWTConfig      `koanf:"jwt"`
	Storage  StorageConfig  `koanf:"storage"` // <-- Add this
}

type StorageConfig struct {
	Provider          string `koanf:"provider"` // local, cloudinary, r2
	LocalDir          string `koanf:"localdir"`
	CloudinaryURL     string `koanf:"cloudinaryurl"`
	R2Endpoint        string `koanf:"r2endpoint"`
	R2AccessKeyID     string `koanf:"r2accesskeyid"`
	R2SecretAccessKey string `koanf:"r2secretaccesskey"`
	R2Bucket          string `koanf:"r2bucket"`
}
```

---

## 📦 Step 3: File Model (`internal/model/file.go`)

Create `internal/model/file.go`:

```go
package model

import (
	"time"
)

type File struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	Filename        string     `json:"filename"`
	StorageProvider string     `json:"storage_provider"` // local, cloudinary, r2
	Bucket          *string    `json:"bucket,omitempty"` // Nullable for providers without buckets
	StorageKey      string     `json:"storage_key"`      // e.g., user/123/docs/resume.pdf
	FileSize        int64      `json:"file_size"`
	ContentType     string     `json:"content_type"`
	IsPublic        bool       `json:"is_public"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"-"`
}
```

---

## 💾 Step 4: Storage Infrastructure (`storage/`)

### 1. Storage Interface (`internal/storage/provider.go`)

Create `internal/storage/provider.go` defining the new interface:

```go
package storage

import (
	"context"
	"io"
	"time"
)

type StorageProvider interface {
	// Upload saves the file content and returns a public URL or identifier path
	Upload(ctx context.Context, storageKey string, content io.Reader, size int64, contentType string) (string, error)
	// Download retrieves the binary data stream for the given storage key
	Download(ctx context.Context, storageKey string) (io.ReadCloser, error)
	// Delete removes the file physically
	Delete(ctx context.Context, storageKey string) error
	// GeneratePresignedUploadURL creates a temporary URL allowing clients to PUT/POST files directly
	GeneratePresignedUploadURL(ctx context.Context, storageKey string, contentType string, expiry time.Duration) (string, error)
	// GeneratePresignedDownloadURL creates a secure temporary URL for downloading private assets
	GeneratePresignedDownloadURL(ctx context.Context, storageKey string, expiry time.Duration) (string, error)
}
```

### 2. Local File Provider (`internal/storage/local/local.go`)

Create `internal/storage/local/local.go`:

```go
package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create base upload directory: %w", err)
	}
	return &LocalStorage{baseDir: baseDir}, nil
}

func (l *LocalStorage) getAbsPath(storageKey string) string {
	return filepath.Join(l.baseDir, storageKey)
}

func (l *LocalStorage) Upload(ctx context.Context, storageKey string, content io.Reader, size int64, contentType string) (string, error) {
	fullPath := l.getAbsPath(storageKey)
	
	// Ensure directory nesting exists
	if err := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create subdirectories: %w", err)
	}

	dest, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create local file: %w", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, content); err != nil {
		return "", fmt.Errorf("failed to write local file: %w", err)
	}

	return fullPath, nil
}

func (l *LocalStorage) Download(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	fullPath := l.getAbsPath(storageKey)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local file: %w", err)
	}
	return file, nil
}

func (l *LocalStorage) Delete(ctx context.Context, storageKey string) error {
	fullPath := l.getAbsPath(storageKey)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete local file: %w", err)
	}
	return nil
}

func (l *LocalStorage) GeneratePresignedUploadURL(ctx context.Context, storageKey string, contentType string, expiry time.Duration) (string, error) {
	// Dev environment shortcut: direct local upload route
	return fmt.Sprintf("/api/v1/files/upload/direct?key=%s", storageKey), nil
}

func (l *LocalStorage) GeneratePresignedDownloadURL(ctx context.Context, storageKey string, expiry time.Duration) (string, error) {
	// Dev environment shortcut: direct file access route
	return fmt.Sprintf("/api/v1/files/download/direct?key=%s", storageKey), nil
}
```

### 3. Cloudinary Provider (`internal/storage/cloudinary/cloudinary.go`)

Create `internal/storage/cloudinary/cloudinary.go`:

```go
package cloudinary

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
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
		UniqueFilename: false,
		Overwrite:      true,
	})
	if err != nil {
		return "", fmt.Errorf("cloudinary upload failed: %w", err)
	}

	return resp.SecureURL, nil
}

func (c *CloudinaryStorage) Download(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	// Cloudinary resources are public secure URLs. To get the stream, perform HTTP request.
	resp, err := c.cld.Admin.Asset(ctx, uploader.AssetParams{
		PublicID: cleanPublicID(storageKey),
	})
	var url string
	if err == nil {
		url = resp.SecureURL
	} else {
		// Fallback manual URL generation
		url = fmt.Sprintf("https://res.cloudinary.com/fallback/image/upload/%s", storageKey)
	}

	httpResp, err := http.Get(url)
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
	// Generate signed upload parameters for clients to upload directly to Cloudinary
	params := map[string]interface{}{
		"public_id": cleanPublicID(storageKey),
		"timestamp": time.Now().Unix(),
	}
	
	signature, err := cloudinary.SignParameters(params, c.cld.Config.APISecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign params: %w", err)
	}

	// Returns parameters formatted for signed client-side requests
	return fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/auto/upload?signature=%s&api_key=%s&timestamp=%d&public_id=%s",
		c.cld.Config.CloudName, signature, c.cld.Config.APIKey, params["timestamp"], params["public_id"]), nil
}

func (c *CloudinaryStorage) GeneratePresignedDownloadURL(ctx context.Context, storageKey string, expiry time.Duration) (string, error) {
	// Cloudinary doesn't use S3-like presigned URLs for private files natively.
	// You can generate authenticated URLs using their SDK:
	url, err := c.cld.Image(cleanPublicID(storageKey))
	if err != nil {
		return "", err
	}
	// Add signature logic or return secure URL directly
	signedURL, err := url.String()
	if err != nil {
		return "", err
	}
	return signedURL, nil
}
```

### 4. Cloudflare R2 Provider (`internal/storage/r2/r2.go`)

Install the AWS SDK modules first:
```powershell
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/credentials
go get github.com/aws/aws-sdk-go-v2/service/s3
```

Create `internal/storage/r2/r2.go`:

```go
package r2

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Storage struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
}

func NewR2Storage(endpoint, accessKey, secretKey, bucket string) (*R2Storage, error) {
	// Custom endpoint resolver specifically targeting Cloudflare R2 subdomain endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint, // e.g. https://<account-id>.r2.cloudflarestorage.com
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("auto"), // R2 region must be "auto"
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(client)

	return &R2Storage{
		client:        client,
		presignClient: presignClient,
		bucket:        bucket,
	}, nil
}

func (r *R2Storage) Upload(ctx context.Context, storageKey string, content io.Reader, size int64, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r.bucket),
		Key:           aws.String(storageKey),
		Body:          content,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object to R2: %w", err)
	}

	return fmt.Sprintf("%s/%s", r.bucket, storageKey), nil
}

func (r *R2Storage) Download(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	resp, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from R2: %w", err)
	}
	return resp.Body, nil
}

func (r *R2Storage) Delete(ctx context.Context, storageKey string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from R2: %w", err)
	}
	return nil
}

func (r *R2Storage) GeneratePresignedUploadURL(ctx context.Context, storageKey string, contentType string, expiry time.Duration) (string, error) {
	req, err := r.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(storageKey),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}
	return req.URL, nil
}

func (r *R2Storage) GeneratePresignedDownloadURL(ctx context.Context, storageKey string, expiry time.Duration) (string, error) {
	req, err := r.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(storageKey),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}
	return req.URL, nil
}
```

---

## 🗄️ Step 5: File Repository (`internal/repository/file_repository.go`)

Create `internal/repository/file_repository.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/archaditya/bytevault/internal/model"
)

type FileRepository struct {
	db *pgxpool.Pool
}

func NewFileRepository(db *pgxpool.Pool) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Create(ctx context.Context, file *model.File) error {
	query := `
		INSERT INTO files (user_id, filename, storage_provider, bucket, storage_key, file_size, content_type, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		file.UserID,
		file.Filename,
		file.StorageProvider,
		file.Bucket,
		file.StorageKey,
		file.FileSize,
		file.ContentType,
		file.IsPublic,
	).Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create file record: %w", err)
	}
	return nil
}

func (r *FileRepository) FindByID(ctx context.Context, id string) (*model.File, error) {
	query := `
		SELECT id, user_id, filename, storage_provider, bucket, storage_key, file_size, content_type, is_public, created_at, updated_at
		FROM files
		WHERE id = $1 AND deleted_at IS NULL
	`
	var file model.File
	err := r.db.QueryRow(ctx, query, id).Scan(
		&file.ID,
		&file.UserID,
		&file.Filename,
		&file.StorageProvider,
		&file.Bucket,
		&file.StorageKey,
		&file.FileSize,
		&file.ContentType,
		&file.IsPublic,
		&file.CreatedAt,
		&file.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find file metadata: %w", err)
	}
	return &file, nil
}

func (r *FileRepository) ListByUserID(ctx context.Context, userID string) ([]*model.File, error) {
	query := `
		SELECT id, user_id, filename, storage_provider, bucket, storage_key, file_size, content_type, is_public, created_at, updated_at
		FROM files
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		var f model.File
		err := rows.Scan(
			&f.ID,
			&f.UserID,
			&f.Filename,
			&f.StorageProvider,
			&f.Bucket,
			&f.StorageKey,
			&f.FileSize,
			&f.ContentType,
			&f.IsPublic,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	return files, nil
}

func (r *FileRepository) UpdatePublicStatus(ctx context.Context, id string, isPublic bool) error {
	query := `UPDATE files SET is_public = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, isPublic, id)
	return err
}

func (r *FileRepository) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE files SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
```

---

## ⚡ Step 6: File Service (`internal/service/file_service.go`)

Create `internal/service/file_service.go`:

```go
package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/storage"
)

type FileService struct {
	repo            *repository.FileRepository
	storage         storage.StorageProvider
	storageProvider string
	bucket          *string
}

func NewFileService(repo *repository.FileRepository, storage storage.StorageProvider, provider string, bucket string) *FileService {
	var bPtr *string
	if bucket != "" {
		bPtr = &bucket
	}
	return &FileService{
		repo:            repo,
		storage:         storage,
		storageProvider: provider,
		bucket:          bPtr,
	}
}

// generateStorageKey builds a structural key: user/{userID}/docs/{filename}
func (s *FileService) generateStorageKey(userID, filename string) string {
	return fmt.Sprintf("user/%s/docs/%s", userID, filepath.Base(filename))
}

func (s *FileService) Upload(ctx context.Context, userID, filename string, size int64, contentType string, content io.Reader) (*model.File, error) {
	storageKey := s.generateStorageKey(userID, filename)

	// 1. Upload file contents
	storagePath, err := s.storage.Upload(ctx, storageKey, content, size, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed upload in storage: %w", err)
	}

	// 2. Save metadata record to DB
	fileMeta := &model.File{
		UserID:          userID,
		Filename:        filename,
		StorageProvider: s.storageProvider,
		Bucket:          s.bucket,
		StorageKey:      storageKey,
		FileSize:        size,
		ContentType:     contentType,
		IsPublic:        false,
	}

	if err := s.repo.Create(ctx, fileMeta); err != nil {
		// Cleanup storage on DB failure
		_ = s.storage.Delete(ctx, storageKey)
		return nil, err
	}

	return fileMeta, nil
}

func (s *FileService) Download(ctx context.Context, fileID, userID string) (io.ReadCloser, *model.File, error) {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	if file == nil {
		return nil, nil, fmt.Errorf("file not found")
	}

	if !file.IsPublic && file.UserID != userID {
		return nil, nil, fmt.Errorf("unauthorized to download this file")
	}

	stream, err := s.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire stream: %w", err)
	}

	return stream, file, nil
}

func (s *FileService) DownloadPublic(ctx context.Context, fileID string) (io.ReadCloser, *model.File, error) {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	if file == nil || !file.IsPublic {
		return nil, nil, fmt.Errorf("file not found or private")
	}

	stream, err := s.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, nil, err
	}

	return stream, file, nil
}

func (s *FileService) ListUserFiles(ctx context.Context, userID string) ([]*model.File, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *FileService) ToggleShareStatus(ctx context.Context, fileID, userID string, isPublic bool) error {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file == nil || file.UserID != userID {
		return fmt.Errorf("file not found or unauthorized")
	}
	return s.repo.UpdatePublicStatus(ctx, fileID, isPublic)
}

func (s *FileService) Delete(ctx context.Context, fileID, userID string) error {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file == nil || file.UserID != userID {
		return fmt.Errorf("file not found or unauthorized")
	}

	if err := s.storage.Delete(ctx, file.StorageKey); err != nil {
		return err
	}

	return s.repo.SoftDelete(ctx, fileID)
}
```

---

## ⚙️ Step 7: File HTTP Handler (`internal/handler/file_handler.go`)

Create `internal/handler/file_handler.go`:

```go
package handler

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/service"
)

type FileHandler struct {
	service *service.FileService
}

func NewFileHandler(service *service.FileService) *FileHandler {
	return &FileHandler{service: service}
}

func (h *FileHandler) Upload(c echo.Context) error {
	userID := c.Get("user_id").(string)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing file form field"})
	}

	src, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to open upload source"})
	}
	defer src.Close()

	fileMeta, err := h.service.Upload(
		c.Request().Context(),
		userID,
		fileHeader.Filename,
		fileHeader.Size,
		fileHeader.Header.Get("Content-Type"),
		src,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "File uploaded successfully",
		"file":    fileMeta,
	})
}

func (h *FileHandler) Download(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	stream, fileMeta, err := h.service.Download(c.Request().Context(), fileID, userID)
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
	}
	defer stream.Close()

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileMeta.Filename)
	c.Response().Header().Set(echo.HeaderContentType, fileMeta.ContentType)
	c.Response().WriteHeader(http.StatusOK)
	
	_, err = io.Copy(c.Response().Writer, stream)
	return err
}

func (h *FileHandler) DownloadPublic(c echo.Context) error {
	fileID := c.Param("id")

	stream, fileMeta, err := h.service.DownloadPublic(c.Request().Context(), fileID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	defer stream.Close()

	c.Response().Header().Set(echo.HeaderContentDisposition, "inline; filename="+fileMeta.Filename)
	c.Response().Header().Set(echo.HeaderContentType, fileMeta.ContentType)
	c.Response().WriteHeader(http.StatusOK)

	_, err = io.Copy(c.Response().Writer, stream)
	return err
}

func (h *FileHandler) List(c echo.Context) error {
	userID := c.Get("user_id").(string)

	files, err := h.service.ListUserFiles(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"files": files})
}

func (h *FileHandler) ToggleShare(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	var req struct {
		IsPublic bool `json:"is_public"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	err := h.service.ToggleShareStatus(c.Request().Context(), fileID, userID, req.IsPublic)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Share status updated successfully"})
}

func (h *FileHandler) Delete(c echo.Context) error {
	fileID := c.Param("id")
	userID := c.Get("user_id").(string)

	err := h.service.Delete(c.Request().Context(), fileID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "File deleted successfully"})
}
```

---

## 🌐 Step 8: Route Integration & Server Setup

Create `internal/server/file_routes.go` to cleanly register these:

```go
package server

import (
	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/handler"
)

func registerFileRoutes(g *echo.Group, fh *handler.FileHandler, authMiddleware echo.MiddlewareFunc) {
	// Public routes
	g.GET("/files/public/:id", fh.DownloadPublic)

	// Protected routes
	filesGroup := g.Group("/files", authMiddleware)
	{
		filesGroup.POST("/upload", fh.Upload)
		filesGroup.GET("", fh.List)
		filesGroup.GET("/:id/download", fh.Download)
		filesGroup.PATCH("/:id/share", fh.ToggleShare)
		filesGroup.DELETE("/:id", fh.Delete)
	}
}
```

In `internal/server/routes.go`, map your imports and initialize:

```go
// Setup inside internal/server/server.go's setup/routing logic:
import (
	"github.com/archaditya/bytevault/internal/storage"
	"github.com/archaditya/bytevault/internal/storage/local"
	"github.com/archaditya/bytevault/internal/storage/cloudinary"
	"github.com/archaditya/bytevault/internal/storage/r2"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
	"github.com/archaditya/bytevault/internal/handler"
)

// Inside routing method initialization:
var store storage.StorageProvider
var err error

switch cfg.Storage.Provider {
case "r2":
	store, err = r2.NewR2Storage(
		cfg.Storage.R2Endpoint,
		cfg.Storage.R2AccessKeyID,
		cfg.Storage.R2SecretAccessKey,
		cfg.Storage.R2Bucket,
	)
case "cloudinary":
	store, err = cloudinary.NewCloudinaryStorage(cfg.Storage.CloudinaryURL)
default:
	store, err = local.NewLocalStorage(cfg.Storage.LocalDir)
}

if err != nil {
	logger.Log.Fatal().Err(err).Msg("Failed to initialize storage provider")
}

fileRepo := repository.NewFileRepository(dbPool)
fileService := service.NewFileService(fileRepo, store, cfg.Storage.Provider, cfg.Storage.R2Bucket)
fileHandler := handler.NewFileHandler(fileService)

// registerFileRoutes(apiGroup, fileHandler, authMiddleware)
```
