#!/bin/bash

set -e

echo "🔧 Starting build process for sec CLI..."

# Define output binary name
BINARY_NAME="sec"

# Check if sec.go exists
if [ ! -f "sec.go" ]; then
  echo "❌ sec.go not found in the current directory. Please run this script from the project root."
  exit 1
fi

# Initialize Go module if go.mod doesn't exist
if [ ! -f "go.mod" ]; then
  echo "📦 Initializing Go module..."
  go mod init example.com/sec
fi

# Fetch dependencies
echo "📥 Tidying up Go module..."
go mod tidy

# Clean previous build
rm -f "$BINARY_NAME"

# Build the binary
echo "🚧 Building binary..."
GOOS=linux GOARCH=amd64 go build -o "$BINARY_NAME" sec.go

echo "✅ Build complete: ./$BINARY_NAME"

# Ask to install globally
read -p "📦 Do you want to install '$BINARY_NAME' to /usr/local/bin? [y/N] " install_choice
if [[ "$install_choice" =~ ^[Yy]$ ]]; then
  sudo mv "$BINARY_NAME" /usr/local/bin/
  echo "✅ Installed to /usr/local/bin/$BINARY_NAME"
else
  echo "ℹ️ Binary left in current directory as './$BINARY_NAME'"
fi

