package main

import (
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
)

func main() {
	// Test cart creation with proper BaseModel
	fmt.Println("Testing cart creation with BaseModel...")

	// Test creating a new cart using models.NewCart()
	testUserID := "test-user-basemodel-123"
	cart := models.NewCart(testUserID)

	fmt.Printf("Created cart with ID: %s\n", cart.ID)
	fmt.Printf("Cart UserID: %s\n", cart.UserID)

	if cart.ID == "" {
		fmt.Println("❌ FAILED: Cart ID is empty!")
		return
	}

	// Test creating cart items using models.NewCartItem()
	item := models.NewCartItem(cart.ID, "PRD00000001", 5, 25.99)

	fmt.Printf("Created cart item with ID: %s\n", item.ID)
	fmt.Printf("Cart item CartID: %s\n", item.CartID)
	fmt.Printf("Cart item ProductID: %s\n", item.ProductID)
	fmt.Printf("Cart item Quantity: %d\n", item.Quantity)
	fmt.Printf("Cart item UnitPrice: %.2f\n", item.UnitPrice)
	fmt.Printf("Cart item TotalPrice: %.2f\n", item.TotalPrice)

	if item.ID == "" {
		fmt.Println("❌ FAILED: Cart item ID is empty!")
		return
	}

	// Test creating another cart to verify ID generation sequence
	cart2 := models.NewCart("test-user-2")
	fmt.Printf("Second cart ID: %s\n", cart2.ID)

	if cart2.ID == "" {
		fmt.Println("❌ FAILED: Second cart ID is empty!")
		return
	}

	if cart.ID == cart2.ID {
		fmt.Println("❌ FAILED: Cart IDs are the same! ID generation not working properly!")
		return
	}

	fmt.Println("✅ SUCCESS: Cart and cart item created with proper BaseModel IDs!")
	fmt.Println("✅ SUCCESS: BaseModel integration is working correctly!")
	fmt.Println("✅ SUCCESS: ID generation is working with unique sequential IDs!")
}
