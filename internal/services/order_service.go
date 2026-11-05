package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OrderServiceInterface defines the contract for order operations
type OrderServiceInterface interface {
	CreateOrderFromCart(ctx context.Context, userID string, cartID string, shippingAddress string, paymentMethod string) (*models.Order, error)
	CreateOrderFromCartWithShipping(ctx context.Context, userID string, shippingAddress string, paymentMethod string, shippingCharges float64) (*models.Order, error)
	// Create order with customer and address data
	CreateOrder(ctx context.Context, userID string, customerData *models.OrderCustomer, addressData *models.OrderAddress, paymentMethod string, shippingCharges float64) (*models.Order, error)
	GetUserCart(ctx context.Context, userID string) (*models.Cart, error)
	GetOrderByID(ctx context.Context, orderID string) (*models.Order, error)
	GetOrderByRazorpayOrderID(ctx context.Context, razorpayOrderID string) (*models.Order, error)
	GetOrderByRazorpayPaymentID(ctx context.Context, razorpayPaymentID string) (*models.Order, error)
	GetOrderByWaybill(ctx context.Context, waybill string) (*models.Order, error)
	GetUserOrders(ctx context.Context, userID string) ([]models.Order, error)
	GetOrderDetails(ctx context.Context, orderID string) (*models.Order, []repositories.OrderItemWithProduct, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status models.OrderStatus) error
	UpdatePaymentStatus(ctx context.Context, orderID string, paymentStatus, paymentID string) error
	UpdatePaymentStatusAndOrderStatus(ctx context.Context, orderID string, paymentStatus, paymentID string, orderStatus models.OrderStatus) error
	GetOrderWithItems(ctx context.Context, orderID string) (*models.Order, []models.OrderItem, error)
	GetOrderWithItemsAndProducts(ctx context.Context, orderID string) (*models.Order, []repositories.OrderItemWithProduct, error)
	ConfirmOrderPayment(ctx context.Context, razorpayOrderID, paymentID string) error
	UpdateOrderNotes(ctx context.Context, orderID string, notes string) error
	ClearUserCart(ctx context.Context, userID string) error
	UpdateOrder(ctx context.Context, order *models.Order) error
	GenerateAndAttachInvoice(ctx context.Context, orderID string) (string, error)
	// Transaction-based methods for payment verification
	CreateOrderFromCartWithShippingAtomic(ctx context.Context, userID string, shippingAddress string, paymentMethod string, shippingCharges float64, razorpayOrderID string) (*models.Order, error)
	UpdateOrderWithShipmentAtomic(ctx context.Context, orderID string, waybill string, trackingURL string, delhiveryStatus string) error
	// Fulfillment helpers (idempotent; shared by verify, webhook and manual retry)
	ApplyShipmentSuccess(ctx context.Context, orderID, waybill, trackingURL, delhiveryStatus string) error
	MarkFulfillmentStatus(ctx context.Context, orderID, status, shipmentErr string) error
	DeductInventoryForOrder(ctx context.Context, orderID string) error
	ClaimOrderForShipment(ctx context.Context, orderID string) (bool, error)
	ClaimOrderForShipmentAdmin(ctx context.Context, orderID string) (bool, error)
	// Address helpers
	GetAddressByID(ctx context.Context, userID, addressID string) (*models.Address, error)
	FormatAddress(addr *models.Address) string
	// Admin listing/stats/export
	ListOrders(ctx context.Context, status, paymentStatus, q string, from, to *time.Time, page, limit int) ([]models.Order, int, error)
	ListUnfulfilledOrders(ctx context.Context, page, limit int) ([]models.Order, int, error)
	GetOrderStats(ctx context.Context) (map[string]interface{}, error)
	ExportOrdersCSV(ctx context.Context) ([]byte, string, error)
}

// OrderService implements order business logic
type OrderService struct {
	orderRepo   *repositories.OrderRepository
	cartService CartServiceInterface
	productRepo repositories.ProductRepositoryInterface
	addressRepo *repositories.AddressRepository
	couponRepo  repositories.CouponRepositoryInterface
	r2Service   *R2Service // Cloudflare R2 storage
	logger      *zap.Logger
}

// NewOrderService creates a new order service
func NewOrderService(orderRepo *repositories.OrderRepository, cartService CartServiceInterface, productRepo repositories.ProductRepositoryInterface, addressRepo *repositories.AddressRepository, couponRepo repositories.CouponRepositoryInterface, r2Service *R2Service, logger *zap.Logger) OrderServiceInterface {
	service := &OrderService{
		orderRepo:   orderRepo,
		cartService: cartService,
		productRepo: productRepo,
		addressRepo: addressRepo,
		couponRepo:  couponRepo,
		r2Service:   r2Service,
		logger:      logger,
	}

	// Initialize order ID counter from existing database records
	service.initializeOrderIDCounter()

	// Initialize order item ID counter from existing database records
	service.initializeOrderItemIDGenerator()

	// Initialize order customer ID counter from existing database records
	service.initializeOrderCustomerIDCounter()

	// Initialize order address ID counter from existing database records
	service.initializeOrderAddressIDCounter()

	return service
}

// initializeOrderIDCounter initializes the order ID counter from existing database records
func (s *OrderService) initializeOrderIDCounter() {
	ctx := context.Background()
	s.logger.Info("Initializing order ID counter")

	// Query all existing order IDs to prevent conflicts (exclude soft-deleted)
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "deleted_at", Operator: base.OpIsNull, Value: nil},
			},
			Logic: base.LogicAnd,
		},
		Page:     1,
		PageSize: 10000,
	}

	orders, err := s.orderRepo.Find(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get existing orders for ID initialization", zap.Error(err))
		// Fallback to empty initialization
		hash.InitializeGlobalCountersFromDatabase("ORDR", []string{}, hash.Large)
	} else {
		// Extract IDs
		existingIDs := make([]string, len(orders))
		for i, order := range orders {
			if order != nil {
				existingIDs[i] = order.GetID()
			}
		}
		s.logger.Info("Found existing order IDs", zap.Int("count", len(existingIDs)))
		hash.InitializeGlobalCountersFromDatabase("ORDR", existingIDs, hash.Large)
	}

	s.logger.Info("Order ID counter initialized")
}

// initializeOrderItemIDGenerator initializes the order item ID generator from existing database records
func (s *OrderService) initializeOrderItemIDGenerator() {
	ctx := context.Background()
	s.logger.Info("Initializing order item ID generator")

	// Query all existing order item IDs to prevent conflicts
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{},
			Logic:      base.LogicAnd,
		},
		Page:     1,
		PageSize: 10000,
	}

	var orderItems []*models.OrderItem
	if err := s.orderRepo.GetOrderItems(ctx, filter, &orderItems); err != nil {
		s.logger.Error("Failed to get existing order items for ID initialization", zap.Error(err))
		// Fallback to empty initialization
		hash.InitializeGlobalCountersFromDatabase("OITM", []string{}, hash.Large)
	} else {
		// Extract IDs
		existingIDs := make([]string, 0, len(orderItems))
		for _, item := range orderItems {
			if item != nil && item.GetID() != "" {
				existingIDs = append(existingIDs, item.GetID())
			}
		}
		s.logger.Info("Found existing order item IDs", zap.Int("count", len(existingIDs)))
		hash.InitializeGlobalCountersFromDatabase("OITM", existingIDs, hash.Large)
	}
	s.logger.Info("Order item ID generator initialized")
}

// initializeOrderCustomerIDCounter initializes the order customer ID counter from existing database records
func (s *OrderService) initializeOrderCustomerIDCounter() {
	ctx := context.Background()
	s.logger.Info("Initializing order customer ID counter")

	// Query all existing order customer IDs to prevent conflicts (exclude soft-deleted)
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "deleted_at", Operator: base.OpIsNull, Value: nil},
			},
			Logic: base.LogicAnd,
		},
		Page:     1,
		PageSize: 10000,
	}

	var orderCustomers []*models.OrderCustomer
	if err := s.orderRepo.GetOrderCustomers(ctx, filter, &orderCustomers); err != nil {
		s.logger.Error("Failed to get existing order customers for ID initialization", zap.Error(err))
		// Fallback to empty initialization
		hash.InitializeGlobalCountersFromDatabase("OCUS", []string{}, hash.Medium)
	} else {
		// Extract IDs
		existingIDs := make([]string, 0, len(orderCustomers))
		for _, customer := range orderCustomers {
			if customer != nil && customer.GetID() != "" {
				existingIDs = append(existingIDs, customer.GetID())
			}
		}
		s.logger.Info("Found existing order customer IDs", zap.Int("count", len(existingIDs)))
		hash.InitializeGlobalCountersFromDatabase("OCUS", existingIDs, hash.Medium)
	}

	s.logger.Info("Order customer ID counter initialized")
}

// initializeOrderAddressIDCounter initializes the order address ID counter from existing database records
func (s *OrderService) initializeOrderAddressIDCounter() {
	ctx := context.Background()
	s.logger.Info("Initializing order address ID counter")

	// Query all existing order address IDs to prevent conflicts (exclude soft-deleted)
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "deleted_at", Operator: base.OpIsNull, Value: nil},
			},
			Logic: base.LogicAnd,
		},
		Page:     1,
		PageSize: 10000,
	}

	var orderAddresses []*models.OrderAddress
	if err := s.orderRepo.GetOrderAddresses(ctx, filter, &orderAddresses); err != nil {
		s.logger.Error("Failed to get existing order addresses for ID initialization", zap.Error(err))
		// Fallback to empty initialization
		hash.InitializeGlobalCountersFromDatabase("OADR", []string{}, hash.Medium)
	} else {
		// Extract IDs
		existingIDs := make([]string, 0, len(orderAddresses))
		for _, address := range orderAddresses {
			if address != nil && address.GetID() != "" {
				existingIDs = append(existingIDs, address.GetID())
			}
		}
		s.logger.Info("Found existing order address IDs", zap.Int("count", len(existingIDs)))
		hash.InitializeGlobalCountersFromDatabase("OADR", existingIDs, hash.Medium)
	}

	s.logger.Info("Order address ID counter initialized")
}

// GetUserCart retrieves the user's cart via cart service
func (s *OrderService) GetUserCart(ctx context.Context, userID string) (*models.Cart, error) {
	return s.cartService.GetUserCart(ctx, userID)
}

// CreateOrderFromCart creates a new order from user's cart
func (s *OrderService) CreateOrderFromCart(ctx context.Context, userID string, cartID string, shippingAddress string, paymentMethod string) (*models.Order, error) {
	// Get user's cart with items
	cart, err := s.cartService.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	if cart == nil {
		return nil, fmt.Errorf("no cart found for user %s", userID)
	}

	if len(cart.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// Calculate total amount including any discounts (no tax)
	cartTotal := cart.GetTotalAmount()
	discountedCartTotal := utils.ApplyDiscount(cartTotal, cart.DiscountAmount)

	// Calculate final total (no tax calculation)
	totalAmount := discountedCartTotal

	// Create order (using legacy method for backward compatibility)
	order := models.NewOrderLegacy(userID, totalAmount, shippingAddress, paymentMethod, models.OrderStatusPending, "pending")

	// Set coupon information in dedicated fields
	if cart.CouponCode != "" {
		order.CouponCode = cart.CouponCode
		order.DiscountAmount = cart.DiscountAmount
	}

	// Add coupon information to notes if applied
	notes := ""
	if cart.CouponCode != "" {
		notes += fmt.Sprintf("Coupon: %s, Discount: %.2f", cart.CouponCode, cart.DiscountAmount)
	}
	order.Notes = notes

	// Save order to database using the repository create method
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Create order items
	var orderItems []models.OrderItem
	for _, cartItem := range cart.Items {
		orderItem := models.NewOrderItem(order.ID, cartItem.ProductID, cartItem.Quantity, cartItem.UnitPrice)
		orderItems = append(orderItems, *orderItem)
	}

	// Save order items
	if err := s.createOrderItems(ctx, orderItems); err != nil {
		s.logger.Error("Failed to create order items", zap.Error(err), zap.String("order_id", order.ID))
		// Don't fail the order creation, but log the error
	}

	s.logger.Info("Order created successfully",
		zap.String("order_id", order.ID),
		zap.String("user_id", userID),
		zap.Float64("total_amount", totalAmount),
		zap.Int("item_count", len(orderItems)))

	return order, nil
}

// CreateOrderFromCartWithShipping creates a new order from user's cart including shipping charges
func (s *OrderService) CreateOrderFromCartWithShipping(ctx context.Context, userID string, shippingAddress string, paymentMethod string, shippingCharges float64) (*models.Order, error) {
	s.logger.Info("Starting order creation",
		zap.String("user_id", userID),
		zap.String("shipping_address", shippingAddress),
		zap.Float64("shipping_charges", shippingCharges))

	// Get user's cart with items
	cart, err := s.cartService.GetUserCart(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user cart",
			zap.String("user_id", userID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	if cart == nil {
		s.logger.Error("No cart found for user", zap.String("user_id", userID))
		return nil, fmt.Errorf("no cart found for user %s", userID)
	}

	s.logger.Info("Found user cart",
		zap.String("user_id", userID),
		zap.String("cart_id", cart.ID),
		zap.Int("item_count", len(cart.Items)))

	if len(cart.Items) == 0 {
		s.logger.Error("Cart is empty", zap.String("user_id", userID), zap.String("cart_id", cart.ID))
		return nil, fmt.Errorf("cart is empty")
	}

	// Calculate cart total amount including any discounts (no tax)
	cartTotal := cart.GetTotalAmount()
	discountedCartTotal := utils.ApplyDiscount(cartTotal, cart.DiscountAmount)

	// Add shipping charges to get final total (no tax)
	totalAmount := utils.AddShippingCharges(discountedCartTotal, shippingCharges)

	s.logger.Info("Calculated amounts",
		zap.String("user_id", userID),
		zap.Float64("cart_total", cartTotal),
		zap.Float64("discounted_cart_total", discountedCartTotal),
		zap.Float64("discount_amount", cart.DiscountAmount),
		zap.Float64("shipping_charges", shippingCharges),
		zap.Float64("total_amount", totalAmount))

	// Create order with total amount including shipping
	order := models.NewOrderLegacy(userID, totalAmount, shippingAddress, paymentMethod, models.OrderStatusPending, "pending")
	// Set shipping metadata
	order.ShippingCarrier = "Delhivery"
	order.ShippingMode = "Surface"
	order.ShippingCharge = shippingCharges

	// Set coupon information in dedicated fields
	if cart.CouponCode != "" {
		order.CouponCode = cart.CouponCode
		order.DiscountAmount = cart.DiscountAmount
	}

	// Add shipping to notes
	notes := ""
	if shippingCharges > 0 {
		notes += fmt.Sprintf("Shipping: %.2f", shippingCharges)
	}
	order.Notes = notes

	s.logger.Info("Created order object",
		zap.String("order_id", order.ID),
		zap.String("user_id", userID),
		zap.Float64("total_amount", totalAmount))

	// Save order to database using the repository create method
	if err := s.orderRepo.Create(ctx, order); err != nil {
		s.logger.Error("Failed to save order to database",
			zap.String("order_id", order.ID),
			zap.String("user_id", userID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	s.logger.Info("Order saved to database successfully",
		zap.String("order_id", order.ID),
		zap.String("user_id", userID))

	// Create order items
	var orderItems []models.OrderItem
	for _, cartItem := range cart.Items {
		orderItem := models.NewOrderItem(order.ID, cartItem.ProductID, cartItem.Quantity, cartItem.UnitPrice)
		orderItems = append(orderItems, *orderItem)
	}

	s.logger.Info("Created order items",
		zap.String("order_id", order.ID),
		zap.Int("item_count", len(orderItems)))

	// Save order items
	if err := s.createOrderItems(ctx, orderItems); err != nil {
		s.logger.Error("Failed to create order items", zap.Error(err), zap.String("order_id", order.ID))
		// Don't fail the order creation, but log the error
	}

	// Update inventory quantities (decrement stock)
	if err := s.updateInventoryForOrderItems(ctx, orderItems); err != nil {
		s.logger.Warn("Inventory update encountered issues", zap.Error(err), zap.String("order_id", order.ID))
		// Do not fail order creation on inventory update issues, but log
	}

	s.logger.Info("Order created successfully with shipping",
		zap.String("order_id", order.ID),
		zap.String("user_id", userID),
		zap.Float64("cart_total", cartTotal),
		zap.Float64("shipping_charges", shippingCharges),
		zap.Float64("total_amount", totalAmount),
		zap.Int("item_count", len(orderItems)))

	return order, nil
}

// updateInventoryForOrderItems decrements stock for each ordered product using an
// atomic per-item UPDATE (no read-modify-write, so concurrent deductions can't lose
// updates). An oversell (stock below the ordered quantity) is logged rather than
// silently hidden; stock is still floored at zero since a paid item can't be un-sold.
func (s *OrderService) updateInventoryForOrderItems(ctx context.Context, items []models.OrderItem) error {
	if s.productRepo == nil {
		return fmt.Errorf("product repository unavailable")
	}
	var firstErr error
	for _, it := range items {
		oversold, err := s.productRepo.DecrementStockAtomic(ctx, nil, it.ProductID, it.Quantity)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to decrement stock for %s: %w", it.ProductID, err)
			}
			continue
		}
		if oversold {
			s.logger.Warn("Inventory oversold (stock floored to zero)",
				zap.String("product_id", it.ProductID),
				zap.Int("quantity", it.Quantity))
		}
	}
	return firstErr
}

// UpdateOrder persists changes to an order
func (s *OrderService) UpdateOrder(ctx context.Context, order *models.Order) error {
	if order == nil || order.ID == "" {
		return fmt.Errorf("invalid order to update")
	}
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}
	return nil
}

// GetAddressByID fetches an address for a user and ensures ownership
func (s *OrderService) GetAddressByID(ctx context.Context, userID, addressID string) (*models.Address, error) {
	if s.addressRepo == nil {
		return nil, fmt.Errorf("address repository unavailable")
	}
	var addr models.Address
	res, err := s.addressRepo.GetByID(ctx, addressID, &addr)
	if err != nil || res == nil {
		return nil, fmt.Errorf("address not found")
	}
	if addr.UserID != userID {
		return nil, fmt.Errorf("forbidden: address does not belong to user")
	}
	return &addr, nil
}

// FormatAddress builds a single-line shipping address string
func (s *OrderService) FormatAddress(addr *models.Address) string {
	if addr == nil {
		return ""
	}
	// Compose without excessive blanks
	line2 := ""
	if addr.AddressLine2 != "" {
		line2 = ", " + addr.AddressLine2
	}
	phone := ""
	if addr.Phone != "" {
		phone = fmt.Sprintf(" (Ph: %s)", addr.Phone)
	}
	name := addr.FirstName
	if addr.LastName != "" {
		name = name + " " + addr.LastName
	}
	return fmt.Sprintf("%s, %s%s, %s, %s %s, %s%s", name, addr.AddressLine1, line2, addr.City, addr.State, addr.PostalCode, addr.Country, phone)
}

// GenerateAndAttachInvoice generates a PDF invoice, uploads to R2 (preferred) or S3, and stores URL on the order
func (s *OrderService) GenerateAndAttachInvoice(ctx context.Context, orderID string) (string, error) {
	// Get order with full details including shipping address from order_addresses table
	order, err := s.orderRepo.GetOrderWithDetails(ctx, orderID)
	if err != nil {
		return "", fmt.Errorf("failed to load order for invoice: %w", err)
	}

	// Populate ShippingAddr from OrderAddress if available (for invoice generation)
	if order.ShippingAddress != nil {
		addr := order.ShippingAddress
		line2 := ""
		if addr.AddressLine2 != "" {
			line2 = ", " + addr.AddressLine2
		}
		order.ShippingAddr = fmt.Sprintf("%s%s, %s, %s %s, %s",
			addr.AddressLine1, line2, addr.City, addr.State, addr.PostalCode, addr.Country)

		s.logger.Info("Using order_addresses for invoice",
			zap.String("order_id", orderID),
			zap.String("address_id", order.ShippingAddressID),
			zap.String("formatted_address", order.ShippingAddr))
	}

	// Populate customer name from OrderCustomer if available
	if order.Customer != nil {
		order.CustomerFirstName = order.Customer.FirstName
		order.CustomerLastName = order.Customer.LastName
		order.CustomerPhone = order.Customer.Phone
	}

	// Create PDF invoice service with R2 storage
	pdfInvoiceService := NewPDFInvoiceService(s.r2Service, s.logger)

	// Convert items to pointers
	items := order.OrderItems
	itemPointers := make([]*models.OrderItem, len(items))
	for i := range items {
		itemPointers[i] = &items[i]
	}

	// Generate PDF invoice
	url, err := pdfInvoiceService.GeneratePDFInvoice(ctx, order, itemPointers, s.productRepo)
	if err != nil {
		return "", fmt.Errorf("failed to generate PDF invoice: %w", err)
	}

	// Update order with invoice URL
	order.InvoiceURL = url
	if err := s.orderRepo.Update(ctx, order); err != nil {
		s.logger.Error("Failed to update order with invoice URL",
			zap.String("order_id", orderID),
			zap.String("invoice_url", url),
			zap.Error(err))
		// Still return the URL even if database update fails
	} else {
		s.logger.Info("Order updated with invoice URL successfully",
			zap.String("order_id", orderID),
			zap.String("invoice_url", url))
	}

	return url, nil
}

// createOrderItems creates multiple order items
func (s *OrderService) createOrderItems(ctx context.Context, orderItems []models.OrderItem) error {
	return s.orderRepo.CreateOrderItems(ctx, orderItems)
}

// GetOrderByID retrieves an order by its ID - using a helper method to call the repository correctly
func (s *OrderService) GetOrderByID(ctx context.Context, orderID string) (*models.Order, error) {
	var order models.Order
	_, err := s.orderRepo.GetByID(ctx, orderID, &order)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	return &order, nil
}

// GetUserOrders retrieves all orders for a specific user with details
func (s *OrderService) GetUserOrders(ctx context.Context, userID string) ([]models.Order, error) {
	orders, err := s.orderRepo.GetOrdersByUserWithDetails(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %w", err)
	}
	return orders, nil
}

// UpdateOrderStatus updates the status of an order
func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID string, status models.OrderStatus) error {
	if err := s.orderRepo.UpdateOrderStatus(ctx, orderID, status); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	s.logger.Info("Order status updated",
		zap.String("order_id", orderID),
		zap.String("status", string(status)))

	return nil
}

// UpdatePaymentStatus updates the payment status and payment ID of an order
func (s *OrderService) UpdatePaymentStatus(ctx context.Context, orderID string, paymentStatus, paymentID string) error {
	if err := s.orderRepo.UpdatePaymentStatus(ctx, orderID, paymentStatus, paymentID); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	s.logger.Info("Payment status updated",
		zap.String("order_id", orderID),
		zap.String("payment_status", paymentStatus),
		zap.String("payment_id", paymentID))

	return nil
}

// UpdatePaymentStatusAndOrderStatus updates both payment status and order status
func (s *OrderService) UpdatePaymentStatusAndOrderStatus(ctx context.Context, orderID string, paymentStatus, paymentID string, orderStatus models.OrderStatus) error {
	// Update payment status first
	if err := s.UpdatePaymentStatus(ctx, orderID, paymentStatus, paymentID); err != nil {
		return err
	}

	// Update order status
	if err := s.orderRepo.UpdateOrderStatus(ctx, orderID, orderStatus); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	s.logger.Info("Order status updated after payment",
		zap.String("order_id", orderID),
		zap.String("order_status", string(orderStatus)),
		zap.String("payment_status", paymentStatus))

	return nil
}

// UpdateOrderNotes updates the notes field of an order
func (s *OrderService) UpdateOrderNotes(ctx context.Context, orderID string, notes string) error {
	if err := s.orderRepo.UpdateOrderNotes(ctx, orderID, notes); err != nil {
		return fmt.Errorf("failed to update order notes: %w", err)
	}
	s.logger.Info("Order notes updated", zap.String("order_id", orderID))
	return nil
}

// ClearUserCart clears the user's cart after successful order/payment
func (s *OrderService) ClearUserCart(ctx context.Context, userID string) error {
	if s.cartService == nil {
		return fmt.Errorf("cart service unavailable")
	}
	return s.cartService.ClearCart(ctx, userID)
}

// GetOrderWithItems retrieves an order with its items
func (s *OrderService) GetOrderWithItems(ctx context.Context, orderID string) (*models.Order, []models.OrderItem, error) {
	return s.orderRepo.GetOrderWithItems(ctx, orderID)
}

// GetOrderWithItemsAndProducts retrieves an order with its items including product details
func (s *OrderService) GetOrderWithItemsAndProducts(ctx context.Context, orderID string) (*models.Order, []repositories.OrderItemWithProduct, error) {
	return s.orderRepo.GetOrderWithItemsAndProducts(ctx, orderID)
}

// ConfirmOrderPayment confirms an order payment and updates relevant statuses
func (s *OrderService) ConfirmOrderPayment(ctx context.Context, razorpayOrderID, paymentID string) error {
	// This is a simplified implementation - you might need to store razorpay_order_id differently
	// For now, let's update the order by searching for pending orders and matching receipt/notes

	s.logger.Info("Confirming order payment",
		zap.String("razorpay_order_id", razorpayOrderID),
		zap.String("payment_id", paymentID))

	// Since we don't have a direct way to query by razorpay order ID,
	// we'll need to implement this differently
	// For now, just log that we received the payment confirmation
	s.logger.Info("Payment confirmation received - order update needed",
		zap.String("razorpay_order_id", razorpayOrderID),
		zap.String("payment_id", paymentID))

	// TODO: Implement proper order lookup and update
	// This would typically involve:
	// 1. Finding the order by razorpay order ID (stored in notes or separate field)
	// 2. Updating payment status to completed
	// 3. Updating order status to confirmed
	// 4. Clearing the user's cart

	return nil
}

// GetOrderByRazorpayOrderID retrieves an order by Razorpay order ID
func (s *OrderService) GetOrderByRazorpayOrderID(ctx context.Context, razorpayOrderID string) (*models.Order, error) {
	if s.orderRepo == nil {
		return nil, fmt.Errorf("order repository unavailable")
	}
	return s.orderRepo.GetOrderByRazorpayOrderID(ctx, razorpayOrderID)
}

// GetOrderByRazorpayPaymentID retrieves an order by Razorpay payment ID (for ownership checks)
func (s *OrderService) GetOrderByRazorpayPaymentID(ctx context.Context, razorpayPaymentID string) (*models.Order, error) {
	if s.orderRepo == nil {
		return nil, fmt.Errorf("order repository unavailable")
	}
	return s.orderRepo.GetOrderByRazorpayPaymentID(ctx, razorpayPaymentID)
}

// GetOrderByWaybill retrieves an order by Delhivery waybill (for ownership checks)
func (s *OrderService) GetOrderByWaybill(ctx context.Context, waybill string) (*models.Order, error) {
	if s.orderRepo == nil {
		return nil, fmt.Errorf("order repository unavailable")
	}
	return s.orderRepo.GetOrderByWaybill(ctx, waybill)
}

// ListOrders returns filtered, paginated orders for admin
func (s *OrderService) ListOrders(ctx context.Context, status, paymentStatus, q string, from, to *time.Time, page, limit int) ([]models.Order, int, error) {
	if s.orderRepo == nil {
		return nil, 0, fmt.Errorf("order repository unavailable")
	}
	conds := []base.FilterCondition{}
	if status != "" {
		conds = append(conds, base.FilterCondition{Field: "status", Operator: base.OpEqual, Value: status})
	}
	if paymentStatus != "" {
		conds = append(conds, base.FilterCondition{Field: "payment_status", Operator: base.OpEqual, Value: paymentStatus})
	}
	if from != nil {
		conds = append(conds, base.FilterCondition{Field: "created_at", Operator: base.OpGreaterEqual, Value: *from})
	}
	if to != nil {
		conds = append(conds, base.FilterCondition{Field: "created_at", Operator: base.OpLessEqual, Value: *to})
	}
	filter := &base.Filter{Group: base.FilterGroup{Conditions: conds, Logic: base.LogicAnd}, Page: page, PageSize: limit}
	// NOTE: q search omitted to avoid portability issues; can be added with builder if needed
	found, err := s.orderRepo.Find(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}
	// Convert []*models.Order to []models.Order if needed
	orders := make([]models.Order, 0, len(found))
	for _, po := range found {
		if po != nil {
			orders = append(orders, *po)
		}
	}
	count, err := s.orderRepo.CountWithFilter(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}
	// load normalized relations for each order (best-effort)
	for i := range orders {
		if orders[i].CustomerID != "" {
			if customer, err := s.orderRepo.GetOrderCustomer(ctx, orders[i].CustomerID); err == nil {
				orders[i].Customer = customer
			}
		}
		if orders[i].ShippingAddressID != "" {
			if addr, err := s.orderRepo.GetOrderAddress(ctx, orders[i].ShippingAddressID); err == nil {
				orders[i].ShippingAddress = addr
			}
		}
	}
	return orders, int(count), nil
}

// ListUnfulfilledOrders returns paid orders that have no shipment yet — i.e. confirmed orders with
// an empty Delhivery waybill. These are the "payment taken but delivery not created" orders that
// need recovery (fix address + retry shipment). Loads customer/address so admins can act on them.
func (s *OrderService) ListUnfulfilledOrders(ctx context.Context, page, limit int) ([]models.Order, int, error) {
	if s.orderRepo == nil {
		return nil, 0, fmt.Errorf("order repository unavailable")
	}
	conds := []base.FilterCondition{
		{Field: "status", Operator: base.OpEqual, Value: string(models.OrderStatusConfirmed)},
		{Field: "delhivery_waybill", Operator: base.OpEqual, Value: ""},
	}
	filter := &base.Filter{Group: base.FilterGroup{Conditions: conds, Logic: base.LogicAnd}, Page: page, PageSize: limit}

	found, err := s.orderRepo.Find(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list unfulfilled orders: %w", err)
	}
	orders := make([]models.Order, 0, len(found))
	for _, po := range found {
		if po != nil {
			orders = append(orders, *po)
		}
	}
	count, err := s.orderRepo.CountWithFilter(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count unfulfilled orders: %w", err)
	}
	for i := range orders {
		if orders[i].CustomerID != "" {
			if customer, err := s.orderRepo.GetOrderCustomer(ctx, orders[i].CustomerID); err == nil {
				orders[i].Customer = customer
			}
		}
		if orders[i].ShippingAddressID != "" {
			if addr, err := s.orderRepo.GetOrderAddress(ctx, orders[i].ShippingAddressID); err == nil {
				orders[i].ShippingAddress = addr
			}
		}
	}
	return orders, int(count), nil
}

// GetOrderStats computes simple aggregates
func (s *OrderService) GetOrderStats(ctx context.Context) (map[string]interface{}, error) {
	if s.orderRepo == nil {
		return nil, fmt.Errorf("order repository unavailable")
	}
	// Fetch recent orders and compute simple counts by status and totals
	filter := &base.Filter{Page: 1, PageSize: 10000}
	orders, err := s.orderRepo.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to load orders: %w", err)
	}
	counts := map[string]int{}
	var revenue float64
	for _, o := range orders {
		if o == nil {
			continue
		}
		counts[string(o.Status)]++
		revenue += o.TotalAmount
	}
	return map[string]interface{}{
		"counts":        counts,
		"revenue_total": revenue,
		"order_count":   len(orders),
	}, nil
}

// ExportOrdersCSV returns CSV bytes and filename
func (s *OrderService) ExportOrdersCSV(ctx context.Context) ([]byte, string, error) {
	if s.orderRepo == nil {
		return nil, "", fmt.Errorf("order repository unavailable")
	}
	orders, err := s.orderRepo.Find(ctx, &base.Filter{Page: 1, PageSize: 100000})
	if err != nil {
		return nil, "", fmt.Errorf("failed to load orders: %w", err)
	}
	var b strings.Builder
	b.WriteString("order_id,user_id,status,payment_status,total_amount,created_at\n")
	for _, o := range orders {
		if o == nil {
			continue
		}
		b.WriteString(fmt.Sprintf("%s,%s,%s,%s,%.2f,%s\n", o.ID, o.UserID, string(o.Status), o.PaymentStatus, o.TotalAmount, o.CreatedAt.Format(time.RFC3339)))
	}
	filename := fmt.Sprintf("orders_export_%d.csv", time.Now().Unix())
	return []byte(b.String()), filename, nil
}

// CreateOrderFromCartWithShippingAtomic creates an order from cart with shipping charges atomically
// This method ensures that duplicate orders cannot be created for the same Razorpay order ID
func (s *OrderService) CreateOrderFromCartWithShippingAtomic(ctx context.Context, userID string, shippingAddress string, paymentMethod string, shippingCharges float64, razorpayOrderID string) (*models.Order, error) {
	if s.orderRepo == nil {
		return nil, fmt.Errorf("order repository unavailable")
	}

	// First check if order already exists for this Razorpay order ID to prevent duplicates
	if existingOrder, _ := s.GetOrderByRazorpayOrderID(ctx, razorpayOrderID); existingOrder != nil {
		return nil, fmt.Errorf("order already exists for razorpay order ID: %s", razorpayOrderID)
	}

	// Get user's cart
	cart, err := s.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	if len(cart.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// Calculate total amount from cart items + shipping - discount (no tax)
	cartTotal := cart.GetTotalAmount()
	discountedCartTotal := utils.ApplyDiscount(cartTotal, cart.DiscountAmount)
	totalAmount := utils.AddShippingCharges(discountedCartTotal, shippingCharges)

	// Create order
	order := models.NewOrderLegacy(userID, totalAmount, shippingAddress, paymentMethod, models.OrderStatusPending, "pending")
	order.ShippingCharge = shippingCharges
	order.RazorpayOrderID = razorpayOrderID
	// Add discount information
	order.CouponCode = cart.CouponCode
	order.DiscountAmount = cart.DiscountAmount

	// Create order with transaction safety
	if err := s.orderRepo.CreateOrderAtomic(ctx, order, cart.Items); err != nil {
		return nil, fmt.Errorf("failed to create order atomically: %w", err)
	}

	// Reduce coupon usage count if a coupon was applied
	if cart.CouponCode != "" {
		if err := s.ReduceCouponUsage(ctx, cart.CouponCode); err != nil {
			s.logger.Error("Failed to reduce coupon usage count after order creation",
				zap.String("order_id", order.ID),
				zap.String("coupon_code", cart.CouponCode),
				zap.Error(err))
			// Don't fail the order creation, but log the error
		}
	}

	s.logger.Info("Order created atomically",
		zap.String("order_id", order.ID),
		zap.String("user_id", userID),
		zap.String("razorpay_order_id", razorpayOrderID),
		zap.Float64("total_amount", totalAmount),
		zap.Float64("shipping_charges", shippingCharges),
		zap.String("coupon_code", cart.CouponCode))

	return order, nil
}

// UpdateOrderWithShipmentAtomic updates order with shipment details atomically
func (s *OrderService) UpdateOrderWithShipmentAtomic(ctx context.Context, orderID string, waybill string, trackingURL string, delhiveryStatus string) error {
	if s.orderRepo == nil {
		return fmt.Errorf("order repository unavailable")
	}

	return s.orderRepo.UpdateOrderShipmentAtomic(ctx, orderID, waybill, trackingURL, delhiveryStatus)
}

// ApplyShipmentSuccess records a successful shipment and advances the order to shipped + fulfilled.
// ClaimOrderForShipment atomically claims an order for a single shipment attempt (see the repository
// method). Returns true only to the one caller that wins the claim.
func (s *OrderService) ClaimOrderForShipment(ctx context.Context, orderID string) (bool, error) {
	if s.orderRepo == nil {
		return false, fmt.Errorf("order repository unavailable")
	}
	return s.orderRepo.ClaimOrderForShipment(ctx, orderID)
}

// ClaimOrderForShipmentAdmin is the admin/manual-retry claim (see the repository method).
func (s *OrderService) ClaimOrderForShipmentAdmin(ctx context.Context, orderID string) (bool, error) {
	if s.orderRepo == nil {
		return false, fmt.Errorf("order repository unavailable")
	}
	return s.orderRepo.ClaimOrderForShipmentAdmin(ctx, orderID)
}

func (s *OrderService) ApplyShipmentSuccess(ctx context.Context, orderID, waybill, trackingURL, delhiveryStatus string) error {
	if s.orderRepo == nil {
		return fmt.Errorf("order repository unavailable")
	}
	return s.orderRepo.ApplyShipmentSuccess(ctx, orderID, waybill, trackingURL, delhiveryStatus)
}

// MarkFulfillmentStatus records the post-payment fulfillment state (and last shipment error).
func (s *OrderService) MarkFulfillmentStatus(ctx context.Context, orderID, status, shipmentErr string) error {
	if s.orderRepo == nil {
		return fmt.Errorf("order repository unavailable")
	}
	return s.orderRepo.SetFulfillmentStatus(ctx, orderID, status, shipmentErr)
}

// DeductInventoryForOrder decrements stock for the order's items exactly once (idempotent).
// It is a no-op if inventory was already deducted for this order, so it is safe to call from
// both the verify path and the webhook/retry paths.
func (s *OrderService) DeductInventoryForOrder(ctx context.Context, orderID string) error {
	if s.orderRepo == nil {
		return fmt.Errorf("order repository unavailable")
	}

	order, items, err := s.orderRepo.GetOrderWithItems(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to load order for inventory deduction: %w", err)
	}
	if order == nil {
		return fmt.Errorf("order %s not found", orderID)
	}
	if order.InventoryDeducted {
		// Fast path — already deducted, nothing to do. (The authoritative guard is
		// the conditional claim inside the transaction below.)
		return nil
	}
	if len(items) == 0 {
		return fmt.Errorf("order %s has no items", orderID)
	}
	if s.productRepo == nil {
		return fmt.Errorf("product repository unavailable")
	}

	pg := s.orderRepo.GetPostgresManager()
	if pg == nil {
		return fmt.Errorf("postgres manager unavailable for inventory deduction")
	}

	// Deduct every item's stock AND flip the inventory_deducted guard in ONE
	// transaction, so a partial failure can't leave stock decremented with the
	// guard still false (which a retry would then double-count). The guard flip is
	// a conditional claim — only the first caller to set it from false proceeds to
	// decrement, so concurrent verify/webhook/admin calls can't double-deduct.
	var (
		claimed     bool
		oversoldIDs []string
	)
	txErr := pg.WithTransaction(ctx, func(tx *gorm.DB) error {
		claim := tx.WithContext(ctx).
			Model(&models.Order{}).
			Where("id = ? AND (inventory_deducted = ? OR inventory_deducted IS NULL)", orderID, false).
			Update("inventory_deducted", true)
		if claim.Error != nil {
			return fmt.Errorf("failed to claim inventory deduction: %w", claim.Error)
		}
		if claim.RowsAffected == 0 {
			// Another call already claimed and deducted — no-op.
			return nil
		}
		claimed = true
		for _, it := range items {
			oversold, derr := s.productRepo.DecrementStockAtomic(ctx, tx, it.ProductID, it.Quantity)
			if derr != nil {
				// Roll back the claim and every prior decrement in this tx.
				return derr
			}
			if oversold {
				oversoldIDs = append(oversoldIDs, it.ProductID)
			}
		}
		return nil
	})
	if txErr != nil {
		return fmt.Errorf("failed to deduct inventory: %w", txErr)
	}
	if !claimed {
		return nil
	}
	if len(oversoldIDs) > 0 {
		s.logger.Warn("Inventory oversold on paid order (stock floored to zero)",
			zap.String("order_id", orderID),
			zap.Strings("oversold_product_ids", oversoldIDs))
	}

	s.logger.Info("Inventory deducted for order", zap.String("order_id", orderID), zap.Int("item_count", len(items)))
	return nil
}

// ReduceCouponUsage reduces the usage count of a coupon when an order is successfully created
func (s *OrderService) ReduceCouponUsage(ctx context.Context, couponCode string) error {
	if couponCode == "" {
		return nil // No coupon applied, nothing to do
	}

	if s.couponRepo == nil {
		s.logger.Warn("Coupon repository not available, skipping coupon usage reduction",
			zap.String("coupon_code", couponCode))
		return nil
	}

	s.logger.Info("Reducing coupon usage count",
		zap.String("coupon_code", couponCode))

	if err := s.couponRepo.IncrementUsage(ctx, couponCode); err != nil {
		s.logger.Error("Failed to reduce coupon usage count",
			zap.String("coupon_code", couponCode),
			zap.Error(err))
		return fmt.Errorf("failed to reduce coupon usage count: %w", err)
	}

	s.logger.Info("Successfully reduced coupon usage count",
		zap.String("coupon_code", couponCode))

	return nil
}

// CreateOrder creates a new order using normalized customer and address data
func (s *OrderService) CreateOrder(ctx context.Context, userID string, customerData *models.OrderCustomer, addressData *models.OrderAddress, paymentMethod string, shippingCharges float64) (*models.Order, error) {
	s.logger.Info("Starting normalized order creation",
		zap.String("user_id", userID),
		zap.String("payment_method", paymentMethod),
		zap.Float64("shipping_charges", shippingCharges))

	// Get user's cart with items
	cart, err := s.cartService.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	if cart == nil {
		return nil, fmt.Errorf("no cart found for user %s", userID)
	}

	if len(cart.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// Calculate cart total amount including any discounts
	cartTotal := cart.GetTotalAmount()
	discountedCartTotal := utils.ApplyDiscount(cartTotal, cart.DiscountAmount)

	// Add shipping charges to get final total (no tax)
	totalAmount := utils.AddShippingCharges(discountedCartTotal, shippingCharges)

	// Create order customer record with retry logic for duplicate key errors
	if err := s.createOrderCustomerWithRetry(ctx, customerData); err != nil {
		return nil, fmt.Errorf("failed to create order customer: %w", err)
	}

	// Create order address record with retry logic for duplicate key errors
	if err := s.createOrderAddressWithRetry(ctx, addressData); err != nil {
		return nil, fmt.Errorf("failed to create order address: %w", err)
	}

	// Create order with normalized references
	order := models.NewOrder(userID, customerData.ID, addressData.ID, totalAmount, paymentMethod, models.OrderStatusPending, "pending")

	// Set shipping metadata
	order.ShippingCarrier = "Delhivery"
	order.ShippingMode = "Surface"
	order.ShippingCharge = shippingCharges

	// Set coupon information in dedicated fields
	if cart.CouponCode != "" {
		order.CouponCode = cart.CouponCode
		order.DiscountAmount = cart.DiscountAmount
	}

	// Add shipping to notes
	notes := ""
	if shippingCharges > 0 {
		notes += fmt.Sprintf("Shipping: %.2f", shippingCharges)
	}
	order.Notes = notes

	// Save order to database
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Create order items
	var orderItems []models.OrderItem
	for _, cartItem := range cart.Items {
		orderItem := models.NewOrderItem(order.ID, cartItem.ProductID, cartItem.Quantity, cartItem.UnitPrice)
		orderItems = append(orderItems, *orderItem)
	}

	// Save order items
	if err := s.createOrderItems(ctx, orderItems); err != nil {
		s.logger.Error("Failed to create order items", zap.Error(err), zap.String("order_id", order.ID))
		// Don't fail the order creation, but log the error
	}

	// Clear user cart after successful order creation
	if err := s.ClearUserCart(ctx, userID); err != nil {
		s.logger.Error("Failed to clear user cart after order creation", zap.Error(err), zap.String("user_id", userID))
		// Don't fail the order creation, but log the error
	}

	s.logger.Info("Normalized order created successfully",
		zap.String("order_id", order.ID),
		zap.String("user_id", userID),
		zap.String("customer_id", customerData.ID),
		zap.String("address_id", addressData.ID),
		zap.Float64("total_amount", totalAmount),
		zap.Int("item_count", len(orderItems)))

	return order, nil
}

// GetOrderDetails retrieves an order with customer, address, and items data including product information
// Returns order and enriched items separately so response builder can use all image URLs
func (s *OrderService) GetOrderDetails(ctx context.Context, orderID string) (*models.Order, []repositories.OrderItemWithProduct, error) {
	// Get order with enhanced product information
	order, enrichedItems, err := s.orderRepo.GetOrderWithItemsAndProducts(ctx, orderID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get order details: %w", err)
	}

	// Get customer data if customer_id exists
	if order.CustomerID != "" {
		customer, err := s.orderRepo.GetOrderCustomer(ctx, order.CustomerID)
		if err != nil {
			s.logger.Warn("Failed to get order customer", zap.String("order_id", orderID), zap.Error(err))
		} else {
			order.Customer = customer
		}
	}

	// Get shipping address data if shipping_address_id exists
	if order.ShippingAddressID != "" {
		address, err := s.orderRepo.GetOrderAddress(ctx, order.ShippingAddressID)
		if err != nil {
			s.logger.Warn("Failed to get order address", zap.String("order_id", orderID), zap.Error(err))
		} else {
			order.ShippingAddress = address
		}
	}

	// Convert enriched items back to regular order items for basic order structure
	var orderItems []models.OrderItem
	for _, enrichedItem := range enrichedItems {
		orderItem := enrichedItem.OrderItem
		if enrichedItem.ProductName != "" || enrichedItem.ProductImageURL != "" {
			product := &models.Product{
				Name:     enrichedItem.ProductName,
				ImageURL: enrichedItem.ProductImageURL, // Primary image
			}
			orderItem.Product = product
		}
		orderItems = append(orderItems, orderItem)
	}

	// Set the enriched order items
	order.OrderItems = orderItems

	// Return enriched items separately so response builder can access all image URLs
	return order, enrichedItems, nil
}

// createOrderCustomerWithRetry creates an order customer with retry logic for duplicate key errors
func (s *OrderService) createOrderCustomerWithRetry(ctx context.Context, customerData *models.OrderCustomer) error {
	maxRetries := 3
	var createErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		createErr = s.orderRepo.CreateOrderCustomer(ctx, customerData)
		if createErr == nil {
			// Successfully created customer, break out of retry loop
			return nil
		}

		// Check if it's a duplicate key error
		errorStr := createErr.Error()
		if strings.Contains(errorStr, "duplicate key value violates unique constraint") &&
			strings.Contains(errorStr, "order_customers_pkey") {
			// Generate a new ID and try again
			s.logger.Warn("Duplicate key error detected for order customer, retrying with new ID",
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries),
				zap.String("old_id", customerData.ID),
				zap.Error(createErr))

			// Create a new customer with a fresh ID
			newCustomer := models.NewOrderCustomer(
				customerData.FirstName,
				customerData.LastName,
				customerData.Phone,
				customerData.PhoneAlt,
				customerData.Email,
			)
			// Copy the new ID to the original customer data
			customerData.ID = newCustomer.ID
			customerData.BaseModel = newCustomer.BaseModel

			// Add a small delay before retry to reduce contention
			time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
			continue
		}

		// If it's not a duplicate key error, return the error immediately
		return createErr
	}

	// If we still have an error after all retries, return it
	return fmt.Errorf("failed to create order customer after %d attempts: %w", maxRetries, createErr)
}

// createOrderAddressWithRetry creates an order address with retry logic for duplicate key errors
func (s *OrderService) createOrderAddressWithRetry(ctx context.Context, addressData *models.OrderAddress) error {
	maxRetries := 3
	var createErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		createErr = s.orderRepo.CreateOrderAddress(ctx, addressData)
		if createErr == nil {
			// Successfully created address, break out of retry loop
			return nil
		}

		// Check if it's a duplicate key error
		errorStr := createErr.Error()
		if strings.Contains(errorStr, "duplicate key value violates unique constraint") &&
			strings.Contains(errorStr, "order_addresses_pkey") {
			// Generate a new ID and try again
			s.logger.Warn("Duplicate key error detected for order address, retrying with new ID",
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries),
				zap.String("old_id", addressData.ID),
				zap.Error(createErr))

			// Create a new address with a fresh ID
			newAddress := models.NewOrderAddress(
				addressData.AddressLine1,
				addressData.AddressLine2,
				addressData.City,
				addressData.State,
				addressData.PostalCode,
				addressData.Country,
			)
			// Copy the new ID to the original address data
			addressData.ID = newAddress.ID
			addressData.BaseModel = newAddress.BaseModel

			// Add a small delay before retry to reduce contention
			time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
			continue
		}

		// If it's not a duplicate key error, return the error immediately
		return createErr
	}

	// If we still have an error after all retries, return it
	return fmt.Errorf("failed to create order address after %d attempts: %w", maxRetries, createErr)
}
