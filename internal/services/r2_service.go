package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

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

// GetPresignedURL returns a short-lived presigned GET URL for an object. Use this
// instead of GetPublicURL for access-controlled files (e.g. invoices) so the object
// does not need to be publicly readable. For the gating to be meaningful the bucket's
// public access must be disabled; otherwise the object is still reachable via its
// public URL regardless of presigning.
func (s *R2Service) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if !s.configured || s.client == nil {
		return "", fmt.Errorf("R2 service is not configured")
	}
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}
	presigner := s3.NewPresignClient(s.client)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to presign object %s: %w", key, err)
	}
	return req.URL, nil
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

// UploadRecipeImage uploads a recipe image to R2 in the review_recipe directory and returns the R2 key
func (s *R2Service) UploadRecipeImage(ctx context.Context, userID string, file multipart.File, header *multipart.FileHeader) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("R2 service not configured")
	}

	// Generate R2 key for recipe image in review_recipe directory
	fileExt := filepath.Ext(header.Filename)
	timestamp := time.Now().Unix()
	r2Key := fmt.Sprintf("review_recipe/%s/recipe_%d%s", userID, timestamp, fileExt)

	s.logger.Info("Uploading recipe image to R2",
		zap.String("bucket", s.bucketName),
		zap.String("key", r2Key),
		zap.String("user_id", userID),
		zap.String("content_type", header.Header.Get("Content-Type")),
		zap.Int64("size_bytes", header.Size))

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		s.logger.Error("Failed to read recipe image file",
			zap.String("key", r2Key),
			zap.Error(err))
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Upload to R2
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(r2Key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(header.Header.Get("Content-Type")),
	})
	if err != nil {
		s.logger.Error("Failed to upload recipe image to R2",
			zap.String("bucket", s.bucketName),
			zap.String("key", r2Key),
			zap.Error(err))
		return "", fmt.Errorf("failed to upload recipe image to R2: %w", err)
	}

	s.logger.Info("Recipe image uploaded to R2 successfully",
		zap.String("bucket", s.bucketName),
		zap.String("key", r2Key),
		zap.String("url", s.GetPublicURL(r2Key)))

	return r2Key, nil
}

// GetRecipeImageURL returns the public URL for a recipe image in R2
func (s *R2Service) GetRecipeImageURL(ctx context.Context, r2Key string, expiration time.Duration) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("R2 service not configured")
	}

	// Return public URL (R2 public domain URLs don't expire, but we keep the signature for consistency)
	return s.GetPublicURL(r2Key), nil
}

// DeleteRecipeImage deletes a recipe image from R2
func (s *R2Service) DeleteRecipeImage(ctx context.Context, r2Key string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("R2 service not configured")
	}

	s.logger.Info("Deleting recipe image from R2",
		zap.String("bucket", s.bucketName),
		zap.String("key", r2Key))

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(r2Key),
	})
	if err != nil {
		s.logger.Error("Failed to delete recipe image from R2",
			zap.String("bucket", s.bucketName),
			zap.String("key", r2Key),
			zap.Error(err))
		return fmt.Errorf("failed to delete recipe image from R2: %w", err)
	}

	s.logger.Info("Recipe image deleted from R2 successfully",
		zap.String("bucket", s.bucketName),
		zap.String("key", r2Key))

	return nil
}

// UploadProductImage uploads a product image to R2 and returns the R2 key
func (s *R2Service) UploadProductImage(ctx context.Context, productID string, file multipart.File, header *multipart.FileHeader) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("R2 service not configured")
	}

	// Generate R2 key for product image
	fileExt := filepath.Ext(header.Filename)
	timestamp := time.Now().Unix()
	r2Key := fmt.Sprintf("images/product-images/%s/product_%d%s", productID, timestamp, fileExt)

	s.logger.Info("Uploading product image to R2",
		zap.String("bucket", s.bucketName),
		zap.String("key", r2Key),
		zap.String("product_id", productID),
		zap.String("content_type", header.Header.Get("Content-Type")),
		zap.Int64("size_bytes", header.Size))

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		s.logger.Error("Failed to read product image file",
			zap.String("key", r2Key),
			zap.Error(err))
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Upload to R2
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(r2Key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(header.Header.Get("Content-Type")),
	})
	if err != nil {
		s.logger.Error("Failed to upload product image to R2",
			zap.String("bucket", s.bucketName),
			zap.String("key", r2Key),
			zap.Error(err))
		return "", fmt.Errorf("failed to upload product image to R2: %w", err)
	}

	s.logger.Info("Product image uploaded to R2 successfully",
		zap.String("bucket", s.bucketName),
		zap.String("key", r2Key),
		zap.String("url", s.GetPublicURL(r2Key)))

	return r2Key, nil
}

// DeleteProductImage deletes a product image from R2
func (s *R2Service) DeleteProductImage(ctx context.Context, r2Key string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("R2 service not configured")
	}

	// Handle case where r2Key might be a full URL
	if strings.HasPrefix(r2Key, "http://") || strings.HasPrefix(r2Key, "https://") {
		// Extract the key from the URL
		// URL format: https://pub-xxx.r2.dev/images/product-images/...
		// or https://bucket.r2.cloudflarestorage.com/images/product-images/...
		if s.publicDomain != "" {
			domain := strings.TrimPrefix(s.publicDomain, "https://")
			domain = strings.TrimPrefix(domain, "http://")
			domain = strings.TrimSuffix(domain, "/")
			prefix := fmt.Sprintf("https://%s/", domain)
			if strings.HasPrefix(r2Key, prefix) {
				r2Key = strings.TrimPrefix(r2Key, prefix)
			}
		}
		// Also try the fallback URL format
		fallbackPrefix := fmt.Sprintf("https://%s.r2.cloudflarestorage.com/", s.bucketName)
		if strings.HasPrefix(r2Key, fallbackPrefix) {
			r2Key = strings.TrimPrefix(r2Key, fallbackPrefix)
		}
	}

	s.logger.Info("Deleting product image from R2",
		zap.String("bucket", s.bucketName),
		zap.String("key", r2Key))

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(r2Key),
	})
	if err != nil {
		s.logger.Error("Failed to delete product image from R2",
			zap.String("bucket", s.bucketName),
			zap.String("key", r2Key),
			zap.Error(err))
		return fmt.Errorf("failed to delete product image from R2: %w", err)
	}

	s.logger.Info("Product image deleted from R2 successfully",
		zap.String("bucket", s.bucketName),
		zap.String("key", r2Key))

	return nil
}

// GetProductImageURL returns the public URL for a product image in R2
func (s *R2Service) GetProductImageURL(r2Key string) string {
	return s.GetPublicURL(r2Key)
}

// UploadProfileImage uploads a profile image to R2 and returns the R2 key
func (s *R2Service) UploadProfileImage(ctx context.Context, userID string, file multipart.File, header *multipart.FileHeader) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("R2 service not configured")
	}

	// Generate R2 key for profile image
	fileExt := filepath.Ext(header.Filename)
	r2Key := fmt.Sprintf("images/profile-images/%s/profile_image%s", userID, fileExt)

	s.logger.Info("Uploading profile image to R2",
		zap.String("bucket", s.bucketName),
		zap.String("key", r2Key),
		zap.String("user_id", userID),
		zap.String("content_type", header.Header.Get("Content-Type")),
		zap.Int64("size_bytes", header.Size))

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		s.logger.Error("Failed to read profile image file",
			zap.String("key", r2Key),
			zap.Error(err))
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Upload to R2
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(r2Key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(header.Header.Get("Content-Type")),
	})
	if err != nil {
		s.logger.Error("Failed to upload profile image to R2",
			zap.String("bucket", s.bucketName),
			zap.String("key", r2Key),
			zap.Error(err))
		return "", fmt.Errorf("failed to upload profile image to R2: %w", err)
	}

	s.logger.Info("Profile image uploaded to R2 successfully",
		zap.String("bucket", s.bucketName),
		zap.String("key", r2Key),
		zap.String("url", s.GetPublicURL(r2Key)))

	return r2Key, nil
}

// GetProfileImageURL returns the public URL for a profile image in R2
// Note: R2 public URLs don't expire, so expiration parameter is ignored but kept for API compatibility
func (s *R2Service) GetProfileImageURL(ctx context.Context, r2Key string, expiration time.Duration) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("R2 service not configured")
	}

	// Return public URL (R2 public domain URLs don't expire)
	return s.GetPublicURL(r2Key), nil
}

// DeleteProfileImage deletes a profile image from R2
func (s *R2Service) DeleteProfileImage(ctx context.Context, r2Key string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("R2 service not configured")
	}

	// Handle case where r2Key might be a full URL
	if strings.HasPrefix(r2Key, "http://") || strings.HasPrefix(r2Key, "https://") {
		if s.publicDomain != "" {
			domain := strings.TrimPrefix(s.publicDomain, "https://")
			domain = strings.TrimPrefix(domain, "http://")
			domain = strings.TrimSuffix(domain, "/")
			prefix := fmt.Sprintf("https://%s/", domain)
			if strings.HasPrefix(r2Key, prefix) {
				r2Key = strings.TrimPrefix(r2Key, prefix)
			}
		}
		fallbackPrefix := fmt.Sprintf("https://%s.r2.cloudflarestorage.com/", s.bucketName)
		if strings.HasPrefix(r2Key, fallbackPrefix) {
			r2Key = strings.TrimPrefix(r2Key, fallbackPrefix)
		}
	}

	s.logger.Info("Deleting profile image from R2",
		zap.String("bucket", s.bucketName),
		zap.String("key", r2Key))

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(r2Key),
	})
	if err != nil {
		s.logger.Error("Failed to delete profile image from R2",
			zap.String("bucket", s.bucketName),
			zap.String("key", r2Key),
			zap.Error(err))
		return fmt.Errorf("failed to delete profile image from R2: %w", err)
	}

	s.logger.Info("Profile image deleted from R2 successfully",
		zap.String("bucket", s.bucketName),
		zap.String("key", r2Key))

	return nil
}
