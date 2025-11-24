#!/bin/bash

# Build script for Specter - RSI Store Checkout Assistant
# Builds binaries for macOS and Windows

set -e

echo "Building Specter for multiple platforms..."
echo ""

# Create build directory
mkdir -p build

# Get version from git or use default
VERSION=$(git describe --tags --always 2>/dev/null || echo "v1.0.0")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# macOS (Intel)
echo "📦 Building for macOS (Intel)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o build/specter-darwin-amd64 .
echo "✓ build/specter-darwin-amd64"

# macOS (Apple Silicon)
echo "📦 Building for macOS (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o build/specter-darwin-arm64 .
echo "✓ build/specter-darwin-arm64"

# Windows
echo "📦 Building for Windows..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o build/specter-windows-amd64.exe .
echo "✓ build/specter-windows-amd64.exe"

# Linux (optional, for completeness)
echo "📦 Building for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o build/specter-linux-amd64 .
echo "✓ build/specter-linux-amd64"

echo ""
echo "✓ All builds completed successfully!"
echo ""
echo "Build outputs:"
ls -lh build/

echo ""
echo "To create distributable packages:"
echo "  - macOS: Use 'create-dmg' or package as .app bundle"
echo "  - Windows: Use NSIS or Inno Setup for installer"
