package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	sdk "github.com/docker/docker/client"

	"docktui/internal/compose"
	"docktui/internal/ui/components"
	composeui "docktui/internal/ui/compose"
	containerui "docktui/internal/ui/container"
	imageui "docktui/internal/ui/image"
	networkui "docktui/internal/ui/network"
	"docktui/internal/docker"
)

// 全局主题颜色定义 - 使用自适应颜色，不硬编码背景色
// 让终端自己处理背景，只设置前景色
var (
	// 主文字颜色 - 使用终端默认前景色（不设置）
	// ThemeTextColor - 不再使用固定颜色
	
	// 次要文字颜色 - 灰色，在亮色和暗色终端都可读
	ThemeTextMuted = lipgloss.Color("245")
	
	// 边框颜色 - 中性灰色
	ThemeBorderColor = lipgloss.Color("240")
	
	// 高亮颜色 - 青色，在两种主题下都醒目
	ThemeHighlight = lipgloss.Color("81")
	
	// 成功颜色 - 绿色
	ThemeSuccess = lipgloss.Color("82")
	
	// 警告颜色 - 黄色
	ThemeWarning = lipgloss.Color("220")
	
	// 错误颜色 - 红色
	ThemeError = lipgloss.Color("196")
	
	// 标题颜色 - 黄色/金色
	ThemeTitleColor = lipgloss.Color("220")
	
	// 按键提示颜色 - 青色
	ThemeKeyColor = lipgloss.Color("81")
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
	// ViewComposeList Compose 项目列表视图
	ViewComposeList
	// ViewImageList 镜像列表视图
	ViewImageList
	// ViewImageDetails 镜像详情视图
	ViewImageDetails
	// ViewNetworkList 网络列表视图
	ViewNetworkList
	// ViewNetworkDetail 网络详情视图
	ViewNetworkDetail
	// ViewComposeDetail Compose 项目详情视图
	ViewComposeDetail
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
	homeView            *HomeView         // 首页导航视图
	containerListView   *containerui.ListView   // 容器列表视图
	containerDetailView *containerui.DetailView // 容器详情视图
	logsView            *containerui.LogsView // 日志视图
	helpView            View              // 帮助视图
	composeListView     *composeui.ListView   // Compose 项目列表视图
	imageListView       *imageui.ListView     // 镜像列表视图
	imageDetailsView    *imageui.DetailsView  // 镜像详情视图
	networkListView     *networkui.ListView   // 网络列表视图
	networkDetailView   *networkui.DetailView // 网络详情视图
	composeDetailView   *composeui.DetailView // Compose 项目详情视图
	shellSelector       *components.ShellSelector // Shell 选择器
	
	// 全局状态字段
	selectedContainerID string   // 当前选中的容器 ID
	previousView        ViewType // 上一个视图（用于返回）
	showShellSelector   bool     // 是否显示 Shell 选择器
	
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
	homeView := NewHomeView(dockerClient)
	containerListView := containerui.NewListView(dockerClient)
	containerDetailView := containerui.NewDetailView(dockerClient)
	logsView := containerui.NewLogsView(dockerClient)
	helpView := NewHelpView(dockerClient)
	imageListView := imageui.NewListView(dockerClient)
	networkListView := networkui.NewListView(dockerClient)
	
	// 初始化 Compose 客户端和视图
	var composeListView *composeui.ListView
	var composeDetailView *composeui.DetailView
	composeClient, err := compose.NewClient()
	if err == nil {
		// 获取 Docker SDK 客户端用于项目发现
		var sdkClient *sdk.Client
		if localClient, ok := dockerClient.(*docker.LocalClient); ok {
			sdkClient = localClient.GetSDKClient()
		}
		composeListView = composeui.NewListView(composeClient, sdkClient)
		composeDetailView = composeui.NewDetailView(composeClient)
	}
	
	// 初始化 Shell 选择器
	shellSelector := components.NewShellSelector(dockerClient)
	
	return Model{
		dockerClient:        dockerClient,
		currentView:         ViewWelcome,
		homeView:            homeView,
		containerListView:   containerListView,
		containerDetailView: containerDetailView,
		logsView:            logsView,
		helpView:            helpView,
		composeListView:     composeListView,
		composeDetailView:   composeDetailView,
		imageListView:       imageListView,
		networkListView:     networkListView,
		shellSelector:       shellSelector,
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
	shell         string // 指定的 Shell 路径
}

// execShellCmd 实现 tea.ExecCommand 接口
type execShellCmd struct {
	dockerClient  docker.Client
	containerID   string
	containerName string
	shell         string // 指定的 Shell 路径
}

// Run 实现 tea.ExecCommand 接口
func (e execShellCmd) Run() error {
	// 清屏（进入 shell 前）
	fmt.Print("\033[2J\033[H")
	
	// 获取 Shell 名称用于显示
	shellName := e.shell
	if shellName == "" {
		shellName = "auto"
	} else {
		// 提取 Shell 名称（如 /bin/bash -> bash）
		parts := strings.Split(shellName, "/")
		if len(parts) > 0 {
			shellName = parts[len(parts)-1]
		}
	}
	
	// 显示提示信息
	fmt.Printf("\n\033[1;36m🐚 进入容器 Shell: %s (%s)\033[0m\n", e.containerName, shellName)
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
		err := e.dockerClient.ExecShell(ctx, e.containerID, e.shell)
		fmt.Print("\033[2J\033[H")
		return err
	}
	
	// 构建 docker exec 命令
	var cmd *exec.Cmd
	if e.shell != "" {
		// 使用指定的 Shell
		cmd = exec.Command(dockerPath, "exec", "-it", e.containerID, e.shell)
	} else {
		// 自动检测 Shell
		cmd = exec.Command(dockerPath, "exec", "-it", e.containerID, "/bin/sh", "-c", 
			"if [ -x /bin/bash ]; then exec /bin/bash; elif [ -x /bin/ash ]; then exec /bin/ash; else exec /bin/sh; fi")
	}
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

// execShell 执行容器 shell（带指定 Shell）
func (m Model) execShellWithShell(containerID, containerName, shell string) tea.Cmd {
	return func() tea.Msg {
		return execShellMsg{
			containerID:   containerID,
			containerName: containerName,
			shell:         shell,
		}
	}
}

// createExecShellCmd 创建执行 shell 的命令
func (m Model) createExecShellCmd(containerID, containerName, shell string) tea.ExecCommand {
	return execShellCmd{
		dockerClient:  m.dockerClient,
		containerID:   containerID,
		containerName: containerName,
		shell:         shell,
	}
}

func (m Model) Init() tea.Cmd {
	// 初始化首页视图，加载统计数据
	if m.homeView != nil {
		return m.homeView.Init()
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case GoBackMsg, imageui.GoBackMsg, networkui.GoBackMsg, composeui.GoBackMsg:
		// 视图请求返回上一级
		return m.goBack()
	
	// ========== 视图切换请求消息 ==========
	case imageui.ViewImageDetailsMsg:
		// 镜像列表视图请求切换到镜像详情
		if msg.Image != nil {
			m.imageDetailsView = imageui.NewDetailsView(m.dockerClient, msg.Image)
			m.imageDetailsView.SetSize(m.width, m.height)
			m.previousView = m.currentView
			m.currentView = ViewImageDetails
			return m, m.imageDetailsView.Init()
		}
		return m, nil
	
	case containerui.ViewDetailsMsg:
		// 容器列表视图请求切换到容器详情
		m.selectedContainerID = msg.ContainerID
		if m.containerDetailView != nil {
			m.containerDetailView.SetContainer(msg.ContainerID, msg.ContainerName)
		}
		m.previousView = m.currentView
		m.currentView = ViewContainerDetail
		var initCmd tea.Cmd
		if m.containerDetailView != nil {
			initCmd = m.containerDetailView.Init()
		}
		return m, initCmd
	
	case containerui.ViewLogsMsg:
		// 容器列表视图请求切换到日志视图
		if m.logsView != nil {
			m.logsView.SetContainer(msg.ContainerID, msg.ContainerName)
		}
		m.previousView = m.currentView
		m.currentView = ViewLogs
		var initCmd tea.Cmd
		if m.logsView != nil {
			initCmd = m.logsView.Init()
		}
		return m, initCmd
	
	case networkui.ViewNetworkDetailsMsg:
		// 网络列表视图请求切换到网络详情
		if msg.Network != nil {
			m.networkDetailView = networkui.NewDetailView(m.dockerClient, msg.Network)
			m.networkDetailView.SetSize(m.width, m.height)
			m.previousView = m.currentView
			m.currentView = ViewNetworkDetail
			return m, m.networkDetailView.Init()
		}
		return m, nil
	
	case GoToComposeDetailMsg:
		// Compose 列表视图请求切换到项目详情
		if msg.Project != nil {
			if project, ok := msg.Project.(*compose.Project); ok {
				if m.composeDetailView != nil {
					m.composeDetailView.SetProject(project)
					m.composeDetailView.SetSize(m.width, m.height)
					m.previousView = m.currentView
					m.currentView = ViewComposeDetail
					return m, m.composeDetailView.Init()
				}
			}
		}
		return m, nil
	
	case composeui.GoToDetailMsg:
		// Compose 列表视图请求切换到项目详情（来自 compose 子包）
		if msg.Project != nil {
			if m.composeDetailView != nil {
				m.composeDetailView.SetProject(msg.Project)
				m.composeDetailView.SetSize(m.width, m.height)
				m.previousView = m.currentView
				m.currentView = ViewComposeDetail
				return m, m.composeDetailView.Init()
			}
		}
		return m, nil
	
	case composeui.GoToContainerDetailMsg:
		// Compose 详情视图请求跳转到容器详情
		m.selectedContainerID = msg.ContainerID
		if m.containerDetailView != nil {
			m.containerDetailView.SetContainer(msg.ContainerID, msg.ContainerName)
		}
		m.previousView = m.currentView
		m.currentView = ViewContainerDetail
		var initCmd tea.Cmd
		if m.containerDetailView != nil {
			initCmd = m.containerDetailView.Init()
		}
		return m, initCmd
	
	case containerui.GoBackMsg:
		// 容器视图请求返回上一级
		return m.goBack()
	
	case execShellMsg:
		// 执行 shell 命令
		// 使用 tea.Exec 来暂时释放终端控制
		return m, tea.Exec(m.createExecShellCmd(msg.containerID, msg.containerName, msg.shell), func(err error) tea.Msg {
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
		if m.homeView != nil {
			m.homeView.SetSize(msg.Width, msg.Height)
		}
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
		if m.composeListView != nil {
			m.composeListView.SetSize(msg.Width, msg.Height)
		}
		if m.imageListView != nil {
			m.imageListView.SetSize(msg.Width, msg.Height)
		}
		if m.networkListView != nil {
			m.networkListView.SetSize(msg.Width, msg.Height)
		}
		if m.networkDetailView != nil {
			m.networkDetailView.SetSize(msg.Width, msg.Height)
		}
		if m.composeDetailView != nil {
			m.composeDetailView.SetSize(msg.Width, msg.Height)
		}
		if m.shellSelector != nil {
			m.shellSelector.SetSize(msg.Width, msg.Height)
		}
		return m, nil
	
	// 处理 Shell 选择器的消息
	case components.ShellsDetectedMsg, components.ShellsDetectErrorMsg:
		if m.showShellSelector && m.shellSelector != nil {
			cmd := m.shellSelector.Update(msg)
			return m, cmd
		}
		return m, nil
		
	case tea.KeyMsg:
		// 如果 Shell 选择器正在显示，优先处理
		if m.showShellSelector && m.shellSelector != nil {
			switch msg.String() {
			case "enter":
				// 选择 Shell 并执行
				shell := m.shellSelector.GetSelectedShell()
				if shell != "" {
					m.showShellSelector = false
					// 获取容器信息
					containerID := m.shellSelector.ContainerID()
					containerName := m.shellSelector.ContainerName()
					return m, m.execShellWithShell(containerID, containerName, shell)
				}
			case "esc", "q":
				// 取消选择
				m.showShellSelector = false
				return m, nil
			default:
				// 其他按键传递给选择器
				cmd := m.shellSelector.Update(msg)
				return m, cmd
			}
			return m, nil
		}
		
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
	// 如果镜像列表视图的拉取输入框或打标签输入框或错误弹窗可见，不处理任何全局快捷键
	if m.currentView == ViewImageList && m.imageListView != nil {
		if m.imageListView.IsPullInputVisible() ||
		   m.imageListView.IsTagInputVisible() ||
		   m.imageListView.HasError() {
			return m, nil
		}
	}
	
	// 如果网络列表视图的错误弹窗或确认对话框可见，不处理任何全局快捷键
	if m.currentView == ViewNetworkList && m.networkListView != nil {
		if m.networkListView.HasError() || m.networkListView.ShowConfirmDialog() || m.networkListView.ShowFilterMenu() || m.networkListView.IsShowingCreateView() {
			return m, nil
		}
	}
	
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
	
	// ESC 键 - 让视图自己处理，视图会发送 GoBackMsg 来请求返回
	// 不在全局处理 ESC，避免复杂的状态检查
	
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
	case ViewComposeList:
		return m.handleComposeListKeys(msg)
	case ViewImageList:
		return m.handleImageListKeys(msg)
	case ViewNetworkList:
		return m.handleNetworkListKeys(msg)
	}
	
	return m, nil
}

// handleWelcomeKeys 处理欢迎界面的快捷键
func (m Model) handleWelcomeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !m.dockerConnected {
		// Docker 未连接时，只支持退出
		return m, nil
	}
	
	// 导航键由 HomeView 处理
	switch msg.String() {
	case "up", "down", "left", "right", "h", "j", "k", "l", "tab":
		if m.homeView != nil {
			m.homeView.Update(msg)
		}
		// 返回空命令，防止 delegateToCurrentView 再次处理
		return m, func() tea.Msg { return nil }
	case "r", "f5":
		// 刷新
		if m.homeView != nil {
			return m, m.homeView.Init()
		}
		return m, nil
	}
	
	switch msg.String() {
	case "enter":
		// 根据选中的卡片进入对应视图
		if m.homeView != nil {
			// 根据选中的资源进入对应视图
			if !m.homeView.IsResourceAvailable() {
				return m, m.SetTemporaryMessage(MsgWarning, "⚠️ 该功能暂不可用", 3)
			}
			
			switch m.homeView.GetSelectedResource() {
			case ResourceContainers:
				return m.enterContainerList()
			case ResourceImages:
				return m.enterImageList()
			case ResourceCompose:
				return m.enterComposeList()
			case ResourceNetworks:
				return m.enterNetworkList()
			case ResourceVolumes:
				return m, m.SetTemporaryMessage(MsgInfo, "💾 卷管理功能开发中...", 3)
			}
		}
		return m, nil
		
	case "1":
		// 直接进入容器列表
		return m.enterContainerList()
		
	case "2":
		// 进入镜像列表
		return m.enterImageList()
	
	case "3", "4":
		// 网络和卷管理（开发中）
		return m, m.SetTemporaryMessage(MsgInfo, "🚧 该功能开发中...", 3)
	
	case "5":
		// 进入 Compose 视图
		return m.enterComposeList()
		
	case "c":
		// 快捷键进入容器列表
		return m.enterContainerList()
	
	case "i":
		// 快捷键进入镜像列表
		return m.enterImageList()
	
	case "n":
		// 快捷键进入网络管理
		return m.enterNetworkList()
	
	case "v":
		// 快捷键进入卷管理（开发中）
		return m, m.SetTemporaryMessage(MsgInfo, "💾 卷管理功能开发中...", 3)
	
	case "o":
		// 快捷键进入 Compose 视图
		return m.enterComposeList()
	}
	
	return m, nil
}

// enterContainerList 进入容器列表视图
func (m Model) enterContainerList() (tea.Model, tea.Cmd) {
	m.previousView = m.currentView
	m.currentView = ViewContainerList
	
	// 触发容器列表视图的初始化，加载数据
	var initCmd tea.Cmd
	if m.containerListView != nil {
		initCmd = m.containerListView.Init()
	}
	
	return m, initCmd
}

// enterComposeList 进入 Compose 项目列表视图
func (m Model) enterComposeList() (tea.Model, tea.Cmd) {
	if m.composeListView == nil {
		return m, m.SetTemporaryMessage(MsgWarning, "⚠️ Docker Compose 未安装或不可用", 3)
	}
	
	m.previousView = m.currentView
	m.currentView = ViewComposeList
	
	// 触发 Compose 列表视图的初始化，扫描项目
	initCmd := m.composeListView.Init()
	
	return m, initCmd
}

// enterImageList 进入镜像列表视图
func (m Model) enterImageList() (tea.Model, tea.Cmd) {
	if m.imageListView == nil {
		return m, m.SetTemporaryMessage(MsgWarning, "⚠️ 镜像列表视图未初始化", 3)
	}
	
	m.previousView = m.currentView
	m.currentView = ViewImageList
	
	// 触发镜像列表视图的初始化，加载数据
	initCmd := m.imageListView.Init()
	
	return m, initCmd
}

// enterNetworkList 进入网络列表视图
func (m Model) enterNetworkList() (tea.Model, tea.Cmd) {
	if m.networkListView == nil {
		return m, m.SetTemporaryMessage(MsgWarning, "⚠️ 网络列表视图未初始化", 3)
	}
	
	m.previousView = m.currentView
	m.currentView = ViewNetworkList
	
	// 触发网络列表视图的初始化，加载数据
	initCmd := m.networkListView.Init()
	
	return m, initCmd
}

// goBack 返回上一级视图
func (m Model) goBack() (tea.Model, tea.Cmd) {
	// 已经在首页，不处理
	if m.currentView == ViewWelcome {
		return m, nil
	}
	
	// 根据当前视图决定返回到哪里（层级导航）
	switch m.currentView {
	case ViewContainerList:
		m.currentView = ViewWelcome
	case ViewContainerDetail:
		m.currentView = ViewContainerList
	case ViewLogs:
		if m.previousView == ViewContainerDetail || m.previousView == ViewContainerList {
			m.currentView = m.previousView
		} else {
			m.currentView = ViewContainerList
		}
	case ViewHelp:
		m.currentView = m.previousView
	case ViewComposeList:
		m.currentView = ViewWelcome
	case ViewComposeDetail:
		m.currentView = ViewComposeList
	case ViewImageList:
		m.currentView = ViewWelcome
	case ViewImageDetails:
		m.currentView = ViewImageList
	case ViewNetworkList:
		m.currentView = ViewWelcome
	case ViewNetworkDetail:
		m.currentView = ViewNetworkList
	default:
		m.currentView = ViewWelcome
	}
	
	// 清除所有临时消息
	m.infoMsg = ""
	m.successMsg = ""
	m.warningMsg = ""
	
	return m, nil
}

// handleContainerListKeys 处理容器列表视图的快捷键
// 注意：大部分按键由视图自己处理，这里只保留需要访问全局状态的快捷键
func (m Model) handleContainerListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 如果处于搜索模式、显示确认对话框、编辑视图、错误弹窗或 JSON 查看器，让视图自己处理
	if m.containerListView != nil {
		if m.containerListView.IsSearching() || m.containerListView.IsEditViewVisible() || m.containerListView.HasError() || m.containerListView.IsShowingJSONViewer() {
			return m, nil  // 返回 nil，让 Update 传递给视图
		}
	}
	
	switch msg.String() {
	case "s":
		// 进入容器 Shell - 显示 Shell 选择器（需要访问全局 shellSelector）
		if m.containerListView != nil {
			if container := m.containerListView.GetSelectedContainer(); container != nil {
				// 检查容器是否在运行
				if container.State != "running" {
					return m, m.SetTemporaryMessage(MsgWarning, "⚠️ 只能在运行中的容器执行 shell", 3)
				}
				
				// 设置选中的容器信息
				m.selectedContainerID = container.ID
				
				// 显示 Shell 选择器
				m.showShellSelector = true
				m.shellSelector.SetContainer(container.ID, container.Name)
				m.shellSelector.SetSize(m.width, m.height)
				m.shellSelector.SetCallbacks(
					func(shell string) {
						// 选择 Shell 后的回调会在 Update 中处理
					},
					func() {
						// 取消选择的回调会在 Update 中处理
					},
				)
				return m, m.shellSelector.Init()
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
			if m.containerDetailView != nil {
				if details := m.containerDetailView.GetDetails(); details != nil {
					containerName = details.Name
				}
			}
			
			// 设置日志视图的容器信息
			if m.logsView != nil {
				m.logsView.SetContainer(m.selectedContainerID, containerName)
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
		// 进入容器 Shell - 显示 Shell 选择器
		if m.selectedContainerID != "" {
			// 从详情视图获取容器名称和状态
			containerName := m.selectedContainerID[:12]
			containerState := "unknown"
			if m.containerDetailView != nil {
				if details := m.containerDetailView.GetDetails(); details != nil {
					containerName = details.Name
					containerState = details.State
				}
			}
			
			// 检查容器是否在运行
			if containerState != "running" {
				return m, m.SetTemporaryMessage(MsgWarning, "⚠️ 只能在运行中的容器执行 shell", 3)
			}
			
			// 显示 Shell 选择器
			m.showShellSelector = true
			m.shellSelector.SetContainer(m.selectedContainerID, containerName)
			m.shellSelector.SetSize(m.width, m.height)
			return m, m.shellSelector.Init()
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

// handleComposeListKeys 处理 Compose 列表视图的快捷键
func (m Model) handleComposeListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Compose 列表视图的按键由视图自己处理
	// 视图会发送 GoToComposeDetailMsg 来请求切换到详情视图
	return m, nil
}

// handleImageListKeys 处理镜像列表视图的快捷键
// handleImageListKeys 处理镜像列表视图的快捷键
// 注意：大部分按键由视图自己处理，这里只保留必要的全局快捷键
func (m Model) handleImageListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 镜像列表视图完全控制自己的按键，不在这里拦截
	return m, nil
}

// handleNetworkListKeys 处理网络列表视图的快捷键
// 注意：大部分按键由视图自己处理，这里只保留必要的全局快捷键
func (m Model) handleNetworkListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 网络列表视图完全控制自己的按键，不在这里拦截
	return m, nil
}

// fillBackground 填充整个屏幕，确保每行宽度一致
// 不强制设置背景色，让终端使用默认背景
func (m Model) fillBackground(content string) string {
	if m.width <= 0 || m.height <= 0 {
		return content
	}
	
	// 将内容按行分割
	lines := strings.Split(content, "\n")
	
	// 处理每一行，确保宽度一致
	var result strings.Builder
	for i := 0; i < m.height; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		
		// 计算可见字符长度（排除 ANSI 转义码）
		visibleLen := visibleLength(line)
		
		// 如果行太短，用空格填充到屏幕宽度
		if visibleLen < m.width {
			padding := m.width - visibleLen
			line = line + strings.Repeat(" ", padding)
		}
		
		result.WriteString(line)
		if i < m.height-1 {
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// visibleLength 计算可见字符长度（排除 ANSI 转义码）
func visibleLength(s string) int {
	inEscape := false
	length := 0
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		length++
	}
	return length
}

func (m Model) View() string {
	// 如果 Shell 选择器正在显示，优先渲染它
	if m.showShellSelector && m.shellSelector != nil {
		return m.shellSelector.View()
	}
	
	var content string
	
	// 根据当前视图类型显示不同内容
	switch m.currentView {
	case ViewWelcome:
		if m.homeView != nil {
			content = m.homeView.View()
		} else {
			content = "🏠 首页视图未初始化"
		}
	case ViewContainerList:
		if m.containerListView != nil {
			content = m.containerListView.View()
		} else {
			content = "📦 容器列表视图未初始化"
		}
	case ViewContainerDetail:
		if m.containerDetailView != nil {
			content = m.containerDetailView.View()
		} else {
			content = "📋 容器详情视图未初始化"
		}
	case ViewLogs:
		if m.logsView != nil {
			content = m.logsView.View()
		} else {
			content = "📜 日志视图未初始化"
		}
	case ViewHelp:
		if m.helpView != nil {
			content = m.helpView.View()
		} else {
			content = "🆘 帮助视图未初始化"
		}
	case ViewComposeList:
		if m.composeListView != nil {
			content = m.composeListView.View()
		} else {
			content = "🧩 Compose 视图未初始化"
		}
	case ViewComposeDetail:
		if m.composeDetailView != nil {
			content = m.composeDetailView.View()
		} else {
			content = "🧩 Compose 详情视图未初始化"
		}
	case ViewImageList:
		if m.imageListView != nil {
			content = m.imageListView.View()
		} else {
			content = "🖼️ 镜像列表视图未初始化"
		}
	case ViewImageDetails:
		if m.imageDetailsView != nil {
			content = m.imageDetailsView.View()
		} else {
			content = "🖼️ 镜像详情视图未初始化"
		}
	case ViewNetworkList:
		if m.networkListView != nil {
			content = m.networkListView.View()
		} else {
			content = "🌐 网络列表视图未初始化"
		}
	case ViewNetworkDetail:
		if m.networkDetailView != nil {
			content = m.networkDetailView.View()
		} else {
			content = "🌐 网络详情视图未初始化"
		}
	default:
		content = "未知视图"
	}
	
	// 添加分级消息显示（非容器列表、Compose 列表、Compose 详情、镜像列表和网络列表视图）
	if m.currentView != ViewContainerList && m.currentView != ViewComposeList && m.currentView != ViewComposeDetail && m.currentView != ViewImageList && m.currentView != ViewNetworkList {
		if m.errorMsg != "" && m.dockerConnected {
			errorStyle := lipgloss.NewStyle().Foreground(ThemeError).Bold(true)
			content = "\n" + errorStyle.Render("❌ 致命错误: "+m.errorMsg) + "\n" + content
		}
		if m.warningMsg != "" {
			warnStyle := lipgloss.NewStyle().Foreground(ThemeWarning).Bold(true)
			content += "\n\n" + warnStyle.Render("⚠️ 警告: "+m.warningMsg)
		}
		if m.infoMsg != "" {
			infoStyle := lipgloss.NewStyle().Foreground(ThemeHighlight)
			content += "\n\n" + infoStyle.Render(m.infoMsg)
		}
		if m.successMsg != "" {
			successStyle := lipgloss.NewStyle().Foreground(ThemeSuccess).Bold(true)
			content += "\n\n" + successStyle.Render(m.successMsg)
		}
	}
	
	// 填充每行到屏幕宽度
	return m.fillBackground(content)
}

// delegateToCurrentView 将消息委托给当前活动的视图处理
func (m Model) delegateToCurrentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch m.currentView {
	case ViewWelcome:
		// ViewWelcome 的按键已经在 handleWelcomeKeys 中处理了
		// 这里只处理非按键消息（如 homeStatsLoadedMsg）
		if _, isKeyMsg := msg.(tea.KeyMsg); !isKeyMsg {
			if m.homeView != nil {
				_, cmd = m.homeView.Update(msg)
			}
		}
	case ViewContainerList:
		if m.containerListView != nil {
			m.containerListView, cmd = m.containerListView.Update(msg)
		}
	case ViewContainerDetail:
		if m.containerDetailView != nil {
			m.containerDetailView, cmd = m.containerDetailView.Update(msg)
		}
	case ViewLogs:
		if m.logsView != nil {
			m.logsView, cmd = m.logsView.Update(msg)
		}
	case ViewComposeList:
		if m.composeListView != nil {
			cmd = m.composeListView.Update(msg)
		}
	case ViewComposeDetail:
		if m.composeDetailView != nil {
			cmd = m.composeDetailView.Update(msg)
		}
	case ViewImageList:
		if m.imageListView != nil {
			m.imageListView, cmd = m.imageListView.Update(msg)
		}
	case ViewImageDetails:
		if m.imageDetailsView != nil {
			m.imageDetailsView, cmd = m.imageDetailsView.Update(msg)
		}
	case ViewNetworkList:
		if m.networkListView != nil {
			m.networkListView, cmd = m.networkListView.Update(msg)
		}
	case ViewNetworkDetail:
		if m.networkDetailView != nil {
			m.networkDetailView, cmd = m.networkDetailView.Update(msg)
		}
	}
	
	return m, cmd
}
