# Docker Images 功能设计方案

## 一、功能概述

Docker Images 功能为 DockTUI 提供完整的镜像管理能力，与现有容器列表功能保持一致的交互风格和操作逻辑。

## 二、TUI 页面位置设计

### 2.1 导航方式
采用多标签页切换的方式集成，与现有容器列表功能并列：

- **顶部导航栏新增标签**：在当前 "Ops/Advanced/View" 区域旁，增加 `<i> Images` 标签
- **快捷键**：`i` 键在 "容器列表页" 和 "镜像列表页" 之间切换
- **状态保留**：切换时保留原页面的筛选/搜索状态，返回时自动恢复

### 2.2 页面布局

```
┌───────────────────────────────────────────────────────────────────────────────┐
│ Docker: Connected  <a> Filter  </> Search  <r> Refresh  <i> Images  <c> Containers │
│ Ops: <d> Delete  <p> Prune  <i> Inspect  <e> Export  <l> Load  <b> Build       │
│ Advanced: <t> Tag  <u> Untag  <P> Push  <p> Pull                               │
│ View: <Enter> Details  <Esc> Back  <q> Quit                                    │
│ Version: v0.1.0  Last Refresh: 0m12s ago  (vim): j/k上下  h/l左右  Enter=选择  │
├───────────────────────────────────────────────────────────────────────────────┤
│ 📦 Total: 8  | 🟢 Active: 3  | 🟡 Dangling: 2  | 🔴 Unused: 3                  │
│ ↔ 水平滚动  Scroll: 0%                                                         │
├───────────────────────────────────────────────────────────────────────────────┤
│ IMAGE ID       REPOSITORY                  TAG                 SIZE    CREATED  │
│ (选中行使用 Reverse 反转色)                                                    │
│ fcdcc70536a1   weshyper-dstore-elasticsearch 20250728112842  1.2GB   2 hour   │
│ ca2326618e11   weshyper-dstore-hop         20250728112842  890MB   2 hour   │
│ <none>         <none>                      <none>            500MB   1 day    │
└───────────────────────────────────────────────────────────────────────────────┘
```

## 三、核心功能需求

### 3.1 基础展示功能

| 展示项 | 说明 | 交互细节 |
|--------|------|----------|
| 镜像列表 | IMAGE ID、REPOSITORY、TAG、SIZE、CREATED | 支持按 Size/Created 排序（快捷键 s/t） |
| 镜像状态标识 | 悬垂镜像（`<none>:<none>`）标灰、活跃镜像（被容器使用）标绿 | 鼠标悬浮时显示 "被 XX 容器引用" 提示 |
| 统计信息 | 总镜像数、活跃数、悬垂数、未使用数 | 点击统计项可快速筛选对应镜像 |

### 3.2 核心操作功能

| 功能 | 快捷键 | 操作场景 | 交互反馈 |
|------|--------|----------|----------|
| 查看镜像详情 | `Enter` | 选中镜像后按回车 | 弹出详情窗口（包含 Digest、Layer 数、创建命令） |
| 删除镜像 | `d` | 选中镜像后按 d | 二次确认（避免误删），成功后列表自动刷新 |
| 清理悬垂镜像 | `p` | 顶部 Ops 栏的 Prune | 弹窗显示 "清理了 X 个镜像，释放 Y 空间" |
| 镜像打标签 | `t` | 选中镜像后按 t | 输入新标签（如 repo:v1.0），成功后列表中 REPOSITORY/TAG 更新 |
| 取消标签 | `u` | 选中镜像后按 u | 输入要删除的标签，确认后删除 |
| 导出镜像 | `e` | 选中镜像后按 e | 输入保存路径（如 ./image.tar），显示导出进度条 |
| 加载本地镜像 | `l` | 顶部 Ops 栏的 Load | 输入本地 tar 包路径，显示加载进度条 |
| 拉取镜像 | `p` | Advanced 栏的 Pull | 输入镜像名称（如 nginx:latest），显示拉取进度 |
| 推送镜像 | `P` | Advanced 栏的 Push | 选中镜像后推送到 registry，需要认证 |

## 四、交互体验优化

### 4.1 与容器页联动
- 在镜像详情中显示 "被哪些容器使用"
- 点击容器名可直接跳转到对应容器页
- 在容器详情中显示使用的镜像，点击可跳转到镜像列表

### 4.2 搜索/筛选增强
- 支持按 REPOSITORY（如输入 `weshyper` 筛选）
- 支持按 TAG（如 `v1.0`）
- 支持按 Size（如 `>1GB`）筛选
- 支持按状态筛选（活跃/悬垂/未使用）

### 4.3 悬垂镜像提醒
- 悬垂镜像行右侧增加 "⚠️" 标识
- 鼠标悬浮时提示 "此镜像无标签，可清理释放空间"
- 在统计栏突出显示悬垂镜像数量

## 五、技术实现建议

### 5.1 复用现有组件
- 基于当前容器列表的 `ScrollableTable` 组件改造
- 新增 IMAGE 相关列定义
- 复用搜索、筛选、排序逻辑

### 5.2 Docker SDK 调用
通过以下接口实现镜像管理：
- `ImageList` - 列出镜像
- `ImageInspect` - 查看详情
- `ImageRemove` - 删除镜像
- `ImagesPrune` - 清理悬垂镜像
- `ImageTag` - 打标签
- `ImageSave` - 导出镜像
- `ImageLoad` - 加载镜像
- `ImagePull` - 拉取镜像
- `ImagePush` - 推送镜像

### 5.3 状态同步
- 镜像操作后自动触发列表刷新
- 复用容器页的 Refresh 逻辑
- 支持手动刷新（快捷键 `r`）

### 5.4 数据结构设计

```go
// Image 镜像信息
type Image struct {
    ID          string    // 镜像 ID
    ShortID     string    // 短 ID（前 12 位）
    Repository  string    // 仓库名
    Tag         string    // 标签
    Size        int64     // 大小（字节）
    Created     time.Time // 创建时间
    Digest      string    // 摘要
    Labels      map[string]string // 标签
    
    // 运行时状态
    InUse       bool      // 是否被容器使用
    Dangling    bool      // 是否为悬垂镜像
    Containers  []string  // 使用此镜像的容器 ID 列表
}

// ImageListView 镜像列表视图
type ImageListView struct {
    dockerClient docker.Client
    
    // UI 尺寸
    width  int
    height int
    
    // 数据状态
    images           []docker.Image
    filteredImages   []docker.Image
    scrollTable      *ScrollableTable
    loading          bool
    errorMsg         string
    successMsg       string
    
    // 搜索状态
    searchQuery string
    isSearching bool
    
    // 筛选状态
    filterType string // "all", "active", "dangling", "unused"
    
    // 排序状态
    sortBy string // "size", "created", "repository"
    
    // 操作状态
    operatingImage *docker.Image
    operationType  string
    
    // 确认对话框
    showConfirmDialog bool
    confirmAction     string
    confirmSelection  int
    
    keys KeyMap
}
```

## 六、实现步骤

### 阶段 1：Docker 客户端层（2-3 天）
1. 在 `internal/docker` 中扩展 Client 接口
2. 实现镜像相关的 API 调用
3. 定义 Image 数据结构
4. 编写单元测试

### 阶段 2：UI 视图层（3-4 天）
1. 创建 `image_list_view.go`
2. 实现镜像列表渲染
3. 实现快捷键处理
4. 实现搜索和筛选功能

### 阶段 3：操作功能（2-3 天）
1. 实现删除镜像（带确认对话框）
2. 实现清理悬垂镜像
3. 实现打标签/取消标签
4. 实现导出/加载镜像

### 阶段 4：集成与优化（1-2 天）
1. 在主 Model 中集成镜像视图
2. 实现容器与镜像的双向跳转
3. 优化错误处理和用户提示
4. 性能测试和优化

### 总计：8-12 天

## 七、测试计划

### 7.1 功能测试
- 镜像列表展示（包含各种状态的镜像）
- 搜索和筛选功能
- 删除镜像（单个/批量）
- 清理悬垂镜像
- 打标签/取消标签
- 导出/加载镜像
- 拉取/推送镜像

### 7.2 交互测试
- 快捷键响应
- 视图切换（容器 ↔ 镜像）
- 确认对话框
- 错误提示
- 进度显示

### 7.3 性能测试
- 大量镜像（100+）的列表渲染
- 频繁刷新的性能
- 搜索和筛选的响应速度

## 八、注意事项

### 8.1 颜色自适应
- 遵循已实现的颜色自适应方案
- 不硬编码背景色
- 使用 `Reverse` 实现选中效果
- 只设置前景色，让终端使用默认背景

### 8.2 错误处理
- 镜像被容器使用时无法删除 → 提示哪些容器正在使用
- 权限不足 → 提示需要的权限
- 网络错误 → 显示详细错误信息和重试建议
- 磁盘空间不足 → 提示清理空间

### 8.3 用户体验
- 操作前二次确认（删除、清理）
- 长时间操作显示进度（导出、加载、拉取）
- 操作成功后自动刷新列表
- 友好的错误提示和解决建议

## 九、未来扩展

### 9.1 镜像构建
- 支持从 Dockerfile 构建镜像
- 显示构建进度和日志
- 支持构建参数配置

### 9.2 镜像历史
- 查看镜像的层历史
- 显示每层的大小和创建命令
- 支持层级展开/折叠

### 9.3 镜像扫描
- 集成漏洞扫描工具
- 显示安全风险评级
- 提供修复建议

### 9.4 Registry 管理
- 支持多个 Registry 配置
- 显示 Registry 中的镜像列表
- 支持 Registry 认证管理
