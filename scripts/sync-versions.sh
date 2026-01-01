#!/bin/bash

# Get the directory of the script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$( dirname "$SCRIPT_DIR" )"

# Read version from root VERSION file
VERSION=$(cat "$ROOT_DIR/VERSION" | tr -d '[:space:]')

if [ -z "$VERSION" ]; then
    echo "Error: VERSION file is empty"
    exit 1
fi

echo "Syncing ecosystem to version: $VERSION"

# 1. Update Go version (CLI)
CLI_VERSION_FILE="$ROOT_DIR/apps/cli/internal/cli/version.go"
if [ -f "$CLI_VERSION_FILE" ]; then
    echo "Updating $CLI_VERSION_FILE..."
    sed -i '' "s/buildVersion = \".*\"/buildVersion = \"$VERSION\"/" "$CLI_VERSION_FILE"
fi

# 2. Update Dashboard version
DASHBOARD_PKC="$ROOT_DIR/apps/dashboard/package.json"
if [ -f "$DASHBOARD_PKC" ]; then
    echo "Updating $DASHBOARD_PKC..."
    sed -i '' "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" "$DASHBOARD_PKC"
fi

# 3. Update VS Code extension version
VSCODE_PKC="$ROOT_DIR/apps/vscode/package.json"
if [ -f "$VSCODE_PKC" ]; then
    echo "Updating $VSCODE_PKC..."
    sed -i '' "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" "$VSCODE_PKC"
fi

# 4. Update Docs version
DOCS_PKC="$ROOT_DIR/apps/docs/package.json"
if [ -f "$DOCS_PKC" ]; then
    echo "Updating $DOCS_PKC..."
    sed -i '' "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" "$DOCS_PKC"
fi

echo "✅ Ecosystem version sync complete."
