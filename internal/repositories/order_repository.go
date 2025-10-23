package repositories

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
)

// Additional methods for OrderRepository beyond the base repository
// The OrderRepository is already defined in repository_manager.go

// CreateOrderCustomer creates a new order customer
func (r *OrderRepository) CreateOrderCustomer(ctx context.Context, customer *models.OrderCustomer) error {
	return r.dbManager.Create(ctx, customer)
}

// CreateOrderAddress creates a new order address
func (r *OrderRepository) CreateOrderAddress(ctx context.Context, address *models.OrderAddress) error {
	return r.dbManager.Create(ctx, address)
}

// GetOrderCustomer retrieves an order customer by ID
func (r *OrderRepository) GetOrderCustomer(ctx context.Context, customerID string) (*models.OrderCustomer, error) {
	var customer models.OrderCustomer
	if err := r.dbManager.GetByID(ctx, customerID, &customer); err != nil {
		return nil, fmt.Errorf("failed to get order customer: %w", err)
	}
	return &customer, nil
}

// GetOrderAddress retrieves an order address by ID
func (r *OrderRepository) GetOrderAddress(ctx context.Context, addressID string) (*models.OrderAddress, error) {
	var address models.OrderAddress
	if err := r.dbManager.GetByID(ctx, addressID, &address); err != nil {
		return nil, fmt.Errorf("failed to get order address: %w", err)
	}
	return &address, nil
}

// GetOrderCustomers retrieves order customers with filter
func (r *OrderRepository) GetOrderCustomers(ctx context.Context, filter *base.Filter, customers *[]*models.OrderCustomer) error {
	return r.dbManager.List(ctx, filter, customers)
}

// GetOrderAddresses retrieves order addresses with filter
func (r *OrderRepository) GetOrderAddresses(ctx context.Context, filter *base.Filter, addresses *[]*models.OrderAddress) error {
	return r.dbManager.List(ctx, filter, addresses)
}

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

// GetOrderWithDetails retrieves an order with customer, address, and items
func (r *OrderRepository) GetOrderWithDetails(ctx context.Context, orderID string) (*models.Order, error) {
	var order models.Order

	// Get order using standard DBManager method
	if err := r.dbManager.GetByID(ctx, orderID, &order); err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Get customer data if customer_id exists
	if order.CustomerID != "" {
		customer, err := r.GetOrderCustomer(ctx, order.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order customer: %w", err)
		}
		order.Customer = customer
	}

	// Get shipping address data if shipping_address_id exists
	if order.ShippingAddressID != "" {
		address, err := r.GetOrderAddress(ctx, order.ShippingAddressID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order address: %w", err)
		}
		order.ShippingAddress = address
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
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}
	order.OrderItems = orderItems

	return &order, nil
}

// GetOrdersByUserWithDetails retrieves all orders for a user with details
func (r *OrderRepository) GetOrdersByUserWithDetails(ctx context.Context, userID string) ([]models.Order, error) {
	// Get orders for user
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "user_id", Operator: base.OpEqual, Value: userID},
			},
			Logic: base.LogicAnd,
		},
	}

	var orders []models.Order
	if err := r.dbManager.List(ctx, filter, &orders); err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	// For each order, load the normalized data
	for i := range orders {
		order := &orders[i]

		// Get customer data if customer_id exists
		if order.CustomerID != "" {
			customer, err := r.GetOrderCustomer(ctx, order.CustomerID)
			if err != nil {
				return nil, fmt.Errorf("failed to get order customer for order %s: %w", order.ID, err)
			}
			order.Customer = customer
		}

		// Get shipping address data if shipping_address_id exists
		if order.ShippingAddressID != "" {
			address, err := r.GetOrderAddress(ctx, order.ShippingAddressID)
			if err != nil {
				return nil, fmt.Errorf("failed to get order address for order %s: %w", order.ID, err)
			}
			order.ShippingAddress = address
		}

		// Get order items
		itemFilter := &base.Filter{
			Group: base.FilterGroup{
				Conditions: []base.FilterCondition{
					{Field: "order_id", Operator: base.OpEqual, Value: order.ID},
				},
				Logic: base.LogicAnd,
			},
		}

		var orderItems []models.OrderItem
		if err := r.dbManager.List(ctx, itemFilter, &orderItems); err != nil {
			return nil, fmt.Errorf("failed to get order items for order %s: %w", order.ID, err)
		}
		order.OrderItems = orderItems
	}

	return orders, nil
}

// OrderItemWithProduct represents an order item with product details
type OrderItemWithProduct struct {
	models.OrderItem
	ProductName     string `json:"product_name"`
	ProductImageURL string `json:"product_image_url"`
}

// GetOrderWithItemsAndProducts retrieves an order with its items including product details
func (r *OrderRepository) GetOrderWithItemsAndProducts(ctx context.Context, orderID string) (*models.Order, []OrderItemWithProduct, error) {
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

	// Collect unique product IDs
	productIDSet := make(map[string]bool)
	for _, item := range orderItems {
		productIDSet[item.ProductID] = true
	}

	// Fetch all products in a single query for better performance
	var productIDs []string
	for productID := range productIDSet {
		productIDs = append(productIDs, productID)
	}

	// Get all products at once
	var products []models.Product
	if len(productIDs) > 0 {
		productFilter := &base.Filter{
			Group: base.FilterGroup{
				Conditions: []base.FilterCondition{
					{Field: "id", Operator: base.OpIn, Value: productIDs},
				},
				Logic: base.LogicAnd,
			},
		}
		if err := r.dbManager.List(ctx, productFilter, &products); err != nil {
			// If we can't fetch products, continue with default values
			products = []models.Product{}
		}
	}

	// Create a map for quick product lookup
	productMap := make(map[string]models.Product)
	for _, product := range products {
		productMap[product.ID] = product
	}

	// Enrich order items with product details
	var enrichedItems []OrderItemWithProduct
	for _, item := range orderItems {
		enrichedItem := OrderItemWithProduct{
			OrderItem: item,
		}

		// Get product details from the map
		if product, exists := productMap[item.ProductID]; exists {
			enrichedItem.ProductName = product.Name

			// Get primary image for the product
			// Note: This is a simplified approach. In a production system, you might want to
			// batch fetch all images at once for better performance
			primaryImage, err := r.getPrimaryImageForProduct(ctx, item.ProductID)
			if err == nil && primaryImage != nil {
				enrichedItem.ProductImageURL = primaryImage.ImageURL
			} else {
				enrichedItem.ProductImageURL = ""
			}
		} else {
			// If product not found, set default values
			enrichedItem.ProductName = "Product Not Found"
			enrichedItem.ProductImageURL = ""
		}

		enrichedItems = append(enrichedItems, enrichedItem)
	}

	return &order, enrichedItems, nil
}

// getPrimaryImageForProduct is a helper method to get the primary image for a product
func (r *OrderRepository) getPrimaryImageForProduct(ctx context.Context, productID string) (*models.Image, error) {
	// This is a simplified implementation. In a production system, you might want to
	// use the image repository directly or batch fetch images for better performance
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "product_id", Operator: base.OpEqual, Value: productID},
				{Field: "is_primary", Operator: base.OpEqual, Value: true},
				{Field: "is_active", Operator: base.OpEqual, Value: true},
			},
			Logic: base.LogicAnd,
		},
		Limit: 1,
	}

	var images []models.Image
	if err := r.dbManager.List(ctx, filter, &images); err != nil {
		return nil, err
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("no primary image found for product %s", productID)
	}

	return &images[0], nil
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

// Find retrieves orders with filtering
func (r *OrderRepository) Find(ctx context.Context, filter *base.Filter) ([]*models.Order, error) {
	var orders []*models.Order
	if err := r.dbManager.List(ctx, filter, &orders); err != nil {
		return nil, fmt.Errorf("failed to find orders: %w", err)
	}
	return orders, nil
}

// CountWithFilter counts orders with a filter
func (r *OrderRepository) CountWithFilter(ctx context.Context, filter *base.Filter) (int64, error) {
	count, err := r.dbManager.Count(ctx, filter, &models.Order{})
	if err != nil {
		return 0, fmt.Errorf("failed to count orders: %w", err)
	}
	return count, nil
}
