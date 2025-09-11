package services

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// PDFTemplateService provides customizable PDF templates
type PDFTemplateService struct{}

// NewPDFTemplateService creates a new PDF template service
func NewPDFTemplateService() *PDFTemplateService {
	return &PDFTemplateService{}
}

// InvoiceTemplateConfig allows customization of the invoice template
type InvoiceTemplateConfig struct {
	CompanyName    string
	CompanyAddress string
	CompanyPhone   string
	CompanyEmail   string
	CompanyWebsite string
	CompanyPAN     string
	CompanyGST     string
	LogoURL        string // Optional logo URL
	Currency       string // Default: ₹
	ShowTax        bool
	TaxRate        float64
	ShowDiscount   bool
	DiscountAmount float64
	ShowCGST       bool
	ShowSGST       bool
	CGSTRate       float64
	SGSTRate       float64
	// New styling options
	PrimaryColor    [3]int // RGB values for primary color
	SecondaryColor  [3]int // RGB values for secondary color
	AccentColor     [3]int // RGB values for accent color
	ShowWatermark   bool
	WatermarkText   string
	HeaderStyle     string // "modern", "classic", "minimal"
	ShowQRCode      bool
	QRCodeData      string
	FooterMessage   string
	ShowBankDetails bool
	BankName        string
	AccountNumber   string
	IFSCCode        string
	BranchName      string
}

// InvoiceData represents the data needed for invoice generation
type InvoiceData struct {
	OrderID         string
	UserID          string
	Date            time.Time
	DueDate         time.Time
	ShippingAddress string
	BillingAddress  string
	ShippingCarrier string
	ShippingMode    string
	ShippingCharge  float64
	EtaMinDays      int
	EtaMaxDays      int
	PaymentMethod   string
	PaymentStatus   string
	TotalAmount     float64
	Items           []InvoiceItem
	CustomerName    string
	CustomerPhone   string
	CustomerEmail   string
	TrackingURL     string
	AWBNumber       string
	Notes           string
	Terms           []string
	// Discount information
	CouponCode      string
	DiscountAmount  float64
	SubtotalAmount  float64
	DiscountedTotal float64
}

// InvoiceItem represents an item in the invoice
type InvoiceItem struct {
	ProductID   string
	ProductName string
	ProductCode string
	Description string
	HSNCode     string
	Quantity    int
	UnitPrice   float64
	Discount    float64
	LineTotal   float64
	TaxRate     float64
	Category    string
}

// GenerateCustomPDFInvoice creates a PDF invoice with professional template
func (s *PDFTemplateService) GenerateCustomPDFInvoice(data *InvoiceData, config *InvoiceTemplateConfig) ([]byte, error) {
	if config == nil {
		config = s.getDefaultConfig()
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Set margins and auto page break
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 25)

	// Add watermark if enabled
	if config.ShowWatermark {
		s.addWatermark(pdf, config)
	}

	// Header Section with modern styling
	s.addModernHeader(pdf, config, data)

	// Company and Customer Details in two columns
	s.addModernCompanyAndCustomerDetails(pdf, config, data)

	// Invoice metadata section
	s.addInvoiceMetadata(pdf, data)

	// Items Table with enhanced styling
	s.addModernItemsTable(pdf, data, config)

	// Summary and totals with modern cards
	s.addModernTotalsSection(pdf, data, config)

	// Terms and conditions
	if len(data.Terms) > 0 {
		s.addTermsAndConditions(pdf, data.Terms)
	}

	// Bank details removed as requested

	// Modern footer
	s.addModernFooter(pdf, config, data)

	// Generate PDF bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// addWatermark adds a watermark to the PDF
func (s *PDFTemplateService) addWatermark(pdf *gofpdf.Fpdf, config *InvoiceTemplateConfig) {
	pdf.SetTextColor(240, 240, 240) // Very light gray
	pdf.SetFont("Arial", "B", 50)

	// Rotate and position watermark
	pdf.TransformBegin()
	pdf.TransformRotate(45, 105, 148) // Rotate 45 degrees at center of page
	pdf.Text(60, 148, config.WatermarkText)
	pdf.TransformEnd()
}

// addModernHeader creates a modern header with better visual hierarchy
func (s *PDFTemplateService) addModernHeader(pdf *gofpdf.Fpdf, config *InvoiceTemplateConfig, data *InvoiceData) {
	// Header background - white for better logo blending
	pdf.SetFillColor(255, 255, 255)
	pdf.Rect(12, 12, 186, 35, "F")

	// Company logo section
	logoPath := "public/logo2.png"
	if _, err := os.Stat(logoPath); err == nil {
		s.addPNGLogo(pdf, logoPath, config)
	} else {
		s.addModernTextLogo(pdf, config)
	}

	// Invoice title with modern styling
	pdf.SetXY(120, 18)
	pdf.SetFont("Arial", "B", 24)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 12, "INVOICE")

	pdf.SetXY(120, 30)
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(64, 64, 64)
	pdf.Cell(0, 6, fmt.Sprintf("INV-%s", data.OrderID))

	// Status badge
	s.addStatusBadge(pdf, data.PaymentStatus, 120, 38)

	pdf.SetY(55) // Reset Y position after header
}

// addPNGLogo adds the PNG logo to the PDF
func (s *PDFTemplateService) addPNGLogo(pdf *gofpdf.Fpdf, logoPath string, config *InvoiceTemplateConfig) {
	// Register the PNG image
	imageInfo := pdf.RegisterImage(logoPath, "")
	if imageInfo == nil {
		// If image registration fails, fallback to text logo
		s.addModernTextLogo(pdf, config)
		return
	}

	// Get image dimensions
	imgWidth, imgHeight := imageInfo.Extent()

	// Calculate logo size (max width 60mm, maintain aspect ratio)
	maxWidth := 60.0
	scale := maxWidth / imgWidth
	logoWidth := imgWidth * scale
	logoHeight := imgHeight * scale

	// Ensure logo doesn't exceed header height
	if logoHeight > 25 {
		scale = 25.0 / imgHeight
		logoWidth = imgWidth * scale
		logoHeight = imgHeight * scale
	}

	// Position logo in header (centered vertically)
	logoX := 18.0
	logoY := 18.0 + (25.0-logoHeight)/2.0

	// Place the PNG logo directly (no background needed with white header)
	pdf.Image(logoPath, logoX, logoY, logoWidth, logoHeight, false, "", 0, "")
}

// addModernLogo adds a modern styled logo (legacy function for SVG)
func (s *PDFTemplateService) addModernLogo(pdf *gofpdf.Fpdf, logoPath string, config *InvoiceTemplateConfig) {
	// Modern logo container
	pdf.SetFillColor(255, 255, 255)
	pdf.RoundedRect(18, 18, 80, 25, 4, "1111", "F")

	// Company name with modern typography
	pdf.SetXY(22, 25)
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(config.PrimaryColor[0], config.PrimaryColor[1], config.PrimaryColor[2])
	pdf.Cell(0, 8, "AMRTI")

	pdf.SetXY(22, 32)
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 4, "E-COMMERCE PLATFORM")
}

// addModernTextLogo creates a modern text-based logo
func (s *PDFTemplateService) addModernTextLogo(pdf *gofpdf.Fpdf, config *InvoiceTemplateConfig) {
	pdf.SetXY(18, 22)
	pdf.SetFont("Arial", "B", 22)
	pdf.SetTextColor(255, 255, 255)
	pdf.Cell(0, 10, "AMRTI")

	pdf.SetXY(18, 32)
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(255, 255, 255)
	pdf.Cell(0, 5, "E-COMMERCE PLATFORM")
}

// addStatusBadge adds a status badge for payment status
func (s *PDFTemplateService) addStatusBadge(pdf *gofpdf.Fpdf, status string, x, y float64) {
	var bgColor [3]int
	switch strings.ToLower(status) {
	case "paid", "completed":
		bgColor = [3]int{34, 197, 94} // Green
	case "pending":
		bgColor = [3]int{251, 191, 36} // Yellow
	case "overdue", "failed":
		bgColor = [3]int{239, 68, 68} // Red
	default:
		bgColor = [3]int{107, 114, 128} // Gray
	}

	pdf.SetFillColor(bgColor[0], bgColor[1], bgColor[2])
	pdf.RoundedRect(x, y, 35, 8, 2, "1111", "F")

	pdf.SetXY(x+2, y+1)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(255, 255, 255)
	pdf.Cell(31, 6, strings.ToUpper(status))
}

// addModernCompanyAndCustomerDetails creates a two-column layout
func (s *PDFTemplateService) addModernCompanyAndCustomerDetails(pdf *gofpdf.Fpdf, config *InvoiceTemplateConfig, data *InvoiceData) {
	currentY := pdf.GetY() + 3

	// Company details card (left column) - reduced height
	s.addInfoCard(pdf, 12, currentY, 88, "SOLD BY", []string{
		config.CompanyName,
		config.CompanyAddress,
		fmt.Sprintf("Phone: %s", config.CompanyPhone),
		fmt.Sprintf("Email: %s", config.CompanyEmail),
		fmt.Sprintf("PAN: %s", config.CompanyPAN),
		fmt.Sprintf("GST: %s", config.CompanyGST),
	}, config)

	// Customer details card (right column) - reduced height
	customerDetails := []string{
		data.CustomerName,
		data.ShippingAddress,
		fmt.Sprintf("Phone: %s", data.CustomerPhone),
	}

	s.addInfoCard(pdf, 110, currentY, 88, "BILL TO", customerDetails, config)

	pdf.SetY(currentY + 55) // Move below both cards - reduced spacing
}

// addInfoCard creates a modern information card
func (s *PDFTemplateService) addInfoCard(pdf *gofpdf.Fpdf, x, y, width float64, title string, details []string, config *InvoiceTemplateConfig) {
	// Card background with subtle shadow effect - reduced height
	pdf.SetFillColor(248, 249, 250)
	pdf.SetDrawColor(229, 231, 235)
	pdf.RoundedRect(x, y, width, 50, 3, "1111", "FD")

	// Card header
	pdf.SetFillColor(config.SecondaryColor[0], config.SecondaryColor[1], config.SecondaryColor[2])
	pdf.RoundedRect(x, y, width, 10, 3, "1100", "F")

	pdf.SetXY(x+5, y+1)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(255, 255, 255)
	pdf.Cell(0, 8, title)

	// Card content
	pdf.SetTextColor(0, 0, 0)
	yOffset := y + 12
	for i, detail := range details {
		if strings.TrimSpace(detail) == "" {
			continue
		}

		pdf.SetXY(x+5, yOffset)
		if i == 0 {
			pdf.SetFont("Arial", "B", 9)
		} else {
			pdf.SetFont("Arial", "", 8)
		}

		// Handle long text with word wrapping
		if len(detail) > 35 {
			lines := s.wrapText(detail, 35)
			for j, line := range lines {
				if j > 0 {
					yOffset += 3
					pdf.SetXY(x+5, yOffset)
				}
				pdf.Cell(0, 3, line)
			}
		} else {
			pdf.Cell(0, 3, detail)
		}
		yOffset += 3
	}
}

// wrapText wraps text to specified width
func (s *PDFTemplateService) wrapText(text string, width int) []string {
	words := strings.Fields(text)
	var lines []string
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// addInvoiceMetadata adds invoice metadata in a modern layout
func (s *PDFTemplateService) addInvoiceMetadata(pdf *gofpdf.Fpdf, data *InvoiceData) {
	currentY := pdf.GetY() + 3

	// Metadata container - reduced height
	pdf.SetFillColor(255, 255, 255)
	pdf.SetDrawColor(229, 231, 235)
	pdf.Rect(12, currentY, 186, 20, "D")

	// Metadata items in grid layout - simplified
	metadata := []struct {
		Label string
		Value string
	}{
		{"Invoice Number", fmt.Sprintf("INV-%s", data.OrderID)},
		{"Invoice Date", data.Date.Format("02 Jan 2006")},
		{"Payment Method", data.PaymentMethod},
		{"Shipping Mode", data.ShippingMode},
	}

	pdf.SetFont("Arial", "", 7)
	for i, item := range metadata {
		x := 16 + float64(i%2)*93
		y := currentY + 3 + float64(i/2)*7

		pdf.SetXY(x, y)
		pdf.SetTextColor(107, 114, 128)
		pdf.Cell(0, 3, item.Label+":")

		pdf.SetXY(x, y+3)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Arial", "B", 7)
		pdf.Cell(0, 3, item.Value)
		pdf.SetFont("Arial", "", 7)
	}

	pdf.SetY(currentY + 25)
}

// addModernItemsTable creates a modern items table
func (s *PDFTemplateService) addModernItemsTable(pdf *gofpdf.Fpdf, data *InvoiceData, config *InvoiceTemplateConfig) {
	// Table title - reduced spacing
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 8, "ITEMS & SERVICES")
	pdf.Ln(8)

	// Table header with modern styling
	pdf.SetFillColor(config.PrimaryColor[0], config.PrimaryColor[1], config.PrimaryColor[2])
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)

	// Define column widths for better alignment - simplified columns
	colWidths := []float64{15, 100, 20, 20, 25}
	headers := []string{"#", "Item Description", "Qty", "Rate", "Amount"}

	for i, header := range headers {
		pdf.CellFormat(colWidths[i], 8, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Table rows with alternating colors and better formatting
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 7)

	for i, item := range data.Items {
		// Alternate row colors
		if i%2 == 0 {
			pdf.SetFillColor(248, 249, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		// Row data - simplified
		rowData := []string{
			strconv.Itoa(i + 1),
			s.truncateText(item.ProductName, 40),
			strconv.Itoa(item.Quantity),
			fmt.Sprintf("%.2f", item.UnitPrice),
			fmt.Sprintf("%.2f", item.LineTotal),
		}

		for j, data := range rowData {
			align := "C"
			if j == 1 { // Item description - left align
				align = "L"
			} else if j >= 2 { // Numeric columns - right align
				align = "R"
			}
			pdf.CellFormat(colWidths[j], 6, data, "1", 0, align, true, 0, "")
		}
		pdf.Ln(-1)
	}

	pdf.Ln(3)
}

// addModernTotalsSection creates a modern totals summary
func (s *PDFTemplateService) addModernTotalsSection(pdf *gofpdf.Fpdf, data *InvoiceData, config *InvoiceTemplateConfig) {
	// Calculate subtotal from items
	subtotal := 0.0
	for _, item := range data.Items {
		subtotal += item.LineTotal
	}

	// Use provided subtotal if available, otherwise calculate from items
	if data.SubtotalAmount > 0 {
		subtotal = data.SubtotalAmount
	}

	// Calculate final total: subtotal + shipping - discount (no tax)
	finalTotal := subtotal + data.ShippingCharge - data.DiscountAmount

	// Use provided discounted total if available
	if data.DiscountedTotal > 0 {
		finalTotal = data.DiscountedTotal
	}

	// Log the calculation for debugging (can be removed in production)
	fmt.Printf("Invoice Total Calculation - Subtotal: %.2f, Shipping: %.2f, Discount: %.2f, Final Total: %.2f\n",
		subtotal, data.ShippingCharge, data.DiscountAmount, finalTotal)

	// Summary card - compact height
	currentY := pdf.GetY()
	cardWidth := 90.0
	cardX := 198 - cardWidth - 12 // Right aligned
	cardHeight := 45.0            // Reduced height since no tax

	// Card background
	pdf.SetFillColor(248, 249, 250)
	pdf.SetDrawColor(229, 231, 235)
	pdf.RoundedRect(cardX, currentY, cardWidth, cardHeight, 3, "1111", "FD")

	// Card header
	pdf.SetFillColor(config.SecondaryColor[0], config.SecondaryColor[1], config.SecondaryColor[2])
	pdf.RoundedRect(cardX, currentY, cardWidth, 10, 3, "1100", "F")

	pdf.SetXY(cardX+5, currentY+2)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(255, 255, 255)
	pdf.Cell(0, 6, "SUMMARY")

	// Summary items - show all relevant amounts
	summaryItems := []struct {
		Label      string
		Amount     float64
		IsBold     bool
		IsDiscount bool
	}{
		{"Subtotal", subtotal, false, false},
	}

	// Add discount if applied
	if data.DiscountAmount > 0 && data.CouponCode != "" {
		summaryItems = append(summaryItems, struct {
			Label      string
			Amount     float64
			IsBold     bool
			IsDiscount bool
		}{fmt.Sprintf("Discount (%s)", data.CouponCode), -data.DiscountAmount, false, true})
	}

	// Add shipping only if it's greater than 0
	if data.ShippingCharge > 0 {
		summaryItems = append(summaryItems, struct {
			Label      string
			Amount     float64
			IsBold     bool
			IsDiscount bool
		}{"Shipping", data.ShippingCharge, false, false})
	}

	// Add total
	summaryItems = append(summaryItems, struct {
		Label      string
		Amount     float64
		IsBold     bool
		IsDiscount bool
	}{"TOTAL", finalTotal, true, false})

	yPos := currentY + 15
	pdf.SetTextColor(0, 0, 0)

	for _, item := range summaryItems {
		if item.IsBold {
			pdf.SetFont("Arial", "B", 11)
			pdf.SetTextColor(config.PrimaryColor[0], config.PrimaryColor[1], config.PrimaryColor[2])
		} else {
			pdf.SetFont("Arial", "", 9)
			if item.IsDiscount {
				pdf.SetTextColor(34, 197, 94) // Green for discount
			} else {
				pdf.SetTextColor(0, 0, 0)
			}
		}

		pdf.SetXY(cardX+5, yPos)
		pdf.Cell(50, 5, item.Label+":")

		pdf.SetXY(cardX+55, yPos)
		amountStr := fmt.Sprintf("%s%.2f", config.Currency, item.Amount)
		if item.Amount < 0 {
			amountStr = fmt.Sprintf("-%s%.2f", config.Currency, -item.Amount)
		}
		pdf.Cell(30, 5, amountStr)

		yPos += 6
	}

	// Amount in words
	pdf.SetY(currentY + cardHeight + 5)
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(107, 114, 128)
	pdf.Cell(0, 5, fmt.Sprintf("Amount in Words: %s only", s.numberToWords(int(finalTotal))))
	pdf.Ln(10)
}

// addTermsAndConditions adds terms and conditions section
func (s *PDFTemplateService) addTermsAndConditions(pdf *gofpdf.Fpdf, terms []string) {
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 8, "TERMS & CONDITIONS")
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(107, 114, 128)

	for i, term := range terms {
		pdf.Cell(5, 5, fmt.Sprintf("%d.", i+1))
		pdf.MultiCell(0, 5, term, "", "", false)
		pdf.Ln(1)
	}
	pdf.Ln(5)
}

// addBankDetails function removed as bank details are no longer needed

// addModernFooter creates a modern footer
func (s *PDFTemplateService) addModernFooter(pdf *gofpdf.Fpdf, config *InvoiceTemplateConfig, data *InvoiceData) {
	currentY := pdf.GetY() + 5

	// Footer background - reduced height
	pdf.SetFillColor(248, 249, 250)
	pdf.Rect(12, currentY, 186, 20, "F")

	// Signature section
	pdf.SetXY(12, currentY+3)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 5, "Authorized Signatory")

	pdf.SetXY(12, currentY+12)
	pdf.SetFont("Arial", "", 7)
	pdf.SetTextColor(107, 114, 128)
	pdf.Cell(0, 3, "Signature")

	// Footer message and generation info
	if config.FooterMessage != "" {
		pdf.SetXY(100, currentY+3)
		pdf.SetFont("Arial", "", 7)
		pdf.SetTextColor(107, 114, 128)
		pdf.MultiCell(86, 3, config.FooterMessage, "", "", false)
	}

	pdf.SetXY(100, currentY+12)
	pdf.SetFont("Arial", "", 6)
	pdf.Cell(0, 3, fmt.Sprintf("Generated on: %s", time.Now().Format("02 Jan 2006, 15:04:05")))

	// Page border for professional look
	pdf.SetDrawColor(229, 231, 235)
	pdf.Rect(12, 12, 186, 285, "D")
}

// truncateText truncates text to specified length
func (s *PDFTemplateService) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// numberToWords converts numbers to words (enhanced implementation)
func (s *PDFTemplateService) numberToWords(num int) string {
	if num == 0 {
		return "Zero"
	}

	ones := []string{"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine"}
	tens := []string{"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety"}
	teens := []string{"Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen", "Seventeen", "Eighteen", "Nineteen"}

	switch {
	case num < 10:
		return ones[num]
	case num < 20:
		return teens[num-10]
	case num < 100:
		result := tens[num/10]
		if num%10 != 0 {
			result += " " + ones[num%10]
		}
		return strings.TrimSpace(result)
	case num < 1000:
		result := ones[num/100] + " Hundred"
		if num%100 != 0 {
			result += " " + s.numberToWords(num%100)
		}
		return strings.TrimSpace(result)
	case num < 100000:
		result := s.numberToWords(num/1000) + " Thousand"
		if num%1000 != 0 {
			result += " " + s.numberToWords(num%1000)
		}
		return strings.TrimSpace(result)
	case num < 10000000:
		result := s.numberToWords(num/100000) + " Lakh"
		if num%100000 != 0 {
			result += " " + s.numberToWords(num%100000)
		}
		return strings.TrimSpace(result)
	case num < 1000000000:
		result := s.numberToWords(num/10000000) + " Crore"
		if num%10000000 != 0 {
			result += " " + s.numberToWords(num%10000000)
		}
		return strings.TrimSpace(result)
	}

	return "Amount"
}

// getDefaultConfig returns enhanced default template configuration
func (s *PDFTemplateService) getDefaultConfig() *InvoiceTemplateConfig {
	return &InvoiceTemplateConfig{
		CompanyName:     "AMRTI - NATURE'S ELIXIR",
		CompanyAddress:  "NANDHI HILLS 5-85/27, Plot No 27 3rd 3rd Hyderabad Nandagiri Hills Rear Road",
		CompanyPhone:    "+91-7386727007",
		CompanyEmail:    "amrticontact@gmail.com",
		CompanyWebsite:  "www.amrti.com",
		CompanyPAN:      "AAWCA6575D",
		CompanyGST:      "36AAWCA6575D1Z7",
		Currency:        "Rs ",
		ShowTax:         true,
		TaxRate:         18.0,
		ShowCGST:        true,
		ShowSGST:        true,
		CGSTRate:        9.0,
		SGSTRate:        9.0,
		ShowDiscount:    false,
		DiscountAmount:  0.0,
		PrimaryColor:    [3]int{64, 64, 64},    // Dark Gray (monochrome)
		SecondaryColor:  [3]int{96, 96, 96},    // Medium Gray
		AccentColor:     [3]int{128, 128, 128}, // Light Gray
		ShowWatermark:   false,
		WatermarkText:   "NATURE'S ELIXIR",
		HeaderStyle:     "modern",
		ShowQRCode:      false,
		QRCodeData:      "",
		FooterMessage:   "Thank you for choosing AMRTI - Nature's Elixir. For any queries, please contact our support team.",
		ShowBankDetails: false,
	}
}
