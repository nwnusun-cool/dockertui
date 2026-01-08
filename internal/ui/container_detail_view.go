package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
)

// ContainerDetailView 容器详情视图
type ContainerDetailView struct {
	dockerClient docker.Client
	
	width  int
	height int
	
	containerID   string
	containerName string
	details       *docker.ContainerDetails
	
	loading    bool
	errorMsg   string
	currentTab int
	
	// 资源监控视图
	statsView *StatsView
	
	keys KeyMap
}

// NewContainerDetailView 创建容器详情视图
func NewContainerDetailView(dockerClient docker.Client) *ContainerDetailView {
	return &ContainerDetailView{
		dockerClient: dockerClient,
		keys:         DefaultKeyMap(),
		width:        100,
		height:       30,
		statsView:    NewStatsView(dockerClient),
	}
}

// SetContainer 设置要查看详情的容器
func (v *ContainerDetailView) SetContainer(containerID, containerName string) {
	v.containerID = containerID
	v.containerName = containerName
	v.statsView.SetContainer(containerID)
}

// Init 初始化
func (v *ContainerDetailView) Init() tea.Cmd {
	if v.containerID == "" {
		return nil
	}
	v.loading = true
	return v.loadDetails
}

// Update 处理消息
func (v *ContainerDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case detailsLoadedMsg:
		v.details = msg.details
		v.loading = false
		v.errorMsg = ""
		return v, nil
		
	case detailsLoadErrorMsg:
		v.loading = false
		v.errorMsg = msg.err.Error()
		return v, nil
	
	// 处理资源监控消息
	case statsLoadedMsg, statsErrorMsg, statsRefreshMsg:
		if v.currentTab == 1 { // 资源监控标签
			cmd := v.statsView.Update(msg)
			return v, cmd
		}
		return v, nil
		
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keys.Refresh):
			v.loading = true
			v.errorMsg = ""
			return v, v.loadDetails
		case msg.String() == "left", msg.String() == "h":
			oldTab := v.currentTab
			if v.currentTab > 0 {
				v.currentTab--
			} else {
				v.currentTab = 5
			}
			return v, v.handleTabChange(oldTab, v.currentTab)
		case msg.String() == "right", msg.String() == "l":
			oldTab := v.currentTab
			v.currentTab = (v.currentTab + 1) % 6
			return v, v.handleTabChange(oldTab, v.currentTab)
		case msg.String() == "tab":
			oldTab := v.currentTab
			v.currentTab = (v.currentTab + 1) % 6
			return v, v.handleTabChange(oldTab, v.currentTab)
		}
	}
	return v, nil
}

// handleTabChange 处理标签页切换
func (v *ContainerDetailView) handleTabChange(oldTab, newTab int) tea.Cmd {
	// 离开资源监控标签时停止监控
	if oldTab == 1 && newTab != 1 {
		v.statsView.Stop()
	}
	
	// 进入资源监控标签时开始监控
	if newTab == 1 && oldTab != 1 {
		return v.statsView.Start()
	}
	
	return nil
}

// View 渲染视图
func (v *ContainerDetailView) View() string {
	// 渲染各部分
	header := v.renderHeader()
	footer := v.renderKeyHints()
	
	// 计算内容区域可用高度
	headerHeight := strings.Count(header, "\n") + 1
	footerHeight := strings.Count(footer, "\n") + 1
	contentHeight := v.height - headerHeight - footerHeight
	if contentHeight < 10 {
		contentHeight = 10
	}
	
	var content string
	if v.loading {
		content = v.renderCenteredState("⏳ 正在加载...", "请稍候，正在获取容器详情", contentHeight)
	} else if v.errorMsg != "" {
		content = v.renderCenteredState("❌ 加载失败", v.errorMsg, contentHeight)
	} else if v.details == nil {
		content = v.renderCenteredState("📭 暂无数据", "按 r 重新加载", contentHeight)
	} else {
		tabBar := v.renderTabBar()
		tabBarHeight := strings.Count(tabBar, "\n") + 1
		tabContent := v.renderTabContent(contentHeight - tabBarHeight)
		content = "\n" + tabBar + tabContent
	}
	
	// 组合布局：header + content + footer
	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

// renderCenteredState 渲染居中的状态提示（加载中/错误/空数据）
func (v *ContainerDetailView) renderCenteredState(title, message string, availableHeight int) string {
	boxWidth := v.width - 8
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 70 {
		boxWidth = 70
	}
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth).
		Align(lipgloss.Center)
	
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	content := titleStyle.Render(title) + "\n\n" + msgStyle.Render(message)
	box := boxStyle.Render(content)
	
	// 计算垂直居中的填充
	boxHeight := strings.Count(box, "\n") + 1
	paddingTop := (availableHeight - boxHeight) / 2
	if paddingTop < 1 {
		paddingTop = 1
	}
	
	// 水平居中
	centeredBox := lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(box)
	
	return strings.Repeat("\n", paddingTop) + centeredBox
}

// renderHeader 渲染顶部标题栏
func (v *ContainerDetailView) renderHeader() string {
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Width(v.width).
		Padding(0, 1)
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("220"))
	
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	if v.details == nil {
		title := titleStyle.Render("📋 " + v.containerName)
		return headerStyle.Render(title)
	}
	
	// 状态徽章
	var statusStyle lipgloss.Style
	var statusText string
	switch v.details.State {
	case "running":
		statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)
		statusText = "● RUNNING"
	case "exited":
		statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
		statusText = "■ STOPPED"
	case "paused":
		statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))
		statusText = "❚❚ PAUSED"
	default:
		statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
		statusText = "✗ " + strings.ToUpper(v.details.State)
	}
	
	// 第一行：名称 + 状态
	title := titleStyle.Render("📋 " + v.details.Name)
	status := statusStyle.Render(statusText)
	line1 := title + "  " + status
	
	// 第二行：ID + 镜像 + 创建时间
	shortID := v.details.ID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	
	// 根据宽度决定显示多少信息
	var line2 string
	if v.width > 80 {
		line2 = infoStyle.Render(fmt.Sprintf("ID: %s  │  镜像: %s  │  创建: %s",
			shortID,
			v.truncate(v.details.Image, 30),
			v.details.Created.Format("2006-01-02 15:04"),
		))
	} else {
		line2 = infoStyle.Render(fmt.Sprintf("ID: %s  │  %s", shortID, v.truncate(v.details.Image, 20)))
	}
	
	content := line1 + "\n" + line2
	return headerStyle.Render(content)
}

// renderTabBar 渲染标签页导航
func (v *ContainerDetailView) renderTabBar() string {
	tabs := []string{"Basic Info", "Resources", "Network", "Storage", "Env Vars", "Labels"}
	
	// 根据宽度决定是否使用简短标签
	if v.width < 80 {
		tabs = []string{"Basic", "Stats", "Network", "Storage", "Env", "Labels"}
	}
	
	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true).
		Underline(true)
	
	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	var parts []string
	for i, tab := range tabs {
		tabNum := fmt.Sprintf("[%d]", i+1)
		if i == v.currentTab {
			parts = append(parts, activeStyle.Render(tabNum+" "+tab))
		} else {
			parts = append(parts, inactiveStyle.Render(tabNum+" "+tab))
		}
	}
	
	tabLine := "  " + strings.Join(parts, "  │  ")
	
	// 底部分隔线
	lineWidth := v.width - 2
	if lineWidth < 40 {
		lineWidth = 40
	}
	line := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(strings.Repeat("─", lineWidth))
	
	return tabLine + "\n" + " " + line + "\n"
}

// renderBasicInfo 渲染基本信息
func (v *ContainerDetailView) renderBasicInfo() string {
	boxWidth := v.width - 4
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 90 {
		boxWidth = 90
	}
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth)
	
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Width(12)
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	row := func(label, value string) string {
		return labelStyle.Render(label) + valueStyle.Render(value)
	}
	
	restartPolicy := v.details.RestartPolicy
	if restartPolicy == "" {
		restartPolicy = "no"
	}
	
	content := lipgloss.JoinVertical(lipgloss.Left,
		row("容器 ID", v.details.ID),
		row("容器名称", v.details.Name),
		row("镜像", v.details.Image),
		row("创建时间", v.details.Created.Format("2006-01-02 15:04:05")),
		row("状态", v.details.Status),
		row("重启策略", restartPolicy),
		row("网络模式", v.details.NetworkMode),
	)
	
	box := boxStyle.Render(content)
	return "\n" + lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(box) + "\n"
}

// renderTabContent 渲染标签页内容
func (v *ContainerDetailView) renderTabContent(availableHeight int) string {
	var content string
	switch v.currentTab {
	case 0:
		content = v.renderBasicInfo()
	case 1:
		content = v.renderStatsTab(availableHeight)
	case 2:
		content = v.renderNetworkInfo()
	case 3:
		content = v.renderStorageInfo()
	case 4:
		content = v.renderEnvInfo()
	case 5:
		content = v.renderLabelsInfo()
	default:
		content = v.renderBasicInfo()
	}
	
	// 确保内容区域填满可用高度
	contentHeight := strings.Count(content, "\n") + 1
	if contentHeight < availableHeight {
		content += strings.Repeat("\n", availableHeight-contentHeight)
	}
	
	return content
}

// renderStatsTab 渲染资源监控标签页
func (v *ContainerDetailView) renderStatsTab(availableHeight int) string {
	// 检查容器是否运行中
	if v.details != nil && v.details.State != "running" {
		return v.renderCenteredState("⚠️ 容器未运行", "资源监控仅在容器运行时可用", availableHeight)
	}
	
	v.statsView.SetSize(v.width, availableHeight)
	return v.statsView.Render()
}

// renderNetworkInfo 渲染网络信息
func (v *ContainerDetailView) renderNetworkInfo() string {
	boxWidth := v.width - 8
	if boxWidth < 60 {
		boxWidth = 60
	}
	if boxWidth > 100 {
		boxWidth = 100
	}
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth)
	
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	var s strings.Builder
	
	// 端口映射
	s.WriteString("\n  " + titleStyle.Render("端口映射") + "\n")
	
	if len(v.details.Ports) == 0 {
		s.WriteString("  " + boxStyle.Render(hintStyle.Render("无端口映射")) + "\n")
	} else {
		var lines []string
		for _, p := range v.details.Ports {
			line := fmt.Sprintf("%-15s → 容器:%d/%s", 
				fmt.Sprintf("%s:%d", p.IP, p.PublicPort),
				p.PrivatePort, p.Type)
			lines = append(lines, valueStyle.Render(line))
		}
		s.WriteString("  " + boxStyle.Render(strings.Join(lines, "\n")) + "\n")
	}
	
	// 网络模式
	s.WriteString("\n  " + titleStyle.Render("网络配置") + "\n")
	s.WriteString("  " + boxStyle.Render(valueStyle.Render("模式: "+v.details.NetworkMode)) + "\n")
	
	return s.String()
}

// renderStorageInfo 渲染存储信息
func (v *ContainerDetailView) renderStorageInfo() string {
	boxWidth := v.width - 8
	if boxWidth < 60 {
		boxWidth = 60
	}
	if boxWidth > 100 {
		boxWidth = 100
	}
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth)
	
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	typeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))
	
	var s strings.Builder
	s.WriteString("\n  " + titleStyle.Render("挂载点") + "\n")
	
	if len(v.details.Mounts) == 0 {
		s.WriteString("  " + boxStyle.Render(hintStyle.Render("无挂载点")) + "\n")
		return s.String()
	}
	
	maxSrcLen := (boxWidth - 30) / 2
	if maxSrcLen < 20 {
		maxSrcLen = 20
	}
	
	var lines []string
	for _, m := range v.details.Mounts {
		src := v.truncate(m.Source, maxSrcLen)
		dst := v.truncate(m.Destination, maxSrcLen)
		line := typeStyle.Render(fmt.Sprintf("[%-6s]", m.Type)) + " " +
			valueStyle.Render(src+" → "+dst) + " " +
			hintStyle.Render("("+m.Mode+")")
		lines = append(lines, line)
	}
	
	s.WriteString("  " + boxStyle.Render(strings.Join(lines, "\n")) + "\n")
	return s.String()
}

// renderEnvInfo 渲染环境变量
func (v *ContainerDetailView) renderEnvInfo() string {
	boxWidth := v.width - 8
	if boxWidth < 60 {
		boxWidth = 60
	}
	if boxWidth > 100 {
		boxWidth = 100
	}
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth)
	
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	var s strings.Builder
	
	if len(v.details.Env) == 0 {
		s.WriteString("\n  " + titleStyle.Render("环境变量") + "\n")
		s.WriteString("  " + boxStyle.Render(hintStyle.Render("无环境变量")) + "\n")
		return s.String()
	}
	
	// 分类
	var appVars, sysVars []string
	sysKeys := map[string]bool{"PATH": true, "HOME": true, "USER": true, "SHELL": true, "TERM": true, "HOSTNAME": true}
	
	for _, env := range v.details.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			if sysKeys[parts[0]] {
				sysVars = append(sysVars, env)
			} else {
				appVars = append(appVars, env)
			}
		}
	}
	
	maxValLen := boxWidth - 25
	if maxValLen < 30 {
		maxValLen = 30
	}
	
	formatEnv := func(env string, isApp bool) string {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			return env
		}
		val := v.truncate(parts[1], maxValLen)
		if isApp {
			return keyStyle.Render(parts[0]) + " = " + valueStyle.Render(val)
		}
		return hintStyle.Render(parts[0] + " = " + val)
	}
	
	// 应用变量
	if len(appVars) > 0 {
		s.WriteString("\n  " + titleStyle.Render("应用环境变量") + "\n")
		var lines []string
		for _, env := range appVars {
			lines = append(lines, formatEnv(env, true))
		}
		s.WriteString("  " + boxStyle.Render(strings.Join(lines, "\n")) + "\n")
	}
	
	// 系统变量
	if len(sysVars) > 0 {
		s.WriteString("\n  " + titleStyle.Render("系统环境变量") + "\n")
		var lines []string
		for _, env := range sysVars {
			lines = append(lines, formatEnv(env, false))
		}
		s.WriteString("  " + boxStyle.Render(strings.Join(lines, "\n")) + "\n")
	}
	
	return s.String()
}

// renderLabelsInfo 渲染标签信息
func (v *ContainerDetailView) renderLabelsInfo() string {
	boxWidth := v.width - 8
	if boxWidth < 60 {
		boxWidth = 60
	}
	if boxWidth > 100 {
		boxWidth = 100
	}
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth)
	
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	var s strings.Builder
	
	if len(v.details.Labels) == 0 {
		s.WriteString("\n  " + titleStyle.Render("标签") + "\n")
		s.WriteString("  " + boxStyle.Render(hintStyle.Render("无标签")) + "\n")
		return s.String()
	}
	
	// 分类
	var customLabels, composeLabels, dockerLabels []struct{ k, v string }
	
	for k, val := range v.details.Labels {
		item := struct{ k, v string }{k, val}
		if strings.HasPrefix(k, "com.docker.compose.") {
			composeLabels = append(composeLabels, item)
		} else if strings.HasPrefix(k, "com.docker.") {
			dockerLabels = append(dockerLabels, item)
		} else {
			customLabels = append(customLabels, item)
		}
	}
	
	maxValLen := boxWidth - 10
	if maxValLen < 40 {
		maxValLen = 40
	}
	
	formatLabel := func(k, val string, highlight bool) string {
		val = v.truncate(val, maxValLen)
		if highlight {
			return keyStyle.Render(k) + "\n  " + valueStyle.Render(val)
		}
		return hintStyle.Render(k) + "\n  " + hintStyle.Render(val)
	}
	
	// 自定义标签
	if len(customLabels) > 0 {
		s.WriteString("\n  " + titleStyle.Render("自定义标签") + "\n")
		var lines []string
		for _, l := range customLabels {
			lines = append(lines, formatLabel(l.k, l.v, true))
		}
		s.WriteString("  " + boxStyle.Render(strings.Join(lines, "\n")) + "\n")
	}
	
	// Compose 标签
	if len(composeLabels) > 0 {
		s.WriteString("\n  " + titleStyle.Render("Docker Compose 标签") + "\n")
		var lines []string
		for _, l := range composeLabels {
			lines = append(lines, formatLabel(l.k, l.v, false))
		}
		s.WriteString("  " + boxStyle.Render(strings.Join(lines, "\n")) + "\n")
	}
	
	// Docker 标签
	if len(dockerLabels) > 0 {
		s.WriteString("\n  " + titleStyle.Render("Docker 系统标签") + "\n")
		var lines []string
		for _, l := range dockerLabels {
			lines = append(lines, formatLabel(l.k, l.v, false))
		}
		s.WriteString("  " + boxStyle.Render(strings.Join(lines, "\n")) + "\n")
	}
	
	return s.String()
}

// renderKeyHints 渲染底部快捷键提示（固定在底部）
func (v *ContainerDetailView) renderKeyHints() string {
	footerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Width(v.width).
		Padding(0, 1)
	
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Bold(true)
	
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	// 根据宽度决定显示多少快捷键
	var items []struct{ key, desc string }
	if v.width > 90 {
		items = []struct{ key, desc string }{
			{"←/→", "切换标签"},
			{"l", "日志"},
			{"s", "终端"},
			{"r", "刷新"},
			{"Esc", "返回"},
			{"q", "退出"},
		}
	} else if v.width > 60 {
		items = []struct{ key, desc string }{
			{"←/→", "标签"},
			{"l", "日志"},
			{"s", "终端"},
			{"Esc", "返回"},
			{"q", "退出"},
		}
	} else {
		items = []struct{ key, desc string }{
			{"←/→", "标签"},
			{"Esc", "返回"},
			{"q", "退出"},
		}
	}
	
	var parts []string
	for _, item := range items {
		parts = append(parts, keyStyle.Render(item.key)+" "+descStyle.Render(item.desc))
	}
	
	line := strings.Join(parts, "  ")
	return footerStyle.Render(line)
}

// truncate 截断字符串
func (v *ContainerDetailView) truncate(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// SetSize 设置视图尺寸
func (v *ContainerDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.statsView.SetSize(width, height-10)
}

// detailsLoadedMsg 详情加载完成消息
type detailsLoadedMsg struct {
	details *docker.ContainerDetails
}

// detailsLoadErrorMsg 详情加载错误消息
type detailsLoadErrorMsg struct {
	err error
}

// loadDetails 加载容器详情
func (v *ContainerDetailView) loadDetails() tea.Msg {
	if v.containerID == "" {
		return detailsLoadErrorMsg{err: fmt.Errorf("容器 ID 为空")}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	details, err := v.dockerClient.ContainerDetails(ctx, v.containerID)
	if err != nil {
		return detailsLoadErrorMsg{err: err}
	}
	
	return detailsLoadedMsg{details: details}
}
