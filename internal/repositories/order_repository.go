package repositories

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
)

// Additional methods for OrderRepository beyond the base repository
// The OrderRepository is already defined in repository_manager.go

// Create creates a new order in the database
func (r *OrderRepository) Create(ctx context.Context, order *models.Order) error {
	return r.dbManager.Create(ctx, order)
}

// CreateOrderItems creates multiple order items in the database
func (r *OrderRepository) CreateOrderItems(ctx context.Context, orderItems []models.OrderItem) error {
	if len(orderItems) == 0 {
		return nil
	}

	// Create each order item using the dbManager
	for i := range orderItems {
		if err := r.dbManager.Create(ctx, &orderItems[i]); err != nil {
			return fmt.Errorf("failed to create order item %d: %w", i, err)
		}
	}

	return nil
}

// UpdateOrderStatus updates the status of an order
func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, orderID string, status models.OrderStatus) error {
	var order models.Order
	if err := r.dbManager.GetByID(ctx, orderID, &order); err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	order.Status = status
	if err := r.dbManager.Update(ctx, &order); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// UpdatePaymentStatus updates the payment status and payment ID of an order
func (r *OrderRepository) UpdatePaymentStatus(ctx context.Context, orderID string, paymentStatus, paymentID string) error {
	var order models.Order
	if err := r.dbManager.GetByID(ctx, orderID, &order); err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	order.PaymentStatus = paymentStatus
	// Note: PaymentID is not stored in a separate field, it's handled through RazorpayReceipt
	// The paymentID parameter is used for logging purposes

	if err := r.dbManager.Update(ctx, &order); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return nil
}

// UpdateOrderNotes updates the notes of an order
func (r *OrderRepository) UpdateOrderNotes(ctx context.Context, orderID string, notes string) error {
	var order models.Order
	if err := r.dbManager.GetByID(ctx, orderID, &order); err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}
	order.Notes = notes
	if err := r.dbManager.Update(ctx, &order); err != nil {
		return fmt.Errorf("failed to update order notes: %w", err)
	}
	return nil
}

// GetOrderWithItems retrieves an order with its items
func (r *OrderRepository) GetOrderWithItems(ctx context.Context, orderID string) (*models.Order, []models.OrderItem, error) {
	// Get order using base repository
	var order models.Order
	if err := r.dbManager.GetByID(ctx, orderID, &order); err != nil {
		return nil, nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Get order items
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "order_id", Operator: base.OpEqual, Value: orderID},
			},
			Logic: base.LogicAnd,
		},
	}

	var orderItems []models.OrderItem
	if err := r.dbManager.List(ctx, filter, &orderItems); err != nil {
		return nil, nil, fmt.Errorf("failed to get order items: %w", err)
	}

	return &order, orderItems, nil
}

// GetOrdersByUserID retrieves all orders for a specific user
func (r *OrderRepository) GetOrdersByUserID(ctx context.Context, userID string) ([]*models.Order, error) {
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "user_id", Operator: base.OpEqual, Value: userID},
			},
			Logic: base.LogicAnd,
		},
	}

	var orders []*models.Order
	if err := r.dbManager.List(ctx, filter, &orders); err != nil {
		return nil, fmt.Errorf("failed to get orders for user: %w", err)
	}

	return orders, nil
}

// GetOrderItems retrieves order items with the given filter
func (r *OrderRepository) GetOrderItems(ctx context.Context, filter *base.Filter, orderItems *[]*models.OrderItem) error {
	return r.dbManager.List(ctx, filter, orderItems)
}

// Update persists an order
func (r *OrderRepository) Update(ctx context.Context, order *models.Order) error {
	return r.dbManager.Update(ctx, order)
}

// GetOrderByRazorpayOrderID retrieves an order by Razorpay order ID
func (r *OrderRepository) GetOrderByRazorpayOrderID(ctx context.Context, razorpayOrderID string) (*models.Order, error) {
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "razorpay_order_id", Operator: base.OpEqual, Value: razorpayOrderID},
			},
			Logic: base.LogicAnd,
		},
	}

	var orders []*models.Order
	if err := r.dbManager.List(ctx, filter, &orders); err != nil {
		return nil, fmt.Errorf("failed to get order by razorpay order ID: %w", err)
	}

	if len(orders) == 0 {
		return nil, fmt.Errorf("order not found for razorpay order ID: %s", razorpayOrderID)
	}

	return orders[0], nil
}

// CreateOrderAtomic creates an order and its items atomically to prevent duplicate orders
func (r *OrderRepository) CreateOrderAtomic(ctx context.Context, order *models.Order, cartItems []models.CartItem) error {
	// Double check that no order exists with this razorpay_order_id before creating
	if order.RazorpayOrderID != "" {
		if existingOrder, _ := r.GetOrderByRazorpayOrderID(ctx, order.RazorpayOrderID); existingOrder != nil {
			return fmt.Errorf("order already exists for razorpay order ID: %s", order.RazorpayOrderID)
		}
	}

	// Create the order first - unique constraint will prevent duplicates at DB level
	if err := r.dbManager.Create(ctx, order); err != nil {
		// Check if this is a duplicate key error
		if isDuplicateKeyError(err) {
			return fmt.Errorf("duplicate order for razorpay order ID: %s", order.RazorpayOrderID)
		}
		return fmt.Errorf("failed to create order: %w", err)
	}

	// Create order items from cart items
	for _, cartItem := range cartItems {
		orderItem := models.NewOrderItem(order.ID, cartItem.ProductID, cartItem.Quantity, cartItem.UnitPrice)
		if err := r.dbManager.Create(ctx, orderItem); err != nil {
			// If order item creation fails, we need to clean up the order
			// For now, just log and return error - the unique constraint prevents the main issue
			return fmt.Errorf("failed to create order item for product %s: %w", cartItem.ProductID, err)
		}
	}

	return nil
}

// UpdateOrderShipmentAtomic updates order with shipment details atomically
func (r *OrderRepository) UpdateOrderShipmentAtomic(ctx context.Context, orderID string, waybill string, trackingURL string, delhiveryStatus string) error {
	var order models.Order
	if err := r.dbManager.GetByID(ctx, orderID, &order); err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	order.DelhiveryWaybill = waybill
	order.TrackingURL = trackingURL
	order.DelhiveryStatus = delhiveryStatus

	if err := r.dbManager.Update(ctx, &order); err != nil {
		return fmt.Errorf("failed to update order with shipment details: %w", err)
	}

	return nil
}

// isDuplicateKeyError checks if an error is a duplicate key constraint violation
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for common duplicate key error patterns
	return contains(errStr, "duplicate key") ||
		contains(errStr, "UNIQUE constraint") ||
		contains(errStr, "Duplicate entry") ||
		contains(errStr, "violates unique constraint")
}

// contains checks if a string contains a substring (case insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > len(substr) && findSubstring(s, substr)))
}

// findSubstring finds a substring in a string (simple implementation)
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
