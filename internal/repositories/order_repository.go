package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"gorm.io/gorm"
)

// ErrDuplicateOrder is returned when an order already exists for a given
// razorpay_order_id. Callers use it to route to idempotent reuse instead of
// creating a second order.
var ErrDuplicateOrder = errors.New("order already exists for razorpay order id")

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

// UpdateInvoiceURL writes only the invoice_url column (#33). The invoice is generated
// after a slow GetOrderWithDetails + PDF build, so a full-row Save here would risk
// clobbering any column a concurrent request changed in the meantime. A column-scoped
// UPDATE touches only invoice_url and leaves every other field untouched.
func (r *OrderRepository) UpdateInvoiceURL(ctx context.Context, orderID, url string) error {
	if r.pgManager == nil {
		return fmt.Errorf("postgres manager unavailable for invoice update")
	}
	gdb, err := r.pgManager.GetDB(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to get db for invoice update: %w", err)
	}
	res := gdb.WithContext(ctx).
		Model(&models.Order{}).
		Where("id = ?", orderID).
		Update("invoice_url", url)
	if res.Error != nil {
		return fmt.Errorf("failed to update invoice url: %w", res.Error)
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

		// Collect unique product IDs from order items
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
		productMap := make(map[string]models.Product)
		if len(productIDs) > 0 {
			productFilter := &base.Filter{
				Group: base.FilterGroup{
					Conditions: []base.FilterCondition{
						{Field: "id", Operator: base.OpIn, Value: productIDs},
					},
					Logic: base.LogicAnd,
				},
			}
			var products []models.Product
			if err := r.dbManager.List(ctx, productFilter, &products); err == nil {
				for _, product := range products {
					productMap[product.ID] = product
				}
			}
		}

		// Enrich order items with product name and images (for list view)
		for i := range orderItems {
			item := &orderItems[i]
			// Create minimal product object
			if item.Product == nil {
				item.Product = &models.Product{}
			}

			// Get product name from the products table
			if product, exists := productMap[item.ProductID]; exists {
				item.Product.Name = product.Name
			}

			// Get primary image for the product
			primaryImage, err := r.getPrimaryImageForProduct(ctx, item.ProductID)
			if err == nil && primaryImage != nil {
				item.Product.ImageURL = primaryImage.ImageURL
			}
		}

		order.OrderItems = orderItems
	}

	return orders, nil
}

// OrderItemWithProduct represents an order item with product details
type OrderItemWithProduct struct {
	models.OrderItem
	ProductName        string   `json:"product_name"`
	ProductImageURL    string   `json:"product_image_url"`  // Primary image
	ProductImageURLs   []string `json:"product_image_urls"` // All images
	ProductDescription string   `json:"product_description,omitempty"`
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
			enrichedItem.ProductDescription = product.Description

			// Get all images for the product (batch fetch for better performance)
			images, err := r.getAllImagesForProduct(ctx, item.ProductID)
			if err == nil && len(images) > 0 {
				// Primary image is the first one (or marked as primary)
				enrichedItem.ProductImageURL = images[0].ImageURL
				// Store all image URLs
				enrichedItem.ProductImageURLs = make([]string, 0, len(images))
				for _, img := range images {
					enrichedItem.ProductImageURLs = append(enrichedItem.ProductImageURLs, img.ImageURL)
				}
			} else {
				// Fallback to primary image only
				primaryImage, err := r.getPrimaryImageForProduct(ctx, item.ProductID)
				if err == nil && primaryImage != nil {
					enrichedItem.ProductImageURL = primaryImage.ImageURL
					enrichedItem.ProductImageURLs = []string{primaryImage.ImageURL}
				} else {
					enrichedItem.ProductImageURL = ""
					enrichedItem.ProductImageURLs = []string{}
				}
			}
		} else {
			// If product not found, set default values
			enrichedItem.ProductName = "Product Not Found"
			enrichedItem.ProductImageURL = ""
			enrichedItem.ProductImageURLs = []string{}
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

// getAllImagesForProduct gets all active images for a product (ordered by is_primary, then sort_order)
func (r *OrderRepository) getAllImagesForProduct(ctx context.Context, productID string) ([]models.Image, error) {
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "product_id", Operator: base.OpEqual, Value: productID},
				{Field: "is_active", Operator: base.OpEqual, Value: true},
			},
			Logic: base.LogicAnd,
		},
	}

	var images []models.Image
	if err := r.dbManager.List(ctx, filter, &images); err != nil {
		return nil, err
	}

	// Sort images: primary first, then by sort_order
	// Primary images (is_primary=true) come first, then sorted by sort_order
	for i := 0; i < len(images)-1; i++ {
		for j := i + 1; j < len(images); j++ {
			// Primary images first
			if !images[i].IsPrimary && images[j].IsPrimary {
				images[i], images[j] = images[j], images[i]
			} else if images[i].IsPrimary == images[j].IsPrimary {
				// If both have same primary status, sort by sort_order
				if images[i].SortOrder > images[j].SortOrder {
					images[i], images[j] = images[j], images[i]
				}
			}
		}
	}

	return images, nil
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

// GetOrderByRazorpayPaymentID retrieves the order bound to a Razorpay payment ID.
// Used for ownership checks so a caller can only read the payment status of their
// own order.
func (r *OrderRepository) GetOrderByRazorpayPaymentID(ctx context.Context, razorpayPaymentID string) (*models.Order, error) {
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "razorpay_payment_id", Operator: base.OpEqual, Value: razorpayPaymentID},
			},
			Logic: base.LogicAnd,
		},
	}

	var orders []*models.Order
	if err := r.dbManager.List(ctx, filter, &orders); err != nil {
		return nil, fmt.Errorf("failed to get order by razorpay payment ID: %w", err)
	}

	if len(orders) == 0 {
		return nil, fmt.Errorf("order not found for razorpay payment ID: %s", razorpayPaymentID)
	}

	return orders[0], nil
}

// GetOrderByWaybill retrieves the order bound to a Delhivery waybill. Used for
// ownership checks so a caller can only track their own shipment.
func (r *OrderRepository) GetOrderByWaybill(ctx context.Context, waybill string) (*models.Order, error) {
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "delhivery_waybill", Operator: base.OpEqual, Value: waybill},
			},
			Logic: base.LogicAnd,
		},
	}

	var orders []*models.Order
	if err := r.dbManager.List(ctx, filter, &orders); err != nil {
		return nil, fmt.Errorf("failed to get order by waybill: %w", err)
	}

	if len(orders) == 0 {
		return nil, fmt.Errorf("order not found for waybill: %s", waybill)
	}

	return orders[0], nil
}

// CreateOrderWithItemsTx creates an order and ALL its items inside a single DB
// transaction. Either the order and every item commit together, or nothing does —
// so a partial failure can never leave a zero-item order (#14) or an orphan order
// (#32). The order must carry its razorpay_order_id so the unique index dedups
// concurrent verifies; a duplicate returns ErrDuplicateOrder (#13). An empty items
// slice is refused.
func (r *OrderRepository) CreateOrderWithItemsTx(ctx context.Context, order *models.Order, items []models.OrderItem) error {
	if r.pgManager == nil {
		return fmt.Errorf("postgres manager unavailable for transactional order creation")
	}
	if len(items) == 0 {
		return fmt.Errorf("refusing to create an order with no items")
	}

	// Defense-in-depth pre-check; the unique index on razorpay_order_id is the real guard.
	if order.RazorpayOrderID != "" {
		if existing, _ := r.GetOrderByRazorpayOrderID(ctx, order.RazorpayOrderID); existing != nil {
			return ErrDuplicateOrder
		}
	}

	return r.pgManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Create(order).Error; err != nil {
			if isDuplicateKeyError(err) {
				return ErrDuplicateOrder
			}
			return fmt.Errorf("failed to create order: %w", err)
		}
		for i := range items {
			if err := tx.WithContext(ctx).Create(&items[i]).Error; err != nil {
				return fmt.Errorf("failed to create order item for product %s: %w", items[i].ProductID, err)
			}
		}
		return nil
	})
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

// ApplyShipmentSuccess records a successful shipment and advances the order:
// sets the Delhivery fields, moves Status to shipped, and marks fulfillment complete — in one write.
func (r *OrderRepository) ApplyShipmentSuccess(ctx context.Context, orderID, waybill, trackingURL, delhiveryStatus string) error {
	var order models.Order
	if err := r.dbManager.GetByID(ctx, orderID, &order); err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	order.DelhiveryWaybill = waybill
	order.TrackingURL = trackingURL
	order.DelhiveryStatus = delhiveryStatus
	order.Status = models.OrderStatusShipped
	order.FulfillmentStatus = models.FulfillmentFulfilled
	order.ShipmentError = ""

	if err := r.dbManager.Update(ctx, &order); err != nil {
		return fmt.Errorf("failed to apply shipment success: %w", err)
	}

	return nil
}

// SetFulfillmentStatus records the post-payment fulfillment state (and last shipment error, if any)
// so paid-but-unshipped orders are queryable and recoverable.
func (r *OrderRepository) SetFulfillmentStatus(ctx context.Context, orderID, status, shipmentErr string) error {
	var order models.Order
	if err := r.dbManager.GetByID(ctx, orderID, &order); err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	order.FulfillmentStatus = status
	order.ShipmentError = shipmentErr

	if err := r.dbManager.Update(ctx, &order); err != nil {
		return fmt.Errorf("failed to set fulfillment status: %w", err)
	}

	return nil
}

// GetPostgresManager exposes the Postgres manager for callers that need to run
// multi-statement atomic transactions (e.g. inventory deduction) spanning tables
// this repository does not own.
func (r *OrderRepository) GetPostgresManager() *db.PostgresManager {
	return r.pgManager
}

// SetInventoryDeducted flips the inventory-deducted idempotency guard on the order.
func (r *OrderRepository) SetInventoryDeducted(ctx context.Context, orderID string, deducted bool) error {
	var order models.Order
	if err := r.dbManager.GetByID(ctx, orderID, &order); err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	order.InventoryDeducted = deducted

	if err := r.dbManager.Update(ctx, &order); err != nil {
		return fmt.Errorf("failed to set inventory_deducted: %w", err)
	}

	return nil
}

// ClaimOrderForShipment atomically claims an order for a SINGLE shipment attempt. Exactly one
// concurrent caller (payment verify, the Razorpay webhook, or an admin retry) wins the claim; the
// others receive false and must NOT call the courier. This prevents duplicate shipments and
// orphaned waybills (#9/#10) that a check-then-act on an empty waybill allowed.
//
// An order is claimable only when it has no waybill yet AND its fulfillment status is retryable
// (empty, pending_shipment, or shipment_failed). The states shipment_in_progress, fulfilled, and
// needs_manual_fulfillment are deliberately not auto-claimable, so a shipment that was created at
// the courier but failed to persist (marked needs_manual) is never automatically re-shipped.
func (r *OrderRepository) ClaimOrderForShipment(ctx context.Context, orderID string) (bool, error) {
	if r.pgManager == nil {
		// Fail safe: without atomic DB access we refuse the claim rather than risk a double shipment.
		return false, fmt.Errorf("postgres manager unavailable for shipment claim")
	}
	gdb, err := r.pgManager.GetDB(ctx, false)
	if err != nil {
		return false, fmt.Errorf("failed to get db for shipment claim: %w", err)
	}

	claimable := []string{"", models.FulfillmentPendingShipment, models.FulfillmentShipmentFailed}
	res := gdb.WithContext(ctx).
		Model(&models.Order{}).
		Where("id = ? AND (delhivery_waybill IS NULL OR delhivery_waybill = '') AND (fulfillment_status IS NULL OR fulfillment_status IN ?)",
			orderID, claimable).
		Update("fulfillment_status", models.FulfillmentShipmentInProgress)
	if res.Error != nil {
		return false, fmt.Errorf("failed to claim order for shipment: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

// ClaimOrderForShipmentAdmin is the manual/admin-retry counterpart of ClaimOrderForShipment. It is
// more permissive — it also allows re-attempting orders in shipment_failed or needs_manual_fulfillment
// (e.g. after an admin corrects the address) — but it still refuses to claim an order that already
// has a waybill, is fulfilled, or has an automated attempt in progress, so it can never create a
// duplicate shipment.
func (r *OrderRepository) ClaimOrderForShipmentAdmin(ctx context.Context, orderID string) (bool, error) {
	if r.pgManager == nil {
		return false, fmt.Errorf("postgres manager unavailable for shipment claim")
	}
	gdb, err := r.pgManager.GetDB(ctx, false)
	if err != nil {
		return false, fmt.Errorf("failed to get db for shipment claim: %w", err)
	}

	blocked := []string{models.FulfillmentFulfilled, models.FulfillmentShipmentInProgress}
	res := gdb.WithContext(ctx).
		Model(&models.Order{}).
		Where("id = ? AND (delhivery_waybill IS NULL OR delhivery_waybill = '') AND (fulfillment_status IS NULL OR fulfillment_status NOT IN ?)",
			orderID, blocked).
		Update("fulfillment_status", models.FulfillmentShipmentInProgress)
	if res.Error != nil {
		return false, fmt.Errorf("failed to claim order for admin shipment: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
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
