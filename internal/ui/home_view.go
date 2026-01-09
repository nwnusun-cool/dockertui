package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
)

// RuntimeType 运行时类型
type RuntimeType int

const (
	RuntimeDocker RuntimeType = iota
	RuntimePodman
	RuntimeContainerd
)

// RuntimeInfo 运行时信息
type RuntimeInfo struct {
	Type      RuntimeType
	Name      string
	Icon      string
	Connected bool
	Version   string
}

// ResourceType 资源类型
type ResourceType int

const (
	ResourceContainers ResourceType = iota
	ResourceImages
	ResourceNetworks
	ResourceVolumes
	ResourceCompose
)

// ResourceInfo 资源信息
type ResourceInfo struct {
	Type        ResourceType
	Name        string
	Icon        string
	Key         string // 快捷键
	Count       int
	ActiveCount int    // 运行中/使用中的数量
	Available   bool   // 是否可用
	Hint        string // 不可用时的提示
}

// HomeView 首页导航视图
type HomeView struct {
	dockerClient docker.Client

	// UI 尺寸
	width  int
	height int

	// 运行时列表
	runtimes        []RuntimeInfo
	selectedRuntime int

	// 资源列表（根据当前运行时动态变化）
	resources        []ResourceInfo
	selectedResource int

	// 焦点区域: 0=运行时, 1=资源
	focusArea int

	// 状态
	loading         bool
	lastRefreshTime time.Time
}

// 首页样式定义
var (
	// 标题样式
	homeTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	// 区域标题样式
	homeSectionTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	// 运行时卡片样式
	runtimeCardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 2).
		Width(20)

	runtimeCardSelectedStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 2).
		Width(20)

	runtimeCardDisabledStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 2).
		Width(20)

	// 资源卡片样式
	resourceCardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(16)

	resourceCardSelectedStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Width(16)

	resourceCardDisabledStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1).
		Width(16)

	// 状态样式
	homeConnectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	homeDisconnectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	homeCountStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	homeActiveCountStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	homeKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	homeHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	// 底部状态栏样式
	homeFooterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 1)
)

// NewHomeView 创建首页视图
func NewHomeView(dockerClient docker.Client) *HomeView {
	v := &HomeView{
		dockerClient:     dockerClient,
		selectedRuntime:  0,
		selectedResource: 0,
		focusArea:        1, // 默认焦点在资源区
	}

	// 初始化运行时列表
	v.runtimes = []RuntimeInfo{
		{Type: RuntimeDocker, Name: "Docker", Icon: "🐳", Connected: false, Version: ""},
		{Type: RuntimePodman, Name: "Podman", Icon: "🦭", Connected: false, Version: ""},
		{Type: RuntimeContainerd, Name: "containerd", Icon: "📦", Connected: false, Version: ""},
	}

	// 初始化资源列表（Docker 的资源）
	v.resources = v.getDockerResources()

	return v
}

// getDockerResources 获取 Docker 运行时的资源列表
func (v *HomeView) getDockerResources() []ResourceInfo {
	return []ResourceInfo{
		{Type: ResourceContainers, Name: "容器", Icon: "📦", Key: "c", Available: true},
		{Type: ResourceImages, Name: "镜像", Icon: "🖼️", Key: "i", Available: true},
		{Type: ResourceNetworks, Name: "网络", Icon: "🌐", Key: "n", Available: false, Hint: "开发中"},
		{Type: ResourceVolumes, Name: "卷", Icon: "💾", Key: "v", Available: false, Hint: "开发中"},
		{Type: ResourceCompose, Name: "Compose", Icon: "🧩", Key: "o", Available: true},
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
		v.loading = false
		v.lastRefreshTime = time.Now()

		// 更新运行时状态
		for i := range v.runtimes {
			if v.runtimes[i].Type == RuntimeDocker {
				v.runtimes[i].Connected = msg.dockerConnected
				v.runtimes[i].Version = msg.dockerVersion
			}
		}

		// 更新资源统计
		for i := range v.resources {
			switch v.resources[i].Type {
			case ResourceContainers:
				v.resources[i].Count = msg.containerCount
				v.resources[i].ActiveCount = msg.runningCount
			case ResourceImages:
				v.resources[i].Count = msg.imageCount
				v.resources[i].ActiveCount = msg.danglingCount
			case ResourceCompose:
				v.resources[i].Count = msg.composeCount
				v.resources[i].ActiveCount = msg.composeRunning
				v.resources[i].Available = msg.composeAvailable
				if !msg.composeAvailable {
					v.resources[i].Hint = "未安装"
				}
			}
		}
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// 切换焦点区域
			v.focusArea = (v.focusArea + 1) % 2
			return v, nil

		case "up", "k":
			if v.focusArea == 1 {
				// 从资源区切换到运行时区
				v.focusArea = 0
			}
			return v, nil

		case "down", "j":
			if v.focusArea == 0 {
				// 从运行时区切换到资源区
				v.focusArea = 1
			}
			return v, nil

		case "left", "h":
			if v.focusArea == 0 {
				// 运行时区左移
				if v.selectedRuntime > 0 {
					v.selectedRuntime--
				}
			} else {
				// 资源区左移
				if v.selectedResource > 0 {
					v.selectedResource--
				}
			}
			return v, nil

		case "right", "l":
			if v.focusArea == 0 {
				// 运行时区右移
				if v.selectedRuntime < len(v.runtimes)-1 {
					v.selectedRuntime++
				}
			} else {
				// 资源区右移
				if v.selectedResource < len(v.resources)-1 {
					v.selectedResource++
				}
			}
			return v, nil

		case "r", "f5":
			// 刷新状态
			v.loading = true
			return v, v.loadStats

		case "1", "2", "3", "4", "5":
			// 数字键快速选择资源
			idx := int(msg.String()[0] - '1')
			if idx >= 0 && idx < len(v.resources) {
				v.selectedResource = idx
				v.focusArea = 1
			}
			return v, nil
		}
	}

	return v, nil
}

// View 渲染首页视图
func (v *HomeView) View() string {
	var content strings.Builder

	// 顶部标题
	content.WriteString(v.renderHeader())
	content.WriteString("\n\n")

	// 运行时区域
	content.WriteString(v.renderRuntimeSection())
	content.WriteString("\n\n")

	// 资源区域
	content.WriteString(v.renderResourceSection())
	content.WriteString("\n")

	// 底部填充
	currentHeight := strings.Count(content.String(), "\n") + 1
	padding := v.height - currentHeight - 2 // 2 是底部状态栏
	if padding > 0 {
		content.WriteString(strings.Repeat("\n", padding))
	}

	// 底部状态栏
	content.WriteString(v.renderFooter())

	return content.String()
}

// renderHeader 渲染顶部标题
func (v *HomeView) renderHeader() string {
	title := homeTitleStyle.Render("🐳 DockTUI")
	version := homeHintStyle.Render("v0.1.0")

	// 加载状态
	var status string
	if v.loading {
		status = homeHintStyle.Render("⏳ 加载中...")
	} else if !v.lastRefreshTime.IsZero() {
		status = homeHintStyle.Render(fmt.Sprintf("最后刷新: %s", v.lastRefreshTime.Format("15:04:05")))
	}

	header := fmt.Sprintf("  %s %s    %s", title, version, status)
	return header
}

// renderRuntimeSection 渲染运行时区域
func (v *HomeView) renderRuntimeSection() string {
	// 区域标题
	sectionTitle := homeSectionTitleStyle.Render("  运行时")
	if v.focusArea == 0 {
		sectionTitle = homeTitleStyle.Render("▶ 运行时")
	}

	// 渲染运行时卡片
	var cards []string
	for i, rt := range v.runtimes {
		cards = append(cards, v.renderRuntimeCard(rt, i == v.selectedRuntime && v.focusArea == 0))
	}

	// 水平排列卡片
	cardsRow := lipgloss.JoinHorizontal(lipgloss.Top, cards...)

	// 居中显示
	centeredCards := lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(cardsRow)

	return sectionTitle + "\n" + centeredCards
}

// renderRuntimeCard 渲染单个运行时卡片
func (v *HomeView) renderRuntimeCard(rt RuntimeInfo, selected bool) string {
	// 选择样式：选中状态优先
	var style lipgloss.Style
	if selected {
		style = runtimeCardSelectedStyle
	} else if !rt.Connected && rt.Type != RuntimeDocker {
		style = runtimeCardDisabledStyle
	} else {
		style = runtimeCardStyle
	}

	// 标题行
	var title string
	if selected {
		title = homeTitleStyle.Render(fmt.Sprintf("%s %s", rt.Icon, rt.Name))
	} else {
		title = fmt.Sprintf("%s %s", rt.Icon, rt.Name)
	}

	// 状态行
	var status string
	if rt.Connected {
		status = homeConnectedStyle.Render("● 已连接")
		if rt.Version != "" {
			status += homeHintStyle.Render(" " + rt.Version)
		}
	} else if rt.Type == RuntimeDocker {
		status = homeDisconnectedStyle.Render("○ 未连接")
	} else {
		status = homeDisconnectedStyle.Render("○ 未安装")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, status)
	return style.Render(content)
}

// renderResourceSection 渲染资源区域
func (v *HomeView) renderResourceSection() string {
	// 获取当前运行时名称
	runtimeName := v.runtimes[v.selectedRuntime].Name

	// 区域标题
	sectionTitle := homeSectionTitleStyle.Render(fmt.Sprintf("  %s 资源", runtimeName))
	if v.focusArea == 1 {
		sectionTitle = homeTitleStyle.Render(fmt.Sprintf("▶ %s 资源", runtimeName))
	}

	// 渲染资源卡片
	var cards []string
	for i, res := range v.resources {
		cards = append(cards, v.renderResourceCard(res, i == v.selectedResource && v.focusArea == 1, i+1))
	}

	// 根据宽度决定布局
	var cardsRow string
	if v.width < 90 {
		// 窄屏：分两行显示
		row1 := lipgloss.JoinHorizontal(lipgloss.Top, cards[:3]...)
		row2 := lipgloss.JoinHorizontal(lipgloss.Top, cards[3:]...)
		cardsRow = lipgloss.JoinVertical(lipgloss.Center, row1, row2)
	} else {
		// 宽屏：一行显示
		cardsRow = lipgloss.JoinHorizontal(lipgloss.Top, cards...)
	}

	// 居中显示
	centeredCards := lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(cardsRow)

	return sectionTitle + "\n" + centeredCards
}

// renderResourceCard 渲染单个资源卡片
func (v *HomeView) renderResourceCard(res ResourceInfo, selected bool, num int) string {
	// 选择样式
	// 选择样式：选中状态优先
	var style lipgloss.Style
	if selected {
		style = resourceCardSelectedStyle
	} else if !res.Available {
		style = resourceCardDisabledStyle
	} else {
		style = resourceCardStyle
	}

	// 标题行（图标 + 名称）
	var title string
	if selected {
		title = homeTitleStyle.Render(fmt.Sprintf("%s %s", res.Icon, res.Name))
	} else {
		title = fmt.Sprintf("%s %s", res.Icon, res.Name)
	}

	// 统计行
	var stats string
	if !res.Available {
		stats = homeHintStyle.Render(res.Hint)
	} else if v.loading {
		stats = homeHintStyle.Render("...")
	} else {
		countStr := homeCountStyle.Render(fmt.Sprintf("%d", res.Count))
		if res.ActiveCount > 0 {
			activeStr := homeActiveCountStyle.Render(fmt.Sprintf("%d", res.ActiveCount))
			switch res.Type {
			case ResourceContainers:
				stats = fmt.Sprintf("%s (%s 运行)", countStr, activeStr)
			case ResourceImages:
				if res.ActiveCount > 0 {
					stats = fmt.Sprintf("%s (%s 悬垂)", countStr, homeHintStyle.Render(fmt.Sprintf("%d", res.ActiveCount)))
				} else {
					stats = countStr
				}
			case ResourceCompose:
				stats = fmt.Sprintf("%s (%s 运行)", countStr, activeStr)
			default:
				stats = countStr
			}
		} else {
			stats = countStr
		}
	}

	// 快捷键提示
	keyHint := homeKeyStyle.Render(res.Key) + homeHintStyle.Render(fmt.Sprintf(" 或 %d", num))

	content := lipgloss.JoinVertical(lipgloss.Left, title, stats, keyHint)
	return style.Render(content)
}

// renderFooter 渲染底部状态栏
func (v *HomeView) renderFooter() string {
	keys := []string{
		homeKeyStyle.Render("↑/↓") + "=切换区域",
		homeKeyStyle.Render("←/→") + "=选择",
		homeKeyStyle.Render("Enter") + "=进入",
		homeKeyStyle.Render("r") + "=刷新",
		homeKeyStyle.Render("?") + "=帮助",
		homeKeyStyle.Render("q") + "=退出",
	}

	footerContent := " " + strings.Join(keys, "  ")

	availableWidth := v.width
	if availableWidth < 60 {
		availableWidth = 60
	}

	return homeFooterStyle.Width(availableWidth).Render(footerContent)
}

// SetSize 设置视图尺寸
func (v *HomeView) SetSize(width, height int) {
	v.width = width
	v.height = height

	// 根据宽度调整卡片宽度
	if width < 80 {
		runtimeCardStyle = runtimeCardStyle.Width(18)
		runtimeCardSelectedStyle = runtimeCardSelectedStyle.Width(18)
		runtimeCardDisabledStyle = runtimeCardDisabledStyle.Width(18)
		resourceCardStyle = resourceCardStyle.Width(14)
		resourceCardSelectedStyle = resourceCardSelectedStyle.Width(14)
		resourceCardDisabledStyle = resourceCardDisabledStyle.Width(14)
	} else {
		runtimeCardStyle = runtimeCardStyle.Width(22)
		runtimeCardSelectedStyle = runtimeCardSelectedStyle.Width(22)
		runtimeCardDisabledStyle = runtimeCardDisabledStyle.Width(22)
		resourceCardStyle = resourceCardStyle.Width(16)
		resourceCardSelectedStyle = resourceCardSelectedStyle.Width(16)
		resourceCardDisabledStyle = resourceCardDisabledStyle.Width(16)
	}
}

// GetSelectedCard 获取当前选中的资源索引（兼容旧接口）
func (v *HomeView) GetSelectedCard() int {
	return v.selectedResource
}

// GetSelectedResource 获取当前选中的资源类型
func (v *HomeView) GetSelectedResource() ResourceType {
	if v.selectedResource >= 0 && v.selectedResource < len(v.resources) {
		return v.resources[v.selectedResource].Type
	}
	return ResourceContainers
}

// GetSelectedRuntime 获取当前选中的运行时类型
func (v *HomeView) GetSelectedRuntime() RuntimeType {
	if v.selectedRuntime >= 0 && v.selectedRuntime < len(v.runtimes) {
		return v.runtimes[v.selectedRuntime].Type
	}
	return RuntimeDocker
}

// IsResourceAvailable 检查当前选中的资源是否可用
func (v *HomeView) IsResourceAvailable() bool {
	if v.selectedResource >= 0 && v.selectedResource < len(v.resources) {
		return v.resources[v.selectedResource].Available
	}
	return false
}

// homeStatsLoadedMsg 首页统计数据加载完成消息
type homeStatsLoadedMsg struct {
	dockerConnected  bool
	dockerVersion    string
	containerCount   int
	runningCount     int
	imageCount       int
	danglingCount    int
	composeCount     int
	composeRunning   int
	composeAvailable bool
}

// loadStats 加载首页统计数据
func (v *HomeView) loadStats() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := homeStatsLoadedMsg{
		dockerConnected:  true,
		composeAvailable: true,
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

	// 获取镜像统计
	images, err := v.dockerClient.ListImages(ctx, true)
	if err == nil {
		result.imageCount = len(images)
		for _, img := range images {
			if img.Dangling {
				result.danglingCount++
			}
		}
	}

	// TODO: 获取 Compose 项目统计
	// 暂时设置为不可用，后续实现
	result.composeAvailable = false
	result.composeCount = 0
	result.composeRunning = 0

	return result
}
