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
	if paymentID != "" {
		order.PaymentMethod = fmt.Sprintf("razorpay:%s", paymentID)
	}

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

// Update persists an order
func (r *OrderRepository) Update(ctx context.Context, order *models.Order) error {
	return r.dbManager.Update(ctx, order)
}
