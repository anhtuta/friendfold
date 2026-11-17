package blob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3 is an S3-compatible store (MinIO, R2, AWS).
type S3 struct {
	client *minio.Client
	bucket string
}

// NewS3 connects and ensures the bucket exists.
func NewS3(ctx context.Context, cfg S3Config) (*S3, error) {
	endpoint, secure, err := parseEndpoint(cfg.Endpoint, cfg.UseSSL)
	if err != nil {
		return nil, err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if bucket == "" {
		return nil, fmt.Errorf("blob: MEDIA_S3_BUCKET is required")
	}
	access := strings.TrimSpace(cfg.AccessKey)
	secret := strings.TrimSpace(cfg.SecretKey)
	if access == "" || secret == "" {
		return nil, fmt.Errorf("blob: MEDIA_S3_ACCESS_KEY and MEDIA_S3_SECRET_KEY are required")
	}
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "us-east-1"
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(access, secret, ""),
		Secure:       secure,
		Region:       region,
		BucketLookup: lookup(cfg.PathStyle),
	})
	if err != nil {
		return nil, fmt.Errorf("blob: s3 client: %w", err)
	}
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("blob: bucket exists: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region}); err != nil {
			return nil, fmt.Errorf("blob: make bucket: %w", err)
		}
	}
	return &S3{client: client, bucket: bucket}, nil
}

func lookup(pathStyle bool) minio.BucketLookupType {
	if pathStyle {
		return minio.BucketLookupPath
	}
	return minio.BucketLookupDNS
}

// Put uploads key. Size is unknown; minio.PutObject reads until EOF.
func (s *S3) Put(ctx context.Context, key, contentType string, r io.Reader) error {
	if err := ValidateKey(key); err != nil {
		return err
	}
	if contentType == "" {
		contentType = jpegType
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, r, -1, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("blob: s3 put: %w", err)
	}
	return nil
}

// Get streams key from the bucket.
func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	if err := ValidateKey(key); err != nil {
		return nil, "", err
	}
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("blob: s3 get: %w", err)
	}
	info, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		if isS3NotFound(err) {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("blob: s3 stat: %w", err)
	}
	ct := info.ContentType
	if ct == "" {
		ct = jpegType
	}
	return obj, ct, nil
}

// Delete removes key.
func (s *S3) Delete(ctx context.Context, key string) error {
	if err := ValidateKey(key); err != nil {
		return err
	}
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isS3NotFound(err) {
			return ErrNotFound
		}
		return fmt.Errorf("blob: s3 stat: %w", err)
	}
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("blob: s3 delete: %w", err)
	}
	return nil
}

func isS3NotFound(err error) bool {
	var e minio.ErrorResponse
	if errors.As(err, &e) {
		return e.Code == "NoSuchKey" || e.StatusCode == 404
	}
	return minio.ToErrorResponse(err).Code == "NoSuchKey"
}
