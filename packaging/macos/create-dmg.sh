#!/bin/bash
# Creates a DMG installer for Mind Palace CLI
# Usage: ./create-dmg.sh <binary-path> <version> <output-name>

set -e

BINARY_PATH="${1:?Binary path required}"
VERSION="${2:?Version required}"
OUTPUT_NAME="${3:?Output name required}"

# Create temporary directory structure
STAGING_DIR=$(mktemp -d)
DMG_DIR="$STAGING_DIR/dmg"
mkdir -p "$DMG_DIR"

# Copy binary
cp "$BINARY_PATH" "$DMG_DIR/palace"
chmod +x "$DMG_DIR/palace"

# Create install script
cat > "$DMG_DIR/install.command" << 'INSTALL_EOF'
#!/bin/bash
# Mind Palace Installer
# This script copies the palace binary to /usr/local/bin

set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "╔══════════════════════════════════════════════╗"
echo "║        Mind Palace CLI Installer             ║"
echo "╚══════════════════════════════════════════════╝"
echo ""

# Check for /usr/local/bin and create if needed
if [ ! -d "/usr/local/bin" ]; then
    echo "Creating /usr/local/bin..."
    sudo mkdir -p /usr/local/bin
fi

# Copy binary
echo "Installing palace to /usr/local/bin..."
sudo cp "$SCRIPT_DIR/palace" /usr/local/bin/palace
sudo chmod +x /usr/local/bin/palace

# Verify installation
if command -v palace &> /dev/null; then
    echo ""
    echo "[OK] Installation successful!"
    echo ""
    palace version
    echo ""
    echo "You can now use 'palace' from any terminal."
else
    echo ""
    echo "[WARNING]  Installation completed, but 'palace' is not in PATH."
    echo "   You may need to restart your terminal or add /usr/local/bin to PATH."
fi

echo ""
echo "Press any key to close..."
read -n 1
INSTALL_EOF

chmod +x "$DMG_DIR/install.command"

# Create README
cat > "$DMG_DIR/README.txt" << README_EOF
Mind Palace CLI - Version $VERSION

INSTALLATION
============
Double-click 'install.command' to install palace to /usr/local/bin.
You will be prompted for your password to copy the file.

MANUAL INSTALLATION
==================
1. Open Terminal
2. Run: sudo cp /Volumes/MindPalace/palace /usr/local/bin/palace
3. Run: sudo chmod +x /usr/local/bin/palace

VERIFICATION
============
After installation, open a new terminal and run:
  palace version

UNINSTALLATION
==============
To uninstall, run:
  sudo rm /usr/local/bin/palace

For more information, visit: https://github.com/koksalmehmet/mind-palace
README_EOF

# Create DMG
echo "Creating DMG..."
hdiutil create -volname "MindPalace" -srcfolder "$DMG_DIR" -ov -format UDZO "$OUTPUT_NAME"

# Cleanup
rm -rf "$STAGING_DIR"

echo "Created: $OUTPUT_NAME"
