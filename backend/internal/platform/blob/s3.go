package blob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3 is an S3-compatible store (local RustFS, R2, AWS).
type S3 struct {
	client *s3.Client
	bucket string
}

// NewS3 connects and ensures the bucket exists.
func NewS3(ctx context.Context, cfg S3Config) (*S3, error) {
	base, err := endpointURL(cfg.Endpoint, cfg.UseSSL)
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
	client := s3.NewFromConfig(aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(access, secret, ""),
	}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(base)
		o.UsePathStyle = cfg.PathStyle
	})
	if err := ensureBucket(ctx, client, bucket, region); err != nil {
		return nil, err
	}
	return &S3{client: client, bucket: bucket}, nil
}

func ensureBucket(ctx context.Context, client *s3.Client, bucket, region string) error {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err == nil {
		return nil
	}
	in := &s3.CreateBucketInput{Bucket: aws.String(bucket)}
	if region != "" && region != "us-east-1" {
		in.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}
	if _, err := client.CreateBucket(ctx, in); err != nil {
		if isS3AlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("blob: make bucket: %w", err)
	}
	return nil
}

func endpointURL(raw string, useSSL bool) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("blob: S3 endpoint is required")
	}
	if !strings.Contains(raw, "://") {
		scheme := "http"
		if useSSL {
			scheme = "https"
		}
		return scheme + "://" + raw, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("blob: parse S3 endpoint: %w", err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("blob: S3 endpoint host is empty")
	}
	scheme := u.Scheme
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("blob: S3 endpoint scheme %q is not http(s)", scheme)
	}
	return scheme + "://" + u.Host, nil
}

// Put uploads key. Unknown-size readers are accepted (SDK buffers if needed).
func (s *S3) Put(ctx context.Context, key, contentType string, r io.Reader) error {
	if err := ValidateKey(key); err != nil {
		return err
	}
	if contentType == "" {
		contentType = jpegType
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String(contentType),
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
	obj, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("blob: s3 get: %w", err)
	}
	ct := aws.ToString(obj.ContentType)
	if ct == "" {
		ct = jpegType
	}
	return obj.Body, ct, nil
}

// Delete removes key.
func (s *S3) Delete(ctx context.Context, key string) error {
	if err := ValidateKey(key); err != nil {
		return err
	}
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return ErrNotFound
		}
		return fmt.Errorf("blob: s3 stat: %w", err)
	}
	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("blob: s3 delete: %w", err)
	}
	return nil
}

func isS3NotFound(err error) bool {
	var nfe *types.NotFound
	var nsk *types.NoSuchKey
	var nsb *types.NoSuchBucket
	if errors.As(err, &nfe) || errors.As(err, &nsk) || errors.As(err, &nsb) {
		return true
	}
	var api smithy.APIError
	if errors.As(err, &api) {
		switch api.ErrorCode() {
		case "NotFound", "NoSuchKey", "NoSuchBucket", "404":
			return true
		}
	}
	return false
}

func isS3AlreadyExists(err error) bool {
	var owned *types.BucketAlreadyOwnedByYou
	var exists *types.BucketAlreadyExists
	if errors.As(err, &owned) || errors.As(err, &exists) {
		return true
	}
	var api smithy.APIError
	if errors.As(err, &api) {
		switch api.ErrorCode() {
		case "BucketAlreadyOwnedByYou", "BucketAlreadyExists":
			return true
		}
	}
	return false
}
