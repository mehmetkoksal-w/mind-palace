#!/usr/bin/env pwsh
# Mind Palace - Run All Tests
# Usage: .\scripts\test-all.ps1 [-Coverage] [-Verbose]

param(
    [switch]$Coverage,
    [switch]$VerboseOutput
)

$ErrorActionPreference = 'Stop'
$script:TotalTests = 0
$script:PassedSuites = 0
$script:FailedSuites = 0

function Write-Section {
    param([string]$Title)
    Write-Host ""
    Write-Host "=" * 60 -ForegroundColor DarkGray
    Write-Host " $Title" -ForegroundColor Cyan
    Write-Host "=" * 60 -ForegroundColor DarkGray
}

function Write-Result {
    param([string]$Name, [bool]$Success, [string]$Details = "")
    if ($Success) {
        Write-Host "[PASS] $Name" -ForegroundColor Green
        $script:PassedSuites++
    } else {
        Write-Host "[FAIL] $Name" -ForegroundColor Red
        $script:FailedSuites++
    }
    if ($Details) {
        Write-Host "       $Details" -ForegroundColor DarkGray
    }
}

# Ensure we're in the project root
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
Set-Location $ProjectRoot

Write-Host ""
Write-Host "Mind Palace - Test Suite" -ForegroundColor White
Write-Host "Project root: $ProjectRoot" -ForegroundColor DarkGray

# =============================================================================
# Go Tests
# =============================================================================
Write-Section "Go Tests"

$goTestArgs = @("test")
if ($VerboseOutput) { $goTestArgs += "-v" }
if ($Coverage) { $goTestArgs += "-coverprofile=coverage.out" }
$goTestArgs += "./apps/cli/..."

Write-Host "Running: go $($goTestArgs -join ' ')" -ForegroundColor DarkGray

$goOutput = & go @goTestArgs 2>&1
$goSuccess = $LASTEXITCODE -eq 0

if ($VerboseOutput -or -not $goSuccess) {
    $goOutput | ForEach-Object { Write-Host $_ }
}

# Count test packages
$okCount = ($goOutput | Select-String -Pattern "^ok\s+" | Measure-Object).Count
$failCount = ($goOutput | Select-String -Pattern "^FAIL\s+" | Measure-Object).Count

Write-Result "Go Unit Tests" $goSuccess "$okCount packages passed, $failCount failed"

if ($Coverage -and $goSuccess) {
    Write-Host "Generating coverage report..." -ForegroundColor DarkGray
    go tool cover -func=coverage.out | Select-Object -Last 1
}

# =============================================================================
# Dashboard Tests (Angular)
# =============================================================================
Write-Section "Dashboard Tests (Angular)"

if (Test-Path "apps\dashboard\node_modules") {
    Push-Location apps\dashboard
    try {
        $dashOutput = npm test -- --watch=false --browsers=ChromeHeadless 2>&1
        $dashSuccess = $LASTEXITCODE -eq 0

        if ($VerboseOutput -or -not $dashSuccess) {
            $dashOutput | ForEach-Object { Write-Host $_ }
        }

        Write-Result "Dashboard Tests" $dashSuccess
    }
    catch {
        Write-Result "Dashboard Tests" $false "Error: $_"
    }
    finally {
        Pop-Location
    }
} else {
    Write-Host "[SKIP] Dashboard tests - node_modules not installed" -ForegroundColor Yellow
    Write-Host "       Run 'npm install' in apps/dashboard first" -ForegroundColor DarkGray
}

# =============================================================================
# VS Code Extension Tests
# =============================================================================
Write-Section "VS Code Extension Tests"

if (Test-Path "apps\vscode\node_modules") {
    Push-Location apps\vscode
    try {
        $vscodeOutput = npm test 2>&1
        $vscodeSuccess = $LASTEXITCODE -eq 0

        if ($VerboseOutput -or -not $vscodeSuccess) {
            $vscodeOutput | ForEach-Object { Write-Host $_ }
        }

        Write-Result "VS Code Extension Tests" $vscodeSuccess
    }
    catch {
        Write-Result "VS Code Extension Tests" $false "Error: $_"
    }
    finally {
        Pop-Location
    }
} else {
    Write-Host "[SKIP] VS Code tests - node_modules not installed" -ForegroundColor Yellow
    Write-Host "       Run 'npm install' in apps/vscode first" -ForegroundColor DarkGray
}

# =============================================================================
# Summary
# =============================================================================
Write-Section "Test Summary"

$totalSuites = $script:PassedSuites + $script:FailedSuites
Write-Host ""
Write-Host "Suites Passed: $($script:PassedSuites)" -ForegroundColor Green
Write-Host "Suites Failed: $($script:FailedSuites)" -ForegroundColor $(if ($script:FailedSuites -gt 0) { "Red" } else { "Green" })
Write-Host ""

if ($script:FailedSuites -gt 0) {
    Write-Host "[FAILED] Some test suites failed" -ForegroundColor Red
    exit 1
} else {
    Write-Host "[OK] All test suites passed" -ForegroundColor Green
    exit 0
}
