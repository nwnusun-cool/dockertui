package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/compose"
	"docktui/internal/docker"
)

// ResourceType 资源类型
type ResourceType int

const (
	ResourceContainers ResourceType = iota
	ResourceImages
	ResourceNetworks
	ResourceCompose
)

// ResourceInfo 资源信息
type ResourceInfo struct {
	Type        ResourceType
	Name        string
	Icon        string
	Key         string
	Count       int
	ActiveCount int
	Available   bool
}

// HomeView 首页导航视图
type HomeView struct {
	dockerClient docker.Client

	width  int
	height int

	resources        []ResourceInfo
	selectedResource int

	loading         bool
	lastRefreshTime time.Time
	dockerConnected bool
	dockerHost      string
}

// NewHomeView 创建首页视图
func NewHomeView(dockerClient docker.Client) *HomeView {
	// 获取 Docker Host
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "Local Docker"
	}

	v := &HomeView{
		dockerClient:     dockerClient,
		selectedResource: 0,
		dockerHost:       dockerHost,
	}

	v.resources = []ResourceInfo{
		{Type: ResourceContainers, Name: "Containers", Icon: "◈", Key: "c", Available: true},
		{Type: ResourceImages, Name: "Images", Icon: "▣", Key: "i", Available: true},
		{Type: ResourceNetworks, Name: "Networks", Icon: "⊕", Key: "n", Available: true},
		{Type: ResourceCompose, Name: "Compose", Icon: "⚙", Key: "o", Available: true},
	}

	return v
}

// Init 初始化
func (v *HomeView) Init() tea.Cmd {
	v.loading = true
	return v.loadStats
}

// Update 处理消息
func (v *HomeView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case homeStatsLoadedMsg:
		v.loading = false
		v.lastRefreshTime = time.Now()
		v.dockerConnected = msg.dockerConnected

		for i := range v.resources {
			switch v.resources[i].Type {
			case ResourceContainers:
				v.resources[i].Count = msg.containerCount
				v.resources[i].ActiveCount = msg.runningCount
			case ResourceImages:
				v.resources[i].Count = msg.imageCount
			case ResourceNetworks:
				v.resources[i].Count = msg.networkCount
			case ResourceCompose:
				v.resources[i].Count = msg.composeCount
				v.resources[i].ActiveCount = msg.composeRunning
				v.resources[i].Available = msg.composeAvailable
			}
		}
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if v.selectedResource > 0 {
				v.selectedResource--
			}
		case "right", "l":
			if v.selectedResource < len(v.resources)-1 {
				v.selectedResource++
			}
		case "1", "2", "3", "4":
			idx := int(msg.String()[0] - '1')
			if idx >= 0 && idx < len(v.resources) {
				v.selectedResource = idx
			}
		case "r", "f5":
			v.loading = true
			return v, v.loadStats
		}
	}

	return v, nil
}

// View 渲染视图
func (v *HomeView) View() string {
	width := v.width
	height := v.height
	if width < 80 {
		width = 80
	}
	if height < 20 {
		height = 20
	}

	// 渲染各部分
	logo := v.renderLogo()
	status := v.renderConnectionStatus()
	cards := v.renderResourceCards()
	footer := v.renderFooter()

	// 计算各部分高度
	logoHeight := strings.Count(logo, "\n") + 1
	statusHeight := 1
	cardsHeight := strings.Count(cards, "\n") + 1
	footerHeight := strings.Count(footer, "\n") + 1

	// 内容总高度
	contentHeight := logoHeight + statusHeight + cardsHeight + 4 // +4 for spacing

	// 计算垂直居中的顶部填充
	topPadding := (height - contentHeight - footerHeight) / 3
	if topPadding < 1 {
		topPadding = 1
	}

	// 计算底部填充（footer 固定在底部）
	bottomPadding := height - topPadding - contentHeight - footerHeight
	if bottomPadding < 1 {
		bottomPadding = 1
	}

	var b strings.Builder

	// 顶部填充
	b.WriteString(strings.Repeat("\n", topPadding))

	// Logo
	b.WriteString(logo)
	b.WriteString("\n\n")

	// 连接状态
	b.WriteString(status)
	b.WriteString("\n\n")

	// 资源卡片
	b.WriteString(cards)

	// 底部填充
	b.WriteString(strings.Repeat("\n", bottomPadding))

	// Footer
	b.WriteString(footer)

	return b.String()
}

// renderLogo 渲染 Logo
func (v *HomeView) renderLogo() string {
	width := v.width
	if width < 80 {
		width = 80
	}

	logoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	versionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	// 根据终端宽度选择 Logo
	var logo string
	if width < 60 {
		// 小终端：简化 Logo
		logo = `
  DockTUI`
	} else {
		// 正常 Logo
		logo = `
    ____             __  ______ __  ______
   / __ \____  _____/ /_/_  __/ / / /  _/
  / / / / __ \/ ___/ //_/ / / / / / // /  
 / /_/ / /_/ / /__/ ,<   / / / /_/ // /   
/_____/\____/\___/_/|_| /_/  \____/___/`
	}

	// Logo 居中
	logoLines := strings.Split(logo, "\n")
	var centeredLogo strings.Builder
	for _, line := range logoLines {
		lineWidth := len(line)
		leftPadding := (width - lineWidth) / 2
		if leftPadding < 0 {
			leftPadding = 0
		}
		centeredLogo.WriteString(strings.Repeat(" ", leftPadding) + line + "\n")
	}

	// 版本信息居中
	subtitle := versionStyle.Render("Docker TUI  v0.1.0")
	subtitleWidth := lipgloss.Width(subtitle)
	subtitlePadding := (width - subtitleWidth) / 2
	if subtitlePadding < 0 {
		subtitlePadding = 0
	}

	return logoStyle.Render(centeredLogo.String()) + strings.Repeat(" ", subtitlePadding) + subtitle
}

// renderHeader 渲染顶部标题（保留兼容）
func (v *HomeView) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Bold(true)

	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	title := titleStyle.Render("🐳 DockTUI")
	version := versionStyle.Render("v0.1.0")

	// 右侧刷新时间
	var rightPart string
	if v.loading {
		rightPart = versionStyle.Render("Loading...")
	} else if !v.lastRefreshTime.IsZero() {
		rightPart = versionStyle.Render("Refresh: " + v.lastRefreshTime.Format("15:04:05"))
	}

	leftPart := title + " " + version
	leftWidth := lipgloss.Width(leftPart)
	rightWidth := lipgloss.Width(rightPart)

	width := v.width
	if width < 80 {
		width = 80
	}

	spacing := width - leftWidth - rightWidth - 4
	if spacing < 2 {
		spacing = 2
	}

	return "  " + leftPart + strings.Repeat(" ", spacing) + rightPart + "  "
}

// renderConnectionStatus 渲染连接状态
func (v *HomeView) renderConnectionStatus() string {
	width := v.width
	if width < 80 {
		width = 80
	}

	var statusIcon, statusText string
	var statusColor lipgloss.Color

	if v.dockerConnected {
		statusIcon = "●"
		statusText = "Docker Connected"
		statusColor = lipgloss.Color("82")
	} else {
		statusIcon = "○"
		statusText = "Docker Disconnected"
		statusColor = lipgloss.Color("196")
	}

	statusStyle := lipgloss.NewStyle().Foreground(statusColor)
	hostStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	content := statusStyle.Render(statusIcon+" "+statusText) + "    " + hostStyle.Render(v.dockerHost)

	// 居中
	contentWidth := lipgloss.Width(content)
	leftPadding := (width - contentWidth) / 2
	if leftPadding < 2 {
		leftPadding = 2
	}

	return strings.Repeat(" ", leftPadding) + content
}

// renderResourceCards 渲染资源卡片
func (v *HomeView) renderResourceCards() string {
	width := v.width
	if width < 80 {
		width = 80
	}

	// 根据终端宽度计算卡片大小
	// 4 张卡片 + 3 个间隔(2字符) + 左右边距(4字符)
	// 可用宽度 = width - 4 - 6 = width - 10
	availableWidth := width - 10
	cardWidth := availableWidth / 4
	if cardWidth < 16 {
		cardWidth = 16
	}
	if cardWidth > 24 {
		cardWidth = 24
	}

	var cards []string
	for i, res := range v.resources {
		isSelected := i == v.selectedResource
		cards = append(cards, v.renderCardWithWidth(res, isSelected, i+1, cardWidth))
	}

	// 水平拼接卡片
	cardsRow := v.joinCardsHorizontal(cards, "  ")

	// 居中
	cardsWidth := v.getFirstLineWidth(cardsRow)
	leftPadding := (width - cardsWidth) / 2
	if leftPadding < 2 {
		leftPadding = 2
	}

	lines := strings.Split(cardsRow, "\n")
	for i, line := range lines {
		lines[i] = strings.Repeat(" ", leftPadding) + line
	}

	return strings.Join(lines, "\n")
}

// renderCardWithWidth 渲染指定宽度的卡片
func (v *HomeView) renderCardWithWidth(res ResourceInfo, selected bool, num int, cardWidth int) string {
	contentWidth := cardWidth - 6 // padding(4) + border(2)
	if contentWidth < 10 {
		contentWidth = 10
	}

	var borderColor lipgloss.Color
	if selected {
		borderColor = lipgloss.Color("81")
	} else {
		borderColor = lipgloss.Color("240")
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 2).
		Width(cardWidth)

	// 标题 (图标 + 名称) - 不使用 Width 自动对齐，手动居中
	titleText := res.Icon + " " + res.Name
	titleStyle := lipgloss.NewStyle()
	if selected {
		titleStyle = titleStyle.Foreground(lipgloss.Color("81")).Bold(true)
	}
	// 手动计算居中填充，假设图标占 1 个显示宽度
	titleDisplayWidth := len(res.Name) + 2 // icon(1) + space(1) + name
	titlePadding := (contentWidth - titleDisplayWidth) / 2
	if titlePadding < 0 {
		titlePadding = 0
	}
	title := strings.Repeat(" ", titlePadding) + titleStyle.Render(titleText)

	// 数量
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	statsStyle := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center)

	var stats string
	if v.loading {
		stats = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("...")
	} else if !res.Available {
		stats = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("Unavailable")
	} else {
		countStr := countStyle.Render(fmt.Sprintf("%d", res.Count))
		if res.ActiveCount > 0 && (res.Type == ResourceContainers || res.Type == ResourceCompose) {
			activeStr := activeStyle.Render(fmt.Sprintf("%d", res.ActiveCount))
			stats = countStr + " (" + activeStr + ")"
		} else {
			stats = countStr
		}
	}
	stats = statsStyle.Render(stats)

	// 快捷键
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	keyHintStyle := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center)
	keyHint := keyHintStyle.Render(keyStyle.Render(res.Key) + hintStyle.Render(" or "+fmt.Sprintf("%d", num)))

	content := lipgloss.JoinVertical(lipgloss.Center, title, stats, keyHint)
	return cardStyle.Render(content)
}

// renderCard 渲染单个卡片 (保留兼容)
func (v *HomeView) renderCard(res ResourceInfo, selected bool, num int) string {
	return v.renderCardWithWidth(res, selected, num, 20)
}

// renderFooter 渲染底部快捷键
func (v *HomeView) renderFooter() string {
	width := v.width
	if width < 80 {
		width = 80
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	keys := []struct{ key, desc string }{
		{"←→", "Select"},
		{"Enter", "Enter"},
		{"r", "Refresh"},
		{"?", "Help"},
		{"q", "Exit"},
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts, keyStyle.Render(k.key)+" "+descStyle.Render(k.desc))
	}

	line := strings.Join(parts, sepStyle.Render("  │  "))

	// 居中
	lineWidth := lipgloss.Width(line)
	leftPadding := (width - lineWidth) / 2
	if leftPadding < 2 {
		leftPadding = 2
	}

	separator := sepStyle.Render(strings.Repeat("─", width-4))

	return "  " + separator + "\n" + strings.Repeat(" ", leftPadding) + line
}

// joinCardsHorizontal 水平拼接卡片
func (v *HomeView) joinCardsHorizontal(cards []string, sep string) string {
	if len(cards) == 0 {
		return ""
	}
	if len(cards) == 1 {
		return cards[0]
	}

	cardLines := make([][]string, len(cards))
	maxLines := 0
	for i, card := range cards {
		cardLines[i] = strings.Split(card, "\n")
		if len(cardLines[i]) > maxLines {
			maxLines = len(cardLines[i])
		}
	}

	cardWidths := make([]int, len(cards))
	for i, lines := range cardLines {
		if len(lines) > 0 {
			cardWidths[i] = lipgloss.Width(lines[0])
		}
	}

	var result []string
	for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
		var lineParts []string
		for cardIdx, lines := range cardLines {
			var line string
			if lineIdx < len(lines) {
				line = lines[lineIdx]
			}
			lineWidth := lipgloss.Width(line)
			if lineWidth < cardWidths[cardIdx] {
				line = line + strings.Repeat(" ", cardWidths[cardIdx]-lineWidth)
			}
			lineParts = append(lineParts, line)
		}
		result = append(result, strings.Join(lineParts, sep))
	}

	return strings.Join(result, "\n")
}

// getFirstLineWidth 获取第一行宽度
func (v *HomeView) getFirstLineWidth(s string) int {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 {
		return lipgloss.Width(lines[0])
	}
	return 0
}

// SetSize 设置尺寸
func (v *HomeView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// GetSelectedResource 获取选中的资源类型
func (v *HomeView) GetSelectedResource() ResourceType {
	if v.selectedResource >= 0 && v.selectedResource < len(v.resources) {
		return v.resources[v.selectedResource].Type
	}
	return ResourceContainers
}

// IsResourceAvailable 检查资源是否可用
func (v *HomeView) IsResourceAvailable() bool {
	if v.selectedResource >= 0 && v.selectedResource < len(v.resources) {
		return v.resources[v.selectedResource].Available
	}
	return false
}

// homeStatsLoadedMsg 统计数据加载完成消息
type homeStatsLoadedMsg struct {
	dockerConnected  bool
	containerCount   int
	runningCount     int
	imageCount       int
	networkCount     int
	composeCount     int
	composeRunning   int
	composeAvailable bool
}

// loadStats 加载统计数据
func (v *HomeView) loadStats() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := homeStatsLoadedMsg{
		dockerConnected:  true,
		composeAvailable: true,
	}

	// 容器统计
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

	// 镜像统计
	images, err := v.dockerClient.ListImages(ctx, true)
	if err == nil {
		result.imageCount = len(images)
	}

	// 网络统计
	networks, err := v.dockerClient.ListNetworks(ctx)
	if err == nil {
		result.networkCount = len(networks)
	}

	// Compose 统计
	composeClient, err := compose.NewClient()
	if err != nil {
		result.composeAvailable = false
	} else {
		result.composeAvailable = true
		_ = composeClient
	}

	return result
}
