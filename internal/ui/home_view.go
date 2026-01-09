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

// 间距常量（终端单位）
const (
	// 模块间大间距（行数）
	spacingModuleLarge = 2
	// 模块内标题与内容间距（行数）
	spacingTitleContent = 1
	// 卡片内边距（字符数）
	paddingCardHorizontal = 2
	paddingCardVertical   = 0
)

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

// 首页样式定义 - 使用自适应颜色
var (
	// 主标题样式
	homeMainTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Bold(true)

	// 区域标题样式（未选中）
	homeSectionTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	// 区域标题样式（选中）
	homeSectionTitleActiveStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("81")).
					Bold(true)

	// 次要文字样式
	homeSubtextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	// 状态样式
	homeConnectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82"))

	homeDisconnectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	// 数量样式
	homeCountStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	homeActiveCountStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82"))

	// 快捷键样式
	homeKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81"))

	// 提示样式
	homeHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// 开发中标记样式
	homeDevTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("208"))
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
			v.focusArea = (v.focusArea + 1) % 2
			return v, nil

		case "up", "k":
			if v.focusArea == 1 {
				v.focusArea = 0
			}
			return v, nil

		case "down", "j":
			if v.focusArea == 0 {
				v.focusArea = 1
			}
			return v, nil

		case "left", "h":
			if v.focusArea == 0 {
				if v.selectedRuntime > 0 {
					v.selectedRuntime--
				}
			} else {
				if v.selectedResource > 0 {
					v.selectedResource--
				}
			}
			return v, nil

		case "right", "l":
			if v.focusArea == 0 {
				if v.selectedRuntime < len(v.runtimes)-1 {
					v.selectedRuntime++
				}
			} else {
				if v.selectedResource < len(v.resources)-1 {
					v.selectedResource++
				}
			}
			return v, nil

		case "r", "f5":
			v.loading = true
			return v, v.loadStats

		case "1", "2", "3", "4", "5":
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

	// 顶部状态栏
	content.WriteString(v.renderHeader())
	content.WriteString(strings.Repeat("\n", spacingModuleLarge))

	// 运行时区域
	content.WriteString(v.renderRuntimeSection())
	content.WriteString(strings.Repeat("\n", spacingModuleLarge))

	// 资源区域
	content.WriteString(v.renderResourceSection())

	// 底部填充
	currentHeight := strings.Count(content.String(), "\n") + 1
	footerHeight := 3 // 底部栏高度
	padding := v.height - currentHeight - footerHeight - spacingModuleLarge
	if padding > 0 {
		content.WriteString(strings.Repeat("\n", padding))
	}

	// 底部间距
	content.WriteString(strings.Repeat("\n", spacingModuleLarge))

	// 底部操作栏
	content.WriteString(v.renderFooter())

	return content.String()
}

// renderHeader 渲染顶部状态栏
func (v *HomeView) renderHeader() string {
	// 确保宽度有效
	width := v.width
	if width < 60 {
		width = 60
	}

	// 左侧：标题 + 版本
	leftPart := homeMainTitleStyle.Render("🐳 DockTUI") + " " + homeSubtextStyle.Render("v0.1.0")

	// 右侧：刷新状态 + 刷新提示
	var rightPart string
	if v.loading {
		rightPart = homeSubtextStyle.Render("⏳ 加载中...")
	} else if !v.lastRefreshTime.IsZero() {
		refreshTime := homeSubtextStyle.Render(fmt.Sprintf("最后刷新: %s", v.lastRefreshTime.Format("15:04:05")))
		refreshHint := homeKeyStyle.Render("r") + homeSubtextStyle.Render("=刷新")
		rightPart = refreshTime + "  " + refreshHint
	}

	// 计算间距，左右对齐
	leftWidth := lipgloss.Width(leftPart)
	rightWidth := lipgloss.Width(rightPart)
	spacing := width - leftWidth - rightWidth - 4 // 4 是左右边距
	if spacing < 2 {
		spacing = 2
	}

	return "  " + leftPart + strings.Repeat(" ", spacing) + rightPart + "  "
}

// renderRuntimeSection 渲染运行时区域
func (v *HomeView) renderRuntimeSection() string {
	// 确保宽度有效
	width := v.width
	if width < 60 {
		width = 60
	}

	// 区域标题
	var sectionTitle string
	if v.focusArea == 0 {
		sectionTitle = homeSectionTitleActiveStyle.Render("▶ 🔧 容器运行时")
	} else {
		sectionTitle = homeSectionTitleStyle.Render("  🔧 容器运行时")
	}

	// 渲染运行时卡片
	var cards []string
	for i, rt := range v.runtimes {
		isSelected := i == v.selectedRuntime && v.focusArea == 0
		cards = append(cards, v.renderRuntimeCard(rt, isSelected))
	}

	// 手动拼接卡片（逐行）
	cardsRow := joinCardsHorizontal(cards, "  ")

	// 居中显示
	cardsWidth := getFirstLineWidth(cardsRow)
	leftPadding := (width - cardsWidth) / 2
	if leftPadding < 2 {
		leftPadding = 2
	}
	
	// 为每行添加左边距
	lines := strings.Split(cardsRow, "\n")
	for i, line := range lines {
		lines[i] = strings.Repeat(" ", leftPadding) + line
	}
	centeredCards := strings.Join(lines, "\n")

	return sectionTitle + strings.Repeat("\n", spacingTitleContent) + centeredCards
}

// joinCardsHorizontal 手动水平拼接多个卡片
func joinCardsHorizontal(cards []string, separator string) string {
	if len(cards) == 0 {
		return ""
	}
	if len(cards) == 1 {
		return cards[0]
	}

	// 将每个卡片分割成行
	cardLines := make([][]string, len(cards))
	maxLines := 0
	for i, card := range cards {
		cardLines[i] = strings.Split(card, "\n")
		if len(cardLines[i]) > maxLines {
			maxLines = len(cardLines[i])
		}
	}

	// 计算每个卡片的宽度（使用第一行）
	cardWidths := make([]int, len(cards))
	for i, lines := range cardLines {
		if len(lines) > 0 {
			cardWidths[i] = lipgloss.Width(lines[0])
		}
	}

	// 逐行拼接
	var result []string
	for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
		var lineParts []string
		for cardIdx, lines := range cardLines {
			var line string
			if lineIdx < len(lines) {
				line = lines[lineIdx]
			}
			// 填充到卡片宽度
			lineWidth := lipgloss.Width(line)
			if lineWidth < cardWidths[cardIdx] {
				line = line + strings.Repeat(" ", cardWidths[cardIdx]-lineWidth)
			}
			lineParts = append(lineParts, line)
		}
		result = append(result, strings.Join(lineParts, separator))
	}

	return strings.Join(result, "\n")
}

// getFirstLineWidth 获取第一行的宽度
func getFirstLineWidth(s string) int {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 {
		return lipgloss.Width(lines[0])
	}
	return 0
}

// renderRuntimeCard 渲染单个运行时卡片
func (v *HomeView) renderRuntimeCard(rt RuntimeInfo, selected bool) string {
	// 边框颜色
	var borderColor lipgloss.Color
	if selected {
		borderColor = lipgloss.Color("81") // 高亮青色
	} else if rt.Connected {
		borderColor = lipgloss.Color("82") // 已连接绿色
	} else {
		borderColor = lipgloss.Color("238") // 未连接灰色
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(paddingCardVertical, paddingCardHorizontal)

	// 标题行
	var title string
	if selected {
		title = homeMainTitleStyle.Render(fmt.Sprintf("%s %s", rt.Icon, rt.Name))
	} else if rt.Connected {
		title = homeConnectedStyle.Render(fmt.Sprintf("%s %s", rt.Icon, rt.Name))
	} else {
		title = homeSubtextStyle.Render(fmt.Sprintf("%s %s", rt.Icon, rt.Name))
	}

	// 状态行
	var status string
	if rt.Connected {
		status = homeConnectedStyle.Render("● 已连接")
		if rt.Version != "" {
			status += " " + homeSubtextStyle.Render(rt.Version)
		}
	} else if rt.Type == RuntimeDocker {
		status = homeDisconnectedStyle.Render("○ 未连接")
	} else {
		status = homeDisconnectedStyle.Render("○ 未安装")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, status)
	return cardStyle.Render(content)
}

// renderResourceSection 渲染资源区域
func (v *HomeView) renderResourceSection() string {
	runtimeName := v.runtimes[v.selectedRuntime].Name
	runtimeIcon := v.runtimes[v.selectedRuntime].Icon

	// 区域标题
	var sectionTitle string
	if v.focusArea == 1 {
		sectionTitle = homeSectionTitleActiveStyle.Render(fmt.Sprintf("▶ %s %s 资源管理", runtimeIcon, runtimeName))
	} else {
		sectionTitle = homeSectionTitleStyle.Render(fmt.Sprintf("  %s %s 资源管理", runtimeIcon, runtimeName))
	}

	// 渲染所有资源卡片
	var cards []string
	for i, res := range v.resources {
		isSelected := i == v.selectedResource && v.focusArea == 1
		cards = append(cards, v.renderResourceCard(res, isSelected, i+1))
	}

	// 手动拼接卡片（逐行）
	cardsRow := joinCardsHorizontal(cards, "  ")

	// 居中显示
	width := v.width
	if width < 60 {
		width = 60
	}
	cardsWidth := getFirstLineWidth(cardsRow)
	leftPadding := (width - cardsWidth) / 2
	if leftPadding < 2 {
		leftPadding = 2
	}

	// 为每行添加左边距
	lines := strings.Split(cardsRow, "\n")
	for i, line := range lines {
		lines[i] = strings.Repeat(" ", leftPadding) + line
	}
	centeredCards := strings.Join(lines, "\n")

	return sectionTitle + strings.Repeat("\n", spacingTitleContent) + centeredCards
}

// renderResourceCard 渲染单个资源卡片
func (v *HomeView) renderResourceCard(res ResourceInfo, selected bool, num int) string {
	// 边框颜色
	var borderColor lipgloss.Color
	if selected {
		borderColor = lipgloss.Color("81") // 高亮青色
	} else if res.Available {
		borderColor = lipgloss.Color("240") // 可用灰色
	} else {
		borderColor = lipgloss.Color("238") // 不可用深灰
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(paddingCardVertical, paddingCardHorizontal)

	// 标题行（图标 + 名称）
	var title string
	if selected {
		title = homeMainTitleStyle.Render(fmt.Sprintf("%s %s", res.Icon, res.Name))
	} else if !res.Available {
		title = homeSubtextStyle.Render(fmt.Sprintf("%s %s", res.Icon, res.Name))
	} else {
		title = fmt.Sprintf("%s %s", res.Icon, res.Name)
	}

	// 统计行
	var stats string
	if !res.Available {
		stats = homeDevTagStyle.Render("🚧 " + res.Hint)
	} else if v.loading {
		stats = homeSubtextStyle.Render("...")
	} else {
		countStr := homeCountStyle.Render(fmt.Sprintf("%d", res.Count))
		switch res.Type {
		case ResourceContainers:
			if res.ActiveCount > 0 {
				activeStr := homeActiveCountStyle.Render(fmt.Sprintf("%d", res.ActiveCount))
				stats = fmt.Sprintf("%s (%s 运行)", countStr, activeStr)
			} else {
				stats = countStr
			}
		case ResourceImages:
			if res.ActiveCount > 0 {
				stats = fmt.Sprintf("%s (%s 悬垂)", countStr, homeSubtextStyle.Render(fmt.Sprintf("%d", res.ActiveCount)))
			} else {
				stats = countStr
			}
		case ResourceCompose:
			if res.ActiveCount > 0 {
				activeStr := homeActiveCountStyle.Render(fmt.Sprintf("%d", res.ActiveCount))
				stats = fmt.Sprintf("%s (%s 运行)", countStr, activeStr)
			} else {
				stats = countStr
			}
		default:
			stats = countStr
		}
	}

	// 快捷键提示（与文字保持间距）
	keyHint := homeKeyStyle.Render(res.Key) + " " + homeSubtextStyle.Render(fmt.Sprintf("或 %d", num))

	content := lipgloss.JoinVertical(lipgloss.Left, title, stats, keyHint)
	return cardStyle.Render(content)
}

// renderFooter 渲染底部操作栏
func (v *HomeView) renderFooter() string {
	// 确保宽度有效
	width := v.width
	if width < 60 {
		width = 60
	}

	// 左侧：快捷键提示
	keys := []string{
		homeKeyStyle.Render("↑↓") + " 切换区域",
		homeKeyStyle.Render("←→") + " 选择",
		homeKeyStyle.Render("Enter") + " 进入",
		homeKeyStyle.Render("?") + " 帮助",
		homeKeyStyle.Render("q") + " 退出",
	}
	leftPart := strings.Join(keys, "  ")

	// 右侧：当前选中提示
	var rightPart string
	if v.focusArea == 1 && v.selectedResource < len(v.resources) {
		res := v.resources[v.selectedResource]
		if res.Available {
			rightPart = homeSubtextStyle.Render("当前: ") +
				homeMainTitleStyle.Render(res.Name) + " " +
				homeKeyStyle.Render(fmt.Sprintf("[%s/%d]", res.Key, v.selectedResource+1))
		} else {
			rightPart = homeSubtextStyle.Render("当前: ") +
				homeDevTagStyle.Render(res.Name+" ("+res.Hint+")")
		}
	} else if v.focusArea == 0 && v.selectedRuntime < len(v.runtimes) {
		rt := v.runtimes[v.selectedRuntime]
		rightPart = homeSubtextStyle.Render("当前: ") + homeMainTitleStyle.Render(rt.Name)
	}

	// 计算间距
	leftWidth := lipgloss.Width(leftPart)
	rightWidth := lipgloss.Width(rightPart)
	spacing := width - leftWidth - rightWidth - 4
	if spacing < 2 {
		spacing = 2
	}

	// 分隔线宽度
	separatorWidth := width - 4
	if separatorWidth < 10 {
		separatorWidth = 10
	}
	separator := homeSubtextStyle.Render(strings.Repeat("─", separatorWidth))

	return "  " + separator + "\n" +
		"  " + leftPart + strings.Repeat(" ", spacing) + rightPart + "  "
}

// SetSize 设置视图尺寸
func (v *HomeView) SetSize(width, height int) {
	v.width = width
	v.height = height
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
	result.composeAvailable = false
	result.composeCount = 0
	result.composeRunning = 0

	return result
}
