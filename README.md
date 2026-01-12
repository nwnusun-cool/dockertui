# DockTUI

🐳 一个现代化的 Docker 终端管理工具，提供类似 [k9s](https://k9scli.io/) 的交互体验。

![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)

✨ 特性
🎨 直观的终端界面 - 基于 Bubble Tea 框架的现代化 TUI 体验，使用 Lipgloss 自适应布局
📦 容器管理 - 查看容器列表、详情、实时日志，支持完整的容器生命周期操作
🖼️ 镜像管理 - 查看镜像列表、详情、删除、清理悬垂镜像、拉取镜像（带进度显示）
🔍 智能搜索 - 支持按名称、镜像、ID 快速搜索容器和镜像
💻 交互式 Shell - 直接进入容器执行命令，支持多种 Shell 选择
📊 资源监控 - 实时查看容器 CPU、内存使用情况和 I/O 统计
⚡ 事件驱动 - 自动监听 Docker 事件，实时更新容器状态
🎯 容器操作 - 启动、停止、重启、暂停、恢复、删除容器
🧩 Compose 支持 - 扫描和管理 Docker Compose 项目（基础功能）
🚀 跨平台 - 支持 Windows、Linux、macOS

## 预览

```
┌─────────────────────────────────────────────────────────────────────────┐
│  🐳 DockTUI                                                              │
├─────────────────────────────────────────────────────────────────────────┤
│  📦 Containers: 12    🖼️ Images: 25    🌐 Networks: 5    💾 Volumes: 8   │
├─────────────────────────────────────────────────────────────────────────┤
│  CONTAINER ID   NAME              IMAGE           STATUS      PORTS     │
│▶ abc123def456   nginx-web         nginx:latest    Up 2h       80/tcp    │
│  def456abc789   mysql-db          mysql:8.0       Up 5h       3306/tcp  │
│  789abc123def   redis-cache       redis:alpine    Up 1h       6379/tcp  │
└─────────────────────────────────────────────────────────────────────────┘
  j/k 移动  Enter 详情  l 日志  s Shell  t 启动  p 停止  / 搜索  ? 帮助
```

## 功能

**容器管理**
- 容器列表、详情、实时日志（Follow 模式）
- 启动、停止、重启、暂停、删除
- 交互式 Shell（支持多种 Shell 选择）
- 实时资源监控（CPU/内存/IO）

**镜像管理**
- 镜像列表、详情、层信息
- 拉取镜像（带进度条）
- 删除、清理悬垂镜像

**网络管理**
- 网络列表、详情
- 创建、删除网络

**Compose 支持**
- 自动发现 Compose 项目
- 项目详情、服务状态
- Up/Down/Restart 操作

## 安装

### 从 Release 下载

前往 [Releases](../../releases) 页面下载对应平台的可执行文件：

- `docktui-windows-amd64.exe` - Windows 64位
- `docktui-linux-amd64` - Linux 64位

### 从源码编译

```bash
git clone https://github.com/yourname/docktui.git
cd docktui
go build -o docktui ./cmd/docktui
```

## 使用

```bash
# 连接本地 Docker
./docktui

# 连接远程 Docker
DOCKER_HOST=tcp://192.168.1.100:2375 ./docktui
```

## 快捷键

### 全局
| 按键 | 功能 |
|------|------|
| `q` / `Ctrl+C` | 退出 |
| `?` | 帮助 |
| `Esc` | 返回上级 |

### 列表导航
| 按键 | 功能 |
|------|------|
| `j` / `↓` | 下移 |
| `k` / `↑` | 上移 |
| `g` / `G` | 首行/末行 |
| `Enter` | 进入详情 |
| `/` | 搜索 |
| `r` | 刷新 |

### 容器操作
| 按键 | 功能 |
|------|------|
| `l` | 查看日志 |
| `s` | 进入 Shell |
| `t` | 启动 |
| `p` | 停止 |
| `P` | 暂停/恢复 |
| `R` | 重启 |
| `Ctrl+D` | 删除 |

### 日志视图
| 按键 | 功能 |
|------|------|
| `f` | Follow 模式 |
| `w` | 自动换行 |
| `Ctrl+d/u` | 翻页 |

## 项目结构

```
docktui/
├── cmd/docktui/          # 程序入口
├── internal/
│   ├── compose/          # Docker Compose 客户端
│   ├── docker/           # Docker API 封装
│   ├── task/             # 后台任务管理
│   └── ui/               # TUI 界面
│       ├── components/   # 通用组件（表格、弹窗、输入框等）
│       ├── compose/      # Compose 视图
│       ├── container/    # 容器视图
│       ├── image/        # 镜像视图
│       └── network/      # 网络视图
└── examples/             # 示例代码
```

## 技术栈

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI 框架
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - 终端样式
- [Docker SDK](https://github.com/docker/docker) - Docker API

## License

MIT
