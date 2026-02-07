#!/bin/bash

# Configuration
APP_NAME="jdoctor"
INSTALL_DIR="$HOME/.local/bin"
SRC_DIR="./cmd/jdoctor"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building $APP_NAME...${NC}"

# Build the project
if go build -o "$APP_NAME" "$SRC_DIR"; then
    echo -e "${GREEN}Build successful!${NC}"
else
    echo -e "${RED}Build failed.${NC}"
    exit 1
fi

# Ensure install directory exists
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating directory $INSTALL_DIR..."
    mkdir -p "$INSTALL_DIR"
fi

# Move executable
echo "Installing to $INSTALL_DIR..."
mv "$APP_NAME" "$INSTALL_DIR/"

# Check if INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${RED}WARNING: $INSTALL_DIR is not in your PATH.${NC}"
    echo "Please add the following line to your shell configuration file (.bashrc, .zshrc, etc.):"
    echo "export PATH=\"\$HOME/.local/bin:\$PATH\""
else
    echo -e "${GREEN}$APP_NAME installed successfully!${NC}"
    echo "You can now run '$APP_NAME' from anywhere."
fi
