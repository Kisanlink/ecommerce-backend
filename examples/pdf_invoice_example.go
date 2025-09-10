package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Kisanlink/ecommerce-backend/internal/services"
)

// Example of how to use the PDF invoice system
func main() {
	// Example invoice data
	invoiceData := &services.InvoiceData{
		OrderID:         "ORDR0000000001",
		UserID:          "USER123",
		Date:            time.Now(),
		ShippingAddress: "John Doe, 123 Main St, City, State 12345, Country",
		ShippingCarrier: "Delhivery",
		ShippingMode:    "Surface",
		ShippingCharge:  50.0,
		EtaMinDays:      3,
		EtaMaxDays:      5,
		PaymentMethod:   "razorpay",
		PaymentStatus:   "successful",
		TotalAmount:     150.0,
		CustomerName:    "John Doe",
		CustomerPhone:   "+91-9876543210",
		Items: []services.InvoiceItem{
			{
				ProductID:   "PROD001",
				Quantity:    2,
				UnitPrice:   25.0,
				LineTotal:   50.0,
				ProductName: "Sample Product 1",
			},
			{
				ProductID:   "PROD002",
				Quantity:    1,
				UnitPrice:   50.0,
				LineTotal:   50.0,
				ProductName: "Sample Product 2",
			},
		},
	}

	// Create template service
	templateService := services.NewPDFTemplateService()

	// Example 1: Default template
	fmt.Println("Generating PDF with default template...")
	pdfBytes, err := templateService.GenerateCustomPDFInvoice(invoiceData, nil)
	if err != nil {
		log.Fatalf("Failed to generate PDF: %v", err)
	}
	fmt.Printf("Generated PDF with %d bytes\n", len(pdfBytes))

	// Example 2: Custom template with company details
	fmt.Println("Generating PDF with custom template...")
	customConfig := &services.InvoiceTemplateConfig{
		CompanyName:    "Amrti E-commerce Pvt Ltd",
		CompanyAddress: "123 Business Park, Tech City, Bangalore, Karnataka 560001, India",
		CompanyPhone:   "+91-80-12345678",
		CompanyEmail:   "billing@amrti.com",
		Currency:       "₹",
		ShowTax:        true,
		TaxRate:        18.0, // 18% GST
		ShowDiscount:   true,
		DiscountAmount: 10.0,
	}

	pdfBytesCustom, err := templateService.GenerateCustomPDFInvoice(invoiceData, customConfig)
	if err != nil {
		log.Fatalf("Failed to generate custom PDF: %v", err)
	}
	fmt.Printf("Generated custom PDF with %d bytes\n", len(pdfBytesCustom))

	// Example 3: US Dollar template
	fmt.Println("Generating PDF with USD template...")
	usdConfig := &services.InvoiceTemplateConfig{
		CompanyName:    "Amrti E-commerce Inc",
		CompanyAddress: "456 Commerce St, New York, NY 10001, USA",
		CompanyPhone:   "+1-555-123-4567",
		CompanyEmail:   "billing@amrti.com",
		Currency:       "$",
		ShowTax:        true,
		TaxRate:        8.25, // 8.25% sales tax
		ShowDiscount:   false,
		DiscountAmount: 0.0,
	}

	pdfBytesUSD, err := templateService.GenerateCustomPDFInvoice(invoiceData, usdConfig)
	if err != nil {
		log.Fatalf("Failed to generate USD PDF: %v", err)
	}
	fmt.Printf("Generated USD PDF with %d bytes\n", len(pdfBytesUSD))

	fmt.Println("All PDF examples generated successfully!")
	fmt.Println("You can save these bytes to files to view the PDFs:")
	fmt.Println("- default_invoice.pdf")
	fmt.Println("- custom_invoice.pdf")
	fmt.Println("- usd_invoice.pdf")
}
