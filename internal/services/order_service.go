package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"go.uber.org/zap"
)

// OrderServiceInterface defines the contract for order operations
type OrderServiceInterface interface {
	CreateOrderFromCart(ctx context.Context, userID string, cartID string, shippingAddress string, paymentMethod string) (*models.Order, error)
	CreateOrderFromCartWithShipping(ctx context.Context, userID string, shippingAddress string, paymentMethod string, shippingCharges float64) (*models.Order, error)
	GetUserCart(ctx context.Context, userID string) (*models.Cart, error)
	GetOrderByID(ctx context.Context, orderID string) (*models.Order, error)
	GetUserOrders(ctx context.Context, userID string) ([]*models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status models.OrderStatus) error
	UpdatePaymentStatus(ctx context.Context, orderID string, paymentStatus, paymentID string) error
	GetOrderWithItems(ctx context.Context, orderID string) (*models.Order, []models.OrderItem, error)
	ConfirmOrderPayment(ctx context.Context, razorpayOrderID, paymentID string) error
	UpdateOrderNotes(ctx context.Context, orderID string, notes string) error
	ClearUserCart(ctx context.Context, userID string) error
	UpdateOrder(ctx context.Context, order *models.Order) error
	GenerateAndAttachInvoice(ctx context.Context, orderID string) (string, error)
	// Address helpers
	GetAddressByID(ctx context.Context, userID, addressID string) (*models.Address, error)
	FormatAddress(addr *models.Address) string
}

// OrderService implements order business logic
type OrderService struct {
	orderRepo   *repositories.OrderRepository
	cartService CartServiceInterface
	productRepo repositories.ProductRepositoryInterface
	addressRepo *repositories.AddressRepository
	s3Service   *S3Service
	logger      *zap.Logger
}

// NewOrderService creates a new order service
func NewOrderService(orderRepo *repositories.OrderRepository, cartService CartServiceInterface, productRepo repositories.ProductRepositoryInterface, addressRepo *repositories.AddressRepository, s3Service *S3Service, logger *zap.Logger) OrderServiceInterface {
	return &OrderService{
		orderRepo:   orderRepo,
		cartService: cartService,
		productRepo: productRepo,
		addressRepo: addressRepo,
		s3Service:   s3Service,
		logger:      logger,
	}
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

	// Calculate total amount including any discounts
	totalAmount := cart.GetTotalAmount()
	if cart.DiscountAmount > 0 {
		totalAmount -= cart.DiscountAmount
		if totalAmount < 0 {
			totalAmount = 0
		}
	}

	// Create order
	order := models.NewOrder(userID, totalAmount, shippingAddress, paymentMethod)
	order.PaymentStatus = "pending"
	order.Status = models.OrderStatusPending

	// Add coupon information to notes if applied
	if cart.CouponCode != "" {
		order.Notes = fmt.Sprintf("Coupon: %s, Discount: %.2f", cart.CouponCode, cart.DiscountAmount)
	}

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

	// Calculate cart total amount including any discounts
	cartTotal := cart.GetTotalAmount()
	if cart.DiscountAmount > 0 {
		cartTotal -= cart.DiscountAmount
		if cartTotal < 0 {
			cartTotal = 0
		}
	}

	// Add shipping charges to get final total
	totalAmount := cartTotal + shippingCharges

	s.logger.Info("Calculated amounts",
		zap.String("user_id", userID),
		zap.Float64("cart_total", cartTotal),
		zap.Float64("discount_amount", cart.DiscountAmount),
		zap.Float64("shipping_charges", shippingCharges),
		zap.Float64("total_amount", totalAmount))

	// Create order with total amount including shipping
	order := models.NewOrder(userID, totalAmount, shippingAddress, paymentMethod)
	order.PaymentStatus = "pending"
	order.Status = models.OrderStatusPending
	// Set shipping metadata
	order.ShippingCarrier = "Delhivery"
	order.ShippingMode = "Surface"
	order.ShippingCharge = shippingCharges

	// Add coupon information and shipping to notes if applicable
	notes := ""
	if cart.CouponCode != "" {
		notes += fmt.Sprintf("Coupon: %s, Discount: %.2f", cart.CouponCode, cart.DiscountAmount)
	}
	if shippingCharges > 0 {
		if notes != "" {
			notes += ", "
		}
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

// updateInventoryForOrderItems decrements stock for each ordered product
func (s *OrderService) updateInventoryForOrderItems(ctx context.Context, items []models.OrderItem) error {
	if s.productRepo == nil {
		return fmt.Errorf("product repository unavailable")
	}
	var firstErr error
	for _, it := range items {
		var prod models.Product
		res, err := s.productRepo.GetByID(ctx, it.ProductID, &prod)
		if err != nil || res == nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("product %s not found: %w", it.ProductID, err)
			}
			continue
		}
		// Decrement stock but not below zero
		newStock := prod.Stock - it.Quantity
		if newStock < 0 {
			newStock = 0
		}
		prod.Stock = newStock
		if newStock <= 0 {
			prod.StockStatus = "Out of Stock"
		}
		if err := s.productRepo.Update(ctx, &prod); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to update stock for %s: %w", it.ProductID, err)
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

// GenerateAndAttachInvoice builds a simple JSON invoice, uploads to S3, and stores URL on the order
func (s *OrderService) GenerateAndAttachInvoice(ctx context.Context, orderID string) (string, error) {
	if s.s3Service == nil {
		return "", fmt.Errorf("s3 service unavailable")
	}
	order, items, err := s.orderRepo.GetOrderWithItems(ctx, orderID)
	if err != nil {
		return "", fmt.Errorf("failed to load order for invoice: %w", err)
	}
	type invoiceItem struct {
		ProductID string  `json:"product_id"`
		Quantity  int     `json:"quantity"`
		UnitPrice float64 `json:"unit_price"`
		LineTotal float64 `json:"line_total"`
	}
	inv := map[string]interface{}{
		"order_id":         order.ID,
		"user_id":          order.UserID,
		"date":             time.Now().Format(time.RFC3339),
		"shipping_address": order.ShippingAddr,
		"shipping_carrier": order.ShippingCarrier,
		"shipping_mode":    order.ShippingMode,
		"shipping_charge":  order.ShippingCharge,
		"eta_min_days":     order.EtaMinDays,
		"eta_max_days":     order.EtaMaxDays,
		"payment_method":   order.PaymentMethod,
		"payment_status":   order.PaymentStatus,
		"total_amount":     order.TotalAmount,
		"items":            []invoiceItem{},
	}
	arr := make([]invoiceItem, 0, len(items))
	for _, it := range items {
		arr = append(arr, invoiceItem{ProductID: it.ProductID, Quantity: it.Quantity, UnitPrice: it.UnitPrice, LineTotal: it.TotalPrice})
	}
	inv["items"] = arr
	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal invoice: %w", err)
	}
	key := fmt.Sprintf("invoices/%s.json", order.ID)
	url, err := s.s3Service.UploadFile(ctx, key, data, "application/json")
	if err != nil {
		return "", fmt.Errorf("failed to upload invoice: %w", err)
	}
	order.InvoiceURL = url
	_ = s.orderRepo.Update(ctx, order)
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

// GetUserOrders retrieves all orders for a specific user
func (s *OrderService) GetUserOrders(ctx context.Context, userID string) ([]*models.Order, error) {
	return s.orderRepo.GetOrdersByUserID(ctx, userID)
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
