@echo off
echo ========================================
echo Building DockTUI for Linux (amd64)
echo ========================================

:: 设置环境变量
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0

:: 编译
echo Compiling...
go build -o docktui-linux-amd64 ./cmd/docktui

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ========================================
    echo Build successful!
    echo Output: docktui-linux-amd64
    echo ========================================
    echo.
    echo To use on Linux:
    echo   1. Copy docktui-linux-amd64 to your Linux machine
    echo   2. chmod +x docktui-linux-amd64
    echo   3. ./docktui-linux-amd64
    echo.
) else (
    echo.
    echo ========================================
    echo Build failed!
    echo ========================================
)

:: 清除环境变量
set GOOS=
set GOARCH=
set CGO_ENABLED=
