package compose

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	sdk "github.com/docker/docker/client"

	composelib "docktui/internal/compose"
)

// ListView Compose 项目列表视图
type ListView struct {
	composeClient composelib.Client
	discovery     *composelib.Discovery

	width  int
	height int

	projects   []*composelib.Project
	tableModel table.Model
	loading    bool
	errorMsg   string
	successMsg string

	operatingProject *composelib.Project
	operationType    string

	lastRefreshTime time.Time
	autoRefresh     bool

	// 操作日志视图
	operationLogView *OperationLogView
	operationStream  *composelib.OperationStream
}

// NewListView 创建 Compose 列表视图
func NewListView(composeClient composelib.Client, dockerCli *sdk.Client) *ListView {
	var discovery *composelib.Discovery
	if dockerCli != nil {
		discovery = composelib.NewDiscovery(dockerCli)
	}

	columns := []table.Column{
		{Title: "Project Name", Width: 20},
		{Title: "Status", Width: 10},
		{Title: "Services", Width: 10},
		{Title: "Path", Width: 40},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
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

	return &ListView{
		composeClient:    composeClient,
		discovery:        discovery,
		tableModel:       t,
		loading:          false,
		autoRefresh:      false,
		operationLogView: NewOperationLogView(),
	}
}

// Init 初始化视图
func (v *ListView) Init() tea.Cmd {
	v.loading = true
	return v.discoverProjects
}

// Update 处理消息并更新视图状态
func (v *ListView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case listScanResultMsg:
		v.loading = false
		if msg.err != nil {
			v.errorMsg = fmt.Sprintf("Failed to discover projects: %v", msg.err)
		} else {
			v.projects = msg.projects
			v.errorMsg = ""
			v.updateTable()
		}
		v.lastRefreshTime = time.Now()
		return nil

	case listOperationResultMsg:
		v.operatingProject = nil
		v.operationType = ""
		if msg.err != nil {
			v.errorMsg = fmt.Sprintf("Operation failed: %v", msg.err)
			v.successMsg = ""
		} else {
			v.successMsg = msg.message
			v.errorMsg = ""
			return v.refreshProjectStatus
		}
		return nil

	case listRefreshStatusMsg:
		v.projects = msg.projects
		v.updateTable()
		return nil

	case listClearMessageMsg:
		v.successMsg = ""
		v.errorMsg = ""
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
		v.operatingProject = nil
		v.operationType = ""
		v.operationStream = nil
		// 刷新项目状态
		return v.refreshProjectStatus

	case tea.KeyMsg:
		// 如果操作日志视图可见，优先处理
		if v.operationLogView != nil && v.operationLogView.IsVisible() {
			if v.operationLogView.Update(msg) {
				// 如果日志视图关闭了，清理状态
				if !v.operationLogView.IsVisible() {
					v.operatingProject = nil
					v.operationType = ""
				}
				return nil
			}
		}

		if v.operatingProject != nil {
			return nil
		}

		switch msg.String() {
		case "esc":
			return func() tea.Msg { return GoBackMsg{} }
		case "j", "down":
			v.tableModel.MoveDown(1)
			return nil
		case "k", "up":
			v.tableModel.MoveUp(1)
			return nil
		case "g":
			v.tableModel.GotoTop()
			return nil
		case "G":
			v.tableModel.GotoBottom()
			return nil
		case "u":
			return v.startOperation("up")
		case "d":
			return v.startOperation("down")
		case "r":
			return v.startOperation("restart")
		case "s":
			return v.startOperation("stop")
		case "t":
			return v.startOperation("start")
		case "R", "f5":
			v.loading = true
			return v.discoverProjects
		case "l":
			v.successMsg = "📜 Log feature in development..."
			return v.clearMessageAfter(3)
		case "enter":
			project := v.GetSelectedProject()
			if project != nil {
				return func() tea.Msg {
					return GoToDetailMsg{Project: project}
				}
			}
		}
	}

	var cmd tea.Cmd
	v.tableModel, cmd = v.tableModel.Update(msg)
	return cmd
}


// View 渲染视图
func (v *ListView) View() string {
	headerHeight := 1
	footerHeight := 3
	messageHeight := 0
	if v.errorMsg != "" || v.successMsg != "" {
		messageHeight = 1
	}

	tableHeight := v.height - headerHeight - footerHeight - messageHeight - 2
	if tableHeight < 5 {
		tableHeight = 5
	}
	v.tableModel.SetHeight(tableHeight)

	header := v.renderHeader()
	content := v.renderContent()
	footer := v.renderFooter()

	baseView := lipgloss.JoinVertical(lipgloss.Left, header, content, footer)

	// 如果操作日志视图可见，叠加显示
	if v.operationLogView != nil && v.operationLogView.IsVisible() {
		return v.operationLogView.Overlay(baseView)
	}

	return baseView
}

// SetSize 设置视图尺寸
func (v *ListView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.updateTableColumns()
	if v.operationLogView != nil {
		v.operationLogView.SetSize(width, height)
	}
}

// GetSelectedProject 获取当前选中的项目
func (v *ListView) GetSelectedProject() *composelib.Project {
	if len(v.projects) == 0 {
		return nil
	}
	idx := v.tableModel.Cursor()
	if idx >= 0 && idx < len(v.projects) {
		return v.projects[idx]
	}
	return nil
}

func (v *ListView) renderHeader() string {
	title := "🧩 Docker Compose Projects"

	runningCount := 0
	for _, p := range v.projects {
		if p.Status == composelib.StatusRunning {
			runningCount++
		}
	}
	stats := fmt.Sprintf("Total %d projects, %d running", len(v.projects), runningCount)

	var refreshInfo string
	if !v.lastRefreshTime.IsZero() {
		refreshInfo = fmt.Sprintf("Last refresh: %s", v.lastRefreshTime.Format("15:04:05"))
	}

	headerContent := fmt.Sprintf(" %s  │  %s  │  %s ", title, stats, refreshInfo)
	return HeaderStyle.Width(v.width).Render(headerContent)
}

func (v *ListView) renderContent() string {
	var content strings.Builder

	if v.errorMsg != "" {
		content.WriteString(ErrorStyle.Render("❌ " + v.errorMsg))
		content.WriteString("\n")
	}
	if v.successMsg != "" {
		content.WriteString(SuccessStyle.Render("✅ " + v.successMsg))
		content.WriteString("\n")
	}

	if v.loading {
		loadingMsg := LoadingStyle.Render("🔄 Discovering Compose projects...")
		centered := lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(loadingMsg)
		content.WriteString("\n\n")
		content.WriteString(centered)
		return content.String()
	}

	if v.operatingProject != nil {
		opMsg := LoadingStyle.Render(fmt.Sprintf("⏳ Executing %s: %s...", v.operationType, v.operatingProject.Name))
		centered := lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(opMsg)
		content.WriteString("\n")
		content.WriteString(centered)
		content.WriteString("\n\n")
	}

	if len(v.projects) == 0 && !v.loading {
		emptyMsg := EmptyStyle.Render("📭 No running Compose projects found\n\nTip: Please start a project with docker compose up -d first")
		centered := lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(emptyMsg)
		content.WriteString("\n\n")
		content.WriteString(centered)
		return content.String()
	}

	content.WriteString(v.tableModel.View())
	return content.String()
}

func (v *ListView) renderFooter() string {
	line1Keys := []string{
		FooterKeyStyle.Render("u") + "=Start",
		FooterKeyStyle.Render("d") + "=Stop",
		FooterKeyStyle.Render("r") + "=Restart",
		FooterKeyStyle.Render("s") + "=Pause",
		FooterKeyStyle.Render("t") + "=Resume",
	}
	line1 := " Ops: " + strings.Join(line1Keys, "  ")

	line2Keys := []string{
		FooterKeyStyle.Render("l") + "=Logs",
		FooterKeyStyle.Render("R") + "=Refresh",
		FooterKeyStyle.Render("Enter") + "=Details",
	}
	line2 := " View: " + strings.Join(line2Keys, "  ")

	line3Keys := []string{
		FooterKeyStyle.Render("j/k") + "=Up/Down",
		FooterKeyStyle.Render("g/G") + "=Top/Bottom",
		FooterKeyStyle.Render("Esc") + "=Back",
		FooterKeyStyle.Render("q") + "=Quit",
	}
	line3 := " Nav: " + strings.Join(line3Keys, "  ")

	return lipgloss.JoinVertical(lipgloss.Left,
		FooterStyle.Width(v.width).Render(line1),
		FooterStyle.Width(v.width).Render(line2),
		FooterStyle.Width(v.width).Render(line3),
	)
}

func (v *ListView) updateTable() {
	rows := make([]table.Row, len(v.projects))
	for i, p := range v.projects {
		var status string
		switch p.Status {
		case composelib.StatusRunning:
			status = "● Running"
		case composelib.StatusPartial:
			status = "◐ Partial"
		case composelib.StatusStopped:
			status = "○ Stopped"
		case composelib.StatusError:
			status = "✗ Error"
		default:
			status = "? Unknown"
		}

		runningServices := 0
		for _, svc := range p.Services {
			if svc.State == "running" || svc.Running > 0 {
				runningServices++
			}
		}
		services := fmt.Sprintf("%d/%d", runningServices, len(p.Services))
		if len(p.Services) == 0 {
			services = "-"
		}

		path := p.Path
		maxPathLen := v.width - 50
		if maxPathLen < 20 {
			maxPathLen = 20
		}
		if len(path) > maxPathLen {
			path = "..." + path[len(path)-maxPathLen+3:]
		}

		rows[i] = table.Row{p.Name, status, services, path}
	}
	v.tableModel.SetRows(rows)
}

func (v *ListView) updateTableColumns() {
	nameWidth := 20
	statusWidth := 10
	servicesWidth := 10
	pathWidth := v.width - nameWidth - statusWidth - servicesWidth - 10

	if pathWidth < 20 {
		pathWidth = 20
	}
	if v.width < 80 {
		nameWidth = 15
		pathWidth = v.width - nameWidth - statusWidth - servicesWidth - 8
	}

	columns := []table.Column{
		{Title: "Project Name", Width: nameWidth},
		{Title: "Status", Width: statusWidth},
		{Title: "Services", Width: servicesWidth},
		{Title: "Path", Width: pathWidth},
	}
	v.tableModel.SetColumns(columns)
}

func (v *ListView) discoverProjects() tea.Msg {
	if v.discovery == nil {
		return listScanResultMsg{err: fmt.Errorf("project discovery not initialized")}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	projects, err := v.discovery.DiscoverProjects(ctx)
	if err != nil {
		return listScanResultMsg{err: err}
	}

	return listScanResultMsg{projects: projects}
}

func (v *ListView) refreshProjectStatus() tea.Msg {
	if v.discovery == nil {
		return listRefreshStatusMsg{projects: v.projects}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projects, err := v.discovery.DiscoverProjects(ctx)
	if err != nil {
		return listRefreshStatusMsg{projects: v.projects}
	}

	return listRefreshStatusMsg{projects: projects}
}

func (v *ListView) startOperation(opType string) tea.Cmd {
	project := v.GetSelectedProject()
	if project == nil {
		v.errorMsg = "Please select a project first"
		return v.clearMessageAfter(3)
	}

	v.operatingProject = project
	v.operationType = opType
	v.errorMsg = ""
	v.successMsg = ""

	// 对于 up 和 down 操作，使用流式执行
	if opType == "up" || opType == "down" {
		// 显示操作日志视图
		opNames := map[string]string{"up": "Starting Project", "down": "Stopping Project"}
		title := opNames[opType] + ": " + project.Name

		if v.operationLogView != nil {
			v.operationLogView.SetSize(v.width, v.height)
			v.operationLogView.Show(title)
		}

		return v.executeOperationStream(project, opType)
	}

	return v.executeOperation(project, opType)
}

func (v *ListView) executeOperation(project *composelib.Project, opType string) tea.Cmd {
	return func() tea.Msg {
		if v.composeClient == nil {
			return listOperationResultMsg{err: fmt.Errorf("Compose client not initialized")}
		}

		var result *composelib.OperationResult
		var err error

		switch opType {
		case "up":
			result, err = v.composeClient.Up(project, composelib.UpOptions{Detach: true})
		case "down":
			result, err = v.composeClient.Down(project, composelib.DownOptions{})
		case "restart":
			result, err = v.composeClient.Restart(project, nil, 10)
		case "stop":
			result, err = v.composeClient.Stop(project, nil, 10)
		case "start":
			result, err = v.composeClient.Start(project, nil)
		default:
			return listOperationResultMsg{err: fmt.Errorf("unknown operation: %s", opType)}
		}

		if err != nil {
			return listOperationResultMsg{err: err}
		}

		if result != nil && !result.Success {
			return listOperationResultMsg{err: fmt.Errorf(result.Message)}
		}

		opNames := map[string]string{
			"up": "Start", "down": "Stop", "restart": "Restart",
			"stop": "Pause", "start": "Resume",
		}
		opName := opNames[opType]
		if opName == "" {
			opName = opType
		}

		return listOperationResultMsg{
			message: fmt.Sprintf("%s project %s succeeded", opName, project.Name),
		}
	}
}

// executeOperationStream 流式执行项目操作
func (v *ListView) executeOperationStream(project *composelib.Project, opType string) tea.Cmd {
	wrapper, ok := v.composeClient.(*composelib.ComposeClientWrapper)
	if !ok {
		// 回退到非流式方法
		return v.executeOperation(project, opType)
	}

	var stream *composelib.OperationStream
	switch opType {
	case "up":
		stream = wrapper.UpStream(project, composelib.UpOptions{Detach: true})
	case "down":
		stream = wrapper.DownStream(project, composelib.DownOptions{})
	default:
		return v.executeOperation(project, opType)
	}

	v.operationStream = stream

	return v.listenOperationStream()
}

// listenOperationStream 监听操作流
func (v *ListView) listenOperationStream() tea.Cmd {
	if v.operationStream == nil {
		return nil
	}

	return func() tea.Msg {
		select {
		case line, ok := <-v.operationStream.LogChan:
			if ok {
				return detailOperationLogMsg{line: line}
			}
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
func (v *ListView) continueListenOperationStream() tea.Cmd {
	if v.operationStream == nil {
		return nil
	}
	return v.listenOperationStream()
}

func (v *ListView) clearMessageAfter(seconds int) tea.Cmd {
	return tea.Tick(time.Duration(seconds)*time.Second, func(t time.Time) tea.Msg {
		return listClearMessageMsg{}
	})
}
