# DockTUI - Docker 终端管理工具

🐳 一个基于 TUI（终端用户界面）的本地 Docker 容器管理工具，提供类似 k9s 的体验，用于本地容器编排和管理。

## ✨ 特性

- 🎨 **直观的终端界面** - 基于 Bubble Tea 框架的现代化 TUI 体验，使用 Lipgloss 自适应布局
- 📦 **容器管理** - 查看容器列表、详情、实时日志，支持完整的容器生命周期操作
- 🖼️ **镜像管理** - 查看镜像列表、详情、删除、清理悬垂镜像、拉取镜像（带进度显示）
- 🔍 **智能搜索** - 支持按名称、镜像、ID 快速搜索容器和镜像
- 💻 **交互式 Shell** - 直接进入容器执行命令，支持多种 Shell 选择
- 📊 **资源监控** - 实时查看容器 CPU、内存使用情况和 I/O 统计
- ⚡ **事件驱动** - 自动监听 Docker 事件，实时更新容器状态
- 🎯 **容器操作** - 启动、停止、重启、暂停、恢复、删除容器
- 🧩 **Compose 支持** - 扫描和管理 Docker Compose 项目（基础功能）
- 🚀 **跨平台** - 支持 Windows、Linux、macOS

## 📸 界面预览

```
┌─────────────────────────────────────────────────────────────────────────┐
│  🐳 DockTUI - Docker 终端管理工具                                        │
├─────────────────────────────────────────────────────────────────────────┤
│  📦 Containers: 12    🖼️ Images: 25    🌐 Networks: 5    💾 Volumes: 8   │
├─────────────────────────────────────────────────────────────────────────┤
│  CONTAINER ID   NAME              IMAGE           STATUS      PORTS     │
│  abc123def456   nginx-web         nginx:latest    Up 2 hours  80/tcp    │
│  def456abc789   mysql-db          mysql:8.0       Up 5 hours  3306/tcp  │
│  ...                                                                     │
└─────────────────────────────────────────────────────────────────────────┘
```

## 🛠️ 技术栈

- **语言**: Go 1.24+
- **TUI 框架**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) v1.3.4
- **UI 组件库**: 
  - [Bubbles](https://github.com/charmbracelet/bubbles) v0.21.0 - 预构建的 TUI 组件
  - [Lipgloss](https://github.com/charmbracelet/lipgloss) v1.1.0 - 样式和布局
- **Docker SDK**: [github.com/docker/docker](https://github.com/docker/docker) v28.0.2

## 📋 前置要求

- Go 1.24 或更高版本
- Docker Desktop (Windows/macOS) 或 Docker Engine (Linux)
- 本地或远程 Docker 守护进程

## 🚀 快速开始

### 安装

```bash
# 克隆仓库
git clone <repository-url>
cd docktui

# 安装依赖
go mod tidy

# 编译
go build -o docktui ./cmd/docktui
```

### 运行

```bash
# Linux/macOS
./docktui

# Windows
.\docktui.exe
```

### 连接远程 Docker

```bash
# 设置环境变量
export DOCKER_HOST=tcp://<remote-ip>:2375
./docktui
```

## 🎮 快捷键说明

### 全局快捷键

| 快捷键 | 功能 |
|--------|------|
| `q` / `Ctrl+C` | 退出程序 |
| `?` | 显示帮助面板 |
| `Esc` | 返回上一级 / 关闭弹窗 |
| `c` | 快速进入容器列表 |
| `i` | 快速进入镜像列表 |

### 首页导航

| 快捷键 | 功能 |
|--------|------|
| `↑/↓/←/→` 或 `h/j/k/l` | 导航选择 |
| `Enter` | 进入选中的功能模块 |
| `1-5` | 快速进入对应模块 |
| `r` | 刷新统计数据 |

### 容器列表视图

| 快捷键 | 功能 |
|--------|------|
| `j/k` 或 `↑/↓` | 上下移动 |
| `Enter` | 查看容器详情 |
| `l` | 查看容器日志 |
| `s` | 进入容器 Shell |
| `t` | 启动容器 |
| `p` | 停止容器 |
| `P` | 暂停/恢复容器 |
| `R` | 重启容器 |
| `Ctrl+D` | 删除容器（带确认） |
| `/` | 搜索容器 |
| `r` | 刷新列表 |

### 镜像列表视图

| 快捷键 | 功能 |
|--------|------|
| `j/k` 或 `↑/↓` | 上下移动 |
| `h/l` 或 `←/→` | 左右滚动表格 |
| `Enter` | 查看镜像详情 |
| `P` | 拉取镜像（弹出输入框） |
| `d` | 删除镜像（带确认） |
| `p` | 清理悬垂镜像 |
| `/` | 搜索镜像 |
| `r` | 刷新列表 |
| `T` | 展开/收起任务进度条 |

### 日志视图

| 快捷键 | 功能 |
|--------|------|
| `f` | 切换 Follow 模式 |
| `w` | 切换自动换行 |
| `j/k` | 上下滚动 |
| `g/G` | 跳到首行/末尾 |
| `Ctrl+d/u` | 向下/上翻页 |

### 容器详情视图

| 快捷键 | 功能 |
|--------|------|
| `←/→` 或 `Tab` | 切换标签页 |
| `l` | 查看日志 |
| `s` | 进入 Shell |
| `r` | 刷新详情 |

## 📁 项目结构

```
docktui/
├── cmd/docktui/           # 主程序入口
├── internal/
│   ├── config/            # 配置管理
│   ├── compose/           # Docker Compose 封装
│   ├── docker/            # Docker 客户端封装
│   │   ├── client.go      # 客户端接口
│   │   ├── exec.go        # Exec Shell 功能
│   │   ├── logs.go        # 日志功能
│   │   ├── pull.go        # 镜像拉取（带进度）
│   │   └── stats.go       # 资源统计
│   ├── task/              # 后台任务管理
│   └── ui/                # TUI 界面
│       ├── ui.go          # 主 Model
│       ├── home_view.go   # 首页导航
│       ├── container_*.go # 容器相关视图
│       ├── image_*.go     # 镜像相关视图
│       ├── logs_view.go   # 日志视图
│       └── ...
├── docs/                  # 文档
└── README.md
```

## 📝 功能状态

### ✅ 已完成功能

**容器管理**
- 容器列表（状态着色、搜索、过滤）
- 容器详情（6 个标签页：基本信息、资源监控、网络端口、存储挂载、环境变量、标签）
- 容器操作（启动、停止、重启、暂停、恢复、删除）
- 容器日志（Follow 模式、自动换行、日志着色）
- 交互式 Shell（多 Shell 选择器）
- 资源监控（CPU/内存图表、I/O 统计）

**镜像管理**
- 镜像列表（状态着色、搜索、水平滚动）
- 镜像详情（6 个标签页）
- 镜像删除（普通删除、强制删除）
- 清理悬垂镜像
- 镜像拉取（带进度条显示）

**Compose 支持**
- 项目扫描（自动发现 compose 文件）
- 项目列表显示
- 基本操作（up/down/restart）

**用户体验**
- vim 风格快捷键
- 自适应终端主题（亮色/暗色）
- 弹窗和对话框
- 后台任务进度条
- 错误提示和成功消息

### 🚧 开发中功能

- 镜像打标签/取消标签
- 镜像导出/导入
- 镜像推送
- Compose 项目详情
- Compose 日志查看
- 网络管理
- 卷管理

## 🗺️ Roadmap

- [ ] 完善镜像管理（打标签、导出/导入）
- [ ] 完善 Compose 支持（详情、日志、服务管理）
- [ ] 网络管理（列表、创建、删除）
- [ ] 卷管理（列表、创建、删除）
- [ ] 多主机支持
- [ ] 配置文件支持
- [ ] 主题定制

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License

## � 致谢结

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - 优秀的 Go TUI 框架
- [Docker Engine API](https://docs.docker.com/engine/api/) - Docker 官方 API
- [k9s](https://k9scli.io/) - 提供了灵感和参考
