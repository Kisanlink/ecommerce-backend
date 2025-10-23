package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/db"
)

// S3Service provides S3 operations for file uploads and management
type S3Service struct {
	s3Manager *db.S3Manager
}

// NewS3Service creates a new S3 service
func NewS3Service(s3Manager *db.S3Manager) *S3Service {
	return &S3Service{
		s3Manager: s3Manager,
	}
}

// UploadFile uploads a file to S3 and returns the file URL
func (s *S3Service) UploadFile(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	if s.s3Manager == nil {
		return "", fmt.Errorf("S3 manager not initialized")
	}

	reader := bytes.NewReader(data)
	err := s.s3Manager.UploadFile(ctx, key, reader, contentType, nil)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Generate presigned URL for access
	url, err := s.s3Manager.GetPresignedURL(ctx, key, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to get file URL: %w", err)
	}

	return url, nil
}

// GeneratePresignedURL generates a presigned URL for file access
func (s *S3Service) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	if s.s3Manager == nil {
		return "", fmt.Errorf("S3 manager not initialized")
	}

	url, err := s.s3Manager.GetPresignedURL(ctx, key, expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url, nil
}

// DeleteFile deletes a file from S3 using the key
func (s *S3Service) DeleteFile(ctx context.Context, key string) error {
	if s.s3Manager == nil {
		return fmt.Errorf("S3 manager not initialized")
	}

	err := s.s3Manager.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

// GetFileURL returns a presigned URL for a file in S3
func (s *S3Service) GetFileURL(ctx context.Context, key string) (string, error) {
	return s.GeneratePresignedURL(ctx, key, 24*time.Hour)
}

// UploadProfileImage uploads a profile image to S3 and returns the S3 key
func (s *S3Service) UploadProfileImage(ctx context.Context, userID string, file multipart.File, header *multipart.FileHeader) (string, error) {
	// Generate S3 key for profile image
	fileExt := filepath.Ext(header.Filename)
	s3Key := fmt.Sprintf("images/profile-images/%s/profile_image%s", userID, fileExt)

	// Check if S3 manager is available
	if s.s3Manager == nil {
		return "", fmt.Errorf("S3 service not available - please configure S3 settings")
	}

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Upload to S3
	reader := bytes.NewReader(data)
	err = s.s3Manager.UploadFile(ctx, s3Key, reader, header.Header.Get("Content-Type"), nil)
	if err != nil {
		return "", fmt.Errorf("failed to upload profile image: %w", err)
	}

	return s3Key, nil
}

// GetProfileImageURL generates a presigned URL for profile image access
func (s *S3Service) GetProfileImageURL(ctx context.Context, s3Key string, expiration time.Duration) (string, error) {
	// Check if S3 manager is available
	if s.s3Manager == nil {
		return "", fmt.Errorf("S3 service not available - please configure S3 settings")
	}

	return s.GeneratePresignedURL(ctx, s3Key, expiration)
}

// UploadProductImage uploads a product image to S3 and returns the S3 key
func (s *S3Service) UploadProductImage(ctx context.Context, productID string, file multipart.File, header *multipart.FileHeader) (string, error) {
	// Generate S3 key for product image
	fileExt := filepath.Ext(header.Filename)
	timestamp := time.Now().Unix()
	s3Key := fmt.Sprintf("images/product-images/%s/product_%d%s", productID, timestamp, fileExt)

	// Check if S3 manager is available
	if s.s3Manager == nil {
		return "", fmt.Errorf("S3 service not available - please configure S3 settings")
	}

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Upload to S3
	reader := bytes.NewReader(data)
	err = s.s3Manager.UploadFile(ctx, s3Key, reader, header.Header.Get("Content-Type"), nil)
	if err != nil {
		return "", fmt.Errorf("failed to upload product image: %w", err)
	}

	return s3Key, nil
}

// GetProductImageURL generates a presigned URL for product image access
func (s *S3Service) GetProductImageURL(ctx context.Context, s3Key string, expiration time.Duration) (string, error) {
	// Check if S3 manager is available
	if s.s3Manager == nil {
		return "", fmt.Errorf("S3 service not available - please configure S3 settings")
	}

	return s.GeneratePresignedURL(ctx, s3Key, expiration)
}

// DeleteProductImage deletes a product image from S3
func (s *S3Service) DeleteProductImage(ctx context.Context, s3Key string) error {
	// Check if S3 manager is available
	if s.s3Manager == nil {
		return fmt.Errorf("S3 service not available - please configure S3 settings")
	}

	return s.DeleteFile(ctx, s3Key)
}

// UploadRecipeImage uploads a recipe image to S3 and returns the S3 key
func (s *S3Service) UploadRecipeImage(ctx context.Context, userID string, file multipart.File, header *multipart.FileHeader) (string, error) {
	// Generate S3 key for recipe image
	fileExt := filepath.Ext(header.Filename)
	timestamp := time.Now().Unix()
	s3Key := fmt.Sprintf("images/recipe-images/%s/recipe_%d%s", userID, timestamp, fileExt)

	// Check if S3 manager is available
	if s.s3Manager == nil {
		return "", fmt.Errorf("S3 service not available - please configure S3 settings")
	}

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Upload to S3
	reader := bytes.NewReader(data)
	err = s.s3Manager.UploadFile(ctx, s3Key, reader, header.Header.Get("Content-Type"), nil)
	if err != nil {
		return "", fmt.Errorf("failed to upload recipe image: %w", err)
	}

	return s3Key, nil
}

// GetRecipeImageURL generates a presigned URL for recipe image access
func (s *S3Service) GetRecipeImageURL(ctx context.Context, s3Key string, expiration time.Duration) (string, error) {
	// Check if S3 manager is available
	if s.s3Manager == nil {
		return "", fmt.Errorf("S3 service not available - please configure S3 settings")
	}

	return s.GeneratePresignedURL(ctx, s3Key, expiration)
}

// DeleteRecipeImage deletes a recipe image from S3
func (s *S3Service) DeleteRecipeImage(ctx context.Context, s3Key string) error {
	// Check if S3 manager is available
	if s.s3Manager == nil {
		return fmt.Errorf("S3 service not available - please configure S3 settings")
	}

	return s.DeleteFile(ctx, s3Key)
}
