package recipes

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

// RegisterRecipeRoutes registers all recipe related routes
func RegisterRecipeRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager) {
	recipeHandler := handlers.NewRecipeHandler(repoManager)

	// Public recipe routes
	recipes := r.Group("/recipes")
	{
		recipes.GET("", recipeHandler.GetRecipes)                    // Get all recipes with filters
		recipes.GET("/:id", recipeHandler.GetRecipeByID)             // Get recipe by ID
		recipes.GET("/search", recipeHandler.GetRecipes)             // Search recipes
		recipes.GET("/category/:category", recipeHandler.GetRecipes) // Get recipes by category
		recipes.GET("/featured", recipeHandler.GetRecipes)           // Get featured recipes
		recipes.GET("/popular", recipeHandler.GetRecipes)            // Get popular recipes
	}

	// Protected recipe routes (requires authentication for creation/modification)
	protected := recipes.Group("/")
	protected.Use(middleware.OptionalFirebaseAuth()) // Optional auth for viewing, required for creating
	{
		protected.POST("", recipeHandler.CreateRecipe)                                            // Create new recipe
		protected.PUT("/:id", middleware.FirebaseAuthMiddleware(), recipeHandler.UpdateRecipe)    // Update recipe (owner only)
		protected.DELETE("/:id", middleware.FirebaseAuthMiddleware(), recipeHandler.DeleteRecipe) // Delete recipe (owner only)
	}

	// Recipe categories
	categories := r.Group("/recipe-categories")
	{
		categories.GET("", recipeHandler.GetRecipes) // Get all recipe categories
		// categories.GET("/:id", categoryHandler.GetCategory)        // Get category details
	}

	// Recipe ratings and reviews
	reviews := recipes.Group("/:id/reviews")
	reviews.Use(middleware.OptionalFirebaseAuth())
	{
		reviews.GET("", recipeHandler.GetRecipes) // Get recipe reviews
		// reviews.POST("", reviewHandler.CreateRecipeReview)         // Create review (requires auth)
		// reviews.PUT("/:reviewId", reviewHandler.UpdateReview)      // Update review (owner only)
		// reviews.DELETE("/:reviewId", reviewHandler.DeleteReview)   // Delete review (owner only)
	}

	// Recipe collections/favorites
	collections := r.Group("/my-recipes")
	collections.Use(middleware.FirebaseAuthMiddleware())
	{
		collections.GET("/created", recipeHandler.GetRecipes)         // Get user's created recipes
		collections.GET("/favorites", recipeHandler.GetRecipes)       // Get user's favorite recipes
		collections.POST("/:id/favorite", recipeHandler.GetRecipes)   // Add to favorites
		collections.DELETE("/:id/favorite", recipeHandler.GetRecipes) // Remove from favorites
	}

	// Recipe ingredients (for shopping integration)
	ingredients := recipes.Group("/:id/ingredients")
	{
		ingredients.GET("", recipeHandler.GetRecipes)                                                   // Get recipe ingredients
		ingredients.POST("/add-to-cart", middleware.FirebaseAuthMiddleware(), recipeHandler.GetRecipes) // Add ingredients to cart
	}
}
