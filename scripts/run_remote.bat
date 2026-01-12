@echo off
REM 远程运行 DockTUI（连接远程 Docker）

echo 构建 DockTUI...
go build -o docktui.exe ../cmd/docktui 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo 构建失败！
    pause
    exit /b 1
)

echo 启动 DockTUI...
set DOCKER_HOST=tcp://192.168.3.49:2375
docktui.exe
