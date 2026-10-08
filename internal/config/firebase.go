package config

import (
	"context"
	"fmt"
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

		// Check if required environment variables are set
		projectID := os.Getenv("FIREBASE_PROJECT_ID")
		privateKey := os.Getenv("FIREBASE_PRIVATE_KEY")
		clientEmail := os.Getenv("FIREBASE_CLIENT_EMAIL")

		if projectID == "" || privateKey == "" || clientEmail == "" {
			log.Printf("Warning: Firebase environment variables not set. Required: FIREBASE_PROJECT_ID, FIREBASE_PRIVATE_KEY, FIREBASE_CLIENT_EMAIL")
			return fmt.Errorf("Firebase configuration incomplete: missing required environment variables")
		}

		// Initialize with environment variables
		config := &firebase.Config{
			ProjectID: projectID,
		}

		opt := option.WithCredentialsJSON([]byte(fmt.Sprintf(`{
			"type": "service_account",
			"project_id": "%s",
			"private_key": "%s",
			"client_email": "%s",
			"auth_uri": "https://accounts.google.com/o/oauth2/auth",
			"token_uri": "https://oauth2.googleapis.com/token"
		}`, projectID, privateKey, clientEmail)))

		app, err := firebase.NewApp(ctx, config, opt)
		if err != nil {
			return fmt.Errorf("failed to initialize Firebase with environment variables: %w", err)
		}
		FirebaseApp = app
	} else {
		// Initialize with service account file
		opt := option.WithCredentialsFile(serviceAccountPath)
		app, err := firebase.NewApp(ctx, nil, opt)
		if err != nil {
			return fmt.Errorf("failed to initialize Firebase with service account file: %w", err)
		}
		FirebaseApp = app
	}

	// Initialize Auth client
	authClient, err := FirebaseApp.Auth(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize Firebase Auth client: %w", err)
	}
	FirebaseAuth = authClient

	log.Println("Firebase initialized successfully")
	return nil
}

// GetFirebaseAuth returns the Firebase Auth client
func GetFirebaseAuth() *auth.Client {
	return FirebaseAuth
}
