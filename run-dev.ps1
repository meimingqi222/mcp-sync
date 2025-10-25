#!/usr/bin/env pwsh
# MCP Sync Development Server Starter

Write-Host "MCP Sync - Development Server" -ForegroundColor Cyan
Write-Host "==============================" -ForegroundColor Cyan
Write-Host ""

# Check prerequisites
Write-Host "Checking prerequisites..." -ForegroundColor Yellow
$goVersion = go version
$nodeVersion = node --version
$pnpmVersion = pnpm --version

Write-Host "Go: $goVersion" -ForegroundColor Green
Write-Host "Node: $nodeVersion" -ForegroundColor Green
Write-Host "pnpm: $pnpmVersion" -ForegroundColor Green
Write-Host ""

# Navigate to project
Set-Location "D:\code\mcp-sync"

Write-Host "Starting Wails development server..." -ForegroundColor Yellow
Write-Host "The application will open at http://localhost:34115" -ForegroundColor Cyan
Write-Host ""

# Run wails dev
wails dev

# Keep window open if error
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error occurred. Press Enter to exit..."
    Read-Host
}
