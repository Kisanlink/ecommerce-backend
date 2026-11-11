package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrOTPCooldown   = errors.New("please wait before requesting another OTP")
	ErrOTPNotFound   = errors.New("verification session not found or expired")
	ErrOTPExpired    = errors.New("verification code has expired")
	ErrOTPMaxRetries = errors.New("maximum verification attempts exceeded, please request a new code")
	ErrOTPInvalid    = errors.New("invalid verification code")
	ErrOTPAlreadyUsed = errors.New("verification code has already been used")
)

// OTPService handles generating, storing, rate-limiting, and verifying OTPs
type OTPService struct {
	pgManager *db.PostgresManager
	logger    *zap.Logger
}

// NewOTPService creates a new OTPService
func NewOTPService(pgManager *db.PostgresManager, logger *zap.Logger) *OTPService {
	return &OTPService{
		pgManager: pgManager,
		logger:    logger,
	}
}

func (s *OTPService) getDB(ctx context.Context) (*gorm.DB, error) {
	return s.pgManager.GetDB(ctx, false)
}

// GenerateAndSaveOTP generates a 6-digit OTP, hashes it, stores it, and returns the plain code and sessionInfo
func (s *OTPService) GenerateAndSaveOTP(ctx context.Context, phoneNumber string) (sessionInfo string, otpCode string, err error) {
	db, err := s.getDB(ctx)
	if err != nil {
		s.logger.Error("Failed to get DB in OTP service", zap.Error(err))
		return "", "", fmt.Errorf("database unavailable: %w", err)
	}

	// Rate limiting: 60-second cooldown per phone number
	cooldownThreshold := time.Now().Add(-60 * time.Second)
	var recentCount int64
	err = db.WithContext(ctx).Model(&models.PhoneVerification{}).
		Where("phone_number = ? AND created_at > ?", phoneNumber, cooldownThreshold).
		Count(&recentCount).Error
	if err != nil {
		s.logger.Error("Failed to check OTP rate limit", zap.String("phone", phoneNumber), zap.Error(err))
		return "", "", fmt.Errorf("failed to check rate limit: %w", err)
	}
	if recentCount > 0 {
		return "", "", ErrOTPCooldown
	}

	// Generate 6-digit cryptographically secure random number (100000 - 999999)
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		s.logger.Error("Failed to generate random OTP", zap.Error(err))
		return "", "", fmt.Errorf("failed to generate OTP: %w", err)
	}
	code := fmt.Sprintf("%06d", n.Int64()+100000)

	// Hash the OTP with bcrypt before storing
	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash OTP", zap.Error(err))
		return "", "", fmt.Errorf("failed to secure OTP: %w", err)
	}

	session := uuid.New().String()
	expiresAt := time.Now().Add(5 * time.Minute)

	verification := models.NewPhoneVerification(phoneNumber, string(codeHash), session, expiresAt)
	if err := db.WithContext(ctx).Create(verification).Error; err != nil {
		s.logger.Error("Failed to save phone verification record", zap.String("phone", phoneNumber), zap.Error(err))
		return "", "", fmt.Errorf("failed to save verification record: %w", err)
	}

	s.logger.Info("Generated OTP verification record",
		zap.String("phone", phoneNumber),
		zap.String("session_info", session),
		zap.Time("expires_at", expiresAt),
	)

	return session, code, nil
}

// DeleteOTP removes an OTP record by sessionInfo — used to roll back when WhatsApp delivery fails
func (s *OTPService) DeleteOTP(ctx context.Context, sessionInfo string) {
	db, err := s.getDB(ctx)
	if err != nil {
		s.logger.Error("Failed to get DB when deleting OTP", zap.String("session", sessionInfo), zap.Error(err))
		return
	}
	if err := db.WithContext(ctx).Where("session_info = ?", sessionInfo).Delete(&models.PhoneVerification{}).Error; err != nil {
		s.logger.Error("Failed to delete OTP record after send failure", zap.String("session", sessionInfo), zap.Error(err))
	}
}

// VerifyOTP verifies a submitted OTP code against the stored hash for given sessionInfo and phoneNumber
func (s *OTPService) VerifyOTP(ctx context.Context, phoneNumber, code, sessionInfo string) error {
	db, err := s.getDB(ctx)
	if err != nil {
		s.logger.Error("Failed to get DB in OTP service", zap.Error(err))
		return fmt.Errorf("database unavailable: %w", err)
	}

	var record models.PhoneVerification
	err = db.WithContext(ctx).Where("session_info = ?", sessionInfo).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOTPNotFound
		}
		s.logger.Error("Database error during OTP lookup", zap.String("session", sessionInfo), zap.Error(err))
		return fmt.Errorf("database error: %w", err)
	}

	if record.IsUsed {
		return ErrOTPAlreadyUsed
	}

	if record.PhoneNumber != phoneNumber {
		s.logger.Warn("OTP phone number mismatch",
			zap.String("record_phone", record.PhoneNumber),
			zap.String("submitted_phone", phoneNumber),
		)
		return ErrOTPNotFound
	}

	if time.Now().After(record.ExpiresAt) {
		return ErrOTPExpired
	}

	// Atomically increment attempts only if under the limit — prevents race condition
	result := db.WithContext(ctx).Model(&record).
		Where("attempts < ?", 5).
		Update("attempts", gorm.Expr("attempts + 1"))
	if result.Error != nil {
		s.logger.Error("Failed to increment OTP attempt counter", zap.String("session", sessionInfo), zap.Error(result.Error))
		return fmt.Errorf("database error: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrOTPMaxRetries
	}

	// Compare bcrypt hash
	if err := bcrypt.CompareHashAndPassword([]byte(record.CodeHash), []byte(code)); err != nil {
		s.logger.Warn("Invalid OTP code attempt",
			zap.String("phone", phoneNumber),
			zap.Int("attempt", record.Attempts+1),
		)
		return ErrOTPInvalid
	}

	// Mark as used
	if err := db.WithContext(ctx).Model(&record).Update("is_used", true).Error; err != nil {
		s.logger.Error("Failed to mark OTP as used", zap.String("session", sessionInfo), zap.Error(err))
	}

	s.logger.Info("OTP verified successfully", zap.String("phone", phoneNumber), zap.String("session", sessionInfo))
	return nil
}
