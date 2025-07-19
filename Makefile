.PHONY: build clean run vm-deploy

# Variables
BIN_DIR := ./bin
DEBUG_BIN := $(BIN_DIR)/debug

# VM deployment variables (customize these)
VM_USER ?= touka
VM_IP ?= 192.168.128.129
VM_PATH ?= /home/$(VM_USER)

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

# VM deployment
vm-deploy: build
	@echo "Deploying to VM..."
	scp -P 22 $(DEBUG_BIN) $(VM_USER)@$(VM_IP):$(VM_PATH)/debug
	@echo "Deploy complete"

# SSH tunnel for testing (background)
vm-tunnel:
	@echo "Starting SSH tunnel (8080:localhost:8080) in background..."
	@ssh -f -L 8080:localhost:8080 $(VM_USER)@$(VM_IP) -N
	@echo "SSH tunnel started. Use 'make vm-tunnel-stop' to stop it."

# Stop SSH tunnel
vm-tunnel-stop:
	@echo "Stopping SSH tunnel..."
	@pkill -f "ssh.*-L 8080:localhost:8080.*$(VM_IP)" || true
	@echo "SSH tunnel stopped."

# Deploy and run on VM
vm-run: vm-deploy
	@echo "Starting server on VM..."
	ssh $(VM_USER)@$(VM_IP) "$(VM_PATH)/debug"

# All-in-one: deploy, tunnel, and run
vm-dev: vm-deploy vm-tunnel
	@echo "Starting server on VM with tunnel..."
	ssh $(VM_USER)@$(VM_IP) "$(VM_PATH)/debug"

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