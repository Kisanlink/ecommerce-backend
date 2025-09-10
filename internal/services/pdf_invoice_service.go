package services

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"go.uber.org/zap"
)

// PDFInvoiceService handles PDF invoice generation
type PDFInvoiceService struct {
	s3Service       *S3Service
	templateService *PDFTemplateService
	logger          *zap.Logger
}

// NewPDFInvoiceService creates a new PDF invoice service
func NewPDFInvoiceService(s3Service *S3Service, logger *zap.Logger) *PDFInvoiceService {
	return &PDFInvoiceService{
		s3Service:       s3Service,
		templateService: NewPDFTemplateService(),
		logger:          logger,
	}
}

// Note: InvoiceData struct is now defined in pdf_template_service.go

// Note: InvoiceItem struct is now defined in pdf_template_service.go

// GeneratePDFInvoice creates a PDF invoice and uploads it to S3
func (s *PDFInvoiceService) GeneratePDFInvoice(ctx context.Context, order *models.Order, items []*models.OrderItem, productRepo ProductRepositoryInterface) (string, error) {
	if s.s3Service == nil {
		return "", fmt.Errorf("s3 service unavailable")
	}

	// Prepare invoice data
	invoiceData := s.prepareInvoiceData(order, items, productRepo, ctx)

	// Generate PDF using template service
	pdfBytes, err := s.templateService.GenerateCustomPDFInvoice(invoiceData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create PDF invoice: %w", err)
	}

	// Upload to S3
	key := fmt.Sprintf("invoices/%s.pdf", order.ID)
	url, err := s.s3Service.UploadFile(ctx, key, pdfBytes, "application/pdf")
	if err != nil {
		return "", fmt.Errorf("failed to upload PDF invoice: %w", err)
	}

	s.logger.Info("PDF invoice generated successfully",
		zap.String("order_id", order.ID),
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
