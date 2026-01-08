#!/usr/bin/env pwsh
# Mind Palace - Interactive Development Menu
# Usage: .\scripts\dev.ps1
#
# Single-key selection - no Enter required!

$ErrorActionPreference = 'Stop'

# Ensure we're in project root
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
Set-Location $ProjectRoot

function Clear-Screen {
    Clear-Host
}

function Write-Header {
    Write-Host ""
    Write-Host "  Mind Palace - Development Console" -ForegroundColor Cyan
    Write-Host "  =================================" -ForegroundColor DarkCyan
    Write-Host "  Press a key to select (no Enter needed)" -ForegroundColor DarkGray
    Write-Host ""
}

function Write-Menu {
    Write-Host "  BUILD" -ForegroundColor Yellow
    Write-Host "    [1] Build All (dashboard + vscode + cli)" -ForegroundColor White
    Write-Host "    [2] Build CLI only" -ForegroundColor White
    Write-Host "    [3] Build Dashboard only" -ForegroundColor White
    Write-Host "    [4] Build VS Code extension only" -ForegroundColor White
    Write-Host "    [5] Build Release (optimized)" -ForegroundColor White
    Write-Host ""
    Write-Host "  TEST" -ForegroundColor Yellow
    Write-Host "    [t] Run All Tests" -ForegroundColor White
    Write-Host "    [g] Run Go Tests only" -ForegroundColor White
    Write-Host "    [d] Run Dashboard Tests only" -ForegroundColor White
    Write-Host "    [v] Run VS Code Tests only" -ForegroundColor White
    Write-Host ""
    Write-Host "  DEVELOPMENT" -ForegroundColor Yellow
    Write-Host "    [r] Run palace (dev mode)" -ForegroundColor White
    Write-Host "    [s] Start dashboard dev server" -ForegroundColor White
    Write-Host "    [w] Watch VS Code extension" -ForegroundColor White
    Write-Host ""
    Write-Host "  UTILITIES" -ForegroundColor Yellow
    Write-Host "    [i] Install all dependencies" -ForegroundColor White
    Write-Host "    [c] Clean build artifacts" -ForegroundColor White
    Write-Host "    [y] Sync versions" -ForegroundColor White
    Write-Host "    [l] Run linters" -ForegroundColor White
    Write-Host ""
    Write-Host "    [q] Quit" -ForegroundColor DarkGray
    Write-Host ""
}

function Read-SingleKey {
    $key = $host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    return $key.Character
}

function Invoke-WithPause {
    param([scriptblock]$Action, [string]$Name)

    Clear-Screen
    Write-Host ""
    Write-Host "  Running: $Name" -ForegroundColor Cyan
    Write-Host "  " + ("-" * 50) -ForegroundColor DarkGray
    Write-Host ""

    try {
        & $Action
        Write-Host ""
        Write-Host "  [OK] $Name completed" -ForegroundColor Green
    }
    catch {
        Write-Host ""
        Write-Host "  [ERROR] $Name failed: $_" -ForegroundColor Red
    }

    Write-Host ""
    Write-Host "  Press any key to continue..." -ForegroundColor DarkGray
    $null = Read-SingleKey
}

function Build-All {
    & "$ScriptDir\build.ps1" all
}

function Build-CLI {
    & "$ScriptDir\build.ps1" cli
}

function Build-Dashboard {
    & "$ScriptDir\build.ps1" dashboard
}

function Build-VSCode {
    & "$ScriptDir\build.ps1" vscode
}

function Build-Release {
    & "$ScriptDir\build.ps1" release
}

function Test-All {
    & "$ScriptDir\test-all.ps1"
}

function Test-Go {
    Write-Host "Running Go tests..." -ForegroundColor Cyan
    go test -v ./apps/cli/...
}

function Test-Dashboard {
    if (Test-Path "apps\dashboard\node_modules") {
        Push-Location apps\dashboard
        npm test -- --watch=false
        Pop-Location
    } else {
        Write-Host "Dashboard dependencies not installed. Run install first." -ForegroundColor Yellow
    }
}

function Test-VSCode {
    if (Test-Path "apps\vscode\node_modules") {
        Push-Location apps\vscode
        npm test
        Pop-Location
    } else {
        Write-Host "VS Code dependencies not installed. Run install first." -ForegroundColor Yellow
    }
}

function Run-Dev {
    Write-Host "Starting palace in dev mode..." -ForegroundColor Cyan
    Write-Host "Press Ctrl+C to stop" -ForegroundColor DarkGray
    go run ./apps/cli serve --dev
}

function Start-DashboardDev {
    Write-Host "Starting dashboard dev server..." -ForegroundColor Cyan
    Write-Host "Press Ctrl+C to stop" -ForegroundColor DarkGray
    Push-Location apps\dashboard
    npm start
    Pop-Location
}

function Watch-VSCode {
    Write-Host "Watching VS Code extension..." -ForegroundColor Cyan
    Write-Host "Press Ctrl+C to stop" -ForegroundColor DarkGray
    Push-Location apps\vscode
    npm run watch
    Pop-Location
}

function Install-Dependencies {
    Write-Host "Installing Go dependencies..." -ForegroundColor Cyan
    go mod download
    go mod tidy

    Write-Host ""
    Write-Host "Installing Dashboard dependencies..." -ForegroundColor Cyan
    Push-Location apps\dashboard
    npm install
    Pop-Location

    Write-Host ""
    Write-Host "Installing VS Code dependencies..." -ForegroundColor Cyan
    Push-Location apps\vscode
    npm install
    Pop-Location
}

function Clean-Artifacts {
    & "$ScriptDir\build.ps1" clean
}

function Sync-Versions {
    Write-Host "Syncing versions..." -ForegroundColor Cyan
    if ($IsWindows -or $env:OS -match "Windows") {
        # PowerShell version for Windows
        $version = (Get-Content VERSION).Trim()
        Write-Host "Version: $version" -ForegroundColor White

        # Update package.json files
        @("apps\dashboard\package.json", "apps\vscode\package.json", "apps\docs\package.json") | ForEach-Object {
            if (Test-Path $_) {
                $content = Get-Content $_ -Raw
                $content = $content -replace '"version":\s*"[^"]+"', "`"version`": `"$version`""
                Set-Content $_ $content -NoNewline
                Write-Host "Updated: $_" -ForegroundColor DarkGray
            }
        }
        Write-Host "[OK] Versions synced to $version" -ForegroundColor Green
    } else {
        bash scripts/sync-versions.sh
    }
}

function Run-Linters {
    Write-Host "Running Go linter..." -ForegroundColor Cyan
    $golangci = Get-Command golangci-lint -ErrorAction SilentlyContinue
    if ($golangci) {
        golangci-lint run ./...
    } else {
        Write-Host "golangci-lint not installed, running go vet..." -ForegroundColor Yellow
        go vet ./...
    }

    Write-Host ""
    Write-Host "Running Dashboard linter..." -ForegroundColor Cyan
    if (Test-Path "apps\dashboard\node_modules") {
        Push-Location apps\dashboard
        npm run lint 2>$null
        Pop-Location
    }

    Write-Host ""
    Write-Host "Running VS Code linter..." -ForegroundColor Cyan
    if (Test-Path "apps\vscode\node_modules") {
        Push-Location apps\vscode
        npm run lint 2>$null
        Pop-Location
    }
}

# Main loop
$running = $true
while ($running) {
    Clear-Screen
    Write-Header
    Write-Menu

    Write-Host "  > " -NoNewline -ForegroundColor Green
    $choice = Read-SingleKey

    switch ($choice) {
        '1' { Invoke-WithPause { Build-All } "Build All" }
        '2' { Invoke-WithPause { Build-CLI } "Build CLI" }
        '3' { Invoke-WithPause { Build-Dashboard } "Build Dashboard" }
        '4' { Invoke-WithPause { Build-VSCode } "Build VS Code" }
        '5' { Invoke-WithPause { Build-Release } "Build Release" }
        't' { Invoke-WithPause { Test-All } "Run All Tests" }
        'T' { Invoke-WithPause { Test-All } "Run All Tests" }
        'g' { Invoke-WithPause { Test-Go } "Go Tests" }
        'G' { Invoke-WithPause { Test-Go } "Go Tests" }
        'd' { Invoke-WithPause { Test-Dashboard } "Dashboard Tests" }
        'D' { Invoke-WithPause { Test-Dashboard } "Dashboard Tests" }
        'v' { Invoke-WithPause { Test-VSCode } "VS Code Tests" }
        'V' { Invoke-WithPause { Test-VSCode } "VS Code Tests" }
        'r' { Invoke-WithPause { Run-Dev } "Palace Dev Mode" }
        'R' { Invoke-WithPause { Run-Dev } "Palace Dev Mode" }
        's' { Invoke-WithPause { Start-DashboardDev } "Dashboard Dev Server" }
        'S' { Invoke-WithPause { Start-DashboardDev } "Dashboard Dev Server" }
        'w' { Invoke-WithPause { Watch-VSCode } "VS Code Watch" }
        'W' { Invoke-WithPause { Watch-VSCode } "VS Code Watch" }
        'i' { Invoke-WithPause { Install-Dependencies } "Install Dependencies" }
        'I' { Invoke-WithPause { Install-Dependencies } "Install Dependencies" }
        'c' { Invoke-WithPause { Clean-Artifacts } "Clean Artifacts" }
        'C' { Invoke-WithPause { Clean-Artifacts } "Clean Artifacts" }
        'y' { Invoke-WithPause { Sync-Versions } "Sync Versions" }
        'Y' { Invoke-WithPause { Sync-Versions } "Sync Versions" }
        'l' { Invoke-WithPause { Run-Linters } "Run Linters" }
        'L' { Invoke-WithPause { Run-Linters } "Run Linters" }
        'q' { $running = $false }
        'Q' { $running = $false }
        ([char]27) { $running = $false }  # Escape key
        default {
            # Invalid key - just refresh menu
        }
    }
}

Clear-Screen
Write-Host ""
Write-Host "  Goodbye!" -ForegroundColor Cyan
Write-Host ""
