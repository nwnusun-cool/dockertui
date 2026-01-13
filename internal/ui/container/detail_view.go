package container

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
	"docktui/internal/ui/components"
)

// DetailView 容器详情视图
type DetailView struct {
	dockerClient docker.Client
	
	width  int
	height int
	
	containerID   string
	containerName string
	details       *docker.ContainerDetails
	
	loading    bool
	errorMsg   string
	currentTab int
	
	// 滚动支持
	scrollOffset int
	maxScroll    int
	
	// 资源监控视图
	statsView *components.StatsView
	
	// 进程列表视图
	processesView *components.ProcessesView
	
	keys components.KeyMap
}

// NewDetailView 创建容器详情视图
func NewDetailView(dockerClient docker.Client) *DetailView {
	return &DetailView{
		dockerClient:  dockerClient,
		keys:          components.DefaultKeyMap(),
		width:         100,
		height:        30,
		statsView:     components.NewStatsView(dockerClient),
		processesView: components.NewProcessesView(dockerClient),
	}
}

// SetContainer 设置要查看详情的容器
func (v *DetailView) SetContainer(containerID, containerName string) {
	v.containerID = containerID
	v.containerName = containerName
	v.statsView.SetContainer(containerID)
	v.processesView.SetContainer(containerID)
}

// Init 初始化
func (v *DetailView) Init() tea.Cmd {
	if v.containerID == "" {
		return nil
	}
	v.loading = true
	return v.loadDetails
}

// Update 处理消息
func (v *DetailView) Update(msg tea.Msg) (*DetailView, tea.Cmd) {
	switch msg := msg.(type) {
	case DetailsLoadedMsg:
		v.details = msg.Details
		v.loading = false
		v.errorMsg = ""
		return v, nil
		
	case DetailsLoadErrorMsg:
		v.loading = false
		v.errorMsg = msg.Err.Error()
		return v, nil
	
	// 处理资源监控消息
	case components.StatsLoadedMsg, components.StatsErrorMsg, components.StatsRefreshMsg:
		if v.currentTab == 1 { // 资源监控标签
			cmd := v.statsView.Update(msg)
			return v, cmd
		}
		return v, nil
	
	// 处理进程列表消息
	case components.ProcessesLoadedMsg, components.ProcessesErrorMsg, components.ProcessesRefreshMsg:
		if v.currentTab == 6 { // 进程列表标签
			cmd := v.processesView.Update(msg)
			return v, cmd
		}
		return v, nil
		
	case tea.KeyMsg:
		// 如果在资源监控标签页，先让 statsView 处理按键
		if v.currentTab == 1 {
			cmd := v.statsView.Update(msg)
			if cmd != nil {
				return v, cmd
			}
		}
		
		switch {
		case msg.String() == "esc":
			// ESC 返回上一级
			return v, func() tea.Msg { return GoBackMsg{} }
		case key.Matches(msg, v.keys.Refresh):
			v.loading = true
			v.errorMsg = ""
			return v, v.loadDetails
		case msg.String() == "left", msg.String() == "h":
			oldTab := v.currentTab
			if v.currentTab > 0 {
				v.currentTab--
			} else {
				v.currentTab = 6
			}
			v.scrollOffset = 0 // 切换标签时重置滚动
			return v, v.handleTabChange(oldTab, v.currentTab)
		case msg.String() == "right", msg.String() == "l":
			oldTab := v.currentTab
			v.currentTab = (v.currentTab + 1) % 7
			v.scrollOffset = 0 // 切换标签时重置滚动
			return v, v.handleTabChange(oldTab, v.currentTab)
		case msg.String() == "tab":
			oldTab := v.currentTab
			v.currentTab = (v.currentTab + 1) % 7
			v.scrollOffset = 0 // 切换标签时重置滚动
			return v, v.handleTabChange(oldTab, v.currentTab)
		case msg.String() == "j", msg.String() == "down":
			if v.scrollOffset < v.maxScroll {
				v.scrollOffset++
			}
			return v, nil
		case msg.String() == "k", msg.String() == "up":
			if v.scrollOffset > 0 {
				v.scrollOffset--
			}
			return v, nil
		case msg.String() == "g":
			v.scrollOffset = 0
			return v, nil
		case msg.String() == "G":
			v.scrollOffset = v.maxScroll
			return v, nil
		}
	}
	return v, nil
}

// handleTabChange 处理标签页切换
func (v *DetailView) handleTabChange(oldTab, newTab int) tea.Cmd {
	// 离开资源监控标签时停止监控
	if oldTab == 1 && newTab != 1 {
		v.statsView.Stop()
	}
	
	// 离开进程列表标签时停止监控
	if oldTab == 6 && newTab != 6 {
		v.processesView.Stop()
	}
	
	// 进入资源监控标签时开始监控
	if newTab == 1 && oldTab != 1 {
		return v.statsView.Start()
	}
	
	// 进入进程列表标签时开始监控
	if newTab == 6 && oldTab != 6 {
		return v.processesView.Start()
	}
	
	return nil
}

// View 渲染视图
func (v *DetailView) View() string {
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
		content = v.renderCenteredState("⏳ Loading...", "Please wait, fetching container details", contentHeight)
	} else if v.errorMsg != "" {
		content = v.renderCenteredState("❌ Load Failed", v.errorMsg, contentHeight)
	} else if v.details == nil {
		content = v.renderCenteredState("📭 No Data", "Press r to reload", contentHeight)
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
func (v *DetailView) renderCenteredState(title, message string, availableHeight int) string {
	boxWidth := v.width - 8
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 70 {
		boxWidth = 70
	}
	
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	content := titleStyle.Render(title) + "\n\n" + msgStyle.Render(message)
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth).
		Align(lipgloss.Center)
	
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
func (v *DetailView) renderHeader() string {
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
		line2 = infoStyle.Render(fmt.Sprintf("ID: %s  │  Image: %s  │  Created: %s",
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
func (v *DetailView) renderTabBar() string {
	tabs := []string{"Basic Info", "Resources", "Network", "Storage", "Env Vars", "Labels", "Processes"}
	
	// 根据宽度决定是否使用简短标签
	if v.width < 80 {
		tabs = []string{"Basic", "Stats", "Net", "Storage", "Env", "Labels", "Proc"}
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
func (v *DetailView) renderBasicInfo() string {
	boxWidth := v.width - 6
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 90 {
		boxWidth = 90
	}
	
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
	
	var lines []string
	lines = append(lines, row("ID", v.details.ID))
	lines = append(lines, row("Name", v.details.Name))
	lines = append(lines, row("Image", v.details.Image))
	lines = append(lines, row("Created", v.details.Created.Format("2006-01-02 15:04:05")))
	lines = append(lines, row("Status", v.details.Status))
	lines = append(lines, row("Restart", restartPolicy))
	lines = append(lines, row("Network", v.details.NetworkMode))
	
	return "\n" + v.wrapInBox("Basic Information", strings.Join(lines, "\n"), boxWidth)
}

// renderTabContent 渲染标签页内容
func (v *DetailView) renderTabContent(availableHeight int) string {
	var content string
	switch v.currentTab {
	case 0:
		content = v.renderBasicInfo()
	case 1:
		content = v.renderStatsTab(availableHeight)
		return content // Resources 标签页不需要滚动处理
	case 2:
		content = v.renderNetworkInfo()
	case 3:
		content = v.renderStorageInfo()
	case 4:
		content = v.renderEnvInfo()
	case 5:
		content = v.renderLabelsInfo()
	case 6:
		content = v.renderProcessesInfo(availableHeight)
	default:
		content = v.renderBasicInfo()
	}
	
	// 应用滚动
	lines := strings.Split(content, "\n")
	totalLines := len(lines)
	visibleLines := availableHeight - 2 // 留出滚动提示的空间
	if visibleLines < 5 {
		visibleLines = 5
	}
	
	// 计算最大滚动值
	v.maxScroll = totalLines - visibleLines
	if v.maxScroll < 0 {
		v.maxScroll = 0
	}
	
	// 确保滚动偏移在有效范围内
	if v.scrollOffset > v.maxScroll {
		v.scrollOffset = v.maxScroll
	}
	
	// 如果内容不需要滚动
	if v.maxScroll == 0 {
		if len(lines) < availableHeight {
			content += strings.Repeat("\n", availableHeight-len(lines))
		}
		return content
	}
	
	// 截取可见部分
	endIdx := v.scrollOffset + visibleLines
	if endIdx > totalLines {
		endIdx = totalLines
	}
	visibleContent := strings.Join(lines[v.scrollOffset:endIdx], "\n")
	
	// 添加滚动提示
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	scrollHint := ""
	if v.scrollOffset > 0 {
		scrollHint += "↑ "
	}
	if v.scrollOffset < v.maxScroll {
		scrollHint += "↓ "
	}
	scrollHint += fmt.Sprintf("(%d/%d) j/k scroll", v.scrollOffset+1, v.maxScroll+1)
	
	result := visibleContent + "\n\n  " + hintStyle.Render(scrollHint)
	
	// 填充剩余空间
	resultLines := strings.Count(result, "\n") + 1
	if resultLines < availableHeight {
		result += strings.Repeat("\n", availableHeight-resultLines)
	}
	
	return result
}

// renderStatsTab 渲染资源监控标签页
func (v *DetailView) renderStatsTab(availableHeight int) string {
	// 检查容器是否运行中
	if v.details != nil && v.details.State != "running" {
		return v.renderCenteredState("⚠️ Container Not Running", "Resource monitoring only available when container is running", availableHeight)
	}
	
	v.statsView.SetSize(v.width, availableHeight)
	return v.statsView.Render()
}

// renderNetworkInfo 渲染网络信息
func (v *DetailView) renderNetworkInfo() string {
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	var s strings.Builder
	
	// 端口映射
	if len(v.details.Ports) == 0 {
		s.WriteString("\n" + v.wrapInBox("Port Mappings", hintStyle.Render("No port mappings"), boxWidth))
	} else {
		var lines []string
		for _, p := range v.details.Ports {
			line := fmt.Sprintf("%-15s → Container:%d/%s", 
				fmt.Sprintf("%s:%d", p.IP, p.PublicPort),
				p.PrivatePort, p.Type)
			lines = append(lines, valueStyle.Render(line))
		}
		s.WriteString("\n" + v.wrapInBox("Port Mappings", strings.Join(lines, "\n"), boxWidth))
	}
	
	// 网络模式
	s.WriteString("\n\n" + v.wrapInBox("Network Config", valueStyle.Render("Mode: "+v.details.NetworkMode), boxWidth))
	
	return s.String()
}

// renderStorageInfo 渲染存储信息
func (v *DetailView) renderStorageInfo() string {
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	typeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))
	
	if len(v.details.Mounts) == 0 {
		return "\n" + v.wrapInBox("Mounts", hintStyle.Render("No mounts"), boxWidth)
	}
	
	var lines []string
	for _, m := range v.details.Mounts {
		// 不截断路径，完整显示
		line := typeStyle.Render(fmt.Sprintf("[%-6s]", m.Type)) + " " +
			valueStyle.Render(m.Source+" → "+m.Destination) + " " +
			hintStyle.Render("("+m.Mode+")")
		lines = append(lines, line)
	}
	
	return "\n" + v.wrapInBox("Mounts", strings.Join(lines, "\n"), boxWidth)
}

// renderEnvInfo 渲染环境变量
func (v *DetailView) renderEnvInfo() string {
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}
	
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	if len(v.details.Env) == 0 {
		return "\n" + v.wrapInBox("Environment Variables", hintStyle.Render("No environment variables"), boxWidth)
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
	
	formatEnv := func(env string, isApp bool) string {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			return env
		}
		// 不截断，完整显示
		if isApp {
			return keyStyle.Render(parts[0]) + " = " + valueStyle.Render(parts[1])
		}
		return hintStyle.Render(parts[0] + " = " + parts[1])
	}
	
	var s strings.Builder
	
	// 应用变量
	if len(appVars) > 0 {
		var lines []string
		for _, env := range appVars {
			lines = append(lines, formatEnv(env, true))
		}
		s.WriteString("\n" + v.wrapInBox(fmt.Sprintf("App Env Vars (%d)", len(appVars)), strings.Join(lines, "\n"), boxWidth))
	}
	
	// 系统变量
	if len(sysVars) > 0 {
		var lines []string
		for _, env := range sysVars {
			lines = append(lines, formatEnv(env, false))
		}
		s.WriteString("\n\n" + v.wrapInBox(fmt.Sprintf("System Env Vars (%d)", len(sysVars)), strings.Join(lines, "\n"), boxWidth))
	}
	
	return s.String()
}

// renderLabelsInfo 渲染标签信息
func (v *DetailView) renderLabelsInfo() string {
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}
	
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	if len(v.details.Labels) == 0 {
		return "\n" + v.wrapInBox("Labels", hintStyle.Render("No labels"), boxWidth)
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
	
	formatLabel := func(k, val string, highlight bool) string {
		// 不截断，完整显示
		if highlight {
			return keyStyle.Render(k) + "\n  " + valueStyle.Render(val)
		}
		return hintStyle.Render(k) + "\n  " + hintStyle.Render(val)
	}
	
	var s strings.Builder
	
	// 自定义标签
	if len(customLabels) > 0 {
		var lines []string
		for _, l := range customLabels {
			lines = append(lines, formatLabel(l.k, l.v, true))
		}
		s.WriteString("\n" + v.wrapInBox(fmt.Sprintf("Custom Labels (%d)", len(customLabels)), strings.Join(lines, "\n"), boxWidth))
	}
	
	// Compose 标签
	if len(composeLabels) > 0 {
		var lines []string
		for _, l := range composeLabels {
			lines = append(lines, formatLabel(l.k, l.v, false))
		}
		s.WriteString("\n\n" + v.wrapInBox(fmt.Sprintf("Docker Compose Labels (%d)", len(composeLabels)), strings.Join(lines, "\n"), boxWidth))
	}
	
	// Docker 标签
	if len(dockerLabels) > 0 {
		var lines []string
		for _, l := range dockerLabels {
			lines = append(lines, formatLabel(l.k, l.v, false))
		}
		s.WriteString("\n\n" + v.wrapInBox(fmt.Sprintf("Docker System Labels (%d)", len(dockerLabels)), strings.Join(lines, "\n"), boxWidth))
	}
	
	return s.String()
}

// renderProcessesInfo 渲染进程信息
func (v *DetailView) renderProcessesInfo(availableHeight int) string {
	// 检查容器是否运行中
	if v.details != nil && v.details.State != "running" {
		return v.renderCenteredState("⚠️ Container Not Running", "Process list only available when container is running", availableHeight)
	}
	
	v.processesView.SetSize(v.width, availableHeight)
	return v.processesView.Render()
}

// renderKeyHints 渲染底部快捷键提示（固定在底部）
func (v *DetailView) renderKeyHints() string {
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
	if v.width > 100 {
		items = []struct{ key, desc string }{
			{"←/→", "Tabs"},
			{"j/k", "Scroll"},
			{"l", "Logs"},
			{"s", "Shell"},
			{"r", "Refresh"},
			{"Esc", "Back"},
			{"q", "Quit"},
		}
	} else if v.width > 70 {
		items = []struct{ key, desc string }{
			{"←/→", "Tabs"},
			{"j/k", "Scroll"},
			{"l", "Logs"},
			{"s", "Shell"},
			{"Esc", "Back"},
		}
	} else {
		items = []struct{ key, desc string }{
			{"←/→", "Tabs"},
			{"j/k", "Scroll"},
			{"Esc", "Back"},
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
func (v *DetailView) truncate(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// wrapInBox 用边框包裹内容（和镜像/网络模块保持一致）
func (v *DetailView) wrapInBox(title, content string, width int) string {
	return components.WrapInBox(title, content, width)
}

// SetSize 设置视图尺寸
func (v *DetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.statsView.SetSize(width, height-10)
}

// loadDetails 加载容器详情
func (v *DetailView) loadDetails() tea.Msg {
	if v.containerID == "" {
		return DetailsLoadErrorMsg{Err: fmt.Errorf("container ID is empty")}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	details, err := v.dockerClient.ContainerDetails(ctx, v.containerID)
	if err != nil {
		return DetailsLoadErrorMsg{Err: err}
	}
	
	return DetailsLoadedMsg{Details: details}
}


// GetDetails 获取容器详情
func (v *DetailView) GetDetails() *docker.ContainerDetails {
	return v.details
}

// GetContainerName 获取容器名称
func (v *DetailView) GetContainerName() string {
	return v.containerName
}
