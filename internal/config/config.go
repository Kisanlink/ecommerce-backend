// config.go - Loads environment variables from .env
// Author: Mani-KisanLink
package config

import (
	"log"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from .env file
func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env, using system env vars")
	}
}
