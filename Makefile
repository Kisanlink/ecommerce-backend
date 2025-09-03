# Ecommerce Backend Makefile

.PHONY: help build run test clean migrate migrate-status migrate-drop

# Default target
help:
	@echo "Available commands:"
	@echo "  build          - Build the application"
	@echo "  run            - Run the server"
	@echo "  test           - Run tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  migrate        - Run database migrations"
	@echo "  migrate-status - Check migration status"
	@echo "  migrate-drop   - Drop all tables (⚠️  DESTRUCTIVE)"
	@echo "  help           - Show this help message"

# Build the application
build:
	go build -o bin/server cmd/server/main.go

# Run the server
run:
	go run cmd/server/main.go

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Database migration targets

# Run migrations
migrate:
	@echo "🔄 Running database migrations..."
	go run cmd/migrate/main.go

# Check migration status
migrate-status:
	@echo "📊 Checking migration status..."
	go run cmd/migrate/main.go -action status

# Drop all tables (with confirmation)
migrate-drop:
	@echo "⚠️  WARNING: This will drop ALL tables!"
	@echo "Make sure you have a backup before proceeding."
	go run cmd/migrate/main.go -action drop

# Development helpers

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Run example
example:
	go run examples/migration_example.go

# Database setup for development
db-setup: migrate
	@echo "✅ Database setup completed!"

# Full development setup
dev-setup: deps db-setup
	@echo "✅ Development environment setup completed!"
