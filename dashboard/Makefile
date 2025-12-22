.PHONY: build run clean test dev

# Build the application
build:
	go build -o dashboard main.go

# Run the application
run: build
	./dashboard

# Run in development mode (rebuild on changes)
dev:
	@echo "Starting development server..."
	@air || (echo "Air not found. Install with: go install github.com/cosmtrek/air@latest" && go run main.go)

# Clean build artifacts
clean:
	rm -f dashboard
	go clean

# Run tests
test:
	go test ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint: fmt
	go vet ./...

# Tidy dependencies
tidy:
	go mod tidy

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the application"
	@echo "  run      - Build and run the application"
	@echo "  dev      - Run in development mode (requires air)"
	@echo "  clean    - Clean build artifacts"
	@echo "  test     - Run tests"
	@echo "  fmt      - Format code"
	@echo "  lint     - Lint and format code"
	@echo "  tidy     - Tidy dependencies"
	@echo "  help     - Show this help"
