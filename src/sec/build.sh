#!/bin/bash
# Exit immediately if a command exits with a non-zero status
set -e

# Print start message
echo "🔧 Starting build process for sec CLI..."

# Define output binary name
BINARY_NAME="sec"

# Check if sec.go exists in the current directory
if [ ! -f "sec.go" ]; then
  echo "❌ sec.go not found in the current directory. Please run this script from the project root."
  exit 1
fi

# Initialize Go module if go.mod doesn't exist
if [ ! -f "go.mod" ]; then
  echo "📦 Initializing Go module..."
  go mod init example.com/sec
fi

# Fetch and tidy Go dependencies
echo "📥 Tidying up Go module..."
go mod tidy

# Remove any previous build of the binary
rm -f "$BINARY_NAME"

# Prompt user to select target OS for build
echo "Select target OS for build:"
echo "1) Ubuntu (linux/amd64)"
echo "2) Windows 11 (windows/amd64)"
read -p "Enter choice [1/2]: " os_choice

# Set GOOS and GOARCH based on user selection
case "$os_choice" in
  1)
    GOOS=linux
    GOARCH=amd64
    ;;
  2)
    GOOS=windows
    GOARCH=amd64
    ;;
  *)
    echo "Invalid choice. Exiting."
    exit 1
    ;;
esac

# Build the Go binary for the selected OS/architecture
echo "🚧 Building binary for $GOOS/$GOARCH..."
go build -o "$BINARY_NAME" sec.go

echo "✅ Build complete: ./$BINARY_NAME"

# Ask user if they want to install the binary globally
read -p "📦 Do you want to install '$BINARY_NAME' to /usr/local/bin? [y/N] " install_choice
if [[ "$install_choice" =~ ^[Yy]$ ]]; then
  sudo mv "$BINARY_NAME" /usr/local/bin/
  echo "✅ Installed to /usr/local/bin/$BINARY_NAME"
else
  echo "ℹ️ Binary left in current directory as './$BINARY_NAME'"
fi
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

echo "Select target OS for build:"
echo "1) Ubuntu (linux/amd64)"
echo "2) Windows 11 (windows/amd64)"
read -p "Enter choice [1/2]: " os_choice

case "$os_choice" in
  1)
    GOOS=linux
    GOARCH=amd64
    ;;
  2)
    GOOS=windows
    GOARCH=amd64
    ;;
  *)
    echo "Invalid choice. Exiting."
    exit 1
    ;;
esac

# Build the binary
echo "🚧 Building binary for $GOOS/$GOARCH..."
go build -o "$BINARY_NAME" sec.go

echo "✅ Build complete: ./$BINARY_NAME"

# Ask to install globally
read -p "📦 Do you want to install '$BINARY_NAME' to /usr/local/bin? [y/N] " install_choice
if [[ "$install_choice" =~ ^[Yy]$ ]]; then
  sudo mv "$BINARY_NAME" /usr/local/bin/
  echo "✅ Installed to /usr/local/bin/$BINARY_NAME"
else
  echo "ℹ️ Binary left in current directory as './$BINARY_NAME'"
fi


