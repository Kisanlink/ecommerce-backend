// main.go - Entry point for amrti-backend server
// Author: Mani-KisanLink
// Initializes config, sets up server, and starts Gin HTTP server

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/server"
)

func main() {
	config.LoadEnv()
	fmt.Println("Starting amrti-backend server...")
	srv, err := server.NewServer()
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
