package main

import (
	"context"
	"log"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.Println("Starting product data seeding...")

	// Database connection string - using the same credentials as .env file
	dsn := "host=localhost user=amrti password=1234 dbname=amrti port=5432 sslmode=disable TimeZone=Asia/Kolkata"

	// Initialize GORM database connection
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Auto-migrate products table
	if err := database.AutoMigrate(&models.Product{}); err != nil {
		log.Fatalf("Failed to migrate products table: %v", err)
	}

	// Seed product data - matching your database table exactly
	products := []*models.Product{
		// Product 1 - Row 1 from table
		{
			BaseModel:       base.NewBaseModel("PROD", hash.Medium),
			Name:            "Papaya Powder",
			Description:     "Premium quality organic papaya powder rich in vitamins A, C, and E. Perfect for digestive health and skin care.",
			Category:        "Superfoods",
			Price:           279.00, // price after discount
			ActualPrice:     359.00, // original price
			DiscountPercent: 22,     // calculated discount
			ImageURL:        "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/web+papaya+powder.jpg",
			ImageURL1:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/web+papaya+powder.jpg",
			ImageURL2:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product+benefits/6.png",
			ImageURL3:       "",
			ImageURL4:       "",
			Stock:           42,
			StockStatus:     "Comming Soon",
			ReviewCount:     98,
			Rating:          4.5,
			IsActive:        false,
		},
		// Product 2 - Row 2 from table
		{
			BaseModel:       base.NewBaseModel("PROD", hash.Medium),
			Name:            "Tomato Powder",
			Description:     "Pure dried tomato powder with intense flavor and rich in lycopene. Ideal for cooking and sauces.",
			Category:        "Spices",
			Price:           249.00, // price after discount
			ActualPrice:     329.00, // original price
			DiscountPercent: 24,     // calculated discount
			ImageURL:        "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/web+Tomato+Powder+(1).jpg",
			ImageURL1:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/web+Tomato+Powder+(1).jpg",
			ImageURL2:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product+benefits/3.png",
			ImageURL3:       "",
			ImageURL4:       "",
			Stock:           42,
			StockStatus:     "Comming Soon",
			ReviewCount:     33,
			Rating:          4.2,
			IsActive:        false,
		},
		// Product 3 - Row 3 from table
		{
			BaseModel:       base.NewBaseModel("PROD", hash.Medium),
			Name:            "Ashwagandha Powder",
			Description:     "High-quality organic ashwagandha root powder known for stress relief and energy enhancement.",
			Category:        "Herbs",
			Price:           249.00, // price after discount
			ActualPrice:     300.00, // original price
			DiscountPercent: 17,     // calculated discount
			ImageURL:        "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/web+aswagandha+powder.jpg",
			ImageURL1:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/web+aswagandha+powder.jpg",
			ImageURL2:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product+benefits/4.png",
			ImageURL3:       "",
			ImageURL4:       "",
			Stock:           60,
			StockStatus:     "Comming Soon",
			ReviewCount:     42,
			Rating:          4.7,
			IsActive:        false,
		},
		// Product 4 - Row 4 from table
		{
			BaseModel:       base.NewBaseModel("PROD", hash.Medium),
			Name:            "Moringa Powder",
			Description:     "Nutrient-rich moringa leaf powder packed with vitamins, minerals, and antioxidants for daily wellness.",
			Category:        "Superfoods",
			Price:           180.00, // price after discount
			ActualPrice:     220.00, // original price
			DiscountPercent: 18,     // calculated discount
			ImageURL:        "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/pouch+front+mockup.jpg",
			ImageURL1:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/pouch+front+mockup.jpg",
			ImageURL2:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/Moringa+bck.png",
			ImageURL3:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product+benefits/1.png",
			ImageURL4:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/pouch+front+mockup.jpg",
			Stock:           29,
			StockStatus:     "In stock",
			ReviewCount:     30,
			Rating:          4.4,
			IsActive:        true,
		},
		// Product 5 - Row 5 from table
		{
			BaseModel:       base.NewBaseModel("PROD", hash.Medium),
			Name:            "Amla Powder",
			Description:     "Pure amla (Indian gooseberry) powder rich in Vitamin C and natural antioxidants for immunity boost.",
			Category:        "Superfoods",
			Price:           95.00,  // price after discount
			ActualPrice:     110.00, // original price
			DiscountPercent: 14,     // calculated discount
			ImageURL:        "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/web+amla+powder.jpg",
			ImageURL1:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/web+amla+powder.jpg",
			ImageURL2:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product+benefits/5.png",
			ImageURL3:       "",
			ImageURL4:       "",
			Stock:           45,
			StockStatus:     "Comming Soon",
			ReviewCount:     35,
			Rating:          4.6,
			IsActive:        false,
		},
		// Product 6 - Row 6 from table
		{
			BaseModel:       base.NewBaseModel("PROD", hash.Medium),
			Name:            "Beetroot Powder",
			Description:     "Premium beetroot powder packed with nitrates, antioxidants and natural minerals for cardiovascular health.",
			Category:        "Superfoods",
			Price:           269.00, // price after discount
			ActualPrice:     349.00, // original price
			DiscountPercent: 23,     // calculated discount
			ImageURL:        "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/web+beetroot+powder.jpg",
			ImageURL1:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/web+beetroot+powder.jpg",
			ImageURL2:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product+benefits/7.png",
			ImageURL3:       "",
			ImageURL4:       "",
			Stock:           54,
			StockStatus:     "Comming Soon",
			ReviewCount:     145,
			Rating:          4.0,
			IsActive:        false, // Row shows false in is_active
		},
		// Product 7 - Row 7 from table
		{
			BaseModel:       base.NewBaseModel("PROD", hash.Medium),
			Name:            "Spinach Powder",
			Description:     "Fresh spinach powder loaded with iron, vitamins, and minerals for enhanced nutrition and energy.",
			Category:        "Superfoods",
			Price:           303.00, // price after discount
			ActualPrice:     399.00, // original price
			DiscountPercent: 24,     // calculated discount
			ImageURL:        "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/web+spinach+powder.jpg",
			ImageURL1:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product-images/web+spinach+powder.jpg",
			ImageURL2:       "https://amrti-ecommerce.s3.eu-north-1.amazonaws.com/images/product+benefits/2.png",
			ImageURL3:       "",
			ImageURL4:       "",
			Stock:           3,
			StockStatus:     "Comming Soon",
			ReviewCount:     89,
			Rating:          4.8,
			IsActive:        false,
		},
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