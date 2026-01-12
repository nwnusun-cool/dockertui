#!/bin/bash
# Cross-platform build script

set -e

APP_NAME="docktui"
BUILD_DIR="build"

mkdir -p $BUILD_DIR

echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o $BUILD_DIR/$APP_NAME-linux-amd64 ./cmd/docktui

echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o $BUILD_DIR/$APP_NAME-windows-amd64.exe ./cmd/docktui

echo "Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -o $BUILD_DIR/$APP_NAME-darwin-amd64 ./cmd/docktui

echo "Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -o $BUILD_DIR/$APP_NAME-darwin-arm64 ./cmd/docktui

echo "Done! Binaries in $BUILD_DIR/"
ls -la $BUILD_DIR/
