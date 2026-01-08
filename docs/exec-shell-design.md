# Exec Shell 交互策略设计 (E1)

## E1.1: 两种交互方案评估

### 方案 A: 内嵌 Shell（在 TUI 内部）

**优点:**
- 用户体验连贯，不需要退出 TUI
- 可以在 TUI 中显示额外的状态信息
- 可以保持 TUI 的上下文状态

**缺点:**
- 实现复杂度高，需要处理终端原始模式切换
- 需要处理终端大小调整、特殊键等复杂场景
- Bubble Tea 框架需要暂停和恢复，容易出现状态问题
- 调试困难，终端状态管理容易出错
- Windows 平台的终端处理更加复杂

**技术挑战:**
- 需要使用 `term.MakeRaw()` 切换终端模式
- 需要正确保存和恢复 Bubble Tea 的状态
- 需要处理 SIGWINCH 信号（终端大小变化）
- 需要处理 Ctrl+C、Ctrl+D 等特殊键

### 方案 B: 退出 TUI 模式（推荐）

**优点:**
- 实现简单，直接调用 Docker SDK 的 ExecShell
- 使用系统原生终端，兼容性好
- 用户体验类似于 `docker exec -it`，符合用户习惯
- 调试简单，问题容易定位
- 跨平台兼容性好（Windows/Linux/macOS）

**缺点:**
- 需要退出 TUI，用户体验略有中断
- 需要在 shell 退出后重新启动 TUI

**技术实现:**
- 使用 `tea.Program.ReleaseTerminal()` 释放终端
- 调用 `docker.Client.ExecShell()` 进入容器
- Shell 退出后使用 `tea.Program.RestoreTerminal()` 恢复 TUI

### 决策: 选择方案 B（退出 TUI 模式）

**理由:**
1. **MVP 优先**: 方案 B 实现简单，可以快速交付功能
2. **稳定性**: 避免复杂的终端状态管理，减少 bug
3. **用户习惯**: 类似 `docker exec -it` 的体验，用户容易理解
4. **可维护性**: 代码简单，易于维护和扩展
5. **跨平台**: Windows 平台的终端处理更可靠

## E1.2: 用户交互流程设计

### 流程图

```
┌─────────────────┐
│  容器列表视图    │
│  或详情视图      │
└────────┬────────┘
         │
         │ 用户按 's' 键
         ▼
┌─────────────────┐
│  显示确认提示    │
│  "进入容器 Shell"│
│  "按 Enter 确认" │
└────────┬────────┘
         │
         │ 用户按 Enter
         ▼
┌─────────────────┐
│  释放终端控制    │
│  tea.Program.   │
│  ReleaseTerminal│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  调用 ExecShell  │
│  进入容器 Shell  │
│  (用户交互)      │
└────────┬────────┘
         │
         │ 用户输入 'exit' 或 Ctrl+D
         ▼
┌─────────────────┐
│  恢复终端控制    │
│  tea.Program.   │
│  RestoreTerminal│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  返回原视图      │
│  显示成功消息    │
└─────────────────┘
```

### 详细步骤

#### 1. 触发 Shell 入口
- **位置**: 容器列表视图或容器详情视图
- **快捷键**: `s` (Shell)
- **前置条件**: 必须选中一个运行中的容器
- **提示**: 显示容器名称和即将执行的 shell 命令

#### 2. 用户确认
- **提示信息**: 
  ```
  🐚 进入容器 Shell: [容器名称]
  
  将执行: /bin/sh (如果 /bin/bash 不可用)
  
  提示:
  - 输入 'exit' 或按 Ctrl+D 退出 shell
  - 退出后将返回 DockTUI
  
  按 Enter 继续，按 Esc 取消
  ```

#### 3. 释放终端
- 调用 `tea.Program.ReleaseTerminal()`
- 清屏并显示过渡信息

#### 4. 执行 Shell
- 尝试 shell 顺序: `/bin/bash` → `/bin/sh` → `/bin/ash`
- 如果失败，显示错误信息并返回 TUI
- 成功则用户进入完全交互式 shell

#### 5. Shell 退出处理
- 检测 shell 退出（exit code）
- 显示退出信息（成功/失败）
- 等待用户按任意键继续

#### 6. 恢复 TUI
- 调用 `tea.Program.RestoreTerminal()`
- 重新渲染当前视图
- 显示成功消息: "✅ 已退出容器 Shell"

### 错误处理

#### 容器未运行
```
❌ 错误: 容器未运行
只能在运行中的容器执行 shell
```

#### Shell 不存在
```
❌ 错误: 容器中没有可用的 shell
尝试的 shell: /bin/bash, /bin/sh, /bin/ash
```

#### 权限不足
```
❌ 错误: 权限不足
请检查 Docker 权限配置
```

#### 网络错误
```
❌ 错误: 无法连接到容器
请检查 Docker 守护进程状态
```

### 用户体验优化

#### 1. 智能 Shell 检测
- 优先尝试 `/bin/bash`（功能更丰富）
- 回退到 `/bin/sh`（POSIX 标准）
- 最后尝试 `/bin/ash`（Alpine Linux）

#### 2. 状态提示
- 进入前: 显示即将执行的命令
- 执行中: 完全交互式，无 TUI 干扰
- 退出后: 显示退出状态和返回提示

#### 3. 快捷键一致性
- `s`: 进入 Shell（在列表和详情视图）
- `Esc`: 取消操作（在确认提示时）
- `Enter`: 确认操作

## 实现计划

### Phase 1: 基础实现 (E2)
- [ ] 在 Docker Client 中完善 ExecShell 方法
- [ ] 添加 shell 自动检测逻辑
- [ ] 处理 Windows 平台兼容性

### Phase 2: TUI 集成 (E3)
- [ ] 在容器列表视图添加 's' 快捷键
- [ ] 在容器详情视图添加 's' 快捷键
- [ ] 实现确认提示界面
- [ ] 实现终端释放和恢复逻辑

### Phase 3: 错误处理和优化
- [ ] 添加完整的错误处理
- [ ] 添加用户友好的提示信息
- [ ] 测试各种边界情况

## 技术参考

### Bubble Tea 终端控制
```go
// 释放终端
program.ReleaseTerminal()

// 执行外部命令
err := dockerClient.ExecShell(ctx, containerID, "/bin/sh")

// 恢复终端
program.RestoreTerminal()
```

### Docker SDK Exec
```go
// 创建 exec
execConfig := container.ExecOptions{
    AttachStdin:  true,
    AttachStdout: true,
    AttachStderr: true,
    Tty:          true,
    Cmd:          []string{"/bin/sh"},
}
```

## 总结

选择"退出 TUI 模式"方案是基于以下考虑：
1. **简单性**: 实现简单，代码量少
2. **可靠性**: 避免复杂的终端状态管理
3. **用户体验**: 符合 Docker 用户的使用习惯
4. **可维护性**: 易于调试和扩展

这个方案可以快速实现 MVP，后续如果需要可以考虑升级到内嵌模式。
