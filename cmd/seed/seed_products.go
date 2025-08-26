package main

import (
	"context"
	"log"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.Println("Starting product data seeding...")

	// Database connection string - using the same credentials as .env file
	dsn := "host=localhost user=postgres password=12345678 dbname=DemoDB port=5432 sslmode=disable TimeZone=Asia/Kolkata"

	// Initialize GORM database connection
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Auto-migrate products table
	if err := database.AutoMigrate(&models.Product{}); err != nil {
		log.Fatalf("Failed to migrate products table: %v", err)
	}

	// Seed product data
	products := []*models.Product{
		models.NewProduct(
			"Turmeric Powder",
			"Premium quality organic turmeric powder with high curcumin content. Perfect for cooking and health benefits.",
			"Spices",
			"https://example.com/images/turmeric-powder.jpg",
			85.00,  // price
			100.00, // actual_price
			100,    // stock
			25,     // review_count
			15,     // discount_percent
			4.5,    // rating
		),
		models.NewProduct(
			"Ginger Powder",
			"Pure dried ginger powder with intense flavor and medicinal properties. Ideal for teas and cooking.",
			"Spices",
			"https://example.com/images/ginger-powder.jpg",
			120.00, // price
			140.00, // actual_price
			75,     // stock
			18,     // review_count
			14,     // discount_percent
			4.3,    // rating
		),
		models.NewProduct(
			"Ashwagandha Powder",
			"High-quality organic ashwagandha root powder known for stress relief and energy enhancement.",
			"Herbs",
			"https://example.com/images/ashwagandha-powder.jpg",
			250.00, // price
			300.00, // actual_price
			60,     // stock
			42,     // review_count
			17,     // discount_percent
			4.7,    // rating
		),
		models.NewProduct(
			"Moringa Powder",
			"Nutrient-rich moringa leaf powder packed with vitamins, minerals, and antioxidants for daily wellness.",
			"Superfoods",
			"https://example.com/images/moringa-powder.jpg",
			180.00, // price
			220.00, // actual_price
			45,     // stock
			30,     // review_count
			18,     // discount_percent
			4.4,    // rating
		),
		models.NewProduct(
			"Amla Powder",
			"Pure amla (Indian gooseberry) powder rich in Vitamin C and natural antioxidants for immunity boost.",
			"Superfoods",
			"https://example.com/images/amla-powder.jpg",
			95.00,  // price
			110.00, // actual_price
			80,     // stock
			35,     // review_count
			14,     // discount_percent
			4.6,    // rating
		),
		models.NewProduct(
			"Spirulina Powder",
			"Premium blue-green spirulina powder loaded with protein, vitamins, and minerals for enhanced nutrition.",
			"Superfoods",
			"https://example.com/images/spirulina-powder.jpg",
			320.00, // price
			380.00, // actual_price
			35,     // stock
			28,     // review_count
			16,     // discount_percent
			4.8,    // rating
		),
		models.NewProduct(
			"Wheatgrass Powder",
			"Fresh wheatgrass powder packed with chlorophyll, enzymes, and nutrients for detoxification.",
			"Superfoods",
			"https://example.com/images/wheatgrass-powder.jpg",
			150.00, // price
			180.00, // actual_price
			50,     // stock
			20,     // review_count
			17,     // discount_percent
			4.2,    // rating
		),
		models.NewProduct(
			"Neem Powder",
			"Pure neem leaf powder with natural antibacterial and antifungal properties for skin and health.",
			"Herbs",
			"https://example.com/images/neem-powder.jpg",
			110.00, // price
			130.00, // actual_price
			65,     // stock
			22,     // review_count
			15,     // discount_percent
			4.1,    // rating
		),
		models.NewProduct(
			"Tulsi Powder",
			"Sacred tulsi (holy basil) powder with adaptogenic properties for stress relief and respiratory health.",
			"Herbs",
			"https://example.com/images/tulsi-powder.jpg",
			135.00, // price
			160.00, // actual_price
			70,     // stock
			32,     // review_count
			16,     // discount_percent
			4.5,    // rating
		),
		models.NewProduct(
			"Brahmi Powder",
			"Pure brahmi (bacopa monnieri) powder for cognitive enhancement, memory improvement, and mental clarity.",
			"Herbs",
			"https://example.com/images/brahmi-powder.jpg",
			190.00, // price
			230.00, // actual_price
			40,     // stock
			15,     // review_count
			17,     // discount_percent
			4.3,    // rating
		),
	}

	ctx := context.Background()

	// Insert each product
	for _, product := range products {
		if err := database.WithContext(ctx).Create(product).Error; err != nil {
			log.Printf("Failed to create product %s: %v", product.Name, err)
			continue
		}

		log.Printf("Successfully seeded product: %s (ID: %s)", product.Name, product.ID)
	}

	log.Println("Product seeding completed successfully!")
}
