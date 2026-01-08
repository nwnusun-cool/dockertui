# DockTUI - Docker 终端管理工具

🐳 一个基于 TUI（终端用户界面）的本地 Docker 容器管理工具，提供类似 k9s 的体验，用于本地容器编排和管理。

## ✨ 特性

- 🎨 **直观的终端界面** - 基于 Bubble Tea 框架的现代化 TUI 体验，使用 Lipgloss 自适应布局
- 📋 **容器管理** - 查看容器列表、详情、实时日志，支持完整的容器生命周期操作
- 🔍 **智能搜索** - 支持按名称、镜像、ID 快速搜索容器
- 💻 **交互式 Shell** - 直接进入容器执行命令，退出后自动返回 TUI
- 📊 **资源监控** - 实时查看容器 CPU、内存使用情况和 I/O 统计
- ⚡ **事件驱动** - 自动监听 Docker 事件，实时更新容器状态
- 🎯 **容器操作** - 启动、停止、重启、暂停、恢复、删除容器
- 🚀 **跨平台** - 支持 Windows、Linux、macOS

## 🛠️ 技术栈

- **语言**: Go 1.24+
- **TUI 框架**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) v1.3.4
- **UI 组件库**: 
  - [Bubbles](https://github.com/charmbracelet/bubbles) v0.21.0 - 预构建的 TUI 组件
    - `table` - 表格组件（容器列表）
    - `viewport` - 滚动视图（日志、详情）
    - `key` - 快捷键管理
    - `help` - 帮助文档生成
  - [Lipgloss](https://github.com/charmbracelet/lipgloss) v1.1.0 - 样式和布局
- **Docker SDK**: [github.com/docker/docker](https://github.com/docker/docker) v28.0.2

## 📋 前置要求

- Go 1.24 或更高版本
- Docker Desktop (Windows/macOS) 或 Docker Engine (Linux)
- 本地或远程 Docker 守护进程

## 🚀 快速开始

### 安装

#### 从源码编译

```bash
# 克隆仓库
git clone <repository-url>
cd demo1

# 安装依赖
go mod tidy

# 编译（当前平台）
go build -o docktui ./cmd/docktui
```

#### 交叉编译到不同平台

**在 Linux/macOS 上编译**:

```bash
# 编译当前平台版本
go build -o docktui ./cmd/docktui

# 交叉编译 Linux 版本（AMD64）
GOOS=linux GOARCH=amd64 go build -o docktui-linux ./cmd/docktui

# 交叉编译 Linux 版本（ARM64，如树莓派）
GOOS=linux GOARCH=arm64 go build -o docktui-linux-arm64 ./cmd/docktui

# 交叉编译 macOS 版本（Intel）
GOOS=darwin GOARCH=amd64 go build -o docktui-mac-amd64 ./cmd/docktui

# 交叉编译 macOS 版本（Apple Silicon M1/M2）
GOOS=darwin GOARCH=arm64 go build -o docktui-mac-arm64 ./cmd/docktui

# 交叉编译 Windows 版本
GOOS=windows GOARCH=amd64 go build -o docktui.exe ./cmd/docktui
```

**在 Windows PowerShell 上编译**:

```powershell
# 编译 Windows 版本（当前平台）


# 交叉编译 Linux 版本（AMD64）
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o docktui-linux ./cmd/docktui

# 交叉编译 Linux 版本（ARM64）
$env:GOOS="linux"; $env:GOARCH="arm64"; go build -o docktui-linux-arm64 ./cmd/docktui

# 交叉编译 macOS 版本（Intel）
$env:GOOS="darwin"; $env:GOARCH="amd64"; go build -o docktui-mac-amd64 ./cmd/docktui

# 交叉编译 macOS 版本（Apple Silicon）
$env:GOOS="darwin"; $env:GOARCH="arm64"; go build -o docktui-mac-arm64 ./cmd/docktui
```

**在 Windows CMD 上编译**:

```cmd
# 编译 Windows 版本
go build -o docktui-win64.exe ./cmd/docktui

# 交叉编译 Linux 版本
set GOOS=linux
set GOARCH=amd64
go build -o docktui-linux ./cmd/docktui
```

> **💡 提示**: 
> - 推荐使用 `./cmd/docktui` 而不是 `cmd/docktui/main.go`，这样可以正确处理包内的所有文件
> - Linux 版本编译后需要添加执行权限: `chmod +x docktui-linux`
> - Windows PowerShell 中设置环境变量使用 `$env:VAR="value"` 语法
> - macOS M1/M2 用户应选择 `arm64` 架构以获得最佳性能

### 运行

#### 连接本地 Docker

```bash
# Linux/macOS
./docktui

# Windows
.\docktui-win64.exe

# 或使用提供的脚本（Windows）
.\run-docktui.ps1
```

#### 连接远程 Docker

**Linux/macOS**:

```bash
# 设置环境变量
export DOCKER_HOST=tcp://<remote-ip>:2375

# 运行
./docktui
```

**Windows PowerShell**:

```powershell
# 方法 1: 设置环境变量并运行
$env:DOCKER_HOST="tcp://<remote-ip>:2375"
.\docktui-win64.exe

# 方法 2: 使用启动脚本（需要先编辑脚本中的 IP 地址）
.\run-docktui.ps1
```

**Windows CMD**:

```cmd
set DOCKER_HOST=tcp://<remote-ip>:2375
docktui-win64.exe
```

> **⚠️ 注意**: 
> - 远程 Docker 需要开启 TCP 端口（默认 2375）
> - 生产环境请配置 TLS 加密（端口 2376）
> - 如果需要连接到其他服务器，请确保防火墙允许该端口通信

## 🎮 快捷键说明

### 全局快捷键

| 快捷键 | 功能 |
|--------|------|
| `q` / `Ctrl+C` | 退出程序 |
| `c` | 进入容器列表视图 |
| `Esc` / `b` | 返回上一个视图 |

### 欢迎界面

| 快捷键 | 功能 |
|--------|------|
| `c` | 进入容器列表 |
| `q` | 退出程序 |

### 容器列表视图

| 快捷键 | 功能 |
|--------|------|
| `j` / `↓` | 向下移动 |
| `k` / `↑` | 向上移动 |
| `Enter` | 查看容器详情 |
| `l` | 查看容器日志 |
| `r` | 手动刷新列表 |
| `/` | 搜索容器（名称/镜像/ID） |
| `s` | 进入容器 Shell |
| `t` | 启动容器 |
| `p` | 停止容器 |
| `P` | 暂停/恢复容器 |
| `R` | 重启容器 |
| `Ctrl+D` | 删除容器（带确认） |
| `Ctrl+A` | 批量操作（开发中） |
| `?` | 显示帮助面板 |
| `Esc` / `b` | 返回 |
| `q` | 退出程序 |

### 容器详情视图

| 快捷键 | 功能 |
|--------|------|
| `←` / `h` | 切换到上一个标签页 |
| `→` / `l` | 切换到下一个标签页 |
| `Tab` | 循环切换标签页 |
| `l` | 查看日志 |
| `s` | 进入 Shell |
| `r` | 刷新详情 |
| `Esc` / `b` | 返回列表 |

**标签页说明**:
- 基本信息 - 容器 ID、名称、镜像、状态、重启策略等
- 资源监控 - 实时 CPU/内存使用率、网络/磁盘 I/O（仅运行中容器）
- 网络端口 - 端口映射和网络配置
- 存储挂载 - 卷挂载和绑定挂载信息
- 环境变量 - 应用和系统环境变量
- 标签 - 自定义标签、Compose 标签、Docker 系统标签

### 日志视图

| 快捷键 | 功能 |
|--------|------|
| `f` | 切换 follow 模式 |
| `w` | 切换自动换行 |
| `j` / `↓` | 向下滚动 |
| `k` / `↑` | 向上滚动 |
| `g` / `Home` | 跳到首行 |
| `G` / `End` | 跳到末尾 |
| `Ctrl+d` | 向下翻页 |
| `Ctrl+u` | 向上翻页 |
| `r` | 刷新日志 |
| `Esc` / `b` | 返回 |

## 📁 项目结构

```
demo1/
├── cmd/
│   ├── docktui/                    # 主程序入口
│   │   └── main.go
│   └── *-demo/                     # 各功能演示程序
├── internal/
│   ├── config/                     # 配置管理
│   │   └── config.go
│   ├── docker/                     # Docker 客户端封装
│   │   ├── client.go               # 客户端接口和实现
│   │   ├── client_test.go          # 单元测试
│   │   ├── exec.go                 # Exec Shell 功能
│   │   ├── logs.go                 # 日志功能
│   │   └── stats.go                # 资源统计功能
│   └── ui/                         # TUI 界面
│       ├── ui.go                   # Bubble Tea 主 Model
│       ├── container_list_view.go  # 容器列表视图
│       ├── container_detail_view.go # 容器详情视图
│       ├── logs_view.go            # 日志视图
│       ├── stats_view.go           # 资源监控视图
│       ├── sparkline.go            # 图表组件
│       └── keys.go                 # 快捷键定义
├── docs/
│   ├── PRD.md                      # 产品需求文档
│   ├── todo.md                     # 详细任务清单
│   ├── exec-shell-design.md        # Shell 功能设计文档
│   └── refactoring-summary.md      # 重构总结
├── go.mod                          # Go 模块定义
├── go.sum                          # 依赖锁定
├── build-linux.bat                 # Linux 构建脚本
├── run-docktui.ps1                 # PowerShell 运行脚本
├── run_local.bat                   # 本地运行脚本（Windows）
└── run_remote.bat                  # 远程运行脚本（Windows）
```

## 🔧 配置

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `DOCKER_HOST` | Docker 守护进程地址 | 自动检测 |

### 自动检测机制

程序会根据操作系统自动选择 Docker 连接方式：

- **Linux**: Unix Socket (`/var/run/docker.sock`)
- **Windows**: Named Pipe (`npipe:////./pipe/docker_engine`)
- **macOS**: Unix Socket (`/var/run/docker.sock`)

如果设置了 `DOCKER_HOST` 环境变量，将优先使用指定的连接方式。

## 🐛 错误处理

程序在启动时会检测 Docker 连接状态：

- ✅ **连接成功**: 正常显示所有功能
- ❌ **连接失败**: 显示详细错误信息和解决建议，但不会立即退出

### 错误提示机制

- **严重错误**: 显示错误对话框，需要用户确认
- **警告消息**: 顶部黄色提示，3秒后自动消失（如"容器已在运行中"）
- **成功消息**: 顶部绿色提示，3秒后自动消失（如"启动容器成功"）

### 常见问题

#### 1. Windows 下无法连接 Docker

**错误信息**: `error during connect: This error may indicate that the docker daemon is not running.`

**解决方法**:
- 确保 Docker Desktop 已启动
- 检查系统托盘中 Docker 图标状态
- 如果使用 WSL2 后端，确保 WSL2 已启动

#### 2. Linux 下权限不足

**错误信息**: `permission denied while trying to connect to the Docker daemon socket`

**解决方法**:
```bash
# 将当前用户添加到 docker 组
sudo usermod -aG docker $USER

# 重新登录或执行
newgrp docker
```

#### 3. 远程连接失败

**错误信息**: `Cannot connect to the Docker daemon at tcp://x.x.x.x:2375`

**解决方法**:
- 确保远程 Docker 已开启 TCP 端口
- 检查防火墙规则
- 验证网络连通性: `telnet <remote-ip> 2375`

## 📝 开发状态

项目正在积极开发中，当前已完成的功能：

### 后端 API
- ✅ Docker 客户端封装
- ✅ 容器列表 API (ListContainers)
- ✅ 容器详情 API (InspectContainer)
- ✅ 容器日志 API (ContainerLogs)
- ✅ Exec Shell API (ExecStart)
- ✅ Docker 事件监听 (WatchEvents)
- ✅ 容器操作 API (Start/Stop/Restart/Pause/Unpause/Remove)
- ✅ 容器资源统计 API (Stats)
- ✅ 远程 Docker 连接支持

### 前端 UI
- ✅ TUI 基础框架搭建 (Bubble Tea)
- ✅ 全局状态与键盘映射
- ✅ K9s 风格界面设计
  - ✅ vim 风格快捷键 (j/k/Enter/q/b)
  - ✅ Emoji 符号可视化
  - ✅ 临时提示自动消失 (3秒)
  - ✅ 帮助面板 (? 触发)
  - ✅ 动态快捷键提示
  - ✅ 错误分级显示（严重错误弹窗，警告消息顶部提示）
- ✅ 容器列表视图
  - ✅ 使用 bubbles/table 组件
  - ✅ 数据加载与显示（匹配 docker ps 格式）
  - ✅ 事件驱动自动刷新
  - ✅ 状态色彩编码（运行=绿色，暂停=黄色，停止=灰色，不健康=红色）
  - ✅ vim 风格导航 (j/k)
  - ✅ 搜索功能 (/ 键搜索容器名称、镜像、ID)
  - ✅ 动态列宽计算（自适应窗口大小）
  - ✅ 友好的时间显示（"11 hours ago"）
- ✅ 容器详情视图
  - ✅ 标签页切换（基本信息、资源监控、网络端口、存储挂载、环境变量、标签）
  - ✅ 左右箭头键/Tab 切换标签页
  - ✅ 分类展示详细信息
  - ✅ 状态着色（运行/停止/错误）
  - ✅ 智能信息分组和格式化
  - ✅ 资源监控标签（CPU/内存图表，I/O 统计）
- ✅ 日志视图
  - ✅ 使用 bubbles/viewport 组件
  - ✅ Follow 模式 (f 键切换)
  - ✅ 自动换行 (w 键切换)
  - ✅ 日志着色 (ERROR/WARN/INFO)
  - ✅ vim 风格滚动 (j/k/g/G/Ctrl+d/Ctrl+u)
- ✅ 帮助面板
  - ✅ 使用 bubbles/help + lipgloss 组件
  - ✅ K9s 风格布局
  - ✅ 分章节展示快捷键
- ✅ 容器操作
  - ✅ 启动容器 (t 键)
  - ✅ 停止容器 (p 键)
  - ✅ 重启容器 (R 键)
  - ✅ 暂停/恢复容器 (P 键)
  - ✅ 删除容器 (Ctrl+D，带确认对话框)
  - ✅ 操作状态提示（成功/失败/警告）
- ✅ 进入容器 Shell
  - ✅ 交互式 Shell (s 键)
  - ✅ 退出后自动返回 TUI
  - ✅ 使用 docker exec 命令（兼容性更好）
- 🚧 批量操作（Ctrl+A，开发中）
- 🚧 Docker Compose 支持（计划中）

详细任务清单请查看 [docs/todo.md](docs/todo.md)

## 🗺️ Roadmap

### MVP 阶段（基本完成）

- [x] 基础架构设计
- [x] Docker SDK 集成
- [x] TUI 框架搭建（Bubble Tea + Lipgloss）
- [x] 容器列表视图（匹配 docker ps 格式）
- [x] 容器详情视图（多标签页）
- [x] 日志查看与跟踪（Follow 模式）
- [x] 搜索功能（名称/镜像/ID）
- [x] 交互式 Shell（docker exec）
- [x] 容器操作（启动/停止/重启/暂停/恢复/删除）
- [x] 资源监控（CPU/内存/I/O）
- [x] 事件驱动自动刷新
- [ ] 批量操作（开发中）
- [ ] Docker Compose 项目管理（计划中）

### 未来计划

- [ ] 镜像管理（列表、拉取、删除、构建）
- [ ] 网络管理（列表、创建、删除）
- [ ] 卷管理（列表、创建、删除）
- [ ] 多主机支持（切换 Docker 主机）
- [ ] 配置文件支持（保存常用设置）
- [ ] 主题定制（颜色方案）
- [ ] 容器日志导出
- [ ] 容器快照/备份

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

[待定]

## 🙏 致谢

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - 优秀的 Go TUI 框架
- [Docker Engine API](https://docs.docker.com/engine/api/) - Docker 官方 API
- [k9s](https://k9scli.io/) - 提供了灵感和参考

---

**注意**: 本项目处于早期开发阶段，API 和功能可能会发生变化。
