package services

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"go.uber.org/zap"
)

// PDFInvoiceService handles PDF invoice generation
type PDFInvoiceService struct {
	r2Service       *R2Service // Cloudflare R2 storage
	templateService *PDFTemplateService
	logger          *zap.Logger
}

// NewPDFInvoiceService creates a new PDF invoice service with R2 storage
func NewPDFInvoiceService(r2Service *R2Service, logger *zap.Logger) *PDFInvoiceService {
	return &PDFInvoiceService{
		r2Service:       r2Service,
		templateService: NewPDFTemplateService(),
		logger:          logger,
	}
}

// Note: InvoiceData struct is now defined in pdf_template_service.go

// Note: InvoiceItem struct is now defined in pdf_template_service.go

// GeneratePDFInvoice creates a PDF invoice and uploads it to R2
func (s *PDFInvoiceService) GeneratePDFInvoice(ctx context.Context, order *models.Order, items []*models.OrderItem, productRepo ProductRepositoryInterface) (string, error) {
	// Check if R2 storage service is available
	if s.r2Service == nil || !s.r2Service.IsConfigured() {
		s.logger.Error("R2 storage service not available", zap.String("order_id", order.ID))
		return "", fmt.Errorf("storage service unavailable")
	}

	s.logger.Info("Starting PDF invoice generation",
		zap.String("order_id", order.ID),
		zap.Int("items_count", len(items)))

	// Prepare invoice data
	invoiceData := s.prepareInvoiceData(order, items, productRepo, ctx)

	// Generate PDF using template service
	pdfBytes, err := s.templateService.GenerateCustomPDFInvoice(invoiceData, nil)
	if err != nil {
		s.logger.Error("Failed to generate PDF content",
			zap.String("order_id", order.ID),
			zap.Error(err))
		return "", fmt.Errorf("failed to create PDF invoice: %w", err)
	}

	s.logger.Info("PDF content generated successfully",
		zap.String("order_id", order.ID),
		zap.Int("pdf_size_bytes", len(pdfBytes)))

	// Upload to R2
	key := fmt.Sprintf("invoices/%s.pdf", order.ID)
	url, err := s.r2Service.UploadFile(ctx, key, pdfBytes, "application/pdf")
	if err != nil {
		s.logger.Error("Failed to upload PDF to R2",
			zap.String("order_id", order.ID),
			zap.String("r2_key", key),
			zap.Error(err))
		return "", fmt.Errorf("failed to upload PDF invoice to R2: %w", err)
	}
	s.logger.Info("PDF invoice uploaded to R2 successfully",
		zap.String("order_id", order.ID),
		zap.String("r2_key", key),
		zap.String("invoice_url", url))

	return url, nil
}

// prepareInvoiceData prepares the data for PDF generation
func (s *PDFInvoiceService) prepareInvoiceData(order *models.Order, items []*models.OrderItem, productRepo ProductRepositoryInterface, ctx context.Context) *InvoiceData {
	invoiceItems := make([]InvoiceItem, 0, len(items))

	// Calculate subtotal from items
	subtotal := 0.0
	for _, item := range items {
		subtotal += item.TotalPrice

		var product models.Product
		productName := "Unknown Product"
		if _, err := productRepo.GetByID(ctx, item.ProductID, &product); err == nil {
			productName = product.Name
		}

		invoiceItems = append(invoiceItems, InvoiceItem{
			ProductID:   item.ProductID,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			LineTotal:   item.TotalPrice,
			ProductName: productName,
		})
	}

	// Calculate discount amount and discounted total
	discountAmount := order.DiscountAmount
	discountedTotal := order.TotalAmount

	return &InvoiceData{
		OrderID:         order.ID,
		UserID:          order.UserID,
		Date:            order.CreatedAt,
		DueDate:         order.CreatedAt.AddDate(0, 0, 30), // 30 days from order date
		ShippingAddress: order.ShippingAddr,
		ShippingCarrier: order.ShippingCarrier,
		ShippingMode:    order.ShippingMode,
		ShippingCharge:  order.ShippingCharge,
		EtaMinDays:      order.EtaMinDays,
		EtaMaxDays:      order.EtaMaxDays,
		PaymentMethod:   order.PaymentMethod,
		PaymentStatus:   order.PaymentStatus,
		TotalAmount:     order.TotalAmount,
		Items:           invoiceItems,
		CustomerName:    fmt.Sprintf("%s %s", order.CustomerFirstName, order.CustomerLastName),
		CustomerPhone:   order.CustomerPhone,
		TrackingURL:     order.TrackingURL,
		AWBNumber:       order.DelhiveryWaybill,
		// Discount information
		CouponCode:      order.CouponCode,
		DiscountAmount:  discountAmount,
		SubtotalAmount:  subtotal,
		DiscountedTotal: discountedTotal,
	}
}

// ProductRepositoryInterface defines the interface for product repository
type ProductRepositoryInterface interface {
	GetByID(ctx context.Context, id string, product *models.Product) (*models.Product, error)
}
