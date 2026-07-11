package r2

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type R2Storage struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
}

func NewR2Storage(endpoint, accessKey, secretKey, bucket string) (*R2Storage, error) {
	// Clean endpoint if it contains the bucket name as a path suffix
	suffix := "/" + bucket
	if strings.HasSuffix(endpoint, suffix) {
		endpoint = strings.TrimSuffix(endpoint, suffix)
	}

	// 1. Load AWS SDK base configuration with static credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("auto"), // R2 region must be "auto"
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	// 2. Initialize S3 client forcing PathStyle for R2 compatibility
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint) // Custom endpoint for Cloudflare R2
		o.UsePathStyle = true                 // CRITICAL: Cloudflare R2 requires path-style addressing
	})
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

func (r *R2Storage) GeneratePresignedDownloadURL(ctx context.Context, storageKey string, expiry time.Duration, filename string, inline bool) (string, error) {
	disposition := "attachment; filename=\"" + filename + "\""
	if inline {
		disposition = "inline; filename=\"" + filename + "\""
	}

	req, err := r.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(r.bucket),
		Key:                        aws.String(storageKey),
		ResponseContentDisposition: aws.String(disposition),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}
	return req.URL, nil
}

func (r *R2Storage) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(r.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(r.bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects from R2: %w", err)
		}
		for _, obj := range page.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}
	}
	return keys, nil
}

// Multipart Upload Flow //
func (r *R2Storage) InitiateMultipartUpload(ctx context.Context, storageKey string, contentType string) (string, error) {
	resp, err := r.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(storageKey),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to initiate multipart upload: %w", err)
	}
	return aws.ToString(resp.UploadId), nil
}

func (r *R2Storage) GeneratePresignedUploadPartURL(ctx context.Context, storageKey string, uploadID string, partNumber int32, expiry time.Duration) (string, error) {
	req, err := r.presignClient.PresignUploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(r.bucket),
		Key:        aws.String(storageKey),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(partNumber),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to presign upload part URL: %w", err)
	}
	return req.URL, nil
}

func (r *R2Storage) CompleteMultipartUpload(ctx context.Context, storageKey string, uploadID string, parts []model.UploadPart) (string, error) {
	var s3Parts []s3types.CompletedPart
	for _, p := range parts {
		s3Parts = append(s3Parts, s3types.CompletedPart{
			PartNumber: aws.Int32(p.PartNumber),
			ETag:       aws.String(p.ETag),
		})
	}

	resp, err := r.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(r.bucket),
		Key:      aws.String(storageKey),
		UploadId: aws.String(uploadID),
		MultipartUpload: &s3types.CompletedMultipartUpload{
			Parts: s3Parts,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to complete multipart upload: %w", err)
	}
	return aws.ToString(resp.Location), nil
}

func (r *R2Storage) AbortMultipartUpload(ctx context.Context, storageKey string, uploadID string) error {
	_, err := r.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(r.bucket),
		Key:      aws.String(storageKey),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}
	return nil
}