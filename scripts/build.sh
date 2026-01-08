#!/bin/bash
# Mind Palace Build Script
# Usage: ./scripts/build.sh [target]
# Targets: all, cli, dashboard, vscode, test, clean, release

set -e

TARGET="${1:-all}"

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# Get version info
VERSION=$(cat VERSION 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS="-s -w -X github.com/koksalmehmet/mind-palace/apps/cli/internal/cli.buildVersion=$VERSION -X github.com/koksalmehmet/mind-palace/apps/cli/internal/cli.buildCommit=$COMMIT -X github.com/koksalmehmet/mind-palace/apps/cli/internal/cli.buildDate=$DATE"

build_dashboard() {
    echo -e "${CYAN}Building dashboard...${NC}"

    if [ ! -d "apps/dashboard/node_modules" ]; then
        echo -e "${YELLOW}Installing dashboard dependencies...${NC}"
        cd apps/dashboard && npm install && cd "$PROJECT_ROOT"
    fi

    cd apps/dashboard && npm run build && cd "$PROJECT_ROOT"

    # Copy dashboard build to CLI embed location
    echo -e "${CYAN}Embedding dashboard assets...${NC}"
    EMBED_DIR="apps/cli/internal/dashboard/dist"

    rm -rf "$EMBED_DIR"
    mkdir -p "$EMBED_DIR"

    # Angular 17+ outputs to dist/dashboard/browser
    DASHBOARD_BUILD="apps/dashboard/dist/dashboard/browser"
    if [ -d "$DASHBOARD_BUILD" ]; then
        cp -r "$DASHBOARD_BUILD"/* "$EMBED_DIR/"
        echo -e "${GREEN}[OK] Dashboard built and embedded${NC}"
    else
        echo -e "${RED}Dashboard build not found at $DASHBOARD_BUILD${NC}"
        exit 1
    fi
}

build_vscode() {
    echo -e "${CYAN}Building VS Code extension...${NC}"

    if [ ! -d "apps/vscode/node_modules" ]; then
        echo -e "${YELLOW}Installing VS Code extension dependencies...${NC}"
        cd apps/vscode && npm install && cd "$PROJECT_ROOT"
    fi

    cd apps/vscode && npm run compile && cd "$PROJECT_ROOT"

    echo -e "${GREEN}[OK] VS Code extension built${NC}"
}

build_cli() {
    local IS_RELEASE="${1:-false}"

    echo -e "${CYAN}Building palace CLI...${NC}"

    # Ensure dashboard is embedded first
    if [ ! -f "apps/cli/internal/dashboard/dist/index.html" ]; then
        echo -e "${YELLOW}Dashboard not embedded. Building dashboard first...${NC}"
        build_dashboard
    fi

    if [ "$IS_RELEASE" = "true" ]; then
        CGO_ENABLED=1 go build -ldflags="$LDFLAGS" -o palace ./apps/cli
    else
        go build -ldflags="$LDFLAGS" -o palace ./apps/cli
    fi

    if [ $? -eq 0 ]; then
        SIZE=$(du -h palace | cut -f1)
        echo -e "${GREEN}[OK] Palace CLI built: ./palace ($SIZE)${NC}"
    else
        echo -e "${RED}CLI build failed${NC}"
        exit 1
    fi
}

run_tests() {
    echo -e "${CYAN}Running all tests...${NC}"

    echo -e "\n${YELLOW}Go tests:${NC}"
    go test -v ./...

    if [ -d "apps/dashboard/node_modules" ]; then
        echo -e "\n${YELLOW}Dashboard tests:${NC}"
        cd apps/dashboard && npm test -- --watch=false && cd "$PROJECT_ROOT"
    fi

    if [ -d "apps/vscode/node_modules" ]; then
        echo -e "\n${YELLOW}VS Code tests:${NC}"
        cd apps/vscode && npm test && cd "$PROJECT_ROOT"
    fi
}

clean_build() {
    echo -e "${CYAN}Cleaning build artifacts...${NC}"

    rm -f palace palace.exe
    rm -rf apps/cli/internal/dashboard/dist
    rm -rf apps/dashboard/dist
    rm -rf apps/vscode/out

    echo -e "${GREEN}[OK] Clean complete${NC}"
}

# Main execution
case "$TARGET" in
    dashboard)
        build_dashboard
        ;;
    vscode)
        build_vscode
        ;;
    cli)
        build_cli
        ;;
    test)
        run_tests
        ;;
    clean)
        clean_build
        ;;
    release)
        build_dashboard
        build_vscode
        build_cli true
        ;;
    all)
        build_dashboard
        build_vscode
        build_cli
        echo -e "\n${GREEN}[OK] Build complete!${NC}"
        ;;
    *)
        echo "Usage: $0 [all|cli|dashboard|vscode|test|clean|release]"
        exit 1
        ;;
esac
