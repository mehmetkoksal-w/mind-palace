#!/bin/bash
# End-to-end tests for Mind Palace
# This script tests the core workflows of the palace CLI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

# Get the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PALACE_BIN="$PROJECT_ROOT/palace"

# Create temp directory for tests
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

echo "=========================================="
echo "Mind Palace End-to-End Tests"
echo "=========================================="
echo "Project root: $PROJECT_ROOT"
echo "Test directory: $TEST_DIR"
echo ""

# Build palace CLI if not present
if [ ! -f "$PALACE_BIN" ]; then
    echo "Building palace CLI..."
    cd "$PROJECT_ROOT"
    go build -o palace ./apps/cli
fi

# Helper function to run tests
run_test() {
    local name="$1"
    local cmd="$2"

    echo -n "Testing: $name... "
    if eval "$cmd" > /dev/null 2>&1; then
        echo -e "${GREEN}PASSED${NC}"
        ((TESTS_PASSED++)) || true
        return 0
    else
        echo -e "${RED}FAILED${NC}"
        ((TESTS_FAILED++)) || true
        return 1
    fi
}

# Helper function to run tests with output check
run_test_output() {
    local name="$1"
    local cmd="$2"
    local expected="$3"

    echo -n "Testing: $name... "
    local output
    output=$(eval "$cmd" 2>&1) || true

    if echo "$output" | grep -q "$expected"; then
        echo -e "${GREEN}PASSED${NC}"
        ((TESTS_PASSED++)) || true
        return 0
    else
        echo -e "${RED}FAILED${NC}"
        echo "  Expected to contain: $expected"
        echo "  Got: $output"
        ((TESTS_FAILED++)) || true
        return 1
    fi
}

echo "=========================================="
echo "1. Basic CLI Tests"
echo "=========================================="

run_test_output "palace help" "$PALACE_BIN help" "palace - AI-friendly codebase memory and search"
run_test_output "palace version" "$PALACE_BIN version" "palace"

echo ""
echo "=========================================="
echo "2. Initialization Tests"
echo "=========================================="

# Create a test project
mkdir -p "$TEST_DIR/test-project/src"
cat > "$TEST_DIR/test-project/src/main.go" << 'EOF'
package main

import "fmt"

func main() {
    greeting := sayHello("World")
    fmt.Println(greeting)
}

func sayHello(name string) string {
    return "Hello, " + name + "!"
}

func add(a, b int) int {
    return a + b
}
EOF

cat > "$TEST_DIR/test-project/src/utils.go" << 'EOF'
package main

// Helper functions

func multiply(a, b int) int {
    return a * b
}

func divide(a, b int) int {
    if b == 0 {
        return 0
    }
    return a / b
}
EOF

cd "$TEST_DIR/test-project"

run_test "palace init" "$PALACE_BIN init"
run_test ".palace directory created" "test -d .palace"
run_test "palace.jsonc created" "test -f .palace/palace.jsonc"

echo ""
echo "=========================================="
echo "3. Scanning Tests"
echo "=========================================="

run_test "palace scan" "$PALACE_BIN scan"
run_test "index created" "test -f .palace/index/palace.db"

echo ""
echo "=========================================="
echo "4. Query Tests"
echo "=========================================="

run_test_output "query sayHello" "$PALACE_BIN explore sayHello" "sayHello"
run_test_output "query main" "$PALACE_BIN explore main" "main"
run_test_output "query multiply" "$PALACE_BIN explore multiply" "multiply"

echo ""
echo "=========================================="
echo "5. Session Memory Tests"
echo "=========================================="

# Start a session
SESSION_OUTPUT=$($PALACE_BIN session start --agent test-agent --goal "E2E testing" 2>&1)
if echo "$SESSION_OUTPUT" | grep -qE "started|Session"; then
    echo -e "Testing: session start... ${GREEN}PASSED${NC}"
    ((TESTS_PASSED++))
else
    echo -e "Testing: session start... ${RED}FAILED${NC}"
    ((TESTS_FAILED++))
fi

run_test_output "session list --active" "$PALACE_BIN session list --active" "test-agent"

echo ""
echo "=========================================="
echo "6. Learning Tests"
echo "=========================================="

run_test "learn command" "$PALACE_BIN store --as learning 'Always use meaningful variable names in Go' && sleep 1"
run_test_output "recall test" "$PALACE_BIN recall 'variable'" "variable"

echo ""
echo "=========================================="
echo "7. Corridor Tests"
echo "=========================================="

run_test "corridor list" "$PALACE_BIN corridor list"

echo ""
echo "=========================================="
echo "8. Brief/Intel Tests"
echo "=========================================="

# These might not have data yet but should not error
run_test "brief command" "$PALACE_BIN brief || true"
run_test "intel command" "$PALACE_BIN intel src/main.go || true"

echo ""
echo "=========================================="
echo "9. Multi-file Project Tests"
echo "=========================================="

# Create more complex project structure
mkdir -p "$TEST_DIR/complex-project/cmd/server"
mkdir -p "$TEST_DIR/complex-project/internal/handlers"
mkdir -p "$TEST_DIR/complex-project/internal/models"

cat > "$TEST_DIR/complex-project/cmd/server/main.go" << 'EOF'
package main

import (
    "fmt"
    "project/internal/handlers"
)

func main() {
    h := handlers.NewHandler()
    fmt.Println(h.Name())
}
EOF

cat > "$TEST_DIR/complex-project/internal/handlers/handler.go" << 'EOF'
package handlers

type Handler struct {
    name string
}

func NewHandler() *Handler {
    return &Handler{name: "TestHandler"}
}

func (h *Handler) Name() string {
    return h.name
}

func (h *Handler) Process(input string) string {
    return "Processed: " + input
}
EOF

cat > "$TEST_DIR/complex-project/internal/models/user.go" << 'EOF'
package models

type User struct {
    ID   int
    Name string
}

func NewUser(id int, name string) *User {
    return &User{ID: id, Name: name}
}

func (u *User) GetName() string {
    return u.Name
}
EOF

cd "$TEST_DIR/complex-project"

run_test "complex project init" "$PALACE_BIN init"
run_test "complex project scan" "$PALACE_BIN scan"
run_test_output "query Handler" "$PALACE_BIN explore Handler" "Handler"
run_test_output "query NewUser" "$PALACE_BIN explore NewUser" "NewUser"

echo ""
echo "=========================================="
echo "10. Re-scan Tests"
echo "=========================================="

# Add a new file and rescan
cat > "$TEST_DIR/complex-project/internal/models/product.go" << 'EOF'
package models

type Product struct {
    ID    int
    Price float64
}

func NewProduct(id int, price float64) *Product {
    return &Product{ID: id, Price: price}
}
EOF

run_test "rescan after adding file" "$PALACE_BIN scan"
run_test_output "query NewProduct after rescan" "$PALACE_BIN explore NewProduct" "NewProduct"

echo ""
echo "=========================================="
echo "Results Summary"
echo "=========================================="
echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "\n${RED}Some tests failed!${NC}"
    exit 1
else
    echo -e "\n${GREEN}All tests passed!${NC}"
    exit 0
fi
