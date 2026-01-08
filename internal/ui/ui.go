package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"

	"docktui/internal/docker"
)

// ViewType 表示当前显示的视图类型
type ViewType int

const (
	// ViewWelcome 欢迎界面
	ViewWelcome ViewType = iota
	// ViewContainerList 容器列表视图
	ViewContainerList
	// ViewContainerDetail 容器详情视图
	ViewContainerDetail
	// ViewLogs 日志视图
	ViewLogs
	// ViewHelp 帮助视图
	ViewHelp
)

// View 接口定义了所有视图必须实现的方法
// 每个视图都应实现 Bubble Tea 的 Init、Update、View 方法
type View interface {
	// Init 初始化视图，返回初始化命令
	Init() tea.Cmd
	
	// Update 处理消息并更新视图状态
	Update(msg tea.Msg) (View, tea.Cmd)
	
	// View 渲染视图内容
	View() string
	
	// SetSize 设置视图尺寸（用于响应式布局）
	SetSize(width, height int)
}

// Model 是 TUI 的主模型，包含全局状态
type Model struct {
	// Docker 客户端
	dockerClient docker.Client
	
	// 当前视图类型
	currentView ViewType
	
	// 视图实例
	containerListView   View // 容器列表视图
	containerDetailView View // 容器详情视图
	logsView            View // 日志视图
	helpView            View // 帮助视图
	
	// 全局状态字段
	selectedContainerID string   // 当前选中的容器 ID
	previousView        ViewType // 上一个视图（用于返回）
	
	// 错误和状态显示
	errorMsg        string    // 错误消息（致命错误，常驻显示）
	warningMsg      string    // 警告消息（5秒后自动消失）
	infoMsg         string    // 信息提示（3秒后自动消失）
	successMsg      string    // 成功提示（3秒后自动消失）
	msgExpireTime   time.Time // 消息过期时间
	ready           bool      // 是否已完成初始化
	dockerConnected bool      // Docker 是否已连接
	
	// 窗口尺寸（用于响应式布局）
	width  int
	height int
}

func NewModel(dockerClient docker.Client) Model {
	// 初始化各个视图
	containerListView := NewContainerListView(dockerClient)
	containerDetailView := NewContainerDetailView(dockerClient)
	logsView := NewLogsView(dockerClient)
	helpView := NewHelpView(dockerClient)
	
	return Model{
		dockerClient:        dockerClient,
		currentView:         ViewWelcome,
		containerListView:   containerListView,
		containerDetailView: containerDetailView,
		logsView:            logsView,
		helpView:            helpView,
		ready:               false,
		dockerConnected:     true, // 默认假设已连接
	}
}

// SetDockerError 设置 Docker 连接错误（致命错误，常驻显示）
func SetDockerError(m Model, errMsg string) Model {
	m.dockerConnected = false
	m.errorMsg = errMsg
	return m
}

// SetTemporaryMessage 设置临时消息（支持自动消失）
type MessageType int

const (
	MsgInfo MessageType = iota
	MsgSuccess
	MsgWarning
	MsgError
)

func (m *Model) SetTemporaryMessage(msgType MessageType, text string, durationSec int) tea.Cmd {
	// 清除其他临时消息
	m.infoMsg = ""
	m.successMsg = ""
	m.warningMsg = ""
	
	// 设置新消息
	switch msgType {
	case MsgInfo:
		m.infoMsg = text
	case MsgSuccess:
		m.successMsg = text
	case MsgWarning:
		m.warningMsg = text
	case MsgError:
		// 致命错误不自动消失
		m.errorMsg = text
		return nil
	}
	
	// 设置过期时间
	m.msgExpireTime = time.Now().Add(time.Duration(durationSec) * time.Second)
	
	// 返回延迟清除命令
	return tea.Tick(time.Duration(durationSec)*time.Second, func(t time.Time) tea.Msg {
		return clearMessageMsg{}
	})
}

// clearMessageMsg 消息清除消息类型
type clearMessageMsg struct{}

// shellExitedMsg shell 退出消息类型
type shellExitedMsg struct {
	err error
}

// execShellMsg 执行 shell 的消息类型
type execShellMsg struct {
	containerID   string
	containerName string
}

// execShellCmd 实现 tea.ExecCommand 接口
type execShellCmd struct {
	dockerClient  docker.Client
	containerID   string
	containerName string
}

// Run 实现 tea.ExecCommand 接口
func (e execShellCmd) Run() error {
	// 清屏（进入 shell 前）
	fmt.Print("\033[2J\033[H")
	
	// 显示提示信息
	fmt.Printf("\n\033[1;36m🐚 进入容器 Shell: %s\033[0m\n", e.containerName)
	fmt.Println("\033[90m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println("\033[33m提示:\033[0m")
	fmt.Println("  • 输入 \033[1mexit\033[0m 或按 \033[1mCtrl+D\033[0m 退出 shell")
	fmt.Println("  • 退出后将自动返回 DockTUI")
	fmt.Println("\033[90m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println()
	
	// 尝试查找 docker 可执行文件
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		// 尝试常见的 Docker 安装路径
		possiblePaths := []string{
			"C:\\Program Files\\Docker\\Docker\\resources\\bin\\docker.exe",
			"C:\\Program Files\\Docker\\Docker\\docker.exe",
		}
		for _, p := range possiblePaths {
			if _, err := os.Stat(p); err == nil {
				dockerPath = p
				break
			}
		}
	}
	
	if dockerPath == "" {
		// 如果找不到 docker，回退到使用 Docker SDK
		fmt.Println("\033[33m使用 Docker SDK 模式...\033[0m")
		ctx := context.Background()
		err := e.dockerClient.ExecShell(ctx, e.containerID, "")
		fmt.Print("\033[2J\033[H")
		return err
	}
	
	// 使用 os/exec 执行 docker exec 命令
	cmd := exec.Command(dockerPath, "exec", "-it", e.containerID, "/bin/sh", "-c", 
		"if [ -x /bin/bash ]; then exec /bin/bash; elif [ -x /bin/ash ]; then exec /bin/ash; else exec /bin/sh; fi")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err = cmd.Run()
	
	// 清屏（退出 shell 后）
	fmt.Print("\033[2J\033[H")
	
	if err != nil {
		// 检查是否是正常退出
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 0 || exitErr.ExitCode() == 130 {
				return nil
			}
		}
		return err
	}
	
	return nil
}

// SetStdin 实现 tea.ExecCommand 接口（可选）
func (e execShellCmd) SetStdin(r io.Reader) {}

// SetStdout 实现 tea.ExecCommand 接口（可选）
func (e execShellCmd) SetStdout(w io.Writer) {}

// SetStderr 实现 tea.ExecCommand 接口（可选）
func (e execShellCmd) SetStderr(w io.Writer) {}

// execShell 执行容器 shell
func (m Model) execShell(containerID, containerName string) tea.Cmd {
	return func() tea.Msg {
		return execShellMsg{
			containerID:   containerID,
			containerName: containerName,
		}
	}
}

// createExecShellCmd 创建执行 shell 的命令
func (m Model) createExecShellCmd(containerID, containerName string) tea.ExecCommand {
	return execShellCmd{
		dockerClient:  m.dockerClient,
		containerID:   containerID,
		containerName: containerName,
	}
}

func (m Model) Init() tea.Cmd {
	// 初始化时不需要执行任何命令
	// 后续可以在这里加载容器列表等异步操作
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case execShellMsg:
		// 执行 shell 命令
		// 使用 tea.Exec 来暂时释放终端控制
		return m, tea.Exec(m.createExecShellCmd(msg.containerID, msg.containerName), func(err error) tea.Msg {
			return shellExitedMsg{err: err}
		})
	
	case shellExitedMsg:
		// Shell 退出后，触发界面刷新
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Shell 执行失败: %v", msg.err)
		}
		// 重新进入 alt screen 并刷新
		return m, tea.Sequence(
			tea.EnterAltScreen,
			tea.ClearScreen,
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)
		
	case clearMessageMsg:
		// 检查是否到达过期时间
		if time.Now().After(m.msgExpireTime) {
			m.infoMsg = ""
			m.successMsg = ""
			m.warningMsg = ""
		}
		return m, nil
		
	case tea.WindowSizeMsg:
		// 窗口尺寸变化，更新模型和所有视图
		m.width = msg.Width
		m.height = msg.Height
		
		// 通知所有视图更新尺寸
		if m.containerListView != nil {
			m.containerListView.SetSize(msg.Width, msg.Height)
		}
		if m.containerDetailView != nil {
			m.containerDetailView.SetSize(msg.Width, msg.Height)
		}
		if m.logsView != nil {
			m.logsView.SetSize(msg.Width, msg.Height)
		}
		if m.helpView != nil {
			m.helpView.SetSize(msg.Width, msg.Height)
		}
		return m, nil
		
	case tea.KeyMsg:
		// 处理全局快捷键
		newModel, cmd := m.handleGlobalKeys(msg)
		if cmd != nil {
			// 如果全局快捷键处理了消息（如退出），直接返回
			return newModel, cmd
		}
		
		// 检查模型是否发生了变化（如视图切换）
		// 将 tea.Model 转换为 Model 类型
		if modelPtr, ok := newModel.(Model); ok {
			if modelPtr.currentView != m.currentView {
				// 视图发生了切换，返回新模型
				return modelPtr, nil
			}
		}
		
		// 否则，将消息传递给当前活动的视图
		return m.delegateToCurrentView(msg)
	}
	
	// 其他消息也传递给当前视图
	return m.delegateToCurrentView(msg)
}

// handleGlobalKeys 处理全局快捷键
func (m Model) handleGlobalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 首先处理无条件全局快捷键（这些键在任何视图都优先处理）
	switch msg.String() {
	case "q", "ctrl+c":
		// 退出程序
		return m, tea.Quit
		
	case "?":
		// 显示帮助面板
		if m.currentView != ViewHelp {
			m.previousView = m.currentView
			m.currentView = ViewHelp
		}
		return m, nil
	}
	
	// ESC/b 键的特殊处理
	if msg.String() == "esc" || msg.String() == "b" {
		// 特殊情况：如果在容器列表的搜索模式，让视图自己处理
		if m.currentView == ViewContainerList {
			if listView, ok := m.containerListView.(*ContainerListView); ok {
				if listView.IsSearching() {
					return m, nil
				}
			}
		}
		
		// 其他情况，执行返回操作
		if m.currentView == ViewWelcome {
			// 已经在欢迎界面，不处理
			return m, nil
		}
		
		// 返回上一个视图
		if m.previousView != ViewWelcome {
			m.currentView = m.previousView
		} else {
			m.currentView = ViewWelcome
		}
		
		// 清除所有临时消息
		m.infoMsg = ""
		m.successMsg = ""
		m.warningMsg = ""
		return m, nil
	}
	
	// 根据当前视图处理不同的快捷键
	switch m.currentView {
	case ViewWelcome:
		return m.handleWelcomeKeys(msg)
	case ViewContainerList:
		return m.handleContainerListKeys(msg)
	case ViewContainerDetail:
		return m.handleContainerDetailKeys(msg)
	case ViewLogs:
		return m.handleLogsKeys(msg)
	case ViewHelp:
		return m.handleHelpKeys(msg)
	}
	
	return m, nil
}

// handleWelcomeKeys 处理欢迎界面的快捷键
func (m Model) handleWelcomeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !m.dockerConnected {
		// Docker 未连接时，只支持退出
		return m, nil
	}
	
	// 调试信息：显示按下的键（临时，可删除）
	// keyStr := msg.String()
	// m.SetTemporaryMessage(MsgInfo, "按下的键: " + keyStr, 2)
	
	switch msg.String() {
	case "c":
		// 切换到容器列表视图
		m.previousView = m.currentView
		m.currentView = ViewContainerList
		
		// 触发容器列表视图的初始化，加载数据
		var initCmd tea.Cmd
		if m.containerListView != nil {
			initCmd = m.containerListView.Init()
		}
		
		return m, initCmd
	}
	
	return m, nil
}

// handleContainerListKeys 处理容器列表视图的快捷键
func (m Model) handleContainerListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 如果处于搜索模式或显示确认对话框，让视图自己处理
	if listView, ok := m.containerListView.(*ContainerListView); ok {
		if listView.IsSearching() || listView.showConfirmDialog {
			return m, nil  // 返回 nil，让 Update 传递给视图
		}
	}
	
	switch msg.String() {
	case "enter":
		// 进入容器详情视图（L3.2）
		// 获取当前选中的容器
		if listView, ok := m.containerListView.(*ContainerListView); ok {
			if container := listView.GetSelectedContainer(); container != nil {
				// 设置选中的容器 ID
				m.selectedContainerID = container.ID
				
				// 设置详情视图的容器信息
				if detailView, ok := m.containerDetailView.(*ContainerDetailView); ok {
					detailView.SetContainer(container.ID, container.Name)
				}
				
				// 切换到详情视图
				m.previousView = m.currentView
				m.currentView = ViewContainerDetail
				
				// 初始化详情视图
				var initCmd tea.Cmd
				if m.containerDetailView != nil {
					initCmd = m.containerDetailView.Init()
				}
				
				// 显示成功消息，包含容器名称
				msg := fmt.Sprintf("✅ 已进入容器详情: %s", container.Name)
				return m, tea.Batch(
					m.SetTemporaryMessage(MsgSuccess, msg, 3),
					initCmd,
				)
			} else {
				// 没有选中的容器
				return m, m.SetTemporaryMessage(MsgWarning, "⚠️ 请先选择一个容器", 3)
			}
		}
		return m, m.SetTemporaryMessage(MsgError, "❌ 视图错误", 3)
		
	case "l":
		// 查看容器日志
		if listView, ok := m.containerListView.(*ContainerListView); ok {
			if container := listView.GetSelectedContainer(); container != nil {
				// 设置日志视图的容器信息
				if logsView, ok := m.logsView.(*LogsView); ok {
					logsView.SetContainer(container.ID, container.Name)
				}
				
				m.previousView = m.currentView
				m.currentView = ViewLogs
				
				// 初始化日志视图
				var initCmd tea.Cmd
				if m.logsView != nil {
					initCmd = m.logsView.Init()
				}
				
				msg := fmt.Sprintf("📜 正在加载容器日志: %s", container.Name)
				return m, tea.Batch(
					m.SetTemporaryMessage(MsgSuccess, msg, 3),
					initCmd,
				)
			} else {
				return m, m.SetTemporaryMessage(MsgWarning, "⚠️ 请先选择一个容器", 3)
			}
		}
		return m, m.SetTemporaryMessage(MsgError, "❌ 视图错误", 3)
		
	case "r":
		// 刷新容器列表（后续实现）
		return m, m.SetTemporaryMessage(MsgInfo, "🔄 正在刷新容器列表...", 3)
		
	case "s":
		// 进入容器 Shell
		if listView, ok := m.containerListView.(*ContainerListView); ok {
			if container := listView.GetSelectedContainer(); container != nil {
				// 检查容器是否在运行
				if container.State != "running" {
					return m, m.SetTemporaryMessage(MsgWarning, "⚠️ 只能在运行中的容器执行 shell", 3)
				}
				
				// 设置选中的容器信息
				m.selectedContainerID = container.ID
				
				// 执行 shell（这里需要特殊处理）
				return m, m.execShell(container.ID, container.Name)
			} else {
				return m, m.SetTemporaryMessage(MsgWarning, "⚠️ 请先选择一个容器", 3)
			}
		}
		return m, m.SetTemporaryMessage(MsgError, "❌ 视图错误", 3)
	}
	
	// 其他按键不处理，返回 nil 让 Update 函数传递给视图
	return m, nil
}

// handleContainerDetailKeys 处理容器详情视图的快捷键
func (m Model) handleContainerDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 只处理特定的快捷键，其他的让视图自己处理
	switch msg.String() {
	case "l":
		// 从详情视图查看容器日志
		if m.selectedContainerID != "" {
			// 从详情视图获取容器名称
			containerName := m.selectedContainerID[:12] // 默认使用短 ID
			if detailView, ok := m.containerDetailView.(*ContainerDetailView); ok {
				if detailView.details != nil {
					containerName = detailView.details.Name
				}
			}
			
			// 设置日志视图的容器信息
			if logsView, ok := m.logsView.(*LogsView); ok {
				logsView.SetContainer(m.selectedContainerID, containerName)
			}
			
			m.previousView = m.currentView
			m.currentView = ViewLogs
			
			// 初始化日志视图
			var initCmd tea.Cmd
			if m.logsView != nil {
				initCmd = m.logsView.Init()
			}
			
			return m, tea.Batch(
				m.SetTemporaryMessage(MsgSuccess, fmt.Sprintf("📜 正在加载容器日志: %s", containerName), 3),
				initCmd,
			)
		}
		return m, m.SetTemporaryMessage(MsgWarning, "⚠️ 未选择容器", 3)
		
	case "s":
		// 进入容器 Shell
		if m.selectedContainerID != "" {
			// 从详情视图获取容器名称和状态
			containerName := m.selectedContainerID[:12]
			containerState := "unknown"
			if detailView, ok := m.containerDetailView.(*ContainerDetailView); ok {
				if detailView.details != nil {
					containerName = detailView.details.Name
					containerState = detailView.details.State
				}
			}
			
			// 检查容器是否在运行
			if containerState != "running" {
				return m, m.SetTemporaryMessage(MsgWarning, "⚠️ 只能在运行中的容器执行 shell", 3)
			}
			
			// 执行 shell
			return m, m.execShell(m.selectedContainerID, containerName)
		}
		return m, m.SetTemporaryMessage(MsgWarning, "⚠️ 未选择容器", 3)
	}
	
	// 其他按键不处理，返回 nil 让消息传递给视图
	return m, nil
}

// handleLogsKeys 处理日志视图的快捷键
func (m Model) handleLogsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 日志视图的按键都由视图自己处理，这里不拦截任何按键
	return m, nil
}

// handleHelpKeys 处理帮助视图的快捷键
func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 帮助视图中，ESC/b 由全局处理
	// 这里不需要处理任何按键
	return m, nil
}

func (m Model) View() string {
	var content string
	
	// 根据当前视图类型显示不同内容
	switch m.currentView {
	case ViewWelcome:
		content = m.renderWelcome()
	case ViewContainerList:
		// 调用容器列表视图的 View 方法
		if m.containerListView != nil {
			content = m.containerListView.View()
		} else {
			content = m.renderContainerList()
		}
	case ViewContainerDetail:
		// 调用容器详情视图的 View 方法
		if m.containerDetailView != nil {
			content = m.containerDetailView.View()
		} else {
			content = m.renderContainerDetail()
		}
	case ViewLogs:
		// 调用日志视图的 View 方法
		if m.logsView != nil {
			content = m.logsView.View()
		} else {
			content = m.renderLogs()
		}
	case ViewHelp:
		// 调用帮助视图的 View 方法
		if m.helpView != nil {
			content = m.helpView.View()
		} else {
			content = "🆘 帮助视图未初始化"
		}
	default:
		content = "未知视图"
	}
	
	// 添加分级消息显示（顶部：致命错误；底部：临时提示）
	// 注意：容器列表视图有自己的消息系统，不需要全局消息
	if m.currentView == ViewContainerList {
		// 容器列表视图自己处理消息显示
		return content
	}
	
	var statusBar string
	
	// 1. 致命错误（顶部常驻）
	if m.errorMsg != "" && m.dockerConnected {
		// Docker 已连接但有致命错误
		statusBar = "\n\033[1;31m❌ 致命错误: " + m.errorMsg + "\033[0m\n" + content
		content = statusBar
	}
	
	// 2. 警告消息（5秒自动消失）
	if m.warningMsg != "" {
		content += "\n\n\033[1;33m⚠️ 警告: " + m.warningMsg + "\033[0m"
	}
	
	// 3. 信息提示（3秒自动消失）
	if m.infoMsg != "" {
		content += "\n\n\033[36m" + m.infoMsg + "\033[0m"
	}
	
	// 4. 成功提示（3秒自动消失）
	if m.successMsg != "" {
		content += "\n\n\033[1;32m" + m.successMsg + "\033[0m"
	}
	
	return content
}

// renderWelcome 渲染欢迎界面（主导航页面）
func (m Model) renderWelcome() string {
	var s string
	
	s += "\n"
	s += "  ╔═══════════════════════════════════════════════════════════════════════════╗\n"
	s += "  ║                                                                           ║\n"
	s += "  ║                  🐳  DockTUI - Docker 管理工具  🐳                        ║\n"
	s += "  ║                                                                           ║\n"
	s += "  ╚═══════════════════════════════════════════════════════════════════════════╝\n"
	s += "\n"
	
	// Docker 连接状态
	if m.dockerConnected {
		s += "  ✅ Docker 守护进程已连接\n"
		s += "\n"
		
		// 主功能导航
		s += "  ╭─────────────────────────────────────────────────────────────────────────╮\n"
		s += "  │                           📋 主功能菜单                                  │\n"
		s += "  ╰─────────────────────────────────────────────────────────────────────────╯\n"
		s += "\n"
		s += "     \033[1;36m[c]\033[0m  📦 容器管理          - 查看、操作 Docker 容器\n"
		s += "     \033[90m[i]\033[0m  🖼️  镜像管理          - 查看、管理 Docker 镜像 \033[90m(待实现)\033[0m\n"
		s += "     \033[90m[n]\033[0m  🌐 网络管理          - 查看、配置 Docker 网络 \033[90m(待实现)\033[0m\n"
		s += "     \033[90m[v]\033[0m  💾 卷管理            - 查看、管理 Docker 卷   \033[90m(待实现)\033[0m\n"
		s += "     \033[90m[p]\033[0m  🐙 Compose 项目      - 管理 docker-compose   \033[90m(待实现)\033[0m\n"
		s += "\n"
		
		// 快捷操作
		s += "  ╭─────────────────────────────────────────────────────────────────────────╮\n"
		s += "  │                           ⚡ 快捷操作                                    │\n"
		s += "  ╰─────────────────────────────────────────────────────────────────────────╯\n"
		s += "\n"
		s += "     \033[1;36m[?]\033[0m  🆘 帮助面板          - 查看所有快捷键和功能说明\n"
		s += "     \033[1;36m[q]\033[0m  ❌ 退出程序          - 退出 DockTUI\n"
		s += "\n"
		
		// 提示信息
		s += "  ╭─────────────────────────────────────────────────────────────────────────╮\n"
		s += "  │                           💡 使用提示                                    │\n"
		s += "  ╰─────────────────────────────────────────────────────────────────────────╯\n"
		s += "\n"
		s += "     • 使用 \033[1mvim 风格\033[0m 快捷键导航 (j/k 上下移动)\n"
		s += "     • 按 \033[1mEnter\033[0m 进入选中项，按 \033[1mEsc/b\033[0m 返回上级\n"
		s += "     • 在容器列表中按 \033[1ms\033[0m 可直接进入容器 Shell\n"
		s += "     • 按 \033[1m?\033[0m 随时查看完整帮助文档\n"
		s += "\n"
		
	} else {
		// Docker 连接失败
		s += "  ❌ 无法连接到 Docker 守护进程\n"
		s += "\n"
		s += "  ╭─────────────────────────────────────────────────────────────────────────╮\n"
		s += "  │                           💡 解决方案                                    │\n"
		s += "  ╰─────────────────────────────────────────────────────────────────────────╯\n"
		s += "\n"
		s += "     1️⃣  确保 Docker Desktop 已启动并运行\n"
		s += "\n"
		s += "     2️⃣  远程连接 Docker (设置环境变量):\n"
		s += "        \033[90mWindows CMD:\033[0m\n"
		s += "        set DOCKER_HOST=tcp://192.168.3.49:2375\n"
		s += "\n"
		s += "        \033[90mWindows PowerShell:\033[0m\n"
		s += "        $env:DOCKER_HOST=\"tcp://192.168.3.49:2375\"\n"
		s += "\n"
		s += "        \033[90mLinux/macOS:\033[0m\n"
		s += "        export DOCKER_HOST=tcp://192.168.3.49:2375\n"
		s += "\n"
		s += "     3️⃣  检查 Docker 服务状态:\n"
		s += "        docker ps\n"
		s += "\n"
		
		if m.errorMsg != "" {
			s += "  ╭─────────────────────────────────────────────────────────────────────────╮\n"
			s += "  │                           📝 错误详情                                    │\n"
			s += "  ╰─────────────────────────────────────────────────────────────────────────╯\n"
			s += "\n"
			s += "     " + m.errorMsg + "\n"
			s += "\n"
		}
		
		s += "  ⚠️  请解决 Docker 连接问题后重新启动程序\n"
		s += "\n"
		s += "     按 \033[1mq\033[0m 退出程序\n"
		s += "\n"
	}
	
	return s
}

// renderContainerList 渲染容器列表视图
func (m Model) renderContainerList() string {
	var s string
	
	s += "\n"
	s += "  ╔═══════════════════════════════════════════════════════════╗\n"
	s += "  ║                    📦 容器列表                      ║\n"
	s += "  ╚═══════════════════════════════════════════════════════════╝\n"
	s += "\n"
	s += "  🚧 此视图尚未实现，请等待 U3/L1 任务完成。\n"
	s += "\n"
	s += "  ⌨️  快捷键：\n"
	s += "     q / Ctrl+C  - 退出程序\n"
	s += "     Esc / b     - 返回欢迎界面\n"
	s += "     r           - 刷新列表（待实现）\n"
	s += "     Enter       - 查看详情（待实现）\n"
	s += "     l           - 查看日志（待实现）\n"
	s += "\n"
	
	return s
}

// renderContainerDetail 渲染容器详情视图
func (m Model) renderContainerDetail() string {
	var s string
	
	s += "\n"
	s += "  ╔═══════════════════════════════════════════════════════════╗\n"
	s += "  ║                    📋 容器详情                      ║\n"
	s += "  ╚═══════════════════════════════════════════════════════════╝\n"
	s += "\n"
	s += "  🚧 此视图尚未实现，请等待 V1/V2 任务完成。\n"
	s += "\n"
	s += "  ⌨️  快捷键：\n"
	s += "     q / Ctrl+C  - 退出程序\n"
	s += "     Esc / b     - 返回列表\n"
	s += "     l           - 查看日志（待实现）\n"
	s += "\n"
	
	return s
}

// renderLogs 渲染日志视图
func (m Model) renderLogs() string {
	var s string
	
	s += "\n"
	s += "  ╔═══════════════════════════════════════════════════════════╗\n"
	s += "  ║                    📜 容器日志                      ║\n"
	s += "  ╚═══════════════════════════════════════════════════════════╝\n"
	s += "\n"
	s += "  🚧 此视图尚未实现，请等待 G1/G2 任务完成。\n"
	s += "\n"
	s += "  ⌨️  快捷键：\n"
	s += "     q / Ctrl+C  - 退出程序\n"
	s += "     Esc / b     - 返回上一个视图\n"
	s += "     f           - 切换 Follow 模式（待实现）\n"
	s += "\n"
	
	return s
}

// delegateToCurrentView 将消息委托给当前活动的视图处理
func (m Model) delegateToCurrentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	// 根据当前视图类型，将消息传递给对应的视图实例
	switch m.currentView {
	case ViewContainerList:
		if m.containerListView != nil {
			var updatedView View
			updatedView, cmd = m.containerListView.Update(msg)
			m.containerListView = updatedView
		}
		
	case ViewContainerDetail:
		if m.containerDetailView != nil {
			var updatedView View
			updatedView, cmd = m.containerDetailView.Update(msg)
			m.containerDetailView = updatedView
		}
		
	case ViewLogs:
		if m.logsView != nil {
			var updatedView View
			updatedView, cmd = m.logsView.Update(msg)
			m.logsView = updatedView
		}
	}
	
	return m, cmd
}
