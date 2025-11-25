.PHONY: all build clean run install deps help

# Default target
all: build

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Build for current platform
build: deps
	@echo "Building for current platform..."
	go build -ldflags="-s -w" -o specter .

# Build for all platforms
build-all: deps
	@echo "Building for all platforms..."
	@mkdir -p build
	@echo "Building for macOS (amd64)..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o build/specter-macos-amd64 .
	@echo "Building for macOS (arm64)..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o build/specter-macos-arm64 .
	@echo "Building for Windows (amd64)..."
	@GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o build/specter-windows-amd64.exe .
	@echo "Building for Linux (amd64)..."
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o build/specter-linux-amd64 .
	@echo "All builds complete! Binaries are in the build/ directory"

# Run the application
run: build
	@./specter

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f specter specter.exe
	@rm -rf build/
	@rm -f config.yaml

# Install on local system (Unix-like systems)
install: build
	@echo "Installing to /usr/local/bin..."
	@sudo cp specter /usr/local/bin/
	@echo "Installation complete!"

# Show help
help:
	@echo "Specter - RSI Store Checkout Assistant"
	@echo ""
	@echo "Available targets:"
	@echo "  make deps       - Install Go dependencies"
	@echo "  make build      - Build for current platform"
	@echo "  make build-all  - Build for all platforms"
	@echo "  make run        - Build and run the application"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make install    - Install to /usr/local/bin (Unix-like)"
	@echo "  make help       - Show this help message"
