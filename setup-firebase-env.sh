#!/bin/bash
echo "Setting up Firebase environment variables..."

# Set Firebase Web API Key
export FIREBASE_WEB_API_KEY="AIzaSyCPzNrFV3Q6a4rcAQF-60G3h3K1WBQqQFI"

# Set Firebase Project ID
export FIREBASE_PROJECT_ID="amrti-m1355"

echo ""
echo "Environment variables set:"
echo "FIREBASE_WEB_API_KEY=$FIREBASE_WEB_API_KEY"
echo "FIREBASE_PROJECT_ID=$FIREBASE_PROJECT_ID"
echo ""
echo "To make these permanent, add them to your ~/.bashrc or ~/.zshrc file."
echo "Or run your Go server from this terminal session."
echo ""
echo "Starting Go server..."
go run cmd/server/main.go
