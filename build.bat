@echo off
REM Build script for Specter - RSI Store Checkout Assistant
REM Builds binaries for macOS and Windows

echo Building Specter for multiple platforms...
echo.

REM Create build directory
if not exist build mkdir build

REM macOS (Intel)
echo Building for macOS (Intel)...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="-s -w" -o build/specter-darwin-amd64 .
echo ✓ build/specter-darwin-amd64
echo.

REM macOS (Apple Silicon)
echo Building for macOS (Apple Silicon)...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags="-s -w" -o build/specter-darwin-arm64 .
echo ✓ build/specter-darwin-arm64
echo.

REM Windows
echo Building for Windows...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o build/specter-windows-amd64.exe .
echo ✓ build/specter-windows-amd64.exe
echo.

REM Linux (optional)
echo Building for Linux...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o build/specter-linux-amd64 .
echo ✓ build/specter-linux-amd64
echo.

echo All builds completed successfully!
echo.
dir build\
echo.
echo To create Windows installer, use NSIS or Inno Setup
pause
