# Docker Compose 功能实现计划

## 概述

本文档详细说明了 DockTUI 中 Docker Compose 支持功能的实现计划，包括功能设计、技术方案、实现顺序和预期工作量。

## 功能优先级

### P0 - 必须实现（MVP）
- ✅ 项目扫描和发现
- ✅ 项目列表展示
- ✅ 基本操作（up/down/restart/stop/start）
- ✅ 容器关联和过滤

### P1 - 重要功能
- ✅ 项目详情视图（服务列表、配置、网络、卷）
- ✅ 服务状态监控
- ✅ 项目和服务日志查看
- ✅ 错误处理和友好提示

### P2 - 可选功能
- ⏳ 开发模式配置
- ⏳ 环境重置和快速重建
- ⏳ 操作历史记录
- ⏳ 配置文件管理

## 核心功能设计

### 1. 项目扫描（C3）

**扫描策略**:
- 从配置的路径递归扫描 compose 文件
- 支持多种文件名：`docker-compose.yml`, `compose.yml`, `docker-compose.yaml`, `compose.yaml`
- 支持 override 文件：`docker-compose.override.yml`, `docker-compose.*.yml`
- 智能忽略：`node_modules`, `.git`, `vendor`, `__pycache__` 等

**项目识别**:
- 同一目录下的 compose 文件归为一个项目
- 项目名称优先使用 compose 文件中的 `name` 字段
- 否则使用目录名作为项目名称

### 2. 项目列表视图（C5）

**列表列定义**:
```
PROJECT NAME | PATH | SERVICES | STATUS | COMPOSE FILES
myapp        | /app | 3/5      | Running| 2
```

**快捷键设计**:
- `j/k/↑/↓` - 上下移动
- `Enter` - 查看项目详情
- `u` - 启动项目（up -d）
- `d` - 停止项目（down）
- `r` - 重启项目
- `s` - 停止但不删除
- `t` - 启动已停止的容器
- `l` - 查看项目日志
- `c` - 跳转到项目容器列表
- `b` - 构建镜像
- `p` - 拉取镜像
- `R` - 刷新列表
- `/` - 搜索项目

**状态颜色**:
- 🟢 Running - 所有服务运行中
- 🟡 Partial - 部分服务运行中
- ⚪ Stopped - 所有服务已停止
- 🔴 Error - 错误状态

### 3. 项目详情视图（C6）

**标签页设计**:
1. **服务列表** - 显示所有服务及其状态、容器数量、端口映射
2. **配置信息** - 显示 compose 文件内容（YAML 高亮）
3. **网络** - 显示项目创建的网络
4. **卷** - 显示项目创建的卷
5. **环境变量** - 显示项目级环境变量

**服务操作**:
- 单个服务的启动/停止/重启
- 查看服务日志
- 跳转到服务容器

### 4. 容器联动（C7）

**项目 → 容器**:
- 在项目列表按 `c` 跳转到容器列表
- 自动过滤显示该项目的容器
- 显示过滤提示："Showing containers for project: myapp"

**容器 → 项目**:
- 在容器详情中显示所属项目信息
- 显示项目名称、服务名称、项目路径
- 支持点击跳转到项目列表

**过滤逻辑**:
- 检查容器 label `com.docker.compose.project`
- 匹配项目名称（不区分大小写）

### 5. 日志查看（C8）

**项目级日志**:
- 聚合显示所有服务的日志
- 带服务名前缀区分
- 支持颜色区分不同服务
- 支持 follow 模式

**服务级日志**:
- 显示单个服务的日志
- 如果有多个副本，聚合显示
- 支持切换到单个容器日志

### 6. 开发友好功能（C9）

**环境重置** (`Ctrl+R`):
1. 停止所有容器
2. 可选删除卷和镜像
3. 重新启动
4. 显示确认对话框

**快速重建** (`Ctrl+B`):
1. 构建镜像
2. 强制重新创建容器
3. 显示构建进度

**开发模式配置**:
- 支持 `.docktui.yml` 配置文件
- 额外的 compose 文件（dev/test）
- 额外的环境变量文件
- 自动挂载代码目录

## 技术实现方案

### 1. 命令检测（C1.1）

```go
// 优先级顺序
1. docker compose (Docker CLI v2 插件)
2. docker-compose (独立工具)
3. 报错提示安装

// 检测逻辑
func detectComposeCommand() (string, error) {
    // 1. 检查 docker compose
    if err := exec.Command("docker", "compose", "version").Run(); err == nil {
        return "docker compose", nil
    }
    
    // 2. 检查 docker-compose
    if err := exec.Command("docker-compose", "--version").Run(); err == nil {
        return "docker-compose", nil
    }
    
    return "", errors.New("docker compose not found")
}
```

### 2. 接口设计（C1.2）

```go
type ComposeClient interface {
    // 项目操作
    Up(ctx context.Context, project *Project, opts UpOptions) error
    Down(ctx context.Context, project *Project, opts DownOptions) error
    Restart(ctx context.Context, project *Project, services []string) error
    Stop(ctx context.Context, project *Project, services []string) error
    Start(ctx context.Context, project *Project, services []string) error
    Pause(ctx context.Context, project *Project, services []string) error
    Unpause(ctx context.Context, project *Project, services []string) error
    
    // 信息查询
    PS(ctx context.Context, project *Project) ([]Service, error)
    Logs(ctx context.Context, project *Project, opts LogOptions) (io.ReadCloser, error)
    Config(ctx context.Context, project *Project) (*ProjectConfig, error)
    
    // 镜像操作
    Build(ctx context.Context, project *Project, services []string) error
    Pull(ctx context.Context, project *Project, services []string) error
    
    // 工具方法
    Version(ctx context.Context) (string, error)
}
```

### 3. 数据模型（C2.1）

```go
type Project struct {
    Name          string
    Path          string
    ComposeFiles  []string
    EnvFiles      []string
    WorkingDir    string
    Labels        map[string]string
    
    // 运行时状态
    Services      []Service
    Status        ProjectStatus
    LastUpdated   time.Time
}

type Service struct {
    Name       string
    Image      string
    State      string
    Containers []string
    Ports      []PortMapping
}

type ProjectStatus int
const (
    StatusUnknown ProjectStatus = iota
    StatusRunning
    StatusPartial
    StatusStopped
    StatusError
)
```

### 4. 错误处理（C10.1）

```go
type ComposeError struct {
    Type       ErrorType
    Message    string
    Details    string
    Suggestion string
}

type ErrorType int
const (
    ErrorUnknown ErrorType = iota
    ErrorConfig      // compose 文件配置错误
    ErrorNetwork     // 网络错误（端口冲突等）
    ErrorImage       // 镜像相关错误
    ErrorRuntime     // 运行时错误
    ErrorPermission  // 权限错误
)
```

## 实现顺序

### 阶段 1：基础 API（3-5 天）
1. C1.1 - 检测 compose 命令
2. C1.2 - 定义接口
3. C2.1 - 定义数据模型
4. C2.2 - 配置集成
5. C4.1 - 实现 Up 操作
6. C4.2 - 实现 Down/Restart/Stop/Start
7. C4.3 - 实现 Logs 和 PS
8. C4.4 - 定义错误结构

### 阶段 2：扫描和列表（3-4 天）
1. C3.1 - 实现文件扫描
2. C3.2 - 实现过滤和去重
3. C3.3 - 提供刷新 API
4. C5.1 - 定义列表 Model
5. C5.2 - 实现列表布局和快捷键
6. C5.3 - 实现状态刷新
7. C5.4 - 集成到主视图

### 阶段 3：详情和联动（2-3 天）
1. C6.1 - 定义详情 Model
2. C6.2 - 实现服务列表标签页
3. C6.3 - 实现配置信息标签页
4. C7.1 - 实现容器过滤
5. C7.2 - 实现跳转功能
6. C7.3 - 在容器详情显示项目信息

### 阶段 4：日志和优化（3-5 天）
1. C8.1 - 实现项目级日志
2. C8.2 - 实现服务级日志
3. C9.1 - 实现环境重置
4. C9.2 - 实现快速重建
5. C9.3 - 实现开发模式配置
6. C10.1 - 实现友好错误提示
7. C10.2 - 实现进度显示
8. C10.3 - 实现操作历史

**总计：11-17 天**

## 技术难点

### 1. Compose 文件解析
- **难点**: 需要解析 YAML 格式，支持变量替换和多文件合并
- **方案**: 使用 `docker compose config` 命令获取合并后的配置

### 2. 多服务日志聚合
- **难点**: 需要同时读取多个服务的日志流，并区分显示
- **方案**: 使用 goroutine 并发读取，通过 channel 聚合，使用颜色区分

### 3. 长时间操作进度
- **难点**: up/build 等操作耗时长，需要实时反馈进度
- **方案**: 流式读取 stdout，解析进度信息，使用 spinner 或进度条显示

### 4. 容器项目关联
- **难点**: 需要准确识别容器属于哪个项目
- **方案**: 使用 Docker label `com.docker.compose.project` 和 `com.docker.compose.service`

## 测试计划

### 测试项目准备
1. **单服务项目** - 简单的 nginx 服务
2. **多服务项目** - web + db + redis
3. **带网络项目** - 自定义网络配置
4. **带卷项目** - 持久化数据
5. **复杂项目** - 多文件、环境变量、构建

### 测试场景
1. **正常流程** - 扫描、启动、停止、查看日志
2. **错误场景** - 端口冲突、镜像缺失、配置错误
3. **性能测试** - 大量项目扫描、频繁刷新
4. **跨平台** - Windows/Linux/macOS 兼容性

### 测试清单
- [ ] 项目扫描准确性
- [ ] 项目状态更新及时性
- [ ] 操作成功率
- [ ] 错误提示友好性
- [ ] 日志显示正确性
- [ ] 容器关联准确性
- [ ] UI 响应速度
- [ ] 内存占用合理性

## 用户体验设计

### 1. 视觉设计
- 使用与容器列表一致的风格
- 状态用颜色区分（绿/黄/灰/红）
- 重要信息高亮显示
- 错误信息醒目但不刺眼

### 2. 交互设计
- vim 风格快捷键保持一致
- 单键快速操作（无需确认）
- 危险操作显示确认对话框
- 支持搜索和过滤

### 3. 反馈设计
- 操作立即反馈（loading 状态）
- 成功消息 3 秒自动消失
- 错误消息持续显示，需手动关闭
- 长时间操作显示进度

### 4. 帮助设计
- 底部显示当前可用快捷键
- 按 `?` 显示完整帮助
- 错误提示包含解决建议
- 空状态显示操作提示

## 配置示例

### .docktui.yml（项目配置）
```yaml
# 项目名称（可选，默认使用目录名）
name: myapp

# 额外的 compose 文件
compose_files:
  - docker-compose.dev.yml
  - docker-compose.local.yml

# 环境变量文件
env_files:
  - .env.local
  - .env.dev

# 开发模式配置
dev_mode:
  # 自动挂载代码目录
  auto_mount: true
  
  # 热重载
  hot_reload: true
  
  # 额外的环境变量
  env:
    DEBUG: "true"
    LOG_LEVEL: "debug"
```

### config.go（全局配置）
```go
type ComposeConfig struct {
    // 扫描路径
    ScanPaths []string `yaml:"scan_paths"`
    
    // 忽略模式
    IgnorePatterns []string `yaml:"ignore_patterns"`
    
    // 最大扫描深度
    MaxDepth int `yaml:"max_depth"`
    
    // 默认环境变量文件
    DefaultEnvFile string `yaml:"default_env_file"`
    
    // 命令超时
    CommandTimeout time.Duration `yaml:"command_timeout"`
    
    // 自动刷新
    AutoRefresh bool `yaml:"auto_refresh"`
}
```

## 后续扩展

### 短期（1-2 个月）
- [ ] 支持 docker compose watch（文件变化自动重启）
- [ ] 支持 docker compose exec（进入服务容器）
- [ ] 支持服务扩缩容（scale）
- [ ] 支持项目模板（快速创建新项目）

### 中期（3-6 个月）
- [ ] 支持多环境管理（dev/test/prod）
- [ ] 支持项目依赖管理
- [ ] 支持项目导入导出
- [ ] 支持 Kubernetes 部署（compose → k8s）

### 长期（6-12 个月）
- [ ] 可视化编辑 compose 文件
- [ ] 项目性能监控和分析
- [ ] 项目成本估算
- [ ] 团队协作功能

## 总结

Docker Compose 支持是 DockTUI 的重要功能，将大大提升本地开发体验。通过分阶段实现，我们可以：

1. **快速交付 MVP** - 先实现核心功能，快速验证
2. **持续优化体验** - 根据反馈逐步完善
3. **保持代码质量** - 每个阶段都有明确的目标和测试
4. **降低实现风险** - 分阶段实现，及时发现和解决问题

预计 2-3 周可以完成 MVP 版本，后续根据用户反馈持续优化。
