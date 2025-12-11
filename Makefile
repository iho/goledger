.PHONY: all build run test test-integration test-coverage coverage-percent migrate-up migrate-down sqlc docker-up docker-down clean lint

# Build variables
BINARY_NAME=goledger
BUILD_DIR=bin

# Default target
all: build

# Generate sqlc code
sqlc:
	cd internal/infrastructure/postgres && sqlc generate

# Generate mocks using gomock
mocks:
	mockgen -source=internal/usecase/interfaces.go -destination=internal/usecase/mocks/mock_interfaces.go -package=mocks

# Generate protobuf code using buf
buf:
	buf generate

# Generate all code (sqlc + mocks + protobuf)
generate: sqlc mocks buf

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
	gotestsum --format testdox -- -race ./internal/domain/... ./internal/usecase/...

# Run integration tests (requires Docker)
test-integration: docker-up
	sleep 2
	gotestsum --format testdox -- -race ./tests/integration/...

# Run tests
test-all:
	gotestsum --format testdox -- -race ./...

# Run database migrations
migrate-up:
	go run ./cmd/cli migrate up

migrate-down:
	go run ./cmd/cli migrate down

# Run tests in Docker
test-docker:
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
	docker-compose -f docker-compose.test.yml down -v

# Run all tests with coverage
test-coverage:
	gotestsum --format testdox -- -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Show coverage percentage summary
coverage-percent:
	@packages=$$(go list ./cmd/... ./internal/... ./proto/... | grep -v '/mocks' | grep -v '/internal/infrastructure/postgres/generated' | grep -v '/proto/'); \
		go test $$packages -coverprofile=coverage.out >/dev/null
	@go tool cover -func=coverage.out | tail -n 1

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
