@echo off
echo Building MCP Sync Project...

echo.
echo Step 1: Tidy Go modules...
cd /d D:\code\mcp-sync
call go mod tidy

echo.
echo Step 2: Install Node dependencies...
cd /d D:\code\mcp-sync\frontend
call npm install

echo.
echo Step 3: Building frontend...
call npm run build

echo.
echo Step 4: Running Wails dev server...
cd /d D:\code\mcp-sync
call wails dev

pause
