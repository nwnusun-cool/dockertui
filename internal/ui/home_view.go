package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/compose"
	"docktui/internal/docker"
)

// HomeView 首页导航视图
type HomeView struct {
	dockerClient  docker.Client
	composeClient compose.Client
	scanner       compose.Scanner

	// UI 尺寸
	width  int
	height int

	// 选中的卡片索引: 0=Docker容器, 1=Docker Compose
	selectedCard int

	// 状态数据
	containerCount    int  // 容器总数
	runningCount      int  // 运行中容器数
	composeCount      int  // Compose 项目数
	composeRunning    int  // 运行中的 Compose 项目数
	dockerConnected   bool // Docker 连接状态
	composeAvailable  bool // Compose 是否可用
	loading           bool // 是否正在加载
	lastRefreshTime   time.Time
}

// 首页样式定义
var (
	// Logo 样式
	logoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6699FF")).
		Bold(true)

	logoTextStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	// 顶部状态栏样式
	homeStatusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	homeStatusConnectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)

	homeStatusDisconnectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	// 卡片样式 - 未选中
	cardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(36)

	// 卡片样式 - 选中
	cardSelectedStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6699FF")).
		Background(lipgloss.Color("236")).
		Padding(1, 2).
		Width(36)

	// 卡片标题样式
	cardTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	cardTitleSelectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6699FF")).
		Bold(true)

	// 卡片状态样式
	cardStatsStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	cardStatsRunningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	// 卡片提示样式
	cardHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	cardHintSelectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	// 底部操作区样式
	homeFooterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	homeFooterKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Bold(true)
)

// ASCII Logo - 鲸鱼造型（优化版，更紧凑）
const asciiLogo = `
       ██╗
      ██╔╝██╗
     ██╔╝ ╚██╗
   ██████████████╗
  ██╔════════════██╗
 ██║  ●           ██████╗
 ██║         ░░░░░░░░░░██╗
 ██║       ░░░░░░░░░░░░░██║
  ██╗    ░░░░░░░░░░░░░░██╔╝
   ╚██████████████████╔╝`

const asciiLogoText = `
 ____             _    _____ _   _ ___ 
|  _ \  ___   ___| | _|_   _| | | |_ _|
| | | |/ _ \ / __| |/ / | | | | | || | 
| |_| | (_) | (__|   <  | | | |_| || | 
|____/ \___/ \___|_|\_\ |_|  \___/|___|`

// 简化版 Logo（窄屏使用）
const asciiLogoSmall = `
  ▄█▀▀▀█▄
 ██  ●  ██▄▄▄
 ██░░░░░░░░░██
  ▀██████████▀`

// NewHomeView 创建首页视图
func NewHomeView(dockerClient docker.Client) *HomeView {
	// 尝试初始化 Compose 客户端
	var composeClient compose.Client
	var scanner compose.Scanner
	composeAvailable := false
	
	client, err := compose.NewClient()
	if err == nil {
		composeClient = client
		scanner = compose.NewScanner(client, compose.DefaultScanConfig())
		composeAvailable = true
	}
	
	return &HomeView{
		dockerClient:     dockerClient,
		composeClient:    composeClient,
		scanner:          scanner,
		selectedCard:     0,
		dockerConnected:  true,
		composeAvailable: composeAvailable,
	}
}

// Init 初始化首页视图
func (v *HomeView) Init() tea.Cmd {
	v.loading = true
	return v.loadStats
}

// Update 处理消息并更新视图状态
func (v *HomeView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case homeStatsLoadedMsg:
		v.containerCount = msg.containerCount
		v.runningCount = msg.runningCount
		v.composeCount = msg.composeCount
		v.composeRunning = msg.composeRunning
		v.dockerConnected = msg.dockerConnected
		v.composeAvailable = msg.composeAvailable
		v.loading = false
		v.lastRefreshTime = time.Now()
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if v.selectedCard > 0 {
				v.selectedCard--
			}
			return v, nil
		case "right", "l", "tab":
			if v.selectedCard < 1 {
				v.selectedCard++
			}
			return v, nil
		case "1":
			v.selectedCard = 0
			return v, nil
		case "2":
			v.selectedCard = 1
			return v, nil
		case "r", "f5":
			// 刷新状态
			v.loading = true
			return v, v.loadStats
		}
	}

	return v, nil
}

// View 渲染首页视图
func (v *HomeView) View() string {
	// 计算各区域高度
	statusBarHeight := 1
	footerHeight := 1
	
	// 内容区域可用高度
	contentHeight := v.height - statusBarHeight - footerHeight - 2 // 2 是上下边距
	if contentHeight < 10 {
		contentHeight = 10
	}
	
	// 顶部状态栏
	statusBar := v.renderStatusBar()
	
	// 底部操作区
	footer := v.renderFooter()
	
	// 中间内容区（Logo + 卡片）
	content := v.renderContent(contentHeight)
	
	// 组合布局：状态栏 + 内容 + 底部
	return lipgloss.JoinVertical(lipgloss.Left,
		statusBar,
		content,
		footer,
	)
}

// renderContent 渲染中间内容区域（Logo + 导航卡片）
func (v *HomeView) renderContent(availableHeight int) string {
	// Logo 区域
	logo := v.renderLogo()
	logoHeight := strings.Count(logo, "\n") + 1
	
	// 导航卡片区域
	cards := v.renderNavigationArea()
	cardsHeight := strings.Count(cards, "\n") + 1
	
	// 计算需要的填充高度
	usedHeight := logoHeight + cardsHeight
	paddingTop := 1
	paddingMiddle := 1
	paddingBottom := availableHeight - usedHeight - paddingTop - paddingMiddle
	if paddingBottom < 0 {
		paddingBottom = 0
	}
	
	// 构建内容区域
	var content strings.Builder
	
	// 顶部填充
	content.WriteString(strings.Repeat("\n", paddingTop))
	
	// Logo
	content.WriteString(logo)
	
	// Logo 和卡片之间的间距
	content.WriteString(strings.Repeat("\n", paddingMiddle+1))
	
	// 导航卡片
	content.WriteString(cards)
	
	// 底部填充（将内容推向上方，footer 固定在底部）
	if paddingBottom > 0 {
		content.WriteString(strings.Repeat("\n", paddingBottom))
	}
	
	return content.String()
}

// renderLogo 渲染 Logo 区域
func (v *HomeView) renderLogo() string {
	// 根据窗口高度决定是否显示 Logo
	if v.height < 20 {
		// 极小窗口：只显示简单标题
		title := logoTextStyle.Render("🐳 DockTUI")
		return lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(title)
	}
	
	// 根据窗口宽度选择 Logo 版本
	var logo string
	if v.width < 50 {
		// 极窄屏：只显示标题
		logo = logoTextStyle.Render("🐳 DockTUI")
	} else if v.width < 70 {
		// 窄屏使用简化版
		logo = logoStyle.Render(asciiLogoSmall)
	} else if v.width >= 100 {
		// 超宽屏：鲸鱼和文字并排，使用 Top 对齐
		whale := logoStyle.Render(asciiLogo)
		text := logoTextStyle.Render(asciiLogoText)
		logo = lipgloss.JoinHorizontal(lipgloss.Top, whale, "    ", text)
	} else {
		// 普通宽屏：只显示鲸鱼 + 简化标题
		whale := logoStyle.Render(asciiLogo)
		title := logoTextStyle.Render("  DockTUI")
		logo = lipgloss.JoinHorizontal(lipgloss.Center, whale, title)
	}

	// 居中显示
	return lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(logo)
}

// SetSize 设置视图尺寸
func (v *HomeView) SetSize(width, height int) {
	v.width = width
	v.height = height

	// 根据宽度调整卡片宽度
	cardWidth := 36
	if width > 90 {
		cardWidth = 40
	}
	if width < 80 {
		cardWidth = 32
	}
	if width < 70 {
		cardWidth = 28
	}

	cardStyle = cardStyle.Width(cardWidth)
	cardSelectedStyle = cardSelectedStyle.Width(cardWidth)
}

// renderStatusBar 渲染顶部状态栏
func (v *HomeView) renderStatusBar() string {
	// 版本信息
	version := "DockTUI v0.1.0"

	// Docker 连接状态
	var connStatus string
	if v.dockerConnected {
		connStatus = homeStatusConnectedStyle.Render("● 已连接")
	} else {
		connStatus = homeStatusDisconnectedStyle.Render("● 未连接")
	}

	// 构建状态栏
	statusContent := fmt.Sprintf(" %s  │  Docker: %s ", version, connStatus)

	// 计算宽度并填充
	availableWidth := v.width
	if availableWidth < 60 {
		availableWidth = 60
	}

	// 使用 lipgloss 渲染状态栏
	statusBar := homeStatusBarStyle.Width(availableWidth).Render(statusContent)

	return statusBar
}

// renderNavigationArea 渲染核心导航区
func (v *HomeView) renderNavigationArea() string {
	// 渲染两个卡片
	card1 := v.renderContainerCard()
	card2 := v.renderComposeCard()

	// 判断是否需要垂直排列（窄屏或矮屏）
	if v.width < 78 || v.height < 25 {
		// 垂直排列
		cards := lipgloss.JoinVertical(lipgloss.Center, card1, "", card2)
		return lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(cards)
	}

	// 水平排列，居中显示
	cards := lipgloss.JoinHorizontal(lipgloss.Top, card1, "  ", card2)

	// 居中
	return lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(cards)
}

// renderContainerCard 渲染 Docker 容器卡片
func (v *HomeView) renderContainerCard() string {
	isSelected := v.selectedCard == 0

	// 选择样式
	style := cardStyle
	titleStyle := cardTitleStyle
	hintStyle := cardHintStyle
	if isSelected {
		style = cardSelectedStyle
		titleStyle = cardTitleSelectedStyle
		hintStyle = cardHintSelectedStyle
	}

	// 标题
	title := titleStyle.Render("🐳 Docker 容器管理")

	// 状态统计
	var stats string
	if v.loading {
		stats = cardStatsStyle.Render("加载中...")
	} else {
		runningText := cardStatsRunningStyle.Render(fmt.Sprintf("%d", v.runningCount))
		stats = fmt.Sprintf("%d 个容器 (%s 运行中)", v.containerCount, runningText)
	}

	// 进入提示
	var hint string
	if isSelected {
		hint = hintStyle.Render("按 Enter 或 1 进入")
	} else {
		hint = hintStyle.Render("按 1 进入")
	}

	// 组合内容
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		stats,
		"",
		hint,
	)

	return style.Render(content)
}

// renderComposeCard 渲染 Docker Compose 卡片
func (v *HomeView) renderComposeCard() string {
	isSelected := v.selectedCard == 1

	// 选择样式
	style := cardStyle
	titleStyle := cardTitleStyle
	hintStyle := cardHintStyle
	if isSelected {
		style = cardSelectedStyle
		titleStyle = cardTitleSelectedStyle
		hintStyle = cardHintSelectedStyle
	}

	// 标题
	title := titleStyle.Render("🧩 Docker Compose")

	// 状态统计
	var stats string
	if v.loading {
		stats = cardStatsStyle.Render("加载中...")
	} else if !v.composeAvailable {
		stats = cardStatsStyle.Render("⚠️ Compose 不可用")
	} else if v.composeCount == 0 {
		stats = cardStatsStyle.Render("未发现项目")
	} else {
		runningText := cardStatsRunningStyle.Render(fmt.Sprintf("%d", v.composeRunning))
		stats = fmt.Sprintf("%d 个项目 (%s 运行中)", v.composeCount, runningText)
	}

	// 进入提示
	var hint string
	if !v.composeAvailable {
		hint = hintStyle.Render("请安装 Docker Compose")
	} else if isSelected {
		hint = hintStyle.Render("按 Enter 或 2 进入")
	} else {
		hint = hintStyle.Render("按 2 进入")
	}

	// 组合内容
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		stats,
		"",
		hint,
	)

	return style.Render(content)
}

// renderFooter 渲染底部操作区
func (v *HomeView) renderFooter() string {
	// 构建操作提示
	keys := []string{
		homeFooterKeyStyle.Render("1") + "=Docker容器",
		homeFooterKeyStyle.Render("2") + "=Docker Compose",
		homeFooterKeyStyle.Render("←/→") + "=切换",
		homeFooterKeyStyle.Render("Enter") + "=进入",
		homeFooterKeyStyle.Render("r") + "=刷新",
		homeFooterKeyStyle.Render("?") + "=帮助",
		homeFooterKeyStyle.Render("q") + "=退出",
	}

	footerContent := " 请选择功能：" + strings.Join(keys, "  ")

	// 计算宽度
	availableWidth := v.width
	if availableWidth < 60 {
		availableWidth = 60
	}

	return homeFooterStyle.Width(availableWidth).Render(footerContent)
}

// GetSelectedCard 获取当前选中的卡片索引
func (v *HomeView) GetSelectedCard() int {
	return v.selectedCard
}

// homeStatsLoadedMsg 首页统计数据加载完成消息
type homeStatsLoadedMsg struct {
	containerCount   int
	runningCount     int
	composeCount     int
	composeRunning   int
	dockerConnected  bool
	composeAvailable bool
}

// loadStats 加载首页统计数据
func (v *HomeView) loadStats() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := homeStatsLoadedMsg{
		dockerConnected:  true,
		composeAvailable: v.composeAvailable,
	}

	// 获取容器统计
	containers, err := v.dockerClient.ListContainers(ctx, true)
	if err != nil {
		result.dockerConnected = false
	} else {
		result.containerCount = len(containers)
		for _, c := range containers {
			if c.State == "running" {
				result.runningCount++
			}
		}
	}

	// 获取 Compose 项目统计
	if v.scanner != nil && v.composeAvailable {
		projects, err := v.scanner.Scan(ctx, []string{"."})
		if err == nil {
			result.composeCount = len(projects)
			// 刷新项目状态并统计运行中的项目
			for i := range projects {
				v.scanner.RefreshProject(ctx, &projects[i])
				if projects[i].Status == compose.StatusRunning || projects[i].Status == compose.StatusPartial {
					result.composeRunning++
				}
			}
		}
	}

	return result
}
