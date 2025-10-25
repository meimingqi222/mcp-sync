@echo off
echo Building MCP Sync...
echo.

cd /d D:\code\mcp-sync

echo Step 1: Go Build
go build ./...
if %errorlevel% neq 0 (
    echo Go build failed!
    pause
    exit /b 1
)
echo [OK] Go compilation successful

echo.
echo Step 2: Frontend Build
cd frontend
call pnpm run build
if %errorlevel% neq 0 (
    echo Frontend build failed!
    pause
    exit /b 1
)
echo [OK] Frontend build successful
cd ..

echo.
echo Step 3: Wails Build
call wails build -clean
if %errorlevel% neq 0 (
    echo Wails build failed!
    pause
    exit /b 1
)
echo [OK] Wails build successful

echo.
echo ===== BUILD COMPLETE =====
echo Executable: build\bin\mcp-sync.exe
echo.
pause
