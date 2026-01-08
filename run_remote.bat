@echo off
REM 远程运行 DockTUI（连接远程 Docker）
echo 启动 DockTUI（连接远程 Docker: 192.168.3.49:2375）...
echo.

REM 设置远程 Docker 地址
set DOCKER_HOST=tcp://192.168.3.49:2375

REM 运行程序
docktui-win64.exe

pause
