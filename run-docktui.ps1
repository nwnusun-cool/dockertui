# DockTUI 启动脚本 - 连接到远程 Docker
# 远程 Docker 主机: 192.168.3.49:2375

Write-Host "🚀 启动 DockTUI..." -ForegroundColor Cyan
Write-Host "📡 连接到远程 Docker: tcp://192.168.3.49:2375" -ForegroundColor Yellow
Write-Host ""

# 设置 Docker 主机环境变量
$env:DOCKER_HOST="tcp://192.168.3.49:2375"

# 启动 DockTUI
.\docktui-win64.exe
