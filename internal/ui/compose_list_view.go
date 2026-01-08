package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/compose"
)

// ComposeListView Compose 项目列表视图
type ComposeListView struct {
	composeClient compose.Client
	scanner       compose.Scanner

	// UI 尺寸
	width  int
	height int

	// 数据状态
	projects   []compose.Project
	tableModel table.Model
	loading    bool
	errorMsg   string
	successMsg string

	// 扫描路径
	scanPaths []string

	// 操作状态
	operatingProject *compose.Project
	operationType    string

	// 刷新状态
	lastRefreshTime time.Time
	autoRefresh     bool
}

// Compose 列表样式定义 - 不设置背景，由全局 fillBackground 处理
var (
	composeHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Padding(0, 1).
		Bold(true)

	composeStatusRunningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)

	composeStatusPartialStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	composeStatusStoppedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	composeStatusErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	composeFooterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 1)

	composeFooterKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Bold(true)

	composeLoadingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Bold(true)

	composeErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	composeSuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)

	composeEmptyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)
)

// NewComposeListView 创建 Compose 列表视图
func NewComposeListView(composeClient compose.Client, scanPaths []string) *ComposeListView {
	// 创建扫描器
	scanner := compose.NewScanner(composeClient, compose.DefaultScanConfig())

	// 创建表格
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

	// 设置表格样式
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

	return &ComposeListView{
		composeClient: composeClient,
		scanner:       scanner,
		scanPaths:     scanPaths,
		tableModel:    t,
		loading:       false,
		autoRefresh:   false,
	}
}


// Init 初始化视图
func (v *ComposeListView) Init() tea.Cmd {
	v.loading = true
	return v.scanProjects
}

// Update 处理消息并更新视图状态
func (v *ComposeListView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case composeScanResultMsg:
		v.loading = false
		if msg.err != nil {
			v.errorMsg = fmt.Sprintf("扫描失败: %v", msg.err)
		} else {
			v.projects = msg.projects
			v.errorMsg = ""
			v.updateTable()
		}
		v.lastRefreshTime = time.Now()
		return v, nil

	case composeOperationResultMsg:
		v.operatingProject = nil
		v.operationType = ""
		if msg.err != nil {
			v.errorMsg = fmt.Sprintf("操作失败: %v", msg.err)
			v.successMsg = ""
		} else {
			v.successMsg = msg.message
			v.errorMsg = ""
			// 操作成功后刷新项目状态
			return v, v.refreshProjectStatus
		}
		return v, nil

	case composeRefreshStatusMsg:
		// 更新项目状态
		for i, p := range v.projects {
			for _, updated := range msg.projects {
				if p.Path == updated.Path {
					v.projects[i] = updated
					break
				}
			}
		}
		v.updateTable()
		return v, nil

	case clearComposeMessageMsg:
		v.successMsg = ""
		v.errorMsg = ""
		return v, nil

	case tea.KeyMsg:
		// 如果正在操作中，忽略按键
		if v.operatingProject != nil {
			return v, nil
		}

		switch msg.String() {
		case "j", "down":
			v.tableModel.MoveDown(1)
			return v, nil
		case "k", "up":
			v.tableModel.MoveUp(1)
			return v, nil
		case "g":
			v.tableModel.GotoTop()
			return v, nil
		case "G":
			v.tableModel.GotoBottom()
			return v, nil
		case "u":
			// 启动项目 (docker compose up -d)
			return v.startOperation("up")
		case "d":
			// 停止项目 (docker compose down)
			return v.startOperation("down")
		case "r":
			// 重启项目 (docker compose restart)
			return v.startOperation("restart")
		case "s":
			// 停止但不删除 (docker compose stop)
			return v.startOperation("stop")
		case "t":
			// 启动已停止的容器 (docker compose start)
			return v.startOperation("start")
		case "R", "f5":
			// 刷新列表
			v.loading = true
			return v, v.scanProjects
		case "l":
			// 查看日志（TODO: 实现）
			v.successMsg = "📜 日志功能开发中..."
			return v, v.clearMessageAfter(3)
		}
	}

	// 更新表格
	var cmd tea.Cmd
	v.tableModel, cmd = v.tableModel.Update(msg)
	return v, cmd
}

// View 渲染视图
func (v *ComposeListView) View() string {
	// 计算各区域高度
	headerHeight := 1
	footerHeight := 3
	messageHeight := 0
	if v.errorMsg != "" || v.successMsg != "" {
		messageHeight = 1
	}

	// 表格可用高度
	tableHeight := v.height - headerHeight - footerHeight - messageHeight - 2
	if tableHeight < 5 {
		tableHeight = 5
	}
	v.tableModel.SetHeight(tableHeight)

	// 渲染各部分
	header := v.renderHeader()
	content := v.renderContent()
	footer := v.renderFooter()

	// 组合布局
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		footer,
	)
}

// SetSize 设置视图尺寸
func (v *ComposeListView) SetSize(width, height int) {
	v.width = width
	v.height = height

	// 更新表格列宽
	v.updateTableColumns()
}

// renderHeader 渲染顶部标题栏
func (v *ComposeListView) renderHeader() string {
	// 标题
	title := "🧩 Docker Compose 项目"

	// 统计信息
	runningCount := 0
	for _, p := range v.projects {
		if p.Status == compose.StatusRunning {
			runningCount++
		}
	}
	stats := fmt.Sprintf("共 %d 个项目，%d 个运行中", len(v.projects), runningCount)

	// 刷新时间
	var refreshInfo string
	if !v.lastRefreshTime.IsZero() {
		refreshInfo = fmt.Sprintf("上次刷新: %s", v.lastRefreshTime.Format("15:04:05"))
	}

	// 构建标题栏
	headerContent := fmt.Sprintf(" %s  │  %s  │  %s ", title, stats, refreshInfo)

	return composeHeaderStyle.Width(v.width).Render(headerContent)
}

// renderContent 渲染内容区域
func (v *ComposeListView) renderContent() string {
	var content strings.Builder

	// 消息区域
	if v.errorMsg != "" {
		content.WriteString(composeErrorStyle.Render("❌ " + v.errorMsg))
		content.WriteString("\n")
	}
	if v.successMsg != "" {
		content.WriteString(composeSuccessStyle.Render("✅ " + v.successMsg))
		content.WriteString("\n")
	}

	// 加载状态
	if v.loading {
		loadingMsg := composeLoadingStyle.Render("🔄 正在扫描 Compose 项目...")
		centered := lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(loadingMsg)
		content.WriteString("\n\n")
		content.WriteString(centered)
		return content.String()
	}

	// 操作中状态
	if v.operatingProject != nil {
		opMsg := composeLoadingStyle.Render(fmt.Sprintf("⏳ 正在执行 %s: %s...", v.operationType, v.operatingProject.Name))
		centered := lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(opMsg)
		content.WriteString("\n")
		content.WriteString(centered)
		content.WriteString("\n\n")
	}

	// 空状态
	if len(v.projects) == 0 && !v.loading {
		emptyMsg := composeEmptyStyle.Render("📭 未找到 Compose 项目\n\n提示：请确保扫描路径下存在 docker-compose.yml 或 compose.yml 文件")
		centered := lipgloss.NewStyle().Width(v.width).Align(lipgloss.Center).Render(emptyMsg)
		content.WriteString("\n\n")
		content.WriteString(centered)
		return content.String()
	}

	// 表格
	content.WriteString(v.tableModel.View())

	return content.String()
}

// renderFooter 渲染底部操作区
func (v *ComposeListView) renderFooter() string {
	// 第一行：基本操作
	line1Keys := []string{
		composeFooterKeyStyle.Render("u") + "=启动",
		composeFooterKeyStyle.Render("d") + "=停止",
		composeFooterKeyStyle.Render("r") + "=重启",
		composeFooterKeyStyle.Render("s") + "=暂停",
		composeFooterKeyStyle.Render("t") + "=恢复",
	}
	line1 := " 操作：" + strings.Join(line1Keys, "  ")

	// 第二行：其他操作
	line2Keys := []string{
		composeFooterKeyStyle.Render("l") + "=日志",
		composeFooterKeyStyle.Render("R") + "=刷新",
		composeFooterKeyStyle.Render("Enter") + "=详情",
	}
	line2 := " 查看：" + strings.Join(line2Keys, "  ")

	// 第三行：导航
	line3Keys := []string{
		composeFooterKeyStyle.Render("j/k") + "=上下移动",
		composeFooterKeyStyle.Render("g/G") + "=首/尾",
		composeFooterKeyStyle.Render("Esc") + "=返回",
		composeFooterKeyStyle.Render("q") + "=退出",
	}
	line3 := " 导航：" + strings.Join(line3Keys, "  ")

	footer := lipgloss.JoinVertical(lipgloss.Left,
		composeFooterStyle.Width(v.width).Render(line1),
		composeFooterStyle.Width(v.width).Render(line2),
		composeFooterStyle.Width(v.width).Render(line3),
	)

	return footer
}

// updateTable 更新表格数据
func (v *ComposeListView) updateTable() {
	rows := make([]table.Row, len(v.projects))
	for i, p := range v.projects {
		// 状态显示
		var status string
		switch p.Status {
		case compose.StatusRunning:
			status = "● 运行中"
		case compose.StatusPartial:
			status = "◐ 部分"
		case compose.StatusStopped:
			status = "○ 已停止"
		case compose.StatusError:
			status = "✗ 错误"
		default:
			status = "? 未知"
		}

		// 服务数量
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

		// 路径（截断显示）
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

// updateTableColumns 更新表格列宽
func (v *ComposeListView) updateTableColumns() {
	// 根据窗口宽度调整列宽
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

// GetSelectedProject 获取当前选中的项目
func (v *ComposeListView) GetSelectedProject() *compose.Project {
	if len(v.projects) == 0 {
		return nil
	}
	idx := v.tableModel.Cursor()
	if idx >= 0 && idx < len(v.projects) {
		return &v.projects[idx]
	}
	return nil
}


// 消息类型定义
type composeScanResultMsg struct {
	projects []compose.Project
	err      error
}

type composeOperationResultMsg struct {
	message string
	err     error
}

type composeRefreshStatusMsg struct {
	projects []compose.Project
}

type clearComposeMessageMsg struct{}

// scanProjects 扫描 Compose 项目
func (v *ComposeListView) scanProjects() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 如果没有指定扫描路径，使用当前目录
	paths := v.scanPaths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	projects, err := v.scanner.Scan(ctx, paths)
	if err != nil {
		return composeScanResultMsg{err: err}
	}

	// 刷新每个项目的状态
	for i := range projects {
		v.scanner.RefreshProject(ctx, &projects[i])
	}

	return composeScanResultMsg{projects: projects}
}

// refreshProjectStatus 刷新项目状态
func (v *ComposeListView) refreshProjectStatus() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 刷新所有项目状态
	for i := range v.projects {
		v.scanner.RefreshProject(ctx, &v.projects[i])
	}

	return composeRefreshStatusMsg{projects: v.projects}
}

// startOperation 开始执行操作
func (v *ComposeListView) startOperation(opType string) (View, tea.Cmd) {
	project := v.GetSelectedProject()
	if project == nil {
		v.errorMsg = "请先选择一个项目"
		return v, v.clearMessageAfter(3)
	}

	v.operatingProject = project
	v.operationType = opType
	v.errorMsg = ""
	v.successMsg = ""

	return v, v.executeOperation(project, opType)
}

// executeOperation 执行操作
func (v *ComposeListView) executeOperation(project *compose.Project, opType string) tea.Cmd {
	return func() tea.Msg {
		if v.composeClient == nil {
			return composeOperationResultMsg{err: fmt.Errorf("Compose 客户端未初始化")}
		}

		var result *compose.OperationResult
		var err error

		switch opType {
		case "up":
			result, err = v.composeClient.Up(project, compose.UpOptions{Detach: true})
		case "down":
			result, err = v.composeClient.Down(project, compose.DownOptions{})
		case "restart":
			result, err = v.composeClient.Restart(project, nil, 10)
		case "stop":
			result, err = v.composeClient.Stop(project, nil, 10)
		case "start":
			result, err = v.composeClient.Start(project, nil)
		default:
			return composeOperationResultMsg{err: fmt.Errorf("未知操作: %s", opType)}
		}

		if err != nil {
			return composeOperationResultMsg{err: err}
		}

		if result != nil && !result.Success {
			return composeOperationResultMsg{err: fmt.Errorf(result.Message)}
		}

		// 构建成功消息
		opNames := map[string]string{
			"up":      "启动",
			"down":    "停止",
			"restart": "重启",
			"stop":    "暂停",
			"start":   "恢复",
		}
		opName := opNames[opType]
		if opName == "" {
			opName = opType
		}

		return composeOperationResultMsg{
			message: fmt.Sprintf("%s 项目 %s 成功", opName, project.Name),
		}
	}
}

// clearMessageAfter 延迟清除消息
func (v *ComposeListView) clearMessageAfter(seconds int) tea.Cmd {
	return tea.Tick(time.Duration(seconds)*time.Second, func(t time.Time) tea.Msg {
		return clearComposeMessageMsg{}
	})
}
