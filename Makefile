.PHONY: build clean run

# Variables
BIN_DIR := ../bin
DEBUG_BIN := $(BIN_DIR)/debug

# Build debug binary
build:
	@echo "Building debug binary..."
	@mkdir -p $(BIN_DIR)
	go build -o $(DEBUG_BIN) ./cmd/debug
	@echo "Build complete: $(DEBUG_BIN)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@echo "Clean complete"

# Run the debug binary
run: build
	@echo "Running debug server..."
	$(DEBUG_BIN)

# Build with race detector
build-race:
	@echo "Building with race detector..."
	@mkdir -p $(BIN_DIR)
	go build -race -o $(DEBUG_BIN) ./cmd/debug
	@echo "Build complete: $(DEBUG_BIN)"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Check for issues
vet:
	@echo "Running go vet..."
	go vet ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# All checks
check: fmt vet test

# Help
help:
	@echo "Available targets:"
	@echo "  build      - Build debug binary"
	@echo "  clean      - Clean build artifacts"
	@echo "  run        - Build and run debug server"
	@echo "  build-race - Build with race detector"
	@echo "  fmt        - Format code"
	@echo "  test       - Run tests"
	@echo "  vet        - Run go vet"
	@echo "  tidy       - Tidy dependencies"
	@echo "  check      - Run fmt, vet, and test"
	@echo "  help       - Show this help"