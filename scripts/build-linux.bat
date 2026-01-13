@echo off
REM Windows batch script to build Linux binary
REM Usage: build-linux.bat [arch]
REM   arch: amd64 (default), arm64

setlocal

set APP_NAME=docktui
set ARCH=%1
if "%ARCH%"=="" set ARCH=amd64

REM Save current directory
set ORIGINAL_DIR=%CD%

REM Change to project root (parent of scripts folder)
cd /d "%~dp0.."

echo Building %APP_NAME% for Linux (%ARCH%)...
echo Project root: %CD%

set GOOS=linux
set GOARCH=%ARCH%
set CGO_ENABLED=0

go build -ldflags="-s -w" -o "scripts\%APP_NAME%-linux-%ARCH%" ./cmd/docktui

if %ERRORLEVEL% neq 0 (
    echo Build failed!
    cd /d "%ORIGINAL_DIR%"
    exit /b 1
)

echo.
echo Build successful!
echo Output: scripts\%APP_NAME%-linux-%ARCH%

cd /d "%ORIGINAL_DIR%"
endlocal
