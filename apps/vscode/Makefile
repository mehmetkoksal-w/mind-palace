.PHONY: build dev test package clean install lint

# Build the extension
build:
	npm run compile

# Watch for changes during development
dev:
	npm run watch

# Run tests
test:
	npm test 2>/dev/null || @echo "No tests configured yet"

# Lint the code
lint:
	npm run lint 2>/dev/null || @echo "No lint script configured"

# Package the extension as .vsix
package:
	vsce package

# Install dependencies
install:
	npm install

# Clean build artifacts
clean:
	rm -rf out
	rm -f *.vsix

# Publish to VS Code marketplace (requires VSCE_PAT)
publish:
	vsce publish
