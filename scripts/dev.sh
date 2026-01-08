#!/bin/bash
# Mind Palace - Interactive Development Menu
# Usage: ./scripts/dev.sh
#
# Single-key selection - no Enter required!

set -e

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# Colors
CYAN='\033[0;36m'
DCYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
WHITE='\033[1;37m'
GRAY='\033[0;90m'
RED='\033[0;31m'
NC='\033[0m'

clear_screen() {
    clear
}

write_header() {
    echo ""
    echo -e "  ${CYAN}Mind Palace - Development Console${NC}"
    echo -e "  ${DCYAN}=================================${NC}"
    echo -e "  ${GRAY}Press a key to select (no Enter needed)${NC}"
    echo ""
}

write_menu() {
    echo -e "  ${YELLOW}BUILD${NC}"
    echo -e "    ${WHITE}[1]${NC} Build All (dashboard + vscode + cli)"
    echo -e "    ${WHITE}[2]${NC} Build CLI only"
    echo -e "    ${WHITE}[3]${NC} Build Dashboard only"
    echo -e "    ${WHITE}[4]${NC} Build VS Code extension only"
    echo -e "    ${WHITE}[5]${NC} Build Release (optimized)"
    echo ""
    echo -e "  ${YELLOW}TEST${NC}"
    echo -e "    ${WHITE}[t]${NC} Run All Tests"
    echo -e "    ${WHITE}[g]${NC} Run Go Tests only"
    echo -e "    ${WHITE}[d]${NC} Run Dashboard Tests only"
    echo -e "    ${WHITE}[v]${NC} Run VS Code Tests only"
    echo ""
    echo -e "  ${YELLOW}DEVELOPMENT${NC}"
    echo -e "    ${WHITE}[r]${NC} Run palace (dev mode)"
    echo -e "    ${WHITE}[s]${NC} Start dashboard dev server"
    echo -e "    ${WHITE}[w]${NC} Watch VS Code extension"
    echo ""
    echo -e "  ${YELLOW}UTILITIES${NC}"
    echo -e "    ${WHITE}[i]${NC} Install all dependencies"
    echo -e "    ${WHITE}[c]${NC} Clean build artifacts"
    echo -e "    ${WHITE}[y]${NC} Sync versions"
    echo -e "    ${WHITE}[l]${NC} Run linters"
    echo ""
    echo -e "    ${GRAY}[q] Quit${NC}"
    echo ""
}

read_single_key() {
    read -n 1 -s key
    echo "$key"
}

invoke_with_pause() {
    local name="$1"
    shift

    clear_screen
    echo ""
    echo -e "  ${CYAN}Running: $name${NC}"
    echo -e "  ${GRAY}--------------------------------------------------${NC}"
    echo ""

    if "$@"; then
        echo ""
        echo -e "  ${GREEN}[OK] $name completed${NC}"
    else
        echo ""
        echo -e "  ${RED}[ERROR] $name failed${NC}"
    fi

    echo ""
    echo -e "  ${GRAY}Press any key to continue...${NC}"
    read -n 1 -s
}

do_build_all() {
    "$SCRIPT_DIR/build.sh" all
}

do_build_cli() {
    "$SCRIPT_DIR/build.sh" cli
}

do_build_dashboard() {
    "$SCRIPT_DIR/build.sh" dashboard
}

do_build_vscode() {
    "$SCRIPT_DIR/build.sh" vscode
}

do_build_release() {
    "$SCRIPT_DIR/build.sh" release
}

do_test_all() {
    "$SCRIPT_DIR/test-all.sh"
}

do_test_go() {
    echo -e "${CYAN}Running Go tests...${NC}"
    go test -v ./apps/cli/...
}

do_test_dashboard() {
    if [ -d "apps/dashboard/node_modules" ]; then
        cd apps/dashboard && npm test -- --watch=false && cd "$PROJECT_ROOT"
    else
        echo -e "${YELLOW}Dashboard dependencies not installed. Run install first.${NC}"
    fi
}

do_test_vscode() {
    if [ -d "apps/vscode/node_modules" ]; then
        cd apps/vscode && npm test && cd "$PROJECT_ROOT"
    else
        echo -e "${YELLOW}VS Code dependencies not installed. Run install first.${NC}"
    fi
}

do_run_dev() {
    echo -e "${CYAN}Starting palace in dev mode...${NC}"
    echo -e "${GRAY}Press Ctrl+C to stop${NC}"
    go run ./apps/cli serve --dev
}

do_start_dashboard() {
    echo -e "${CYAN}Starting dashboard dev server...${NC}"
    echo -e "${GRAY}Press Ctrl+C to stop${NC}"
    cd apps/dashboard && npm start
}

do_watch_vscode() {
    echo -e "${CYAN}Watching VS Code extension...${NC}"
    echo -e "${GRAY}Press Ctrl+C to stop${NC}"
    cd apps/vscode && npm run watch
}

do_install_deps() {
    echo -e "${CYAN}Installing Go dependencies...${NC}"
    go mod download
    go mod tidy

    echo ""
    echo -e "${CYAN}Installing Dashboard dependencies...${NC}"
    cd apps/dashboard && npm install && cd "$PROJECT_ROOT"

    echo ""
    echo -e "${CYAN}Installing VS Code dependencies...${NC}"
    cd apps/vscode && npm install && cd "$PROJECT_ROOT"
}

do_clean() {
    "$SCRIPT_DIR/build.sh" clean
}

do_sync_versions() {
    "$SCRIPT_DIR/sync-versions.sh"
}

do_run_linters() {
    echo -e "${CYAN}Running Go linter...${NC}"
    if command -v golangci-lint &> /dev/null; then
        golangci-lint run ./...
    else
        echo -e "${YELLOW}golangci-lint not installed, running go vet...${NC}"
        go vet ./...
    fi

    echo ""
    echo -e "${CYAN}Running Dashboard linter...${NC}"
    if [ -d "apps/dashboard/node_modules" ]; then
        cd apps/dashboard && npm run lint 2>/dev/null || true && cd "$PROJECT_ROOT"
    fi

    echo ""
    echo -e "${CYAN}Running VS Code linter...${NC}"
    if [ -d "apps/vscode/node_modules" ]; then
        cd apps/vscode && npm run lint 2>/dev/null || true && cd "$PROJECT_ROOT"
    fi
}

# Main loop
running=true
while $running; do
    clear_screen
    write_header
    write_menu

    echo -ne "  ${GREEN}> ${NC}"
    choice=$(read_single_key)

    case "$choice" in
        1) invoke_with_pause "Build All" do_build_all ;;
        2) invoke_with_pause "Build CLI" do_build_cli ;;
        3) invoke_with_pause "Build Dashboard" do_build_dashboard ;;
        4) invoke_with_pause "Build VS Code" do_build_vscode ;;
        5) invoke_with_pause "Build Release" do_build_release ;;
        t|T) invoke_with_pause "Run All Tests" do_test_all ;;
        g|G) invoke_with_pause "Go Tests" do_test_go ;;
        d|D) invoke_with_pause "Dashboard Tests" do_test_dashboard ;;
        v|V) invoke_with_pause "VS Code Tests" do_test_vscode ;;
        r|R) invoke_with_pause "Palace Dev Mode" do_run_dev ;;
        s|S) invoke_with_pause "Dashboard Dev Server" do_start_dashboard ;;
        w|W) invoke_with_pause "VS Code Watch" do_watch_vscode ;;
        i|I) invoke_with_pause "Install Dependencies" do_install_deps ;;
        c|C) invoke_with_pause "Clean Artifacts" do_clean ;;
        y|Y) invoke_with_pause "Sync Versions" do_sync_versions ;;
        l|L) invoke_with_pause "Run Linters" do_run_linters ;;
        q|Q) running=false ;;
        $'\e') running=false ;;  # Escape key
        *) ;;  # Invalid key - just refresh menu
    esac
done

clear_screen
echo ""
echo -e "  ${CYAN}Goodbye!${NC}"
echo ""
