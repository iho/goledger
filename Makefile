.PHONY: all build run test test-integration test-coverage sqlc docker-up docker-down clean lint

# Build variables
BINARY_NAME=goledger
BUILD_DIR=bin

# Default target
all: build

# Generate sqlc code
sqlc:
	cd internal/infrastructure/postgres && sqlc generate

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build the application
build: deps
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

# Run the application
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run unit tests
test:
	go test -v -race ./internal/domain/... ./internal/usecase/...

# Run integration tests (requires Docker)
test-integration: docker-up
	go test -v -race -tags=integration ./tests/integration/...

# Run all tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race detector
test-race:
	go test -v -race ./...

# Start development environment
docker-up:
	docker-compose up -d

# Stop development environment
docker-down:
	docker-compose down -v

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...
