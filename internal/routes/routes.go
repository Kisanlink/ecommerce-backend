package routes

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, repoManager *repositories.RepositoryManager) {
	// Firebase Auth endpoints
	firebaseAuth := handlers.NewFirebaseAuthHandler()
	auth := r.Group("/auth")
	{
		auth.POST("/signup", firebaseAuth.Signup)
		auth.POST("/verify-token", firebaseAuth.VerifyToken)
		auth.GET("/profile", middleware.FirebaseAuthMiddleware(), firebaseAuth.GetProfile)
		auth.PUT("/profile", middleware.FirebaseAuthMiddleware(), firebaseAuth.UpdateProfile)
		auth.DELETE("/account", middleware.FirebaseAuthMiddleware(), firebaseAuth.DeleteAccount)
	}

	// Create handlers with repository dependencies
	userHandler := handlers.NewUserHandler(repoManager.UserRepository)
	recipeHandler := handlers.NewRecipeHandler(repoManager) // May need multiple repos
	
	// Legacy auth endpoints (if you want to keep them)
	r.POST("/register", userHandler.Register)
	r.POST("/login", userHandler.Login)

	// User endpoints (protected with Firebase auth)
	userRoutes := r.Group("/users")
	userRoutes.Use(middleware.FirebaseAuthMiddleware())
	{
		userRoutes.GET("/:id", userHandler.GetUser)
		userRoutes.PUT("/:id", userHandler.UpdateUser)
		// Add more user-specific routes here
	}

	// Recipe endpoints (public for demo)
	recipe := r.Group("/recipes")
	{
		recipe.GET("", recipeHandler.GetRecipes)
		recipe.GET("/:id", recipeHandler.GetRecipeByID)
		recipe.POST("", middleware.OptionalFirebaseAuth(), recipeHandler.CreateRecipe)
		recipe.PUT("/:id", middleware.FirebaseAuthMiddleware(), recipeHandler.UpdateRecipe)
		recipe.DELETE("/:id", middleware.FirebaseAuthMiddleware(), recipeHandler.DeleteRecipe)
	}
}
