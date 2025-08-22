package routes

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Auth endpoints
	r.POST("/register", handlers.Register)
	r.POST("/login", handlers.Login)

	// User endpoints (protected)
	handlers.RegisterUserRoutes(r, middleware.AuthMiddleware(), middleware.AdminMiddleware())

	// Recipe endpoints (public for demo)
	recipe := r.Group("/recipes")
	{
		recipe.GET("", handlers.GetRecipes)
		recipe.GET(":id", handlers.GetRecipeByID)
		recipe.POST("", handlers.CreateRecipe)
		recipe.PUT(":id", handlers.UpdateRecipe)
		recipe.DELETE(":id", handlers.DeleteRecipe)
	}
}
