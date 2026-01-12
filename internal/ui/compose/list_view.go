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
}

// NewListView 创建 Compose 列表视图
func NewListView(composeClient composelib.Client, dockerCli *sdk.Client) *ListView {
	var discovery *composelib.Discovery
	if dockerCli != nil {
		discovery = composelib.NewDiscovery(dockerCli)
	}

	columns := []table.Column{
		{Title: "项目名称", Width: 20},
		{Title: "状态", Width: 10},
		{Title: "服务", Width: 10},
		{Title: "路径", Width: 40},
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
		composeClient: composeClient,
		discovery:     discovery,
		tableModel:    t,
		loading:       false,
		autoRefresh:   false,
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
			v.errorMsg = fmt.Sprintf("发现项目失败: %v", msg.err)
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
			v.errorMsg = fmt.Sprintf("操作失败: %v", msg.err)
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

	case tea.KeyMsg:
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
			v.successMsg = "📜 日志功能开发中..."
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

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

// SetSize 设置视图尺寸
func (v *ListView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.updateTableColumns()
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
	title := "🧩 Docker Compose 项目"

	runningCount := 0
	for _, p := range v.projects {
		if p.Status == composelib.StatusRunning {
			runningCount++
		}
	}
	stats := fmt.Sprintf("共 %d 个项目，%d 个运行中", len(v.projects), runningCount)

	var refreshInfo string
	if !v.lastRefreshTime.IsZero() {
		refreshInfo = fmt.Sprintf("上次刷新: %s", v.lastRefreshTime.Format("15:04:05"))
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
		loadingMsg := LoadingStyle.Render("🔄 正在发现 Compose 项目...")
		centered := lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(loadingMsg)
		content.WriteString("\n\n")
		content.WriteString(centered)
		return content.String()
	}

	if v.operatingProject != nil {
		opMsg := LoadingStyle.Render(fmt.Sprintf("⏳ 正在执行 %s: %s...", v.operationType, v.operatingProject.Name))
		centered := lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(opMsg)
		content.WriteString("\n")
		content.WriteString(centered)
		content.WriteString("\n\n")
	}

	if len(v.projects) == 0 && !v.loading {
		emptyMsg := EmptyStyle.Render("📭 未发现运行中的 Compose 项目\n\n提示：请先使用 docker compose up -d 启动项目")
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
		FooterKeyStyle.Render("u") + "=启动",
		FooterKeyStyle.Render("d") + "=停止",
		FooterKeyStyle.Render("r") + "=重启",
		FooterKeyStyle.Render("s") + "=暂停",
		FooterKeyStyle.Render("t") + "=恢复",
	}
	line1 := " 操作：" + strings.Join(line1Keys, "  ")

	line2Keys := []string{
		FooterKeyStyle.Render("l") + "=日志",
		FooterKeyStyle.Render("R") + "=刷新",
		FooterKeyStyle.Render("Enter") + "=详情",
	}
	line2 := " 查看：" + strings.Join(line2Keys, "  ")

	line3Keys := []string{
		FooterKeyStyle.Render("j/k") + "=上下移动",
		FooterKeyStyle.Render("g/G") + "=首/尾",
		FooterKeyStyle.Render("Esc") + "=返回",
		FooterKeyStyle.Render("q") + "=退出",
	}
	line3 := " 导航：" + strings.Join(line3Keys, "  ")

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
			status = "● 运行中"
		case composelib.StatusPartial:
			status = "◐ 部分"
		case composelib.StatusStopped:
			status = "○ 已停止"
		case composelib.StatusError:
			status = "✗ 错误"
		default:
			status = "? 未知"
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
		{Title: "项目名称", Width: nameWidth},
		{Title: "状态", Width: statusWidth},
		{Title: "服务", Width: servicesWidth},
		{Title: "路径", Width: pathWidth},
	}
	v.tableModel.SetColumns(columns)
}

func (v *ListView) discoverProjects() tea.Msg {
	if v.discovery == nil {
		return listScanResultMsg{err: fmt.Errorf("项目发现器未初始化")}
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
		v.errorMsg = "请先选择一个项目"
		return v.clearMessageAfter(3)
	}

	v.operatingProject = project
	v.operationType = opType
	v.errorMsg = ""
	v.successMsg = ""

	return v.executeOperation(project, opType)
}

func (v *ListView) executeOperation(project *composelib.Project, opType string) tea.Cmd {
	return func() tea.Msg {
		if v.composeClient == nil {
			return listOperationResultMsg{err: fmt.Errorf("Compose 客户端未初始化")}
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
			return listOperationResultMsg{err: fmt.Errorf("未知操作: %s", opType)}
		}

		if err != nil {
			return listOperationResultMsg{err: err}
		}

		if result != nil && !result.Success {
			return listOperationResultMsg{err: fmt.Errorf(result.Message)}
		}

		opNames := map[string]string{
			"up": "启动", "down": "停止", "restart": "重启",
			"stop": "暂停", "start": "恢复",
		}
		opName := opNames[opType]
		if opName == "" {
			opName = opType
		}

		return listOperationResultMsg{
			message: fmt.Sprintf("%s 项目 %s 成功", opName, project.Name),
		}
	}
}

func (v *ListView) clearMessageAfter(seconds int) tea.Cmd {
	return tea.Tick(time.Duration(seconds)*time.Second, func(t time.Time) tea.Msg {
		return listClearMessageMsg{}
	})
}
