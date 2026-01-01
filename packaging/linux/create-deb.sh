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
License: PolyForm-Shield-1.0.0
 PolyForm Shield License 1.0.0
 .
 https://polyformproject.org/licenses/shield/1.0.0
 .
 1.  Rights Granted. Licensor grants you a non-exclusive, royalty-free,
     worldwide, non-sublicensable, non-transferable license to use, modify,
     and distribute the Software, provided that you do not use the Software
     to create, provide, or otherwise make available a Service that competes
     with the Software.
 .
 2.  Conditions.
     a.  If you distribute the Software, you must provide a copy of this
         license and retain all copyright, patent, trademark, and
         attribution notices.
     b.  You may not use the licensor's trademarks or logos except as
         required for reasonable and customary use in describing the
         origin of the Software.
 .
 3.  Limitations.
     a.  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
         EXPRESS OR IMPLIED.
     b.  IN NO EVENT SHALL LICENSOR BE LIABLE FOR ANY CLAIM, DAMAGES OR
         OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR
         OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE.
 .
 4.  Definitions.
     a.  "Licensor" means the copyright owner or entity authorized by the
         copyright owner that is granting the License.
     b.  "Software" means the Mind Palace software and documentation.
     c.  "Service" means a product or service that allows third parties to
         use the functionality of the Software.
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
