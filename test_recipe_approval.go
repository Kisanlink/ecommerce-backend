package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Test data structures
type TestRecipe struct {
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	PrepTime     string   `json:"prep_time"`
	Servings     int      `json:"servings"`
	Difficulty   string   `json:"difficulty"`
	Ingredients  []string `json:"ingredients"`
	Instructions []string `json:"instructions"`
}

type TestResponse struct {
	Message string `json:"message"`
	Review  struct {
		ReviewID string `json:"review_id"`
		Status   string `json:"status"`
	} `json:"review"`
}

func main() {
	baseURL := "http://localhost:8080/api/v1"

	// Test recipe data
	testRecipe := TestRecipe{
		Name:        "Test Chocolate Cake",
		Category:    "Dessert",
		Description: "A delicious chocolate cake recipe for testing",
		PrepTime:    "45 minutes",
		Servings:    8,
		Difficulty:  "Medium",
		Ingredients: []string{
			"2 cups all-purpose flour",
			"1 cup sugar",
			"1/2 cup cocoa powder",
			"1 tsp baking soda",
			"1/2 tsp salt",
			"1 cup water",
			"1/3 cup vegetable oil",
			"1 tsp vanilla extract",
		},
		Instructions: []string{
			"Preheat oven to 350°F (175°C)",
			"Mix dry ingredients in a large bowl",
			"Add wet ingredients and mix until smooth",
			"Pour into greased 9-inch pan",
			"Bake for 30-35 minutes until toothpick comes out clean",
		},
	}

	fmt.Println("🧪 Testing Recipe Approval System")
	fmt.Println("==================================")

	// Test 1: Submit recipe for review
	fmt.Println("\n1. Testing recipe submission...")
	reviewID := testRecipeSubmission(baseURL, testRecipe)
	if reviewID == "" {
		fmt.Println("❌ Recipe submission failed")
		return
	}
	fmt.Printf("✅ Recipe submitted successfully. Review ID: %s\n", reviewID)

	// Test 2: Get pending reviews (admin endpoint)
	fmt.Println("\n2. Testing admin pending reviews endpoint...")
	testGetPendingReviews(baseURL)

	// Test 3: Get specific pending review
	fmt.Println("\n3. Testing get specific pending review...")
	testGetSpecificReview(baseURL, reviewID)

	// Test 4: Approve recipe
	fmt.Println("\n4. Testing recipe approval...")
	testApproveRecipe(baseURL, reviewID)

	// Test 5: Verify recipe is now in main recipes table
	fmt.Println("\n5. Testing recipe retrieval from main table...")
	testGetRecipeFromMainTable(baseURL, reviewID)

	fmt.Println("\n🎉 All tests completed!")
	fmt.Println("\nNote: This is a basic test script. In a real environment:")
	fmt.Println("- You would need valid Firebase authentication tokens")
	fmt.Println("- Admin endpoints require admin role verification")
	fmt.Println("- The server needs to be running with all dependencies")
}

func testRecipeSubmission(baseURL string, recipe TestRecipe) string {
	url := baseURL + "/recipes/submit"

	jsonData, err := json.Marshal(recipe)
	if err != nil {
		fmt.Printf("Error marshaling recipe: %v\n", err)
		return ""
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error submitting recipe: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return ""
	}

	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Unexpected status code: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
		return ""
	}

	var response TestResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error unmarshaling response: %v\n", err)
		return ""
	}

	return response.Review.ReviewID
}

func testGetPendingReviews(baseURL string) {
	url := baseURL + "/admin/recipes/pending"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error getting pending reviews: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Pending reviews endpoint accessible")
	} else {
		fmt.Printf("❌ Unexpected status code: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
	}
}

func testGetSpecificReview(baseURL, reviewID string) {
	url := baseURL + "/admin/recipes/pending/" + reviewID

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error getting specific review: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Specific review endpoint accessible")
	} else {
		fmt.Printf("❌ Unexpected status code: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
	}
}

func testApproveRecipe(baseURL, reviewID string) {
	url := baseURL + "/admin/recipes/pending/" + reviewID + "/approve"

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Printf("Error approving recipe: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Recipe approval endpoint accessible")
	} else {
		fmt.Printf("❌ Unexpected status code: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
	}
}

func testGetRecipeFromMainTable(baseURL, reviewID string) {
	// Note: In a real implementation, we'd need to track the actual recipe ID
	// For this test, we'll just check if the endpoint is accessible
	url := baseURL + "/recipes"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error getting recipes: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Main recipes endpoint accessible")
	} else {
		fmt.Printf("❌ Unexpected status code: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
	}
}
