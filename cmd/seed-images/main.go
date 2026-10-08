package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.Println("Starting image data seeding...")

	// Database connection string - try multiple configurations
	dsn := getDatabaseDSN()

	// Initialize GORM database connection
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Auto-migrate images table
	if err := database.AutoMigrate(&models.Image{}); err != nil {
		log.Fatalf("Failed to migrate images table: %v", err)
	}

	ctx := context.Background()

	// Clear existing images first (optional - remove this if you want to keep existing data)
	log.Println("Clearing existing images...")
	database.WithContext(ctx).Where("1 = 1").Delete(&models.Image{})

	// Product ID mapping based on the database table
	productMapping := map[string]string{
		"Amla":        "PROD00000005", // Amla Powder
		"Ashwagandha": "PROD00000003", // Ashwagandha Powder
		"Beetroot":    "PROD00000006", // Beetroot Powder
		"Moringa":     "PROD00000004", // Moringa Powder
		"Papaya":      "PROD00000001", // Papaya Powder
		"Spinach":     "PROD00000007", // Spinach Powder
		"Tomato":      "PROD00000002", // Tomato Powder
	}

	// Image data organized by product - Complete data from user
	imageData := map[string][]string{
		"Amla": {
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Amla/10.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Amla/amla+backside.jpg",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Amla/amla+powder.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Amla/web+amla+powder.jpg",
		},
		"Ashwagandha": {
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Aswagandha/8.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Aswagandha/9.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Aswagandha/web+aswagandha+powder.jpg",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Aswagandha/aswagandha+backside.jpg",
		},
		"Beetroot": {
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Beetroot/14.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Beetroot/15.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Beetroot/web+beetroot+powder.jpg",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Beetroot/beetroot+backside.jpg",
		},
		"Moringa": {
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Moringa/1.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Moringa/Copy+of+pouch+front+mockup.jpg",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Moringa/2.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Moringa/3.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Moringa/pouch+back+mockup.jpg",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Moringa/moringa+pouch+PACKANGING+back.jpg",
		},
		"Papaya": {
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Papaya/12.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Papaya/13.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Papaya/moringa+backside.jpg",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Papaya/web+papaya+powder.jpg",
		},
		"Spinach": {
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Spinach/4.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Spinach/5.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Spinach/web+spinach+powder.jpg",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Spinach/spinach+backside.jpg",
		},
		"Tomato": {
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Tomato/6.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Tomato/7.png",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Tomato/tomato+backside.jpg",
			"https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Tomato/web+Tomato+Powder.jpg",
		},
	}

	totalImages := 0

	// Insert images for each product
	for productName, productID := range productMapping {
		images := imageData[productName]
		log.Printf("Processing %d images for %s (ID: %s)", len(images), productName, productID)

		for i, imageURL := range images {
			// Create image with proper metadata
			image := &models.Image{
				BaseModel: base.NewBaseModel("IMG", hash.Large),
				ProductID: productID,
				ImageURL:  imageURL,
				ImageType: models.ImageTypeProduct,
				AltText:   generateAltText(productName, i),
				Caption:   generateCaption(productName, i),
				SortOrder: i,
				IsActive:  true,
				IsPrimary: i == 0, // First image is primary
				MimeType:  getMimeType(imageURL),
			}

			// Insert image into database
			if err := database.WithContext(ctx).Create(image).Error; err != nil {
				log.Printf("Failed to create image for %s: %v", productName, err)
				continue
			}

			log.Printf("Successfully seeded image %d for %s: %s (ID: %s)", i+1, productName, imageURL, image.ID)
			totalImages++
		}
	}

	log.Printf("Image seeding completed successfully! Seeded %d images for %d products.", totalImages, len(productMapping))
}

// generateAltText creates alt text for the image
func generateAltText(productName string, index int) string {
	altTexts := map[string][]string{
		"Amla":        {"Amla Powder Product", "Amla Powder Back View", "Amla Powder Close-up", "Amla Powder Web Image"},
		"Ashwagandha": {"Ashwagandha Powder Product", "Ashwagandha Powder Detail", "Ashwagandha Powder Web Image", "Ashwagandha Powder Back View"},
		"Beetroot":    {"Beetroot Powder Product", "Beetroot Powder Detail", "Beetroot Powder Web Image", "Beetroot Powder Back View"},
		"Moringa":     {"Moringa Powder Product", "Moringa Pouch Front Mockup", "Moringa Powder Detail", "Moringa Powder Close-up", "Moringa Pouch Back Mockup", "Moringa Powder Packaging"},
		"Papaya":      {"Papaya Powder Product", "Papaya Powder Detail", "Papaya Powder Back View", "Papaya Powder Web Image"},
		"Spinach":     {"Spinach Powder Product", "Spinach Powder Detail", "Spinach Powder Web Image", "Spinach Powder Back View"},
		"Tomato":      {"Tomato Powder Product", "Tomato Powder Detail", "Tomato Powder Back View", "Tomato Powder Web Image"},
	}

	if texts, exists := altTexts[productName]; exists && index < len(texts) {
		return texts[index]
	}
	return productName + " Powder Product Image"
}

// generateCaption creates caption for the image
func generateCaption(productName string, index int) string {
	captions := map[string][]string{
		"Amla":        {"Premium Amla Powder", "Amla Powder Back View", "Amla Powder Detail", "Amla Powder Product"},
		"Ashwagandha": {"Premium Ashwagandha Powder", "Ashwagandha Powder Detail", "Ashwagandha Powder Product", "Ashwagandha Powder Back View"},
		"Beetroot":    {"Premium Beetroot Powder", "Beetroot Powder Detail", "Beetroot Powder Product", "Beetroot Powder Back View"},
		"Moringa":     {"Premium Moringa Powder", "Moringa Pouch Front Mockup", "Moringa Powder Detail", "Moringa Powder Close-up", "Moringa Pouch Back Mockup", "Moringa Powder Packaging"},
		"Papaya":      {"Premium Papaya Powder", "Papaya Powder Detail", "Papaya Powder Back View", "Papaya Powder Product"},
		"Spinach":     {"Premium Spinach Powder", "Spinach Powder Detail", "Spinach Powder Product", "Spinach Powder Back View"},
		"Tomato":      {"Premium Tomato Powder", "Tomato Powder Detail", "Tomato Powder Back View", "Tomato Powder Product"},
	}

	if texts, exists := captions[productName]; exists && index < len(texts) {
		return texts[index]
	}
	return "Premium " + productName + " Powder"
}

// getMimeType determines MIME type based on file extension
func getMimeType(imageURL string) string {
	if len(imageURL) < 4 {
		return "image/jpeg"
	}

	extension := imageURL[len(imageURL)-4:]
	switch extension {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg" // Default fallback
	}
}

// getDatabaseDSN returns the database connection string with fallback options
func getDatabaseDSN() string {
	// Try environment variables first
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		log.Printf("Using POSTGRES_DSN from environment")
		return dsn
	}

	// Try individual environment variables
	host := getEnvOrDefault("DB_POSTGRES_HOST", "localhost")
	port := getEnvOrDefault("DB_POSTGRES_PORT", "5555") // Docker Compose port
	user := getEnvOrDefault("DB_POSTGRES_USER", "postgres")
	password := getEnvOrDefault("DB_POSTGRES_PASSWORD", "12345678")
	dbname := getEnvOrDefault("DB_POSTGRES_DBNAME", "DemoDB")
	sslmode := getEnvOrDefault("DB_POSTGRES_SSLMODE", "require")
	channelBinding := os.Getenv("DB_POSTGRES_CHANNEL_BINDING")

	// Try Docker Compose configuration first
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Kolkata",
		host, user, password, dbname, port, sslmode)
	if channelBinding != "" {
		dsn = fmt.Sprintf("%s channel_binding=%s", dsn, channelBinding)
	}

	log.Printf("Using database connection: host=%s, port=%s, user=%s, dbname=%s", host, port, user, dbname)
	return dsn
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
