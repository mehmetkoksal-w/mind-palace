#!/bin/bash
# Mind Palace - Run All Tests
# Usage: ./scripts/test-all.sh [-c|--coverage] [-v|--verbose]

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
GRAY='\033[0;90m'
NC='\033[0m'

# Parse arguments
COVERAGE=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--coverage) COVERAGE=true; shift ;;
        -v|--verbose) VERBOSE=true; shift ;;
        *) shift ;;
    esac
done

# Counters
PASSED_SUITES=0
FAILED_SUITES=0

# Get project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

write_section() {
    echo ""
    echo -e "${GRAY}============================================================${NC}"
    echo -e "${CYAN} $1${NC}"
    echo -e "${GRAY}============================================================${NC}"
}

write_result() {
    local name="$1"
    local success="$2"
    local details="$3"

    if [ "$success" = "true" ]; then
        echo -e "${GREEN}[PASS]${NC} $name"
        ((PASSED_SUITES++))
    else
        echo -e "${RED}[FAIL]${NC} $name"
        ((FAILED_SUITES++))
    fi

    if [ -n "$details" ]; then
        echo -e "${GRAY}       $details${NC}"
    fi
}

echo ""
echo -e "${CYAN}Mind Palace - Test Suite${NC}"
echo -e "${GRAY}Project root: $PROJECT_ROOT${NC}"

# =============================================================================
# Go Tests
# =============================================================================
write_section "Go Tests"

GO_ARGS="test"
if [ "$VERBOSE" = "true" ]; then GO_ARGS="$GO_ARGS -v"; fi
if [ "$COVERAGE" = "true" ]; then GO_ARGS="$GO_ARGS -coverprofile=coverage.out"; fi
GO_ARGS="$GO_ARGS ./apps/cli/..."

echo -e "${GRAY}Running: go $GO_ARGS${NC}"

if GO_OUTPUT=$(go $GO_ARGS 2>&1); then
    GO_SUCCESS=true
else
    GO_SUCCESS=false
fi

if [ "$VERBOSE" = "true" ] || [ "$GO_SUCCESS" = "false" ]; then
    echo "$GO_OUTPUT"
fi

OK_COUNT=$(echo "$GO_OUTPUT" | grep -c "^ok" || true)
FAIL_COUNT=$(echo "$GO_OUTPUT" | grep -c "^FAIL" || true)

write_result "Go Unit Tests" "$GO_SUCCESS" "$OK_COUNT packages passed, $FAIL_COUNT failed"

if [ "$COVERAGE" = "true" ] && [ "$GO_SUCCESS" = "true" ]; then
    echo -e "${GRAY}Generating coverage report...${NC}"
    go tool cover -func=coverage.out | tail -1
fi

# =============================================================================
# Dashboard Tests (Angular)
# =============================================================================
write_section "Dashboard Tests (Angular)"

if [ -d "apps/dashboard/node_modules" ]; then
    cd apps/dashboard
    if DASH_OUTPUT=$(npm test -- --watch=false --browsers=ChromeHeadless 2>&1); then
        DASH_SUCCESS=true
    else
        DASH_SUCCESS=false
    fi
    cd "$PROJECT_ROOT"

    if [ "$VERBOSE" = "true" ] || [ "$DASH_SUCCESS" = "false" ]; then
        echo "$DASH_OUTPUT"
    fi

    write_result "Dashboard Tests" "$DASH_SUCCESS"
else
    echo -e "${YELLOW}[SKIP]${NC} Dashboard tests - node_modules not installed"
    echo -e "${GRAY}       Run 'npm install' in apps/dashboard first${NC}"
fi

# =============================================================================
# VS Code Extension Tests
# =============================================================================
write_section "VS Code Extension Tests"

if [ -d "apps/vscode/node_modules" ]; then
    cd apps/vscode
    if VSCODE_OUTPUT=$(npm test 2>&1); then
        VSCODE_SUCCESS=true
    else
        VSCODE_SUCCESS=false
    fi
    cd "$PROJECT_ROOT"

    if [ "$VERBOSE" = "true" ] || [ "$VSCODE_SUCCESS" = "false" ]; then
        echo "$VSCODE_OUTPUT"
    fi

    write_result "VS Code Extension Tests" "$VSCODE_SUCCESS"
else
    echo -e "${YELLOW}[SKIP]${NC} VS Code tests - node_modules not installed"
    echo -e "${GRAY}       Run 'npm install' in apps/vscode first${NC}"
fi

# =============================================================================
# Summary
# =============================================================================
write_section "Test Summary"

echo ""
echo -e "${GREEN}Suites Passed: $PASSED_SUITES${NC}"
if [ "$FAILED_SUITES" -gt 0 ]; then
    echo -e "${RED}Suites Failed: $FAILED_SUITES${NC}"
else
    echo -e "${GREEN}Suites Failed: $FAILED_SUITES${NC}"
fi
echo ""

if [ "$FAILED_SUITES" -gt 0 ]; then
    echo -e "${RED}[FAILED] Some test suites failed${NC}"
    exit 1
else
    echo -e "${GREEN}[OK] All test suites passed${NC}"
    exit 0
fi
