#!/bin/bash

# Build script for MudBot
set -e

echo "Building MudBot..."

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${PROJECT_ROOT}"

# Download dependencies
echo "Downloading dependencies..."
go mod tidy

# Build for current platform
echo "Building binary..."
go build -o bin/mudbot ./cmd/mudbot

# Make binary executable
chmod +x bin/mudbot

echo "Build complete! Binary at: bin/mudbot"
echo ""
echo "To run:"
echo "  ./bin/mudbot -token YOUR_BOT_TOKEN"
echo ""
echo "Or set environment variables:"
echo "  export DISCORD_TOKEN=YOUR_BOT_TOKEN"
echo "  export DISCORD_GUILD_ID=YOUR_GUILD_ID"
echo "  ./bin/mudbot"