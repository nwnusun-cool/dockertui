@echo off
REM 本地运行 DockTUI（使用本地 Docker Desktop）
echo 启动 DockTUI（连接本地 Docker Desktop）...
echo.

REM 清除 DOCKER_HOST 环境变量，使用本地 Docker
set DOCKER_HOST=

REM 运行程序
docktui.exe

pause
