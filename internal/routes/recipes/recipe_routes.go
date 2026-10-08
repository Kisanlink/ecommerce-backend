package recipes

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// RegisterRecipeRoutes registers all recipe related routes
func RegisterRecipeRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) {
	recipeHandler := handlers.NewRecipeHandler(repoManager)
	userRecipeHandler := handlers.NewUserRecipeHandler(repoManager, serviceManager.R2Service)

	// Public recipe routes
	recipes := r.Group("/recipes")
	{
		recipes.GET("", recipeHandler.GetRecipes)                    // Get all recipes with filters
		recipes.GET("/:id", recipeHandler.GetRecipeByID)             // Get recipe by ID            // Search recipes
		recipes.GET("/category/:category", recipeHandler.GetRecipes) // Get recipes by category           // Get popular recipes
	}

	// User recipe submission routes (requires authentication)
	userRecipes := r.Group("/recipes")
	userRecipes.Use(middleware.FirebaseAuthMiddleware())
	{
		userRecipes.POST("/submit", userRecipeHandler.SubmitRecipeForReview)          // Submit recipe for review
		userRecipes.GET("/my-submissions", userRecipeHandler.GetUserSubmittedRecipes) // Get user's submitted recipes
		userRecipes.POST("/images/upload", userRecipeHandler.UploadRecipeImage)       // Upload recipe image
		userRecipes.DELETE("/images/:s3_key", userRecipeHandler.DeleteRecipeImage)    // Delete recipe image
	}

	// Admin-only recipe routes (requires authentication + admin role)
	adminRecipes := recipes.Group("/")
	adminRecipes.Use(middleware.FirebaseAuthMiddleware())
	adminRecipes.Use(middleware.AdminGuard(repoManager))
	{
		adminRecipes.POST("", recipeHandler.CreateRecipe)       // Create new recipe (admin only)
		adminRecipes.PUT("/:id", recipeHandler.UpdateRecipe)    // Update recipe (admin only)
		adminRecipes.DELETE("/:id", recipeHandler.DeleteRecipe) // Delete recipe (admin only)
	}

	// Recipe categories
	categories := r.Group("/recipe-categories")
	{
		categories.GET("", recipeHandler.GetRecipes) // Get all recipe categories
		// categories.GET("/:id", categoryHandler.GetCategory)        // Get category details
	}

	// Recipe ratings and reviews
	//reviews := recipes.Group("/:id/reviews")
	//reviews.Use(middleware.OptionalFirebaseAuth())
	//{
	//reviews.GET("", recipeHandler.GetRecipes) // Get recipe reviews
	// reviews.POST("", reviewHandler.CreateRecipeReview)         // Create review (requires auth)
	// reviews.PUT("/:reviewId", reviewHandler.UpdateReview)      // Update review (owner only)
	// reviews.DELETE("/:reviewId", reviewHandler.DeleteReview)   // Delete review (owner only)
	//}
}
