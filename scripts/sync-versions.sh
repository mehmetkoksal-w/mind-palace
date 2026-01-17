#!/bin/bash
set -e  # Exit on error

# Get the directory of the script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$( dirname "$SCRIPT_DIR" )"

# Read version from root VERSION file
VERSION=$(cat "$ROOT_DIR/VERSION" | tr -d '[:space:]')

if [ -z "$VERSION" ]; then
    echo "[ERROR] Error: VERSION file is empty"
    exit 1
fi

# Validate version format
if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
    echo "[ERROR] Error: Invalid VERSION format: $VERSION"
    echo "Expected format: X.Y.Z or X.Y.Z-suffix"
    exit 1
fi

echo "Syncing ecosystem to version: $VERSION"

# 1. Update Go version (CLI) - NO LONGER NEEDED (Dynamic detector in internal/cli/version.go)
# CLI_VERSION_FILE="$ROOT_DIR/apps/cli/internal/cli/version.go"
# if [ -f "$CLI_VERSION_FILE" ]; then
#     echo "Updating $CLI_VERSION_FILE..."
#     sed -i '' "s/buildVersion = \".*\"/buildVersion = \"$VERSION\"/" "$CLI_VERSION_FILE"
# fi

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

echo ""
echo "Verifying synchronization..."

# Verify all versions match
ERRORS=0

if [ -f "$DASHBOARD_PKC" ]; then
    DASH_VER=$(grep -oP '"version":\s*"\K[^"]+' "$DASHBOARD_PKC" || echo "")
    if [ "$DASH_VER" != "$VERSION" ]; then
        echo "[ERROR] Dashboard version mismatch: $DASH_VER (expected: $VERSION)"
        ERRORS=$((ERRORS + 1))
    else
        echo "[OK] Dashboard: $DASH_VER"
    fi
fi

if [ -f "$VSCODE_PKC" ]; then
    VSCODE_VER=$(grep -oP '"version":\s*"\K[^"]+' "$VSCODE_PKC" || echo "")
    if [ "$VSCODE_VER" != "$VERSION" ]; then
        echo "[ERROR] VS Code version mismatch: $VSCODE_VER (expected: $VERSION)"
        ERRORS=$((ERRORS + 1))
    else
        echo "[OK] VS Code: $VSCODE_VER"
    fi
fi

if [ -f "$DOCS_PKC" ]; then
    DOCS_VER=$(grep -oP '"version":\s*"\K[^"]+' "$DOCS_PKC" || echo "")
    if [ "$DOCS_VER" != "$VERSION" ]; then
        echo "[ERROR] Docs version mismatch: $DOCS_VER (expected: $VERSION)"
        ERRORS=$((ERRORS + 1))
    else
        echo "[OK] Docs: $DOCS_VER"
    fi
fi

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "[ERROR] Synchronization failed with $ERRORS error(s)"
    exit 1
fi

echo ""
echo "[OK] Ecosystem version sync complete - all versions synchronized to $VERSION"
