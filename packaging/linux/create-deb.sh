#!/bin/bash
# Creates a .deb package for Mind Palace CLI
# Usage: ./create-deb.sh <binary-path> <version> <architecture> <output-name>

set -e

BINARY_PATH="${1:?Binary path required}"
VERSION="${2:?Version required}"
ARCH="${3:?Architecture required}"  # amd64 or arm64
OUTPUT_NAME="${4:?Output name required}"

# Map architecture names
case "$ARCH" in
    amd64) DEB_ARCH="amd64" ;;
    arm64) DEB_ARCH="arm64" ;;
    *) echo "Unknown architecture: $ARCH"; exit 1 ;;
esac

# Create package directory structure
PKG_DIR=$(mktemp -d)
mkdir -p "$PKG_DIR/DEBIAN"
mkdir -p "$PKG_DIR/usr/local/bin"
mkdir -p "$PKG_DIR/usr/share/doc/palace"

# Copy binary
cp "$BINARY_PATH" "$PKG_DIR/usr/local/bin/palace"
chmod 755 "$PKG_DIR/usr/local/bin/palace"

# Create control file
cat > "$PKG_DIR/DEBIAN/control" << CONTROL_EOF
Package: palace
Version: ${VERSION#v}
Section: devel
Priority: optional
Architecture: $DEB_ARCH
Maintainer: Mind Palace <hello@mindpalace.dev>
Homepage: https://github.com/koksalmehmet/mind-palace
Description: AI-Friendly Codebase Context Manager
 Mind Palace is a CLI tool that helps you manage your codebase context
 for AI-assisted development. It provides intelligent code analysis,
 context collection, and seamless integration with AI coding assistants.
 .
 Features:
  - Intelligent code scanning and indexing
  - Context pack generation for AI assistants
  - Built-in search with FTS5
  - MCP server integration
  - Interactive dashboard
CONTROL_EOF

# Create copyright file
cat > "$PKG_DIR/usr/share/doc/palace/copyright" << COPYRIGHT_EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: palace
Upstream-Contact: Mind Palace <hello@mindpalace.dev>
Source: https://github.com/koksalmehmet/mind-palace

Files: *
Copyright: 2024 Mind Palace
License: Apache-2.0
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
 .
     http://www.apache.org/licenses/LICENSE-2.0
 .
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
COPYRIGHT_EOF

# Create changelog
cat > "$PKG_DIR/usr/share/doc/palace/changelog.Debian" << CHANGELOG_EOF
palace (${VERSION#v}) unstable; urgency=low

  * Release ${VERSION}

 -- Mind Palace <hello@mindpalace.dev>  $(date -R)
CHANGELOG_EOF
gzip -9 "$PKG_DIR/usr/share/doc/palace/changelog.Debian"

# Create postinst script (optional - for showing installation message)
cat > "$PKG_DIR/DEBIAN/postinst" << 'POSTINST_EOF'
#!/bin/bash
set -e
echo ""
echo "╔══════════════════════════════════════════════╗"
echo "║   Mind Palace installed successfully!        ║"
echo "╚══════════════════════════════════════════════╝"
echo ""
echo "Run 'palace version' to verify installation."
echo "Run 'palace init' to initialize a new project."
echo ""
POSTINST_EOF
chmod 755 "$PKG_DIR/DEBIAN/postinst"

# Build the package
dpkg-deb --build --root-owner-group "$PKG_DIR" "$OUTPUT_NAME"

# Cleanup
rm -rf "$PKG_DIR"

echo "Created: $OUTPUT_NAME"
