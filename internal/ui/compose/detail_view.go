package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	composelib "docktui/internal/compose"
)

// Tab 索引常量
const (
	tabServices = iota
	tabConfig
	tabLogs
	tabInfo
)

// 布局常量
const (
	detailHeaderHeight   = 2
	detailTabBarHeight   = 2
	detailMinFooterHeight = 1
	detailMaxFooterHeight = 2
	detailMinContentHeight = 8
	detailMinPanelWidth  = 30
)

// DetailView Compose 项目详情视图
type DetailView struct {
	composeClient composelib.Client

	width  int
	height int

	project  *composelib.Project
	services []composelib.Service

	// Config Tab 数据
	envContent      string
	envFileName     string
	ymlContent      string
	ymlFileName     string
	configFocusLeft bool
	envScrollOffset int
	ymlScrollOffset int

	loading    bool
	errorMsg   string
	successMsg string
	currentTab int

	serviceTable table.Model

	operatingService string
	operationType    string
	
	// 操作日志视图
	operationLogView *OperationLogView
	operationStream  *composelib.OperationStream
}

// NewDetailView 创建 Compose 详情视图
func NewDetailView(composeClient composelib.Client) *DetailView {
	t := table.New(
		table.WithColumns([]table.Column{}),
		table.WithFocused(true),
		table.WithHeight(8),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return &DetailView{
		composeClient:    composeClient,
		serviceTable:     t,
		currentTab:       tabServices,
		configFocusLeft:  true,
		operationLogView: NewOperationLogView(),
	}
}

// SetProject 设置要查看的项目
func (v *DetailView) SetProject(project *composelib.Project) {
	v.project = project
	v.services = project.Services
	v.envContent = ""
	v.ymlContent = ""
	v.envFileName = ""
	v.ymlFileName = ""
	v.envScrollOffset = 0
	v.ymlScrollOffset = 0
	v.configFocusLeft = true
	v.errorMsg = ""
	v.successMsg = ""
	v.updateServiceTable()
}

// Init 初始化视图
func (v *DetailView) Init() tea.Cmd {
	if v.project == nil {
		return nil
	}
	v.loading = true
	return tea.Batch(v.refreshServices, v.detectConfigFiles)
}

// detectConfigFiles 检测配置文件
func (v *DetailView) detectConfigFiles() tea.Msg {
	if v.project == nil || v.project.Path == "" {
		return nil
	}

	envNames := []string{".env", ".env.local", "env"}
	for _, name := range envNames {
		envPath := filepath.Join(v.project.Path, name)
		if _, err := os.Stat(envPath); err == nil {
			v.envFileName = name
			break
		}
	}

	if len(v.project.ComposeFiles) == 0 {
		ymlNames := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
		for _, name := range ymlNames {
			ymlPath := filepath.Join(v.project.Path, name)
			if _, err := os.Stat(ymlPath); err == nil {
				v.ymlFileName = name
				break
			}
		}
	} else {
		v.ymlFileName = v.project.ComposeFiles[0]
	}

	return nil
}


// Update 处理消息
func (v *DetailView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case detailServicesMsg:
		v.loading = false
		if msg.err != nil {
			v.errorMsg = fmt.Sprintf("Failed to refresh services: %v", msg.err)
		} else {
			v.services = msg.services
			v.updateServiceTable()
			v.errorMsg = ""
		}
		return nil

	case detailConfigFilesMsg:
		v.loading = false
		if msg.err != nil {
			v.errorMsg = fmt.Sprintf("Failed to load config: %v", msg.err)
		} else {
			v.envContent = msg.envContent
			v.envFileName = msg.envFileName
			v.ymlContent = msg.ymlContent
			v.ymlFileName = msg.ymlFileName
			v.envScrollOffset = 0
			v.ymlScrollOffset = 0
			v.errorMsg = ""
		}
		return nil

	case detailOperationMsg:
		v.operatingService = ""
		v.operationType = ""
		if msg.err != nil {
			v.errorMsg = fmt.Sprintf("Operation failed: %v", msg.err)
			v.successMsg = ""
		} else {
			v.successMsg = msg.message
			v.errorMsg = ""
			return v.refreshServices
		}
		return nil

	case detailOperationLogMsg:
		// 追加日志行
		if v.operationLogView != nil {
			v.operationLogView.AppendLog(msg.line)
		}
		// 继续监听更多日志
		return v.continueListenOperationStream()

	case detailOperationDoneMsg:
		// 操作完成
		if v.operationLogView != nil && msg.result != nil {
			v.operationLogView.SetComplete(msg.result.Success, msg.result.Message)
		}
		v.operatingService = ""
		v.operationType = ""
		v.operationStream = nil
		// 刷新服务列表
		return v.refreshServices

	case detailClearMessageMsg:
		v.successMsg = ""
		v.errorMsg = ""
		return nil

	case tea.KeyMsg:
		// 如果操作日志视图可见，优先处理
		if v.operationLogView != nil && v.operationLogView.IsVisible() {
			if v.operationLogView.Update(msg) {
				// 如果日志视图关闭了，清理状态
				if !v.operationLogView.IsVisible() {
					v.operatingService = ""
					v.operationType = ""
				}
				return nil
			}
		}

		if v.operatingService != "" {
			return nil
		}

		switch msg.String() {
		case "esc":
			return func() tea.Msg { return GoBackMsg{} }

		case "tab":
			v.currentTab = (v.currentTab + 1) % 4
			return v.handleTabChange()

		case "shift+tab":
			if v.currentTab > 0 {
				v.currentTab--
			} else {
				v.currentTab = 3
			}
			return v.handleTabChange()

		case "1":
			v.currentTab = tabServices
			return nil
		case "2":
			v.currentTab = tabConfig
			return v.loadConfigFiles
		case "3":
			v.currentTab = tabLogs
			return nil
		case "4":
			v.currentTab = tabInfo
			return nil

		case "R", "f5":
			v.loading = true
			if v.currentTab == tabConfig {
				return v.loadConfigFiles
			}
			return v.refreshServices

		case "u":
			if v.currentTab == tabServices {
				return v.startServiceOperation("start")
			}
		case "s":
			if v.currentTab == tabServices {
				return v.startServiceOperation("stop")
			}
		case "r":
			if v.currentTab == tabServices {
				return v.startServiceOperation("restart")
			}

		case "U":
			return v.startProjectOperation("up")
		case "D":
			return v.startProjectOperation("down")

		case "enter":
			if v.currentTab == tabServices {
				return v.enterContainerDetail()
			}

		case "L":
			// 查看容器日志
			if v.currentTab == tabServices {
				return v.viewContainerLogs()
			}

		case "S":
			// 进入容器 Shell
			if v.currentTab == tabServices {
				return v.execContainerShell()
			}

		case "h", "left":
			if v.currentTab == tabConfig {
				v.configFocusLeft = true
				return nil
			}
		case "l", "right":
			if v.currentTab == tabServices {
				// 查看容器日志
				return v.viewContainerLogs()
			} else if v.currentTab == tabConfig {
				v.configFocusLeft = false
				return nil
			}

		case "j", "down":
			if v.currentTab == tabServices {
				v.serviceTable.MoveDown(1)
				return nil
			} else if v.currentTab == tabConfig {
				v.scrollCurrentPanel(1)
			}
		case "k", "up":
			if v.currentTab == tabServices {
				v.serviceTable.MoveUp(1)
				return nil
			} else if v.currentTab == tabConfig {
				v.scrollCurrentPanel(-1)
			}
		case "g":
			if v.currentTab == tabServices {
				v.serviceTable.GotoTop()
				return nil
			} else if v.currentTab == tabConfig {
				v.scrollCurrentPanelToStart()
			}
		case "G":
			if v.currentTab == tabServices {
				v.serviceTable.GotoBottom()
				return nil
			} else if v.currentTab == tabConfig {
				v.scrollCurrentPanelToEnd()
			}
		case "ctrl+d", "pgdown":
			if v.currentTab == tabConfig {
				v.scrollCurrentPanel(10)
			}
		case "ctrl+u", "pgup":
			if v.currentTab == tabConfig {
				v.scrollCurrentPanel(-10)
			}
		}
	}

	if v.currentTab == tabServices {
		var cmd tea.Cmd
		v.serviceTable, cmd = v.serviceTable.Update(msg)
		return cmd
	}

	return nil
}

// View 渲染视图
func (v *DetailView) View() string {
	header := v.renderHeader()
	tabBar := v.renderTabBar()
	content := v.renderTabContent()
	footer := v.renderFooter()

	baseView := lipgloss.JoinVertical(lipgloss.Left, header, tabBar, content, footer)

	// 如果操作日志视图可见，叠加显示
	if v.operationLogView != nil && v.operationLogView.IsVisible() {
		return v.operationLogView.Overlay(baseView)
	}

	return baseView
}

// SetSize 设置视图尺寸
func (v *DetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.updateTableColumns()
	if v.operationLogView != nil {
		v.operationLogView.SetSize(width, height)
	}
}

// GetSelectedService 获取选中的服务
func (v *DetailView) GetSelectedService() *composelib.Service {
	if len(v.services) == 0 {
		return nil
	}
	idx := v.serviceTable.Cursor()
	if idx >= 0 && idx < len(v.services) {
		return &v.services[idx]
	}
	return nil
}


// 滚动相关方法
func (v *DetailView) scrollCurrentPanel(delta int) {
	if v.configFocusLeft {
		v.scrollPanel(&v.envScrollOffset, v.envContent, delta)
	} else {
		v.scrollPanel(&v.ymlScrollOffset, v.ymlContent, delta)
	}
}

func (v *DetailView) scrollCurrentPanelToStart() {
	if v.configFocusLeft {
		v.envScrollOffset = 0
	} else {
		v.ymlScrollOffset = 0
	}
}

func (v *DetailView) scrollCurrentPanelToEnd() {
	visibleLines := v.getConfigPanelVisibleLines()
	if v.configFocusLeft {
		v.scrollPanelToEnd(&v.envScrollOffset, v.envContent, visibleLines)
	} else {
		v.scrollPanelToEnd(&v.ymlScrollOffset, v.ymlContent, visibleLines)
	}
}

func (v *DetailView) scrollPanel(offset *int, content string, delta int) {
	if content == "" {
		return
	}
	lines := strings.Split(content, "\n")
	visibleLines := v.getConfigPanelVisibleLines()
	maxOffset := len(lines) - visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}

	*offset += delta
	if *offset < 0 {
		*offset = 0
	}
	if *offset > maxOffset {
		*offset = maxOffset
	}
}

func (v *DetailView) scrollPanelToEnd(offset *int, content string, visibleLines int) {
	if content == "" {
		return
	}
	lines := strings.Split(content, "\n")
	maxOffset := len(lines) - visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}
	*offset = maxOffset
}

func (v *DetailView) getConfigPanelVisibleLines() int {
	contentHeight := v.getContentHeight()
	visibleLines := contentHeight - 4
	if visibleLines < 3 {
		visibleLines = 3
	}
	return visibleLines
}

func (v *DetailView) getContentHeight() int {
	msgHeight := 0
	if v.errorMsg != "" || v.successMsg != "" {
		msgHeight = 1
	}
	footerHeight := v.getFooterHeight()

	contentHeight := v.height - detailHeaderHeight - detailTabBarHeight - footerHeight - msgHeight
	if contentHeight < detailMinContentHeight {
		contentHeight = detailMinContentHeight
	}
	return contentHeight
}

func (v *DetailView) getFooterHeight() int {
	if v.currentTab == tabServices && v.width >= 60 {
		return detailMaxFooterHeight
	}
	return detailMinFooterHeight
}

// 渲染方法
func (v *DetailView) renderHeader() string {
	if v.project == nil {
		return HeaderStyle.Width(v.width).Render("🧩 Compose Project Details")
	}

	var statusStyle lipgloss.Style
	var statusText string
	switch v.project.Status {
	case composelib.StatusRunning:
		statusStyle = StatusRunningStyle
		statusText = "● Running"
	case composelib.StatusPartial:
		statusStyle = StatusPartialStyle
		statusText = "◐ Partial"
	case composelib.StatusStopped:
		statusStyle = StatusStoppedStyle
		statusText = "○ Stopped"
	default:
		statusStyle = StatusErrorStyle
		statusText = "✗ Error"
	}

	title := fmt.Sprintf("🧩 %s", v.project.Name)
	status := statusStyle.Render(statusText)

	runningCount := 0
	for _, svc := range v.services {
		if svc.State == "running" || svc.Running > 0 {
			runningCount++
		}
	}
	stats := fmt.Sprintf("Services: %d/%d", runningCount, len(v.services))

	var headerContent string
	if v.width >= 60 {
		headerContent = fmt.Sprintf(" %s  │  %s  │  %s ", title, status, stats)
	} else {
		headerContent = fmt.Sprintf(" %s  %s ", title, status)
	}
	return HeaderStyle.Width(v.width).Render(headerContent)
}

func (v *DetailView) renderTabBar() string {
	var tabs []string
	if v.width >= 70 {
		tabs = []string{"Services", "Config", "Logs", "Info"}
	} else {
		tabs = []string{"Svc", "Cfg", "Log", "Info"}
	}

	var parts []string
	for i, tab := range tabs {
		tabNum := fmt.Sprintf("[%d]", i+1)
		if i == v.currentTab {
			parts = append(parts, TabActiveStyle.Render(tabNum+" "+tab))
		} else {
			parts = append(parts, TabInactiveStyle.Render(tabNum+" "+tab))
		}
	}

	separator := "  │  "
	if v.width < 60 {
		separator = " │ "
	}

	tabLine := " " + strings.Join(parts, separator)
	lineWidth := v.width - 2
	if lineWidth < 20 {
		lineWidth = 20
	}
	line := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("─", lineWidth))

	return tabLine + "\n " + line
}

func (v *DetailView) renderTabContent() string {
	contentHeight := v.getContentHeight()

	var msgArea string
	if v.errorMsg != "" {
		msgArea = ErrorStyle.Render("❌ " + v.errorMsg) + "\n"
	} else if v.successMsg != "" {
		msgArea = SuccessStyle.Render("✅ " + v.successMsg) + "\n"
	}

	if v.loading {
		return msgArea + v.renderCentered("🔄 Loading...", contentHeight)
	}

	if v.operatingService != "" {
		opMsg := fmt.Sprintf("⏳ Executing %s: %s...", v.operationType, v.operatingService)
		return msgArea + v.renderCentered(opMsg, contentHeight)
	}

	var content string
	switch v.currentTab {
	case tabServices:
		content = v.renderServicesTab(contentHeight)
	case tabConfig:
		content = v.renderConfigTab(contentHeight)
	case tabLogs:
		content = v.renderLogsTab(contentHeight)
	case tabInfo:
		content = v.renderInfoTab(contentHeight)
	}

	return msgArea + content
}

func (v *DetailView) renderServicesTab(contentHeight int) string {
	if len(v.services) == 0 {
		return v.renderCentered("📭 No service info", contentHeight)
	}

	tableHeight := contentHeight - 1
	if tableHeight < 3 {
		tableHeight = 3
	}
	v.serviceTable.SetHeight(tableHeight)

	return "\n" + v.serviceTable.View()
}


func (v *DetailView) renderConfigTab(contentHeight int) string {
	if v.width < 80 {
		return v.renderConfigTabNarrow(contentHeight)
	}
	return v.renderConfigTabWide(contentHeight)
}

func (v *DetailView) renderConfigTabWide(contentHeight int) string {
	totalWidth := v.width - 4
	panelWidth := (totalWidth - 2) / 2
	if panelWidth < detailMinPanelWidth {
		panelWidth = detailMinPanelWidth
	}

	visibleLines := v.getConfigPanelVisibleLines()

	leftPanel := v.renderConfigPanel(
		v.envFileName, v.envContent, v.envScrollOffset,
		panelWidth, visibleLines, v.configFocusLeft, "No env file",
	)

	rightPanel := v.renderConfigPanel(
		v.ymlFileName, v.ymlContent, v.ymlScrollOffset,
		panelWidth, visibleLines, !v.configFocusLeft, "No Compose file",
	)

	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
	hint := ConfigHintStyle.Render(" h/l=Switch panel  j/k=Scroll  g/G=Top/Bottom")

	return "\n" + combined + "\n" + hint
}

func (v *DetailView) renderConfigTabNarrow(contentHeight int) string {
	panelWidth := v.width - 4
	if panelWidth < 30 {
		panelWidth = 30
	}

	visibleLines := v.getConfigPanelVisibleLines()

	var envTab, ymlTab string
	if v.configFocusLeft {
		envTab = TabActiveStyle.Render("[" + v.getDisplayFileName(v.envFileName, ".env") + "]")
		ymlTab = TabInactiveStyle.Render("[" + v.getDisplayFileName(v.ymlFileName, "compose.yml") + "]")
	} else {
		envTab = TabInactiveStyle.Render("[" + v.getDisplayFileName(v.envFileName, ".env") + "]")
		ymlTab = TabActiveStyle.Render("[" + v.getDisplayFileName(v.ymlFileName, "compose.yml") + "]")
	}
	fileTabs := " " + envTab + "  " + ymlTab

	var content, fileName, emptyMsg string
	var scrollOffset int
	if v.configFocusLeft {
		content = v.envContent
		fileName = v.envFileName
		scrollOffset = v.envScrollOffset
		emptyMsg = "No env file"
	} else {
		content = v.ymlContent
		fileName = v.ymlFileName
		scrollOffset = v.ymlScrollOffset
		emptyMsg = "No Compose file"
	}

	panel := v.renderConfigPanel(fileName, content, scrollOffset, panelWidth, visibleLines, true, emptyMsg)
	hint := ConfigHintStyle.Render(" h/l=Switch file  j/k=Scroll")

	return "\n" + fileTabs + "\n" + panel + "\n" + hint
}

func (v *DetailView) renderConfigPanel(fileName, content string, scrollOffset, width, visibleLines int, focused bool, emptyMsg string) string {
	boxStyle := BoxStyle
	if focused {
		boxStyle = BoxFocusedStyle
	}

	displayName := v.getDisplayFileName(fileName, "")
	if displayName == "" {
		displayName = emptyMsg
	}
	title := ConfigTitleStyle.Render("─ " + displayName + " ")

	contentWidth := width - 4

	if content == "" {
		emptyContent := ConfigHintStyle.Render(emptyMsg)
		innerHeight := visibleLines
		if innerHeight < 1 {
			innerHeight = 1
		}
		box := boxStyle.Width(width).Height(innerHeight + 2).Render(emptyContent)
		return title + "\n" + box
	}

	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	startLine := scrollOffset
	endLine := startLine + visibleLines
	if endLine > totalLines {
		endLine = totalLines
	}
	if startLine > endLine {
		startLine = endLine
	}

	var visibleContent string
	if startLine < len(lines) {
		visibleContent = strings.Join(lines[startLine:endLine], "\n")
	}

	visibleContent = v.truncateLines(visibleContent, contentWidth)

	var scrollInfo string
	if totalLines > visibleLines {
		scrollInfo = fmt.Sprintf(" [%d-%d/%d]", startLine+1, endLine, totalLines)
	}

	styledContent := ConfigContentStyle.Width(contentWidth).Render(visibleContent)
	box := boxStyle.Width(width).Render(styledContent)

	return title + scrollInfo + "\n" + box
}

func (v *DetailView) truncateLines(content string, maxWidth int) string {
	if maxWidth < 10 {
		maxWidth = 10
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if len(line) > maxWidth {
			lines[i] = line[:maxWidth-3] + "..."
		}
	}
	return strings.Join(lines, "\n")
}

func (v *DetailView) getDisplayFileName(fileName, defaultName string) string {
	if fileName == "" {
		return defaultName
	}
	return filepath.Base(fileName)
}

func (v *DetailView) renderLogsTab(contentHeight int) string {
	return v.renderCentered("📜 Log feature in development...\n\nTip: Use docker compose logs to view logs", contentHeight)
}

func (v *DetailView) renderInfoTab(contentHeight int) string {
	if v.project == nil {
		return v.renderCentered("No project info", contentHeight)
	}

	boxWidth := v.width - 4
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 90 {
		boxWidth = 90
	}

	labelWidth := 14
	if v.width < 70 {
		labelWidth = 10
	}

	labelStyle := LabelStyle.Width(labelWidth)

	row := func(label, value string) string {
		maxValueLen := boxWidth - labelWidth - 6
		if maxValueLen < 20 {
			maxValueLen = 20
		}
		if len(value) > maxValueLen {
			value = value[:maxValueLen-3] + "..."
		}
		return labelStyle.Render(label) + ValueStyle.Render(value)
	}

	composeFiles := "-"
	if v.ymlFileName != "" {
		composeFiles = v.ymlFileName
	} else if len(v.project.ComposeFiles) > 0 {
		composeFiles = strings.Join(v.project.ComposeFiles, ", ")
	}

	envFiles := "-"
	if v.envFileName != "" {
		envFiles = v.envFileName
	} else if len(v.project.EnvFiles) > 0 {
		envFiles = strings.Join(v.project.EnvFiles, ", ")
	}

	path := v.project.Path
	if path == "" {
		path = "-"
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		row("Project Name", v.project.Name),
		row("Project Path", path),
		row("Compose File", composeFiles),
		row("Env File", envFiles),
		row("Service Count", fmt.Sprintf("%d", len(v.services))),
		row("Status", v.project.Status.String()),
	)

	infoBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)

	return "\n" + lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(
		infoBoxStyle.Width(boxWidth).Render(content),
	)
}


func (v *DetailView) renderFooter() string {
	if v.width >= 100 {
		return v.renderFooterWide()
	} else if v.width >= 70 {
		return v.renderFooterMedium()
	}
	return v.renderFooterNarrow()
}

func (v *DetailView) renderFooterWide() string {
	var line1, line2 string

	if v.currentTab == tabServices {
		line1Keys := []string{
			FooterKeyStyle.Render("u") + "=Start",
			FooterKeyStyle.Render("s") + "=Stop",
			FooterKeyStyle.Render("r") + "=Restart",
			FooterKeyStyle.Render("l") + "=Logs",
			FooterKeyStyle.Render("S") + "=Shell",
			FooterKeyStyle.Render("Enter") + "=Details",
		}
		line1 = " Service: " + strings.Join(line1Keys, "  ")
	}

	line2Keys := []string{
		FooterKeyStyle.Render("U") + "=Start project",
		FooterKeyStyle.Render("D") + "=Stop project",
		FooterKeyStyle.Render("1-4") + "=Tabs",
		FooterKeyStyle.Render("R") + "=Refresh",
		FooterKeyStyle.Render("Esc") + "=Back",
	}
	line2 = " Project: " + strings.Join(line2Keys, "  ")

	footer := FooterStyle.Width(v.width).Render(line2)
	if line1 != "" {
		footer = FooterStyle.Width(v.width).Render(line1) + "\n" + footer
	}
	return footer
}

func (v *DetailView) renderFooterMedium() string {
	var keys []string

	if v.currentTab == tabServices {
		keys = []string{
			FooterKeyStyle.Render("u/s/r") + "=Service",
			FooterKeyStyle.Render("l") + "=Logs",
			FooterKeyStyle.Render("S") + "=Shell",
			FooterKeyStyle.Render("U/D") + "=Project",
			FooterKeyStyle.Render("Esc") + "=Back",
		}
	} else {
		keys = []string{
			FooterKeyStyle.Render("U/D") + "=Project ops",
			FooterKeyStyle.Render("1-4") + "=Tabs",
			FooterKeyStyle.Render("R") + "=Refresh",
			FooterKeyStyle.Render("Esc") + "=Back",
		}
	}

	return FooterStyle.Width(v.width).Render(" " + strings.Join(keys, "  "))
}

func (v *DetailView) renderFooterNarrow() string {
	keys := []string{
		FooterKeyStyle.Render("1-4") + "=Tab",
		FooterKeyStyle.Render("Esc") + "=Back",
	}
	return FooterStyle.Width(v.width).Render(" " + strings.Join(keys, " "))
}

func (v *DetailView) renderCentered(msg string, height int) string {
	style := lipgloss.NewStyle().
		Width(v.width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(lipgloss.Color("245"))
	return style.Render(msg)
}

// 表格相关方法
func (v *DetailView) updateServiceTable() {
	v.updateTableColumns()

	rows := make([]table.Row, len(v.services))
	for i, svc := range v.services {
		var status string
		switch svc.State {
		case "running":
			status = "● Running"
		case "exited":
			status = "○ Stopped"
		case "partial":
			status = "◐ Partial"
		case "paused":
			status = "❚❚ Paused"
		default:
			status = "? " + svc.State
		}

		replicas := fmt.Sprintf("%d/%d", svc.Running, svc.Replicas)

		image := svc.Image
		imageWidth := v.getImageColumnWidth()
		if len(image) > imageWidth-3 && imageWidth > 6 {
			image = image[:imageWidth-6] + "..."
		}

		rows[i] = table.Row{svc.Name, status, replicas, image}
	}
	v.serviceTable.SetRows(rows)
}

func (v *DetailView) getImageColumnWidth() int {
	nameWidth, statusWidth, replicasWidth := v.getColumnWidths()
	imageWidth := v.width - nameWidth - statusWidth - replicasWidth - 8
	if imageWidth < 15 {
		imageWidth = 15
	}
	return imageWidth
}

func (v *DetailView) getColumnWidths() (nameWidth, statusWidth, replicasWidth int) {
	if v.width >= 120 {
		return 25, 12, 8
	} else if v.width >= 100 {
		return 22, 10, 8
	} else if v.width >= 80 {
		return 18, 10, 6
	} else if v.width >= 60 {
		return 15, 8, 6
	}
	return 12, 8, 5
}

func (v *DetailView) updateTableColumns() {
	nameWidth, statusWidth, replicasWidth := v.getColumnWidths()
	imageWidth := v.getImageColumnWidth()

	columns := []table.Column{
		{Title: "Service Name", Width: nameWidth},
		{Title: "Status", Width: statusWidth},
		{Title: "Replicas", Width: replicasWidth},
		{Title: "Image", Width: imageWidth},
	}
	v.serviceTable.SetColumns(columns)
}

func (v *DetailView) handleTabChange() tea.Cmd {
	if v.currentTab == tabConfig && v.ymlContent == "" {
		return v.loadConfigFiles
	}
	return nil
}

// 数据加载方法
func (v *DetailView) refreshServices() tea.Msg {
	if v.composeClient == nil || v.project == nil {
		return detailServicesMsg{err: fmt.Errorf("client or project not initialized")}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx

	services, err := v.composeClient.PS(v.project)
	if err != nil {
		return detailServicesMsg{err: err}
	}

	return detailServicesMsg{services: services}
}

func (v *DetailView) loadConfigFiles() tea.Msg {
	if v.project == nil {
		return detailConfigFilesMsg{err: fmt.Errorf("project not initialized")}
	}

	result := detailConfigFilesMsg{}

	// 读取 env 文件
	if len(v.project.EnvFiles) > 0 {
		envFile := v.project.EnvFiles[0]
		envPath := envFile
		if !filepath.IsAbs(envFile) && v.project.Path != "" {
			envPath = filepath.Join(v.project.Path, envFile)
		}
		if content, err := os.ReadFile(envPath); err == nil {
			result.envContent = string(content)
			result.envFileName = envFile
		}
	} else if v.project.Path != "" {
		defaultEnvPath := filepath.Join(v.project.Path, ".env")
		if content, err := os.ReadFile(defaultEnvPath); err == nil {
			result.envContent = string(content)
			result.envFileName = ".env"
		}
	}

	// 读取 compose yml 文件
	if len(v.project.ComposeFiles) > 0 {
		ymlFile := v.project.ComposeFiles[0]
		ymlPath := ymlFile
		if !filepath.IsAbs(ymlFile) && v.project.Path != "" {
			ymlPath = filepath.Join(v.project.Path, ymlFile)
		}
		if content, err := os.ReadFile(ymlPath); err == nil {
			result.ymlContent = string(content)
			result.ymlFileName = ymlFile
		}
	} else if v.project.Path != "" {
		defaultNames := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
		for _, name := range defaultNames {
			ymlPath := filepath.Join(v.project.Path, name)
			if content, err := os.ReadFile(ymlPath); err == nil {
				result.ymlContent = string(content)
				result.ymlFileName = name
				break
			}
		}
	}

	return result
}

// 操作方法
func (v *DetailView) startServiceOperation(opType string) tea.Cmd {
	svc := v.GetSelectedService()
	if svc == nil {
		v.errorMsg = "Please select a service first"
		return v.clearMessageAfter(3)
	}

	v.operatingService = svc.Name
	v.operationType = opType
	v.errorMsg = ""
	v.successMsg = ""

	// 显示操作日志视图
	opNames := map[string]string{"start": "Starting", "stop": "Stopping", "restart": "Restarting"}
	title := opNames[opType]
	if title == "" {
		title = opType
	}
	title = title + " Service: " + svc.Name

	if v.operationLogView != nil {
		v.operationLogView.SetSize(v.width, v.height)
		v.operationLogView.Show(title)
	}

	return v.executeServiceOperationStream(svc.Name, opType)
}

func (v *DetailView) startProjectOperation(opType string) tea.Cmd {
	if v.project == nil {
		v.errorMsg = "Project not initialized"
		return v.clearMessageAfter(3)
	}

	v.operatingService = v.project.Name
	v.operationType = opType
	v.errorMsg = ""
	v.successMsg = ""

	// 显示操作日志视图
	opNames := map[string]string{"up": "Starting Project", "down": "Stopping Project"}
	title := opNames[opType]
	if title == "" {
		title = opType + " Project"
	}
	title = title + ": " + v.project.Name

	if v.operationLogView != nil {
		v.operationLogView.SetSize(v.width, v.height)
		v.operationLogView.Show(title)
	}

	return v.executeProjectOperationStream(opType)
}

func (v *DetailView) executeServiceOperation(serviceName, opType string) tea.Cmd {
	return func() tea.Msg {
		if v.composeClient == nil {
			return detailOperationMsg{err: fmt.Errorf("client not initialized")}
		}

		var result *composelib.OperationResult
		var err error
		services := []string{serviceName}

		switch opType {
		case "start":
			result, err = v.composeClient.Start(v.project, services)
		case "stop":
			result, err = v.composeClient.Stop(v.project, services, 10)
		case "restart":
			result, err = v.composeClient.Restart(v.project, services, 10)
		default:
			return detailOperationMsg{err: fmt.Errorf("unknown operation: %s", opType)}
		}

		if err != nil {
			return detailOperationMsg{err: err}
		}
		if result != nil && !result.Success {
			return detailOperationMsg{err: fmt.Errorf(result.Message)}
		}

		opNames := map[string]string{"start": "Start", "stop": "Stop", "restart": "Restart"}
		return detailOperationMsg{message: fmt.Sprintf("%s service %s succeeded", opNames[opType], serviceName)}
	}
}

// executeServiceOperationStream 流式执行服务操作
func (v *DetailView) executeServiceOperationStream(serviceName, opType string) tea.Cmd {
	wrapper, ok := v.composeClient.(*composelib.ComposeClientWrapper)
	if !ok {
		// 回退到非流式方法
		return v.executeServiceOperation(serviceName, opType)
	}

	services := []string{serviceName}
	var stream *composelib.OperationStream

	switch opType {
	case "start":
		stream = wrapper.StartStream(v.project, services)
	case "stop":
		stream = wrapper.StopStream(v.project, services, 10)
	case "restart":
		stream = wrapper.RestartStream(v.project, services, 10)
	default:
		v.errorMsg = "Unknown operation: " + opType
		return nil
	}

	v.operationStream = stream
	return v.listenOperationStream()
}

func (v *DetailView) executeProjectOperation(opType string) tea.Cmd {
	return func() tea.Msg {
		if v.composeClient == nil {
			return detailOperationMsg{err: fmt.Errorf("client not initialized")}
		}

		var result *composelib.OperationResult
		var err error

		switch opType {
		case "up":
			result, err = v.composeClient.Up(v.project, composelib.UpOptions{Detach: true})
		case "down":
			result, err = v.composeClient.Down(v.project, composelib.DownOptions{})
		default:
			return detailOperationMsg{err: fmt.Errorf("unknown operation: %s", opType)}
		}

		if err != nil {
			return detailOperationMsg{err: err}
		}
		if result != nil && !result.Success {
			return detailOperationMsg{err: fmt.Errorf(result.Message)}
		}

		opNames := map[string]string{"up": "Start", "down": "Stop"}
		return detailOperationMsg{message: fmt.Sprintf("%s project succeeded", opNames[opType])}
	}
}

// executeProjectOperationStream 流式执行项目操作
func (v *DetailView) executeProjectOperationStream(opType string) tea.Cmd {
	// 获取 composeClient 的底层类型以调用流式方法
	wrapper, ok := v.composeClient.(*composelib.ComposeClientWrapper)
	if !ok {
		// 回退到非流式方法
		return v.executeProjectOperation(opType)
	}

	var stream *composelib.OperationStream
	switch opType {
	case "up":
		stream = wrapper.UpStream(v.project, composelib.UpOptions{Detach: true})
	case "down":
		stream = wrapper.DownStream(v.project, composelib.DownOptions{})
	default:
		v.errorMsg = "Unknown operation: " + opType
		return nil
	}

	v.operationStream = stream

	// 返回一个命令来监听日志
	return v.listenOperationStream()
}

// listenOperationStream 监听操作流
func (v *DetailView) listenOperationStream() tea.Cmd {
	if v.operationStream == nil {
		return nil
	}

	return func() tea.Msg {
		select {
		case line, ok := <-v.operationStream.LogChan:
			if ok {
				return detailOperationLogMsg{line: line}
			}
			// LogChan 关闭，等待完成
			return nil
		case result, ok := <-v.operationStream.DoneChan:
			if ok {
				return detailOperationDoneMsg{result: result}
			}
			return nil
		}
	}
}

// continueListenOperationStream 继续监听操作流
func (v *DetailView) continueListenOperationStream() tea.Cmd {
	if v.operationStream == nil {
		return nil
	}
	return v.listenOperationStream()
}

func (v *DetailView) enterContainerDetail() tea.Cmd {
	svc := v.GetSelectedService()
	if svc == nil || len(svc.Containers) == 0 {
		v.errorMsg = "This service has no running containers"
		return v.clearMessageAfter(3)
	}

	containerID := svc.Containers[0]
	return func() tea.Msg {
		return GoToContainerDetailMsg{
			ContainerID:   containerID,
			ContainerName: svc.Name,
		}
	}
}

func (v *DetailView) viewContainerLogs() tea.Cmd {
	svc := v.GetSelectedService()
	if svc == nil || len(svc.Containers) == 0 {
		v.errorMsg = "This service has no running containers"
		return v.clearMessageAfter(3)
	}

	containerID := svc.Containers[0]
	return func() tea.Msg {
		return GoToContainerLogsMsg{
			ContainerID:   containerID,
			ContainerName: svc.Name,
		}
	}
}

func (v *DetailView) execContainerShell() tea.Cmd {
	svc := v.GetSelectedService()
	if svc == nil || len(svc.Containers) == 0 {
		v.errorMsg = "This service has no running containers"
		return v.clearMessageAfter(3)
	}

	// 只有运行中的容器才能执行 shell
	if svc.State != "running" && svc.Running == 0 {
		v.errorMsg = "Can only execute shell in running containers"
		return v.clearMessageAfter(3)
	}

	containerID := svc.Containers[0]
	return func() tea.Msg {
		return ExecContainerShellMsg{
			ContainerID:   containerID,
			ContainerName: svc.Name,
		}
	}
}

func (v *DetailView) clearMessageAfter(seconds int) tea.Cmd {
	return tea.Tick(time.Duration(seconds)*time.Second, func(t time.Time) tea.Msg {
		return detailClearMessageMsg{}
	})
}
