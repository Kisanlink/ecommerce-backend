package services

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
)

// R2Service provides Cloudflare R2 operations for file uploads and management
// R2 is S3-compatible, so we use the AWS SDK with a custom endpoint
type R2Service struct {
	client       *s3.Client
	bucketName   string
	publicDomain string
	logger       *zap.Logger
	configured   bool
}

// NewR2Service creates a new Cloudflare R2 service
func NewR2Service(logger *zap.Logger) (*R2Service, error) {
	cfg := config.GetR2Config()

	if !cfg.IsR2Configured() {
		return nil, fmt.Errorf("R2 configuration incomplete: missing required environment variables (R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY)")
	}

	// Create custom resolver for R2 endpoint
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: cfg.Endpoint,
		}, nil
	})

	// Load AWS config with custom credentials and endpoint
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithEndpointResolverWithOptions(r2Resolver),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
		awsconfig.WithRegion("auto"), // R2 uses "auto" as region
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for R2: %w", err)
	}

	// Create S3 client for R2
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // R2 requires path-style addressing
	})

	logger.Info("R2 service initialized",
		zap.String("bucket", cfg.BucketName),
		zap.String("endpoint", cfg.Endpoint),
		zap.String("public_domain", cfg.PublicDomain))

	return &R2Service{
		client:       client,
		bucketName:   cfg.BucketName,
		publicDomain: cfg.PublicDomain,
		logger:       logger,
		configured:   true,
	}, nil
}

// IsConfigured returns true if the R2 service is properly configured
func (s *R2Service) IsConfigured() bool {
	return s != nil && s.configured && s.client != nil
}

// GetBucketName returns the configured bucket name
func (s *R2Service) GetBucketName() string {
	if s == nil {
		return ""
	}
	return s.bucketName
}

// UploadFile uploads a file to R2 and returns the public URL
func (s *R2Service) UploadFile(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("R2 service not configured")
	}

	s.logger.Info("Uploading file to R2",
		zap.String("bucket", s.bucketName),
		zap.String("key", key),
		zap.String("content_type", contentType),
		zap.Int("size_bytes", len(data)))

	// Upload to R2
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		s.logger.Error("Failed to upload file to R2",
			zap.String("bucket", s.bucketName),
			zap.String("key", key),
			zap.Error(err))
		return "", fmt.Errorf("failed to upload file to R2: %w", err)
	}

	// Generate public URL
	url := s.GetPublicURL(key)

	s.logger.Info("File uploaded to R2 successfully",
		zap.String("bucket", s.bucketName),
		zap.String("key", key),
		zap.String("url", url))

	return url, nil
}

// GetPublicURL returns the public URL for a file in R2
func (s *R2Service) GetPublicURL(key string) string {
	if s.publicDomain != "" {
		domain := s.publicDomain
		// Strip protocol if already present to avoid double https://
		domain = strings.TrimPrefix(domain, "https://")
		domain = strings.TrimPrefix(domain, "http://")
		// Remove trailing slash if present
		domain = strings.TrimSuffix(domain, "/")
		return fmt.Sprintf("https://%s/%s", domain, key)
	}
	// Fallback: construct URL without public domain (requires public bucket or signed URLs)
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s", s.bucketName, key)
}

// DeleteFile deletes a file from R2
func (s *R2Service) DeleteFile(ctx context.Context, key string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("R2 service not configured")
	}

	s.logger.Info("Deleting file from R2",
		zap.String("bucket", s.bucketName),
		zap.String("key", key))

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		s.logger.Error("Failed to delete file from R2",
			zap.String("bucket", s.bucketName),
			zap.String("key", key),
			zap.Error(err))
		return fmt.Errorf("failed to delete file from R2: %w", err)
	}

	s.logger.Info("File deleted from R2 successfully",
		zap.String("bucket", s.bucketName),
		zap.String("key", key))

	return nil
}

// FileExists checks if a file exists in R2
func (s *R2Service) FileExists(ctx context.Context, key string) (bool, error) {
	if !s.IsConfigured() {
		return false, fmt.Errorf("R2 service not configured")
	}

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error
		return false, nil
	}

	return true, nil
}
