package config

import (
	"context"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

var (
	FirebaseApp  *firebase.App
	FirebaseAuth *auth.Client
)

// InitFirebase initializes Firebase Admin SDK
func InitFirebase() error {
	ctx := context.Background()

	// Check if service account key file exists
	serviceAccountPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH")
	if serviceAccountPath == "" {
		serviceAccountPath = "internal/config/firebase_service_account.json"
	}

	// Check if file exists
	if _, err := os.Stat(serviceAccountPath); os.IsNotExist(err) {
		log.Printf("Firebase service account file not found at %s, using environment variables", serviceAccountPath)
		// Try to initialize with environment variables
		app, err := firebase.NewApp(ctx, nil)
		if err != nil {
			return err
		}
		FirebaseApp = app
	} else {
		// Initialize with service account file
		opt := option.WithCredentialsFile(serviceAccountPath)
		app, err := firebase.NewApp(ctx, nil, opt)
		if err != nil {
			return err
		}
		FirebaseApp = app
	}

	// Initialize Auth client
	authClient, err := FirebaseApp.Auth(ctx)
	if err != nil {
		return err
	}
	FirebaseAuth = authClient

	log.Println("Firebase initialized successfully")
	return nil
}

// GetFirebaseAuth returns the Firebase Auth client
func GetFirebaseAuth() *auth.Client {
	return FirebaseAuth
}
