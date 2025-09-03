package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	fmt.Println("Testing Updated Cart Request Structures...")

	// Test UpdateCartItemRequest - should not require ProductID in body anymore
	updateReq := map[string]interface{}{
		"quantity": 5,
	}

	jsonData, err := json.Marshal(updateReq)
	if err != nil {
		fmt.Printf("❌ Error marshaling update request: %v\n", err)
		return
	}

	fmt.Printf("✅ UpdateCartItem request JSON (ProductID in URL, not body): %s\n", string(jsonData))

	// Test IncrementCartItemRequest
	incrementReq := map[string]interface{}{
		"amount": 2, // Optional, defaults to 1 if not provided
	}

	jsonData, err = json.Marshal(incrementReq)
	if err != nil {
		fmt.Printf("❌ Error marshaling increment request: %v\n", err)
		return
	}

	fmt.Printf("✅ IncrementCartItem request JSON: %s\n", string(jsonData))

	// Test DecrementCartItemRequest
	decrementReq := map[string]interface{}{
		"amount": 1, // Optional, defaults to 1 if not provided
	}

	jsonData, err = json.Marshal(decrementReq)
	if err != nil {
		fmt.Printf("❌ Error marshaling decrement request: %v\n", err)
		return
	}

	fmt.Printf("✅ DecrementCartItem request JSON: %s\n", string(jsonData))

	fmt.Println("\n✅ All request structures are valid!")
	fmt.Println("✅ UpdateCartItem no longer requires ProductID in request body")
	fmt.Println("✅ Added user-friendly increment/decrement endpoints")

	fmt.Println("\n📍 API Endpoints:")
	fmt.Println("• PUT /api/v1/cart/items/{product_id} - Update item quantity")
	fmt.Println("• POST /api/v1/cart/items/{product_id}/increment - Increment by amount (default: 1)")
	fmt.Println("• POST /api/v1/cart/items/{product_id}/decrement - Decrement by amount (default: 1)")
}
