@echo off
REM 远程运行 DockTUI（连接远程 Docker）
REM 使用前请修改 DOCKER_HOST 为你的远程 Docker 地址
echo 启动 DockTUI（连接远程 Docker）...
echo.

REM 设置远程 Docker 地址
set DOCKER_HOST=tcp://your-docker-host:2375

REM 运行程序
docktui.exe

pause
