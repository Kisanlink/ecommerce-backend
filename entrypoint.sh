#!/bin/sh
set -e

# Create directory for Firebase service account JSON if it doesn't exist
mkdir -p /app/internal/config

# Write Firebase service account JSON from environment variable to file
# This allows us to store the secret in AWS Secrets Manager and create the file at runtime
if [ -n "$FIREBASE_SERVICE_ACCOUNT_JSON_CONTENT" ]; then
    echo "Writing Firebase service account JSON to file..."
    echo "$FIREBASE_SERVICE_ACCOUNT_JSON_CONTENT" > /app/internal/config/firebase_service_account.json
    echo "Firebase service account JSON file created successfully"
else
    echo "Warning: FIREBASE_SERVICE_ACCOUNT_JSON_CONTENT environment variable not set"
    echo "Firebase initialization may fail if the file doesn't exist"
fi

# Start the application
echo "Starting application..."
exec ./main
