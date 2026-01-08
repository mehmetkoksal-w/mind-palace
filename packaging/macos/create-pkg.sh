#!/bin/bash
# Creates a PKG installer for Mind Palace CLI
# Usage: ./create-pkg.sh <binary-path> <version> <output-name>

set -e

BINARY_PATH="${1:?Binary path required}"
VERSION="${2:?Version required}"
OUTPUT_NAME="${3:?Output name required}"

# Remove 'v' prefix if present
VERSION="${VERSION#v}"

# Create temporary directory structure
STAGING_DIR=$(mktemp -d)
PAYLOAD_DIR="$STAGING_DIR/payload"
SCRIPTS_DIR="$STAGING_DIR/scripts"
RESOURCES_DIR="$STAGING_DIR/resources"

mkdir -p "$PAYLOAD_DIR/usr/local/bin"
mkdir -p "$SCRIPTS_DIR"
mkdir -p "$RESOURCES_DIR"

# Copy binary
cp "$BINARY_PATH" "$PAYLOAD_DIR/usr/local/bin/palace"
chmod +x "$PAYLOAD_DIR/usr/local/bin/palace"

# Create postinstall script (runs after installation)
cat > "$SCRIPTS_DIR/postinstall" << 'POSTINSTALL_EOF'
#!/bin/bash
# Ensure /usr/local/bin is in PATH for all users
if [ -f /etc/paths ]; then
    if ! grep -q "/usr/local/bin" /etc/paths; then
        echo "/usr/local/bin" >> /etc/paths
    fi
fi
exit 0
POSTINSTALL_EOF
chmod +x "$SCRIPTS_DIR/postinstall"

# Create welcome message
cat > "$RESOURCES_DIR/welcome.html" << 'WELCOME_EOF'
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; padding: 20px; }
        h1 { color: #6b21a8; }
        .feature { margin: 10px 0; }
    </style>
</head>
<body>
    <h1>Mind Palace</h1>
    <p>AI-native codebase memory for intelligent agents.</p>
    <div class="feature">
        <strong>What you'll get:</strong>
        <ul>
            <li>The <code>palace</code> CLI tool</li>
            <li>MCP server for AI agents</li>
            <li>Web dashboard for visualization</li>
        </ul>
    </div>
    <p>Click <strong>Continue</strong> to install Mind Palace.</p>
</body>
</html>
WELCOME_EOF

# Create conclusion message
cat > "$RESOURCES_DIR/conclusion.html" << 'CONCLUSION_EOF'
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; padding: 20px; }
        h1 { color: #22c55e; }
        code { background: #f3f4f6; padding: 2px 6px; border-radius: 4px; }
        .command { background: #1f2937; color: #e5e7eb; padding: 10px; border-radius: 6px; margin: 10px 0; }
    </style>
</head>
<body>
    <h1>Installation Complete!</h1>
    <p>Mind Palace has been installed successfully.</p>
    <p><strong>Get started:</strong></p>
    <div class="command">
        <code>palace init</code> &nbsp;&mdash;&nbsp; Initialize a project<br>
        <code>palace scan</code> &nbsp;&mdash;&nbsp; Index your codebase<br>
        <code>palace dashboard</code> &nbsp;&mdash;&nbsp; Open the dashboard
    </div>
    <p>Open a new Terminal window and run <code>palace --help</code> for more options.</p>
</body>
</html>
CONCLUSION_EOF

# Create license (MIT License)
cat > "$RESOURCES_DIR/license.txt" << 'LICENSE_EOF'
MIT License

Copyright (c) 2026 Mind Palace Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
LICENSE_EOF

# Create distribution XML for productbuild
cat > "$STAGING_DIR/distribution.xml" << DIST_EOF
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
    <title>Mind Palace</title>
    <organization>com.mindpalace</organization>
    <domains enable_localSystem="true"/>
    <options customize="never" require-scripts="true" rootVolumeOnly="true"/>

    <welcome file="welcome.html"/>
    <license file="license.txt"/>
    <conclusion file="conclusion.html"/>

    <choices-outline>
        <line choice="default">
            <line choice="com.mindpalace.palace"/>
        </line>
    </choices-outline>

    <choice id="default"/>
    <choice id="com.mindpalace.palace" visible="false">
        <pkg-ref id="com.mindpalace.palace"/>
    </choice>

    <pkg-ref id="com.mindpalace.palace" version="$VERSION" onConclusion="none">palace-component.pkg</pkg-ref>
</installer-gui-script>
DIST_EOF

# Build component package
echo "Building component package..."
pkgbuild \
    --root "$PAYLOAD_DIR" \
    --scripts "$SCRIPTS_DIR" \
    --identifier "com.mindpalace.palace" \
    --version "$VERSION" \
    --install-location "/" \
    "$STAGING_DIR/palace-component.pkg"

# Build final product package with GUI
echo "Building product package..."
productbuild \
    --distribution "$STAGING_DIR/distribution.xml" \
    --resources "$RESOURCES_DIR" \
    --package-path "$STAGING_DIR" \
    "$OUTPUT_NAME"

# Cleanup
rm -rf "$STAGING_DIR"

echo "Created: $OUTPUT_NAME"
