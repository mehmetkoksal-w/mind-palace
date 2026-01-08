#!/usr/bin/env pwsh
# Mind Palace Build Script for Windows
# Usage: .\build.ps1 [target]
# Targets: all, cli, dashboard, vscode, test, clean

param(
    [Parameter(Position = 0)]
    [ValidateSet('all', 'cli', 'dashboard', 'vscode', 'test', 'clean', 'release')]
    [string]$Target = 'all'
)

$ErrorActionPreference = 'Stop'

# Get version info
$VERSION = if (Test-Path VERSION) { (Get-Content VERSION).Trim() } else { "dev" }
$COMMIT = try { git rev-parse --short HEAD 2>$null } catch { "unknown" }
$DATE = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ" -AsUTC
$LDFLAGS = "-s -w -X github.com/koksalmehmet/mind-palace/apps/cli/internal/cli.buildVersion=$VERSION -X github.com/koksalmehmet/mind-palace/apps/cli/internal/cli.buildCommit=$COMMIT -X github.com/koksalmehmet/mind-palace/apps/cli/internal/cli.buildDate=$DATE"

function Build-Dashboard {
    Write-Host "Building dashboard..." -ForegroundColor Cyan
    
    if (-not (Test-Path "apps\dashboard\node_modules")) {
        Write-Host "Installing dashboard dependencies..." -ForegroundColor Yellow
        Push-Location apps\dashboard
        npm install
        Pop-Location
    }
    
    Push-Location apps\dashboard
    npm run build
    Pop-Location
    
    # Copy dashboard build to CLI embed location
    Write-Host "Embedding dashboard assets..." -ForegroundColor Cyan
    $embedDir = "apps\cli\internal\dashboard\dist"
    
    if (Test-Path $embedDir) {
        Remove-Item -Recurse -Force $embedDir
    }
    New-Item -ItemType Directory -Force -Path $embedDir | Out-Null
    
    # Angular 17+ outputs to dist/dashboard/browser
    $dashboardBuild = "apps\dashboard\dist\dashboard\browser"
    if (Test-Path $dashboardBuild) {
        Copy-Item -Recurse "$dashboardBuild\*" $embedDir
        Write-Host "[OK] Dashboard built and embedded" -ForegroundColor Green
    }
    else {
        Write-Error "Dashboard build not found at $dashboardBuild"
    }
}

function Build-VSCode {
    Write-Host "Building VS Code extension..." -ForegroundColor Cyan
    
    if (-not (Test-Path "apps\vscode\node_modules")) {
        Write-Host "Installing VS Code extension dependencies..." -ForegroundColor Yellow
        Push-Location apps\vscode
        npm install
        Pop-Location
    }
    
    Push-Location apps\vscode
    npm run compile
    Pop-Location
    
    Write-Host "[OK] VS Code extension built" -ForegroundColor Green
}

function Build-CLI {
    param([bool]$IsRelease = $false)
    
    Write-Host "Building palace CLI..." -ForegroundColor Cyan
    
    # Ensure dashboard is embedded first
    if (-not (Test-Path "apps\cli\internal\dashboard\dist\index.html")) {
        Write-Host "Dashboard not embedded. Building dashboard first..." -ForegroundColor Yellow
        Build-Dashboard
    }
    
    if ($IsRelease) {
        $env:CGO_ENABLED = "1"
        go build -ldflags="$LDFLAGS" -o palace.exe .\apps\cli
    }
    else {
        go build -ldflags="$LDFLAGS" -o palace.exe .\apps\cli
    }
    
    if ($LASTEXITCODE -eq 0) {
        $size = (Get-Item palace.exe).Length / 1MB
        Write-Host "[OK] Palace CLI built: palace.exe ($([math]::Round($size, 2)) MB)" -ForegroundColor Green
    }
    else {
        Write-Error "CLI build failed"
    }
}

function Run-Tests {
    Write-Host "Running all tests..." -ForegroundColor Cyan
    
    Write-Host "`nGo tests:" -ForegroundColor Yellow
    go test -v ./...
    
    if (Test-Path "apps\dashboard\node_modules") {
        Write-Host "`nDashboard tests:" -ForegroundColor Yellow
        Push-Location apps\dashboard
        npm test -- --watch=false
        Pop-Location
    }
    
    if (Test-Path "apps\vscode\node_modules") {
        Write-Host "`nVS Code tests:" -ForegroundColor Yellow
        Push-Location apps\vscode
        npm test
        Pop-Location
    }
}

function Clean-Build {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Cyan
    
    if (Test-Path "palace.exe") { Remove-Item palace.exe }
    if (Test-Path "palace") { Remove-Item palace }
    if (Test-Path "apps\cli\internal\dashboard\dist") { 
        Remove-Item -Recurse -Force "apps\cli\internal\dashboard\dist" 
    }
    if (Test-Path "apps\dashboard\dist") { 
        Remove-Item -Recurse -Force "apps\dashboard\dist" 
    }
    if (Test-Path "apps\vscode\out") { 
        Remove-Item -Recurse -Force "apps\vscode\out" 
    }
    
    Write-Host "[OK] Clean complete" -ForegroundColor Green
}

# Main execution
switch ($Target) {
    'dashboard' { Build-Dashboard }
    'vscode' { Build-VSCode }
    'cli' { Build-CLI }
    'test' { Run-Tests }
    'clean' { Clean-Build }
    'release' { 
        Build-Dashboard
        Build-VSCode
        Build-CLI -IsRelease $true
    }
    'all' {
        Build-Dashboard
        Build-VSCode
        Build-CLI
        Write-Host "`n[OK] Build complete!" -ForegroundColor Green
    }
}
