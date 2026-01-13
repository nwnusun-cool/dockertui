@echo off
REM Windows batch script to build Linux binary
REM Usage: build-linux.bat [arch]
REM   arch: amd64 (default), arm64

setlocal enabledelayedexpansion

set APP_NAME=docktui
set ARCH=%1

if "%ARCH%"=="" set ARCH=amd64

REM Get script directory
set SCRIPT_DIR=%~dp0

echo Building %APP_NAME% for Linux (%ARCH%)...
set GOOS=linux
set GOARCH=%ARCH%
set CGO_ENABLED=0

go build -ldflags="-s -w" -o "%SCRIPT_DIR%%APP_NAME%-linux-%ARCH%" .\cmd\docktui

if %ERRORLEVEL% neq 0 (
    echo Build failed!
    exit /b 1
)

echo.
echo Build successful!
echo Output: %SCRIPT_DIR%%APP_NAME%-linux-%ARCH%
dir "%SCRIPT_DIR%%APP_NAME%-linux-%ARCH%"

endlocal
