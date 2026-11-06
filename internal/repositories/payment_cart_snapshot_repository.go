package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PaymentCartSnapshotRepository persists and retrieves the cart snapshots used to rebuild
// a confirmed order from a webhook when the browser never reached /verify. It uses direct
// GORM access (no BaseFilterableRepository) because the access pattern is a single
// unique-key lookup plus an insert and a one-field status update.
type PaymentCartSnapshotRepository struct {
	pgManager *db.PostgresManager
	logger    *zap.Logger
}

// NewPaymentCartSnapshotRepository creates the repository.
func NewPaymentCartSnapshotRepository(pgManager *db.PostgresManager, logger *zap.Logger) *PaymentCartSnapshotRepository {
	return &PaymentCartSnapshotRepository{pgManager: pgManager, logger: logger}
}

// Create inserts a snapshot. A duplicate razorpay_order_id is treated as success (the
// snapshot already exists), since create-order is the only writer and must not fail
// checkout over a retried intent.
func (r *PaymentCartSnapshotRepository) Create(ctx context.Context, snap *models.PaymentCartSnapshot) error {
	if r.pgManager == nil {
		return fmt.Errorf("postgres manager unavailable for snapshot creation")
	}
	gdb, err := r.pgManager.GetDB(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to get DB for snapshot creation: %w", err)
	}
	if err := gdb.WithContext(ctx).Create(snap).Error; err != nil {
		if isDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("failed to create payment cart snapshot: %w", err)
	}
	return nil
}

// GetByRazorpayOrderID returns the snapshot for a Razorpay order id, or (nil, nil) when
// none exists (so the caller can distinguish "no snapshot" from a real error).
func (r *PaymentCartSnapshotRepository) GetByRazorpayOrderID(ctx context.Context, rzpOrderID string) (*models.PaymentCartSnapshot, error) {
	if r.pgManager == nil {
		return nil, fmt.Errorf("postgres manager unavailable for snapshot lookup")
	}
	gdb, err := r.pgManager.GetDB(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get DB for snapshot lookup: %w", err)
	}
	var snap models.PaymentCartSnapshot
	err = gdb.WithContext(ctx).Where("razorpay_order_id = ?", rzpOrderID).First(&snap).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to look up payment cart snapshot: %w", err)
	}
	return &snap, nil
}

// MarkConsumed flags the snapshot as turned into an order (idempotency/audit). Non-critical:
// a failure here does not affect the order that was already committed.
func (r *PaymentCartSnapshotRepository) MarkConsumed(ctx context.Context, id, paymentID string) error {
	if r.pgManager == nil {
		return fmt.Errorf("postgres manager unavailable for snapshot update")
	}
	gdb, err := r.pgManager.GetDB(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to get DB for snapshot update: %w", err)
	}
	now := time.Now()
	return gdb.WithContext(ctx).
		Model(&models.PaymentCartSnapshot{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":              models.SnapshotStatusConsumed,
			"consumed_at":         &now,
			"razorpay_payment_id": paymentID,
		}).Error
}

// DeleteStalePending removes PENDING (never-consumed) snapshots created before cutoff, so
// abandoned checkouts don't accumulate. Consumed snapshots are retained as an audit trail.
// Returns the number of rows removed.
func (r *PaymentCartSnapshotRepository) DeleteStalePending(ctx context.Context, cutoff time.Time) (int64, error) {
	if r.pgManager == nil {
		return 0, fmt.Errorf("postgres manager unavailable for snapshot cleanup")
	}
	gdb, err := r.pgManager.GetDB(ctx, false)
	if err != nil {
		return 0, fmt.Errorf("failed to get DB for snapshot cleanup: %w", err)
	}
	res := gdb.WithContext(ctx).
		Where("status = ? AND created_at < ?", models.SnapshotStatusPending, cutoff).
		Delete(&models.PaymentCartSnapshot{})
	return res.RowsAffected, res.Error
}
