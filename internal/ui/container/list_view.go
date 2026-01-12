package container

import (
	"context"
	"fmt"
	"strings"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
	"docktui/internal/ui/components"
)

// ListView 容器列表视图
type ListView struct {
	dockerClient docker.Client
	
	// UI 尺寸
	width  int
	height int
	
	// 数据状态
	containers         []docker.Container
	filteredContainers []docker.Container
	tableModel         table.Model
	scrollTable        *components.ScrollableTable
	loading            bool
	errorMsg           string
	successMsg         string
	successMsgTime     time.Time
	
	// 搜索状态
	searchQuery string
	isSearching bool
	
	// 筛选状态
	filterType string // "all", "running", "exited", "paused"
	
	// 刷新状态
	lastRefreshTime time.Time
	
	// 事件监听状态
	eventListening bool
	
	// 确认对话框状态
	showConfirmDialog bool
	confirmAction     string
	confirmContainer  *docker.Container
	confirmSelection  int
	
	// 多选功能
	selectedContainers map[string]bool
	
	// 编辑视图
	editView *EditView
	
	// 错误弹窗
	errorDialog *components.ErrorDialog
	
	// JSON 查看器
	jsonViewer *components.JSONViewer
	
	// 快捷键管理
	keys components.KeyMap
}

// NewListView 创建容器列表视图
func NewListView(dockerClient docker.Client) *ListView {
	columns := []table.Column{
		{Title: "CONTAINER ID", Width: 14},
		{Title: "NAMES", Width: 18},
		{Title: "IMAGE", Width: 25},
		{Title: "COMMAND", Width: 22},
		{Title: "CREATED", Width: 14},
		{Title: "STATUS", Width: 22},
		{Title: "PORTS", Width: 40},
	}
	
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
	
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(s)
	
	scrollColumns := []components.TableColumn{
		{Title: "SEL", Width: 3},
		{Title: "CONTAINER ID", Width: 14},
		{Title: "NAMES", Width: 20},
		{Title: "IMAGE", Width: 30},
		{Title: "COMMAND", Width: 25},
		{Title: "CREATED", Width: 16},
		{Title: "STATUS", Width: 25},
		{Title: "PORTS", Width: 50},
	}
	scrollTable := components.NewScrollableTable(scrollColumns)
	
	return &ListView{
		dockerClient:       dockerClient,
		tableModel:         t,
		scrollTable:        scrollTable,
		keys:               components.DefaultKeyMap(),
		searchQuery:        "",
		isSearching:        false,
		filterType:         "all",
		selectedContainers: make(map[string]bool),
		editView:           NewEditView(),
		errorDialog:        components.NewErrorDialog(),
		jsonViewer:         components.NewJSONViewer(),
	}
}

// Init 初始化容器列表视图
func (v *ListView) Init() tea.Cmd {
	v.loading = true
	return tea.Batch(
		v.loadContainers,
		v.watchDockerEvents(),
	)
}

// Update 处理消息并更新视图状态
func (v *ListView) Update(msg tea.Msg) (*ListView, tea.Cmd) {
	// 如果显示 JSON 查看器，优先处理
	if v.jsonViewer != nil && v.jsonViewer.IsVisible() {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if v.jsonViewer.Update(keyMsg) {
				return v, nil
			}
		}
	}

	switch msg := msg.(type) {
	case ContainersLoadedMsg:
		v.containers = msg.Containers
		v.loading = false
		v.errorMsg = ""
		v.lastRefreshTime = time.Now()
		v.applyFilters()
		v.updateColumnWidths()
		return v, nil
		
	case ContainersLoadErrorMsg:
		v.loading = false
		v.errorMsg = msg.Err.Error()
		return v, nil
		
	case ContainerEventMsg:
		event := msg.Event
		switch event.Action {
		case "start", "die", "stop", "rename", "create", "destroy":
			return v, tea.Batch(v.loadContainers, v.watchDockerEvents())
		}
		return v, v.watchDockerEvents()
		
	case ContainerEventErrorMsg:
		return v, v.watchDockerEvents()
		
	case ContainerOperationSuccessMsg:
		v.successMsg = fmt.Sprintf("✅ %s容器成功: %s", msg.Operation, msg.Container)
		v.successMsgTime = time.Now()
		v.errorMsg = ""
		return v, tea.Batch(
			v.loadContainers,
			v.clearSuccessMessageAfter(3*time.Second),
		)
		
	case ContainerOperationErrorMsg:
		errMsg := fmt.Sprintf("%s失败 (%s): %v", msg.Operation, msg.Container, msg.Err)
		if v.errorDialog != nil {
			v.errorDialog.ShowError(errMsg)
		}
		v.successMsg = ""
		return v, nil
	
	case ContainerOperationWarningMsg:
		v.successMsg = "⚠️ " + msg.Message
		v.successMsgTime = time.Now()
		v.errorMsg = ""
		return v, v.clearSuccessMessageAfter(3*time.Second)

	case ContainerBatchOperationMsg:
		if msg.FailedCount > 0 {
			v.successMsg = fmt.Sprintf("⚠️ %s: 成功 %d 个, 失败 %d 个", msg.Operation, msg.SuccessCount, msg.FailedCount)
			if msg.Err != nil && v.errorDialog != nil {
				v.errorDialog.ShowError(fmt.Sprintf("%s失败 (%s): %v", msg.Operation, strings.Join(msg.FailedNames, ", "), msg.Err))
			}
		} else {
			v.successMsg = fmt.Sprintf("✅ %s成功: %d 个容器", msg.Operation, msg.SuccessCount)
		}
		v.successMsgTime = time.Now()
		v.errorMsg = ""
		return v, tea.Batch(
			v.loadContainers,
			v.clearSuccessMessageAfter(3*time.Second),
		)
		
	case ClearSuccessMessageMsg:
		if time.Since(v.successMsgTime) >= 3*time.Second {
			v.successMsg = ""
		}
		return v, nil
	
	case ContainerInspectMsg:
		if v.jsonViewer != nil {
			v.jsonViewer.SetSize(v.width, v.height)
			v.jsonViewer.Show("Container Inspect: "+msg.ContainerName, msg.JSONContent)
		}
		return v, nil

	case ContainerInspectErrorMsg:
		if v.errorDialog != nil {
			v.errorDialog.ShowError(fmt.Sprintf("获取容器信息失败: %v", msg.Err))
		}
		return v, nil

	case ContainerEditReadyMsg:
		if v.editView != nil {
			v.editView.Show(msg.Container, msg.Details)
		}
		return v, nil
		
	case tea.KeyMsg:
		// 优先处理错误弹窗
		if v.errorDialog != nil && v.errorDialog.IsVisible() {
			if v.errorDialog.Update(msg) {
				return v, nil
			}
		}
		
		// 优先处理编辑视图
		if v.editView != nil && v.editView.IsVisible() {
			confirmed, handled, cmd := v.editView.Update(msg)
			if confirmed {
				return v, v.updateContainerConfig()
			}
			if handled {
				return v, cmd
			}
		}
		
		// 优先处理确认对话框
		if v.showConfirmDialog {
			switch msg.Type {
			case tea.KeyLeft, tea.KeyRight, tea.KeyTab:
				v.confirmSelection = 1 - v.confirmSelection
				return v, nil
			case tea.KeyEnter:
				if v.confirmSelection == 1 {
					action := v.confirmAction
					container := v.confirmContainer
					v.showConfirmDialog = false
					v.confirmAction = ""
					v.confirmContainer = nil
					v.confirmSelection = 0
					if action == "remove" && container != nil {
						return v, v.removeContainer(container)
					}
				} else {
					v.showConfirmDialog = false
					v.confirmAction = ""
					v.confirmContainer = nil
					v.confirmSelection = 0
				}
				return v, nil
			case tea.KeyEsc:
				v.showConfirmDialog = false
				v.confirmAction = ""
				v.confirmContainer = nil
				v.confirmSelection = 0
				return v, nil
			case tea.KeyRunes:
				keyStr := msg.String()
				if keyStr == "h" || keyStr == "l" {
					v.confirmSelection = 1 - v.confirmSelection
					return v, nil
				}
			}
			return v, nil
		}
		
		// ESC 键处理
		if msg.String() == "esc" {
			if v.isSearching {
				v.isSearching = false
				v.searchQuery = ""
				v.applyFilters()
				v.updateColumnWidths()
				return v, nil
			}
			if v.searchQuery != "" {
				v.searchQuery = ""
				v.applyFilters()
				v.updateColumnWidths()
				return v, nil
			}
			if v.filterType != "all" {
				v.filterType = "all"
				v.applyFilters()
				v.updateColumnWidths()
				return v, nil
			}
			return v, func() tea.Msg { return GoBackMsg{} }
		}
		
		// 搜索模式
		if v.isSearching {
			switch msg.String() {
			case "enter":
				v.isSearching = false
				return v, nil
			case "backspace":
				if len(v.searchQuery) > 0 {
					v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]
					v.applyFilters()
					v.updateColumnWidths()
				}
				return v, nil
			default:
				if len(msg.String()) == 1 {
					v.searchQuery += msg.String()
					v.applyFilters()
					v.updateColumnWidths()
				}
				return v, nil
			}
		}
		
		// 快捷键处理
		switch {
		case key.Matches(msg, v.keys.Refresh):
			v.loading = true
			v.errorMsg = ""
			return v, v.loadContainers
		case msg.String() == "f":
			switch v.filterType {
			case "all":
				v.filterType = "running"
			case "running":
				v.filterType = "exited"
			case "exited":
				v.filterType = "paused"
			case "paused":
				v.filterType = "all"
			default:
				v.filterType = "all"
			}
			v.applyFilters()
			v.updateColumnWidths()
			return v, nil
		case msg.String() == "/":
			v.isSearching = true
			v.searchQuery = ""
			return v, nil
		case msg.String() == "left", msg.String() == "h":
			if v.scrollTable != nil {
				v.scrollTable.ScrollLeft()
			}
			return v, nil
		case msg.String() == "right", msg.String() == "l":
			if v.scrollTable != nil {
				v.scrollTable.ScrollRight()
			}
			return v, nil
		case msg.String() == "j", msg.String() == "down":
			if v.scrollTable != nil {
				v.scrollTable.MoveDown(1)
			}
			v.tableModel.MoveDown(1)
			return v, nil
		case msg.String() == "k", msg.String() == "up":
			if v.scrollTable != nil {
				v.scrollTable.MoveUp(1)
			}
			v.tableModel.MoveUp(1)
			return v, nil
		case msg.String() == "g":
			if v.scrollTable != nil {
				v.scrollTable.GotoTop()
			}
			v.tableModel.GotoTop()
			return v, nil
		case msg.String() == "G":
			if v.scrollTable != nil {
				v.scrollTable.GotoBottom()
			}
			v.tableModel.GotoBottom()
			return v, nil
		case msg.String() == "t":
			return v, v.startSelectedContainer()
		case msg.String() == "o":
			return v, v.stopSelectedContainer()
		case msg.String() == "u":
			return v, v.togglePauseContainer()
		case msg.String() == "R":
			return v, v.restartSelectedContainer()
		case msg.String() == "ctrl+d":
			return v, v.showRemoveConfirmDialog()
		case msg.String() == "e":
			return v, v.showEditView()
		case msg.String() == "i":
			return v, v.inspectContainer()
		case msg.String() == " ":
			container := v.GetSelectedContainer()
			if container != nil {
				if v.selectedContainers[container.ID] {
					delete(v.selectedContainers, container.ID)
				} else {
					v.selectedContainers[container.ID] = true
				}
				v.updateTableData()
			}
			return v, nil
		case msg.String() == "a":
			allSelected := true
			for _, c := range v.filteredContainers {
				if !v.selectedContainers[c.ID] {
					allSelected = false
					break
				}
			}
			if allSelected && len(v.filteredContainers) > 0 {
				v.selectedContainers = make(map[string]bool)
			} else {
				for _, c := range v.filteredContainers {
					v.selectedContainers[c.ID] = true
				}
			}
			v.updateTableData()
			return v, nil
		case msg.String() == "enter":
			container := v.GetSelectedContainer()
			if container == nil {
				return v, nil
			}
			return v, func() tea.Msg {
				return ViewDetailsMsg{
					ContainerID:   container.ID,
					ContainerName: container.Name,
				}
			}
		case msg.String() == "L":
			container := v.GetSelectedContainer()
			if container == nil {
				return v, nil
			}
			return v, func() tea.Msg {
				return ViewLogsMsg{
					ContainerID:   container.ID,
					ContainerName: container.Name,
				}
			}
		default:
			v.tableModel, _ = v.tableModel.Update(msg)
			return v, nil
		}
	}
	
	return v, nil
}

// View 渲染容器列表视图
func (v *ListView) View() string {
	if v.jsonViewer != nil && v.jsonViewer.IsVisible() {
		return v.jsonViewer.View()
	}

	var s string
	s += v.renderStatusBar()
	
	if v.successMsg != "" {
		msgStyle := SuccessMsgStyle
		if strings.HasPrefix(v.successMsg, "⚠️") {
			msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
		}
		s += "\n  " + msgStyle.Render(v.successMsg) + "\n"
	}
	
	totalCount := len(v.containers)
	showingCount := len(v.filteredContainers)
	
	runningCount := 0
	stoppedCount := 0
	for _, c := range v.containers {
		if c.State == "running" {
			runningCount++
		} else {
			stoppedCount++
		}
	}
	
	totalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	runningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	stoppedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	
	statsContent := totalStyle.Render(fmt.Sprintf("📦 Total: %d", totalCount)) +
		separatorStyle.Render("  │  ") +
		runningStyle.Render(fmt.Sprintf("✓ Running: %d", runningCount)) +
		separatorStyle.Render("  │  ") +
		stoppedStyle.Render(fmt.Sprintf("■ Stopped: %d", stoppedCount))
	
	if showingCount != totalCount || (!v.isSearching && v.searchQuery != "") {
		filterParts := []string{}
		if showingCount != totalCount {
			filterParts = append(filterParts, fmt.Sprintf("Showing: %d", showingCount))
		}
		if !v.isSearching && v.searchQuery != "" {
			filterParts = append(filterParts, fmt.Sprintf("Search: \"%s\"", v.searchQuery))
		}
		filterInfo := SearchHintStyle.Render("  [" + strings.Join(filterParts, " | ") + "]")
		statsContent += filterInfo
	}
	
	lineWidth := v.width - 6
	if lineWidth < 60 {
		lineWidth = 60
	}
	line := lineStyle.Render(strings.Repeat("─", lineWidth))
	statsLine := lipgloss.NewStyle().Width(lineWidth).Align(lipgloss.Center).Render(statsContent)
	
	s += "\n  " + line + "\n"
	s += "  " + statsLine + "\n"
	s += "  " + line + "\n"
	
	if v.loading {
		loadingContent := lipgloss.JoinVertical(lipgloss.Center,
			"",
			StatusBarKeyStyle.Render("⏳ 正在加载容器列表..."),
			"",
			SearchHintStyle.Render("请稍候，正在从 Docker 获取数据"),
			"",
		)
		s += "\n  " + StateBoxStyle.Render(loadingContent) + "\n"
		return s
	}
	
	if v.errorMsg != "" && len(v.containers) == 0 {
		errLines := []string{""}
		errText := strings.TrimPrefix(v.errorMsg, "❌ ")
		maxLineLen := 70
		for len(errText) > maxLineLen {
			errLines = append(errLines, ErrorMsgStyle.Render(errText[:maxLineLen]))
			errText = errText[maxLineLen:]
		}
		if errText != "" {
			errLines = append(errLines, ErrorMsgStyle.Render(errText))
		}
		errLines = append(errLines,
			"",
			StatusBarKeyStyle.Render("按 r 重新加载") + SearchHintStyle.Render(" 或 ") + StatusBarKeyStyle.Render("按 Esc 返回"),
			"",
		)
		errorContent := lipgloss.JoinVertical(lipgloss.Left, errLines...)
		s += "\n  " + StateBoxStyle.Width(v.width-10).Render(errorContent) + "\n"
		return s
	}
	
	if len(v.containers) == 0 {
		emptyContent := lipgloss.JoinVertical(lipgloss.Left,
			"",
			SearchHintStyle.Render("📦 暂无容器"),
			"",
			StatusBarLabelStyle.Render("💡 快速开始:"),
			"",
			StatusBarKeyStyle.Render("1.") + SearchHintStyle.Render(" 启动一个测试容器:"),
			SearchHintStyle.Render("   docker run -d --name test nginx"),
			"",
			StatusBarKeyStyle.Render("2.") + SearchHintStyle.Render(" 刷新容器列表:"),
			SearchHintStyle.Render("   按 r 键刷新"),
			"",
			SearchHintStyle.Render("提示: 容器列表会自动刷新（事件驱动模式）"),
			"",
		)
		s += "\n  " + StateBoxStyle.Render(emptyContent) + "\n"
		return s
	}
	
	if len(v.filteredContainers) == 0 {
		var filterHints []string
		filterHints = append(filterHints, "", SearchHintStyle.Render("🔍 No matching containers"), "")
		filterHints = append(filterHints, StatusBarLabelStyle.Render("Current search:"))
		if v.searchQuery != "" {
			filterHints = append(filterHints, SearchHintStyle.Render("   • Keyword: ")+StatusBarKeyStyle.Render("\""+v.searchQuery+"\""))
		}
		filterHints = append(filterHints, "", StatusBarLabelStyle.Render("💡 Tips:"))
		if v.searchQuery != "" {
			filterHints = append(filterHints, SearchHintStyle.Render("   • Press ")+StatusBarKeyStyle.Render("ESC")+SearchHintStyle.Render(" to clear search"))
		} else {
			filterHints = append(filterHints, SearchHintStyle.Render("   • Press ")+StatusBarKeyStyle.Render("/")+SearchHintStyle.Render(" to search"))
		}
		filterHints = append(filterHints, SearchHintStyle.Render("   • Press ")+StatusBarKeyStyle.Render("r")+SearchHintStyle.Render(" to refresh"), "")
		emptyFilterContent := lipgloss.JoinVertical(lipgloss.Left, filterHints...)
		s += "\n  " + StateBoxStyle.Render(emptyFilterContent) + "\n"
		return s
	}
	
	if v.scrollTable != nil {
		s += v.scrollTable.View() + "\n"
	} else {
		s += "  " + v.tableModel.View() + "\n"
	}
	
	s += "\n"
	
	if v.isSearching {
		searchLine := "\n  " + strings.Repeat("─", 67) + "\n"
		searchPrompt := "  " + SearchPromptStyle.Render("Search:") + " "
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		searchInput := v.searchQuery + cursor
		cancelHint := SearchHintStyle.Render("[Enter=Confirm | ESC=Cancel]")
		totalWidth := 70
		usedWidth := 10 + len(v.searchQuery) + 1 + 28
		padding := ""
		if totalWidth > usedWidth {
			padding = strings.Repeat(" ", totalWidth-usedWidth)
		}
		s += searchLine + searchPrompt + searchInput + padding + cancelHint + "\n"
	}
	
	if !v.isSearching && v.filterType != "all" {
		filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
		s += "  " + filterStyle.Render("[Filter: "+v.filterType+"]") + "  " + SearchHintStyle.Render("按 ESC 清除筛选，按 f 切换") + "\n"
	}
	
	if v.showConfirmDialog {
		s = v.overlayDialog(s)
	}
	
	if v.editView != nil && v.editView.IsVisible() {
		s = v.overlayEditView(s)
	}
	
	if v.errorDialog != nil && v.errorDialog.IsVisible() {
		s = v.errorDialog.Overlay(s)
	}
	
	return s
}

// overlayDialog 将对话框叠加到现有内容上
func (v *ListView) overlayDialog(baseContent string) string {
	return components.OverlayCentered(baseContent, v.renderConfirmDialogContent(), v.width, v.height)
}

// renderConfirmDialogContent 渲染对话框内容
func (v *ListView) renderConfirmDialogContent() string {
	if v.confirmContainer == nil {
		return ""
	}

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(56)
	
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	
	cancelBtnStyle := lipgloss.NewStyle().Padding(0, 2)
	okBtnStyle := lipgloss.NewStyle().Padding(0, 2)
	
	if v.confirmSelection == 0 {
		cancelBtnStyle = cancelBtnStyle.Reverse(true).Bold(true)
		okBtnStyle = okBtnStyle.Foreground(lipgloss.Color("245"))
	} else {
		cancelBtnStyle = cancelBtnStyle.Foreground(lipgloss.Color("245"))
		okBtnStyle = okBtnStyle.Reverse(true).Bold(true)
	}
	
	containerName := v.confirmContainer.Name
	if len(containerName) > 35 {
		containerName = containerName[:32] + "..."
	}
	
	warningText := "This action cannot be undone!"
	if v.confirmContainer.State == "running" {
		warningText = "⚠️  容器正在运行，将强制删除！"
	}
	
	title := titleStyle.Render("⚠️  Delete Container: " + containerName)
	warning := warningStyle.Render(warningText)
	
	cancelBtn := cancelBtnStyle.Render("< Cancel >")
	okBtn := okBtnStyle.Render("< OK >")
	buttons := cancelBtn + "    " + okBtn
	buttonsLine := lipgloss.NewStyle().Width(52).Align(lipgloss.Center).Render(buttons)
	
	content := title + "\n\n" + warning + "\n\n" + buttonsLine
	dialog := dialogStyle.Render(content)
	
	if v.width > 60 {
		leftPadding := (v.width - 60) / 2
		lines := strings.Split(dialog, "\n")
		var result strings.Builder
		for i, line := range lines {
			result.WriteString(strings.Repeat(" ", leftPadding))
			result.WriteString(line)
			if i < len(lines)-1 {
				result.WriteString("\n")
			}
		}
		return result.String()
	}
	
	return dialog
}

// renderStatusBar 渲染顶部状态栏
func (v *ListView) renderStatusBar() string {
	width := v.width
	if width < 80 {
		width = 80
	}
	
	availableWidth := width - 4
	if availableWidth < 60 {
		availableWidth = 60
	}
	
	labelColWidth := 20
	shortcutsWidth := availableWidth - labelColWidth
	
	itemsPerRow := 4
	if shortcutsWidth < 60 {
		itemsPerRow = 3
	}
	if shortcutsWidth < 45 {
		itemsPerRow = 2
	}
	
	itemWidth := shortcutsWidth / itemsPerRow
	if itemWidth < 12 {
		itemWidth = 12
	}
	
	labelStyle := lipgloss.NewStyle().Width(labelColWidth).Foreground(lipgloss.Color("220")).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	itemStyle := lipgloss.NewStyle().Width(itemWidth)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	
	makeItem := func(key, desc string) string {
		return itemStyle.Render(keyStyle.Render(key) + descStyle.Render(" "+desc))
	}
	
	var lines []string
	
	row1Label := labelStyle.Render("📦 Containers")
	row1Keys := makeItem("<f>", "Filter") + makeItem("</>", "Search") + makeItem("<r>", "Refresh")
	lines = append(lines, "  "+row1Label+row1Keys)
	
	row2Label := labelStyle.Render("Ops:")
	row2Keys := makeItem("<t>", "Start") + makeItem("<o>", "Stop") + makeItem("<u>", "Pause") + makeItem("<R>", "Restart")
	lines = append(lines, "  "+row2Label+row2Keys)
	
	row3Label := labelStyle.Render("Advanced:")
	row3Keys := makeItem("<Ctrl+D>", "Delete") + makeItem("<e>", "Edit") + makeItem("<i>", "Inspect") + makeItem("<L>", "Logs")
	lines = append(lines, "  "+row3Label+row3Keys)

	row4Label := labelStyle.Render("Select:")
	row4Keys := makeItem("<Space>", "Toggle") + makeItem("<a>", "All")
	lines = append(lines, "  "+row4Label+row4Keys)
	
	refreshInfo := "-"
	if !v.lastRefreshTime.IsZero() {
		refreshInfo = formatDuration(time.Since(v.lastRefreshTime)) + " ago"
	}
	
	row5Label := labelStyle.Render("Last Refresh:")
	row5Info := hintStyle.Render(refreshInfo) + "    " + 
		hintStyle.Render("j/k=上下  Enter=详情  Esc=返回  q=退出")
	
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	if len(v.selectedContainers) > 0 {
		row5Info += "    " + selectedStyle.Render(fmt.Sprintf("[已选: %d]", len(v.selectedContainers)))
	}
	
	lines = append(lines, "  "+row5Label+row5Info)
	
	return "\n" + strings.Join(lines, "\n") + "\n"
}

// containersToRows 将容器数据转换为 table.Row
func (v *ListView) containersToRows(containers []docker.Container) []table.Row {
	rows := make([]table.Row, len(containers))
	
	exitedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	pausedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	unhealthyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	
	for i, c := range containers {
		created := formatCreatedTime(c.Created)
		ports := c.Ports
		if ports == "" {
			ports = ""
		}
		
		var rowStyle lipgloss.Style
		var needsStyle bool
		
		switch {
		case strings.Contains(strings.ToLower(c.Status), "unhealthy"):
			rowStyle = unhealthyStyle
			needsStyle = true
		case c.State == "paused":
			rowStyle = pausedStyle
			needsStyle = true
		case c.State == "exited":
			rowStyle = exitedStyle
			needsStyle = true
		default:
			needsStyle = false
		}
		
		if needsStyle {
			rows[i] = table.Row{
				rowStyle.Render(c.ShortID),
				rowStyle.Render(c.Name),
				rowStyle.Render(c.Image),
				rowStyle.Render(c.Command),
				rowStyle.Render(created),
				rowStyle.Render(c.Status),
				rowStyle.Render(ports),
			}
		} else {
			rows[i] = table.Row{
				c.ShortID,
				c.Name,
				c.Image,
				c.Command,
				created,
				c.Status,
				ports,
			}
		}
	}
	
	return rows
}

// formatCreatedTime 格式化创建时间
func formatCreatedTime(t time.Time) string {
	d := time.Since(t)
	
	if d < time.Minute {
		return fmt.Sprintf("%d seconds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	}
	if d < 48*time.Hour {
		return "1 day ago"
	}
	if d < 30*24*time.Hour {
		return fmt.Sprintf("%d days ago", int(d.Hours())/24)
	}
	if d < 60*24*time.Hour {
		return "1 month ago"
	}
	return fmt.Sprintf("%d months ago", int(d.Hours())/(24*30))
}

// formatDuration 格式化时间差
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "just now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// SetSize 设置视图尺寸
func (v *ListView) SetSize(width, height int) {
	v.width = width
	v.height = height
	
	tableHeight := height - 15
	if tableHeight < 5 {
		tableHeight = 5
	}
	v.tableModel.SetHeight(tableHeight)
	
	if v.scrollTable != nil {
		v.scrollTable.SetSize(width-4, tableHeight)
	}
	
	if v.editView != nil {
		v.editView.SetWidth(width)
	}
	
	if v.errorDialog != nil {
		v.errorDialog.SetWidth(width)
	}
	
	v.updateColumnWidths()
	StateBoxStyle = StateBoxStyle.Width(width - 10)
}

// updateColumnWidths 根据实际数据计算并更新列宽
func (v *ListView) updateColumnWidths() {
	maxID := 12
	maxImage := 5
	maxCommand := 7
	maxCreated := 7
	maxStatus := 6
	maxPorts := 5
	maxNames := 5
	
	for _, c := range v.filteredContainers {
		if len(c.Image) > maxImage {
			maxImage = len(c.Image)
		}
		if len(c.Command) > maxCommand {
			maxCommand = len(c.Command)
		}
		created := formatCreatedTime(c.Created)
		if len(created) > maxCreated {
			maxCreated = len(created)
		}
		if len(c.Status) > maxStatus {
			maxStatus = len(c.Status)
		}
		if len(c.Ports) > maxPorts {
			maxPorts = len(c.Ports)
		}
		if len(c.Name) > maxNames {
			maxNames = len(c.Name)
		}
	}
	
	statusAnsiPadding := 20
	availableWidth := v.width - 10
	idWidth := maxID + 2
	totalNeeded := idWidth + maxImage + maxCommand + maxCreated + (maxStatus + statusAnsiPadding) + maxPorts + maxNames + 14
	
	if totalNeeded <= availableWidth {
		v.tableModel.SetColumns([]table.Column{
			{Title: "CONTAINER ID", Width: idWidth},
			{Title: "NAMES", Width: maxNames + 2},
			{Title: "IMAGE", Width: maxImage + 2},
			{Title: "COMMAND", Width: maxCommand + 2},
			{Title: "CREATED", Width: maxCreated + 2},
			{Title: "STATUS", Width: maxStatus + 2 + statusAnsiPadding},
			{Title: "PORTS", Width: maxPorts + 2},
		})
	} else {
		flexWidth := availableWidth - idWidth - statusAnsiPadding - 6
		totalVar := maxImage + maxCommand + maxCreated + maxStatus + maxPorts + maxNames
		if totalVar == 0 {
			totalVar = 1
		}
		
		imageWidth := flexWidth * maxImage / totalVar
		commandWidth := flexWidth * maxCommand / totalVar
		createdWidth := flexWidth * maxCreated / totalVar
		statusWidth := flexWidth * maxStatus / totalVar + statusAnsiPadding
		portsWidth := flexWidth * maxPorts / totalVar
		namesWidth := flexWidth * maxNames / totalVar
		
		if imageWidth < 15 { imageWidth = 15 }
		if commandWidth < 12 { commandWidth = 12 }
		if createdWidth < 12 { createdWidth = 12 }
		if statusWidth < 15 + statusAnsiPadding { statusWidth = 15 + statusAnsiPadding }
		if portsWidth < 20 { portsWidth = 20 }
		if namesWidth < 12 { namesWidth = 12 }
		
		v.tableModel.SetColumns([]table.Column{
			{Title: "CONTAINER ID", Width: idWidth},
			{Title: "NAMES", Width: namesWidth},
			{Title: "IMAGE", Width: imageWidth},
			{Title: "COMMAND", Width: commandWidth},
			{Title: "CREATED", Width: createdWidth},
			{Title: "STATUS", Width: statusWidth},
			{Title: "PORTS", Width: portsWidth},
		})
	}
	
	if v.scrollTable != nil {
		v.scrollTable.SetColumns([]components.TableColumn{
			{Title: "SEL", Width: 3},
			{Title: "CONTAINER ID", Width: maxID + 2},
			{Title: "NAMES", Width: maxNames + 2},
			{Title: "IMAGE", Width: maxImage + 2},
			{Title: "COMMAND", Width: maxCommand + 2},
			{Title: "CREATED", Width: maxCreated + 2},
			{Title: "STATUS", Width: maxStatus + 2},
			{Title: "PORTS", Width: maxPorts + 2},
		})
		
		if len(v.filteredContainers) > 0 {
			rows := make([]components.TableRow, len(v.filteredContainers))
			
			exitedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
			pausedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
			unhealthyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
			
			for i, c := range v.filteredContainers {
				created := formatCreatedTime(c.Created)
				ports := c.Ports
				if ports == "" {
					ports = "-"
				}
				
				selMark := " "
				if v.selectedContainers[c.ID] {
					selMark = selectedStyle.Render("✓")
				}
				
				var rowStyle lipgloss.Style
				var needsStyle bool
				
				switch {
				case strings.Contains(strings.ToLower(c.Status), "unhealthy"):
					rowStyle = unhealthyStyle
					needsStyle = true
				case c.State == "paused":
					rowStyle = pausedStyle
					needsStyle = true
				case c.State == "exited":
					rowStyle = exitedStyle
					needsStyle = true
				default:
					needsStyle = false
				}
				
				if needsStyle {
					rows[i] = components.TableRow{
						selMark,
						rowStyle.Render(c.ShortID),
						rowStyle.Render(c.Name),
						rowStyle.Render(c.Image),
						rowStyle.Render(c.Command),
						rowStyle.Render(created),
						rowStyle.Render(c.Status),
						rowStyle.Render(ports),
					}
				} else {
					rows[i] = components.TableRow{
						selMark,
						c.ShortID,
						c.Name,
						c.Image,
						c.Command,
						created,
						c.Status,
						ports,
					}
				}
			}
			v.scrollTable.SetRows(rows)
		} else {
			v.scrollTable.SetRows([]components.TableRow{})
		}
	}
	
	if len(v.filteredContainers) > 0 {
		rows := v.containersToRows(v.filteredContainers)
		v.tableModel.SetRows(rows)
	} else {
		v.tableModel.SetRows([]table.Row{})
	}
}

// GetSelectedContainer 获取当前选中的容器
func (v *ListView) GetSelectedContainer() *docker.Container {
	if len(v.filteredContainers) == 0 {
		return nil
	}
	var selectedIndex int
	if v.scrollTable != nil {
		selectedIndex = v.scrollTable.Cursor()
	} else {
		selectedIndex = v.tableModel.Cursor()
	}
	if selectedIndex < 0 || selectedIndex >= len(v.filteredContainers) {
		return nil
	}
	return &v.filteredContainers[selectedIndex]
}

// IsSearching 返回是否处于搜索模式
func (v *ListView) IsSearching() bool {
	return v.isSearching
}

// applyFilters 应用搜索和状态过滤
func (v *ListView) applyFilters() {
	v.filteredContainers = make([]docker.Container, 0)
	
	for _, container := range v.containers {
		switch v.filterType {
		case "running":
			if container.State != "running" {
				continue
			}
		case "exited":
			if container.State != "exited" {
				continue
			}
		case "paused":
			if container.State != "paused" {
				continue
			}
		}
		
		if v.searchQuery != "" {
			query := strings.ToLower(v.searchQuery)
			if !strings.Contains(strings.ToLower(container.Name), query) &&
			   !strings.Contains(strings.ToLower(container.Image), query) &&
			   !strings.Contains(strings.ToLower(container.ID), query) {
				continue
			}
		}
		
		v.filteredContainers = append(v.filteredContainers, container)
	}
}

// loadContainers 加载容器列表
func (v *ListView) loadContainers() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	containers, err := v.dockerClient.ListContainers(ctx, true)
	if err != nil {
		return ContainersLoadErrorMsg{Err: err}
	}
	
	return ContainersLoadedMsg{Containers: containers}
}

// watchDockerEvents 监听 Docker 容器事件
func (v *ListView) watchDockerEvents() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		eventChan, errorChan := v.dockerClient.WatchEvents(ctx)
		
		select {
		case event, ok := <-eventChan:
			if !ok {
				return ContainerEventErrorMsg{Err: fmt.Errorf("事件通道关闭")}
			}
			return ContainerEventMsg{Event: event}
		case err, ok := <-errorChan:
			if !ok || err == nil {
				return nil
			}
			return ContainerEventErrorMsg{Err: err}
		}
	}
}

// startSelectedContainer 启动选中的容器
func (v *ListView) startSelectedContainer() tea.Cmd {
	containers := v.getSelectedOrCurrentContainers()
	if len(containers) == 0 {
		return func() tea.Msg {
			return ContainerOperationErrorMsg{Operation: "启动容器", Container: "", Err: fmt.Errorf("请先选择一个容器")}
		}
	}

	var toStart []docker.Container
	for _, c := range containers {
		if c.State != "running" {
			toStart = append(toStart, c)
		}
	}

	if len(toStart) == 0 {
		return func() tea.Msg {
			return ContainerOperationWarningMsg{Message: "所有选中的容器都已在运行中"}
		}
	}

	return v.batchContainerOperation("启动", toStart, func(ctx context.Context, id string) error {
		return v.dockerClient.StartContainer(ctx, id)
	})
}

// stopSelectedContainer 停止选中的容器
func (v *ListView) stopSelectedContainer() tea.Cmd {
	containers := v.getSelectedOrCurrentContainers()
	if len(containers) == 0 {
		return func() tea.Msg {
			return ContainerOperationErrorMsg{Operation: "停止容器", Container: "", Err: fmt.Errorf("请先选择一个容器")}
		}
	}

	var toStop []docker.Container
	for _, c := range containers {
		if c.State == "running" {
			toStop = append(toStop, c)
		}
	}

	if len(toStop) == 0 {
		return func() tea.Msg {
			return ContainerOperationWarningMsg{Message: "所有选中的容器都未在运行"}
		}
	}

	return v.batchContainerOperation("停止", toStop, func(ctx context.Context, id string) error {
		return v.dockerClient.StopContainer(ctx, id, 10)
	})
}

// restartSelectedContainer 重启选中的容器
func (v *ListView) restartSelectedContainer() tea.Cmd {
	containers := v.getSelectedOrCurrentContainers()
	if len(containers) == 0 {
		return func() tea.Msg {
			return ContainerOperationErrorMsg{Operation: "重启容器", Container: "", Err: fmt.Errorf("请先选择一个容器")}
		}
	}

	return v.batchContainerOperation("重启", containers, func(ctx context.Context, id string) error {
		return v.dockerClient.RestartContainer(ctx, id, 10)
	})
}

// showRemoveConfirmDialog 显示删除确认对话框
func (v *ListView) showRemoveConfirmDialog() tea.Cmd {
	container := v.GetSelectedContainer()
	if container == nil {
		return func() tea.Msg {
			return ContainerOperationErrorMsg{Operation: "删除容器", Container: "", Err: fmt.Errorf("请先选择一个容器")}
		}
	}

	v.showConfirmDialog = true
	v.confirmAction = "remove"
	v.confirmContainer = container
	v.confirmSelection = 0
	return nil
}

// removeContainer 删除容器
func (v *ListView) removeContainer(container *docker.Container) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		force := container.State == "running"
		err := v.dockerClient.RemoveContainer(ctx, container.ID, force, false)
		if err != nil {
			return ContainerOperationErrorMsg{Operation: "删除容器", Container: container.Name, Err: err}
		}

		return ContainerOperationSuccessMsg{Operation: "删除", Container: container.Name}
	}
}

// clearSuccessMessageAfter 在指定时间后清除成功消息
func (v *ListView) clearSuccessMessageAfter(duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(duration)
		return ClearSuccessMessageMsg{}
	}
}

// togglePauseContainer 暂停/恢复选中的容器
func (v *ListView) togglePauseContainer() tea.Cmd {
	container := v.GetSelectedContainer()
	if container == nil {
		return func() tea.Msg {
			return ContainerOperationErrorMsg{Operation: "暂停/恢复容器", Container: "", Err: fmt.Errorf("请先选择一个容器")}
		}
	}

	if container.State != "running" && container.State != "paused" {
		return func() tea.Msg {
			return ContainerOperationWarningMsg{
				Message: fmt.Sprintf("容器 %s 状态为 %s，只能暂停运行中的容器或恢复已暂停的容器", container.Name, container.State),
			}
		}
	}

	isPaused := container.State == "paused"

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		var operation string

		if isPaused {
			err = v.dockerClient.UnpauseContainer(ctx, container.ID)
			operation = "恢复"
		} else {
			err = v.dockerClient.PauseContainer(ctx, container.ID)
			operation = "暂停"
		}

		if err != nil {
			return ContainerOperationErrorMsg{Operation: operation + "容器", Container: container.Name, Err: err}
		}

		return ContainerOperationSuccessMsg{Operation: operation, Container: container.Name}
	}
}

// showEditView 显示编辑视图
func (v *ListView) showEditView() tea.Cmd {
	container := v.GetSelectedContainer()
	if container == nil {
		return func() tea.Msg {
			return ContainerOperationErrorMsg{Operation: "编辑容器", Container: "", Err: fmt.Errorf("请先选择一个容器")}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		details, err := v.dockerClient.ContainerDetails(ctx, container.ID)
		if err != nil {
			return ContainerOperationErrorMsg{Operation: "获取容器详情", Container: container.Name, Err: err}
		}

		return ContainerEditReadyMsg{Container: container, Details: details}
	}
}

// inspectContainer 获取容器的原始 JSON
func (v *ListView) inspectContainer() tea.Cmd {
	container := v.GetSelectedContainer()
	if container == nil {
		return nil
	}

	containerID := container.ID
	containerName := container.Name

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		jsonContent, err := v.dockerClient.InspectContainerRaw(ctx, containerID)
		if err != nil {
			return ContainerInspectErrorMsg{Err: err}
		}

		return ContainerInspectMsg{ContainerName: containerName, JSONContent: jsonContent}
	}
}

// updateContainerConfig 更新容器配置
func (v *ListView) updateContainerConfig() tea.Cmd {
	if v.editView == nil {
		return nil
	}

	containerID := v.editView.GetContainerID()
	containerName := v.editView.GetContainerName()
	config := v.editView.GetConfig()
	v.editView.Hide()

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := v.dockerClient.UpdateContainer(ctx, containerID, config)
		if err != nil {
			return ContainerOperationErrorMsg{Operation: "更新容器配置", Container: containerName, Err: err}
		}

		return ContainerOperationSuccessMsg{Operation: "更新配置", Container: containerName}
	}
}

// overlayEditView 将编辑视图叠加到现有内容上
func (v *ListView) overlayEditView(baseContent string) string {
	if v.editView == nil {
		return baseContent
	}
	return components.OverlayCentered(baseContent, v.editView.View(), v.width, v.height)
}

// IsEditViewVisible 返回编辑视图是否可见
func (v *ListView) IsEditViewVisible() bool {
	return v.editView != nil && v.editView.IsVisible()
}

// HasError 返回是否有错误信息显示
func (v *ListView) HasError() bool {
	return v.errorDialog != nil && v.errorDialog.IsVisible()
}

// IsShowingJSONViewer 返回是否正在显示 JSON 查看器
func (v *ListView) IsShowingJSONViewer() bool {
	return v.jsonViewer != nil && v.jsonViewer.IsVisible()
}

// getSelectedOrCurrentContainers 获取选中的容器列表
func (v *ListView) getSelectedOrCurrentContainers() []docker.Container {
	if len(v.selectedContainers) > 0 {
		var containers []docker.Container
		for _, c := range v.filteredContainers {
			if v.selectedContainers[c.ID] {
				containers = append(containers, c)
			}
		}
		return containers
	}
	
	container := v.GetSelectedContainer()
	if container != nil {
		return []docker.Container{*container}
	}
	return nil
}

// batchContainerOperation 批量执行容器操作
func (v *ListView) batchContainerOperation(opName string, containers []docker.Container, op func(ctx context.Context, id string) error) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		successCount := 0
		var lastError error
		var failedNames []string

		for _, c := range containers {
			err := op(ctx, c.ID)
			if err != nil {
				lastError = err
				failedNames = append(failedNames, c.Name)
			} else {
				successCount++
			}
		}

		v.selectedContainers = make(map[string]bool)

		if len(failedNames) > 0 {
			if successCount > 0 {
				return ContainerBatchOperationMsg{
					Operation:    opName,
					SuccessCount: successCount,
					FailedCount:  len(failedNames),
					FailedNames:  failedNames,
					Err:          lastError,
				}
			}
			return ContainerOperationErrorMsg{
				Operation: opName + "容器",
				Container: strings.Join(failedNames, ", "),
				Err:       lastError,
			}
		}

		if len(containers) == 1 {
			return ContainerOperationSuccessMsg{Operation: opName, Container: containers[0].Name}
		}
		return ContainerBatchOperationMsg{Operation: opName, SuccessCount: successCount, FailedCount: 0}
	}
}

// updateTableData 更新表格数据
func (v *ListView) updateTableData() {
	if v.scrollTable == nil || len(v.filteredContainers) == 0 {
		return
	}

	rows := make([]components.TableRow, len(v.filteredContainers))
	
	exitedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	pausedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	unhealthyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	
	for i, c := range v.filteredContainers {
		created := formatCreatedTime(c.Created)
		ports := c.Ports
		if ports == "" {
			ports = "-"
		}
		
		selMark := " "
		if v.selectedContainers[c.ID] {
			selMark = selectedStyle.Render("✓")
		}
		
		var rowStyle lipgloss.Style
		var needsStyle bool
		
		switch {
		case strings.Contains(strings.ToLower(c.Status), "unhealthy"):
			rowStyle = unhealthyStyle
			needsStyle = true
		case c.State == "paused":
			rowStyle = pausedStyle
			needsStyle = true
		case c.State == "exited":
			rowStyle = exitedStyle
			needsStyle = true
		default:
			needsStyle = false
		}
		
		if needsStyle {
			rows[i] = components.TableRow{
				selMark,
				rowStyle.Render(c.ShortID),
				rowStyle.Render(c.Name),
				rowStyle.Render(c.Image),
				rowStyle.Render(c.Command),
				rowStyle.Render(created),
				rowStyle.Render(c.Status),
				rowStyle.Render(ports),
			}
		} else {
			rows[i] = components.TableRow{
				selMark,
				c.ShortID,
				c.Name,
				c.Image,
				c.Command,
				created,
				c.Status,
				ports,
			}
		}
	}
	v.scrollTable.SetRows(rows)
}

// GetSelectedCount 获取选中的容器数量
func (v *ListView) GetSelectedCount() int {
	return len(v.selectedContainers)
}
