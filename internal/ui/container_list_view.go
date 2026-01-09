package ui

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
)

// 容器列表视图样式定义 - 使用自适应颜色，不硬编码背景色
var (
	// 状态栏样式
	statusBarLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	statusBarValueStyle = lipgloss.NewStyle()
	
	statusBarKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))
	
	// 标题栏样式
	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	// 过滤状态样式
	filterAllStyle = lipgloss.NewStyle()
	
	filterRunningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)
	
	filterExitedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	// 成功/错误消息样式
	successMsgStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)
	
	errorMsgStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	
	// 搜索栏样式
	searchPromptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Bold(true)
	
	searchHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	// 对话框样式 - 使用边框区分，不设置背景
	dialogStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)
	
	dialogTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	dialogWarningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	// 按钮样式 - 使用 Reverse 实现选中效果
	buttonActiveStyle = lipgloss.NewStyle().
		Reverse(true).
		Bold(true).
		Padding(0, 2)
	
	buttonInactiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 2)
	
	// 加载/空状态框样式
	stateBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(66)
)

// ContainerListView 容器列表视图
type ContainerListView struct {
	dockerClient docker.Client
	
	// UI 尺寸
	width  int
	height int
	
	// 数据状态（L1.1）
	containers    []docker.Container // 容器列表数据（原始）
	filteredContainers []docker.Container // 过滤后的容器列表
	tableModel    table.Model        // bubbles/table 组件（保留兼容）
	scrollTable   *ScrollableTable   // 可水平滚动的表格
	loading       bool               // 是否正在加载
	errorMsg      string             // 错误信息（初始加载失败时使用）
	successMsg    string             // 成功消息
	successMsgTime time.Time         // 成功消息显示时间
	
	// 搜索状态（L4）
	searchQuery   string // 搜索关键字
	isSearching   bool   // 是否处于搜索模式
	
	// 筛选状态
	filterType    string // "all", "running", "exited", "paused"
	
	// 刷新状态
	lastRefreshTime   time.Time // 上次刷新时间
	
	// 事件监听状态（E2）
	eventListening bool // 是否正在监听事件
	
	// 确认对话框状态（O2）
	showConfirmDialog bool   // 是否显示确认对话框
	confirmAction     string // 确认的操作类型: "remove"
	confirmContainer  *docker.Container // 待操作的容器
	confirmSelection  int    // 确认对话框中的选择: 0=Cancel, 1=OK
	
	// 编辑视图
	editView *ContainerEditView // 容器配置编辑视图
	
	// 错误弹窗
	errorDialog *ErrorDialog // 错误弹窗组件
	
	// JSON 查看器
	jsonViewer *JSONViewer // JSON 查看器
	
	// 快捷键管理（R3）
	keys KeyMap
}

// NewContainerListView 创建容器列表视图
func NewContainerListView(dockerClient docker.Client) *ContainerListView {
	// 定义表格列（NAME 移到第二列）
	columns := []table.Column{
		{Title: "CONTAINER ID", Width: 14},
		{Title: "NAMES", Width: 18},
		{Title: "IMAGE", Width: 25},
		{Title: "COMMAND", Width: 22},
		{Title: "CREATED", Width: 14},
		{Title: "STATUS", Width: 22},
		{Title: "PORTS", Width: 40},
	}
	
	// 创建表格样式
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
	
	// 初始化表格组件
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(s)
	
	// 创建可滚动表格（NAME 移到第二列）
	scrollColumns := []TableColumn{
		{Title: "CONTAINER ID", Width: 14},
		{Title: "NAMES", Width: 20},
		{Title: "IMAGE", Width: 30},
		{Title: "COMMAND", Width: 25},
		{Title: "CREATED", Width: 16},
		{Title: "STATUS", Width: 25},
		{Title: "PORTS", Width: 50},
	}
	scrollTable := NewScrollableTable(scrollColumns)
	
	return &ContainerListView{
		dockerClient: dockerClient,
		tableModel:   t,
		scrollTable:  scrollTable,
		keys:         DefaultKeyMap(),
		searchQuery:  "",
		isSearching:  false,
		filterType:   "all",
		editView:     NewContainerEditView(),
		errorDialog:  NewErrorDialog(),
		jsonViewer:   NewJSONViewer(),
	}
}

// Init 初始化容器列表视图
func (v *ContainerListView) Init() tea.Cmd {
	// 加载容器列表数据，并启动事件监听（E2.2 + E3.1）
	v.loading = true
	return tea.Batch(
		v.loadContainers,
		v.watchDockerEvents(), // 仅使用事件驱动，移除定时轮询
	)
}

// Update 处理消息并更新视图状态
func (v *ContainerListView) Update(msg tea.Msg) (View, tea.Cmd) {
	// 如果显示 JSON 查看器，优先处理
	if v.jsonViewer != nil && v.jsonViewer.IsVisible() {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if v.jsonViewer.Update(keyMsg) {
				return v, nil
			}
		}
	}

	switch msg := msg.(type) {
	case containersLoadedMsg:
		// 容器列表加载完成
		v.containers = msg.containers
		v.loading = false
		v.errorMsg = ""
		v.lastRefreshTime = time.Now()
		
		// 应用过滤和搜索
		v.applyFilters()
		
		// 根据数据内容更新列宽，然后渲染表格
		v.updateColumnWidths()
		
		return v, nil
		
	case containersLoadErrorMsg:
		// 容器列表加载失败
		v.loading = false
		v.errorMsg = msg.err.Error()
		return v, nil
		
	case containerEventMsg:
		// Docker 容器事件（E2.3 - 增量更新）
		event := msg.event
		
		// 根据事件类型处理
		switch event.Action {
		case "start", "die", "stop", "rename":
			// 容器状态变化，重新加载列表
			return v, tea.Batch(v.loadContainers, v.watchDockerEvents())
			
		case "create":
			// 新容器创建，重新加载列表
			return v, tea.Batch(v.loadContainers, v.watchDockerEvents())
			
		case "destroy":
			// 容器删除，重新加载列表
			return v, tea.Batch(v.loadContainers, v.watchDockerEvents())
		}
		// 其他事件，继续监听
		return v, v.watchDockerEvents()
		
	case containerEventErrorMsg:
		// 事件监听错误，记录错误信息但不影响正常使用
		// 这里可以记录日志或显示提示
		// fmt.Println("事件监听错误:", msg.err)
		// 尝试重新启动监听
		return v, v.watchDockerEvents()
		
	case containerOperationSuccessMsg:
		// 容器操作成功，显示成功消息并刷新列表
		v.successMsg = fmt.Sprintf("✅ %s容器成功: %s", msg.operation, msg.container)
		v.successMsgTime = time.Now()
		v.errorMsg = "" // 清除错误消息
		return v, tea.Batch(
			v.loadContainers,
			v.clearSuccessMessageAfter(3 * time.Second),
		)
		
	case containerOperationErrorMsg:
		// 容器操作失败，显示错误弹窗
		errMsg := fmt.Sprintf("%s失败 (%s): %v", msg.operation, msg.container, msg.err)
		if v.errorDialog != nil {
			v.errorDialog.ShowError(errMsg)
		}
		v.successMsg = "" // 清除成功消息
		return v, nil
	
	case containerOperationWarningMsg:
		// 容器操作警告，显示为成功消息样式（黄色/橙色提示）
		v.successMsg = "⚠️ " + msg.message
		v.successMsgTime = time.Now()
		v.errorMsg = "" // 清除错误消息
		return v, v.clearSuccessMessageAfter(3 * time.Second)
		
	case clearSuccessMessageMsg:
		// 清除成功消息
		if time.Since(v.successMsgTime) >= 3*time.Second {
			v.successMsg = ""
		}
		return v, nil
	
	case containerInspectMsg:
		// 显示 JSON 查看器
		if v.jsonViewer != nil {
			v.jsonViewer.SetSize(v.width, v.height)
			v.jsonViewer.Show("Container Inspect: "+msg.containerName, msg.jsonContent)
		}
		return v, nil

	case containerInspectErrorMsg:
		if v.errorDialog != nil {
			v.errorDialog.ShowError(fmt.Sprintf("获取容器信息失败: %v", msg.err))
		}
		return v, nil

	case containerEditReadyMsg:
		// 容器详情获取成功，显示编辑视图
		if v.editView != nil {
			v.editView.Show(msg.container, msg.details)
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
				// 用户确认修改
				return v, v.updateContainerConfig()
			}
			if handled {
				return v, cmd
			}
		}
		
		// 优先处理确认对话框的按键
		if v.showConfirmDialog {
			// 检测所有可能的方向键表示方式
			switch msg.Type {
			case tea.KeyLeft, tea.KeyRight, tea.KeyTab:
				// 方向键和 Tab 切换选择
				v.confirmSelection = 1 - v.confirmSelection
				return v, nil
			case tea.KeyEnter:
				// 确认选择
				if v.confirmSelection == 1 {
					// 选择了 OK，执行操作
					action := v.confirmAction
					container := v.confirmContainer
					
					// 重置对话框状态
					v.showConfirmDialog = false
					v.confirmAction = ""
					v.confirmContainer = nil
					v.confirmSelection = 0
					
					// 执行操作
					if action == "remove" && container != nil {
						return v, v.removeContainer(container)
					}
				} else {
					// 选择了 Cancel，取消操作
					v.showConfirmDialog = false
					v.confirmAction = ""
					v.confirmContainer = nil
					v.confirmSelection = 0
				}
				return v, nil
			case tea.KeyEsc:
				// ESC 直接取消
				v.showConfirmDialog = false
				v.confirmAction = ""
				v.confirmContainer = nil
				v.confirmSelection = 0
				return v, nil
			case tea.KeyRunes:
				// 处理字符按键 h/l
				keyStr := msg.String()
				if keyStr == "h" || keyStr == "l" {
					v.confirmSelection = 1 - v.confirmSelection
					return v, nil
				}
			}
			// 在对话框模式下，忽略其他按键
			return v, nil
		}
		
		// 优先处理 ESC 键（清除搜索/筛选或返回）
		if msg.String() == "esc" {
			if v.isSearching {
				// 如果在搜索模式，退出搜索
				v.isSearching = false
				v.searchQuery = ""
				v.applyFilters()
				v.updateColumnWidths()
				return v, nil
			}
			// 如果有搜索词，先清除搜索
			if v.searchQuery != "" {
				v.searchQuery = ""
				v.applyFilters()
				v.updateColumnWidths()
				return v, nil
			}
			// 如果有筛选条件，先清除筛选
			if v.filterType != "all" {
				v.filterType = "all"
				v.applyFilters()
				v.updateColumnWidths()
				return v, nil
			}
			// 没有搜索和筛选条件，发送 GoBackMsg 请求返回上一级
			return v, func() tea.Msg { return GoBackMsg{} }
		}
		
		// 如果处于搜索模式，处理搜索输入
		if v.isSearching {
			switch msg.String() {
			case "enter":
				// 确认搜索
				v.isSearching = false
				return v, nil
			case "backspace":
				// 删除字符
				if len(v.searchQuery) > 0 {
					v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]
					v.applyFilters()
					v.updateColumnWidths()
				}
				return v, nil
			default:
				// 添加字符到搜索查询
				if len(msg.String()) == 1 {
					v.searchQuery += msg.String()
					v.applyFilters()
					v.updateColumnWidths()
				}
				return v, nil
			}
		}
		
		// 使用 bubbles/key 处理快捷键（R3）
		switch {
		case key.Matches(msg, v.keys.Refresh):
			// 手动刷新列表（E3.2 - 保留手动刷新）
			v.loading = true
			v.errorMsg = "" // 清除错误信息
			return v, v.loadContainers
		case msg.String() == "f":
			// 切换筛选状态：all -> running -> exited -> paused -> all
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
			// 进入搜索模式（L4.2）
			v.isSearching = true
			v.searchQuery = ""
			return v, nil
		case msg.String() == "left", msg.String() == "h":
			// 水平向左滚动
			if v.scrollTable != nil {
				v.scrollTable.ScrollLeft()
			}
			return v, nil
		case msg.String() == "right", msg.String() == "l":
			// 水平向右滚动
			if v.scrollTable != nil {
				v.scrollTable.ScrollRight()
			}
			return v, nil
		case msg.String() == "j", msg.String() == "down":
			// 向下移动
			if v.scrollTable != nil {
				v.scrollTable.MoveDown(1)
			}
			v.tableModel.MoveDown(1)
			return v, nil
		case msg.String() == "k", msg.String() == "up":
			// 向上移动
			if v.scrollTable != nil {
				v.scrollTable.MoveUp(1)
			}
			v.tableModel.MoveUp(1)
			return v, nil
		case msg.String() == "g":
			// 跳转到顶部
			if v.scrollTable != nil {
				v.scrollTable.GotoTop()
			}
			v.tableModel.GotoTop()
			return v, nil
		case msg.String() == "G":
			// 跳转到底部
			if v.scrollTable != nil {
				v.scrollTable.GotoBottom()
			}
			v.tableModel.GotoBottom()
			return v, nil
		case msg.String() == "t":
			// 启动容器（Start）
			return v, v.startSelectedContainer()
		case msg.String() == "p":
			// 停止容器（Stop）
			return v, v.stopSelectedContainer()
		case msg.String() == "P":
			// 暂停/恢复容器（Pause/Unpause）- 大写 P
			return v, v.togglePauseContainer()
		case msg.String() == "R":
			// 重启容器（Restart）- 大写 R
			return v, v.restartSelectedContainer()
		case msg.String() == "ctrl+d":
			// 删除容器（Delete）- Ctrl+D
			return v, v.showRemoveConfirmDialog()
		case msg.String() == "e":
			// 编辑容器配置（Edit）
			return v, v.showEditView()
		case msg.String() == "i":
			// 检查容器（显示 JSON）
			return v, v.inspectContainer()
		default:
			// 其他按键交给 table 处理
			v.tableModel, _ = v.tableModel.Update(msg)
			return v, nil
		}
	}
	
	return v, nil
}

// View 渲染容器列表视图
func (v *ContainerListView) View() string {
	// 如果显示 JSON 查看器
	if v.jsonViewer != nil && v.jsonViewer.IsVisible() {
		return v.jsonViewer.View()
	}

	var s string
	
	// 顶部状态栏和操作提示
	s += v.renderStatusBar()
	
	// 显示成功消息（如果有）
	if v.successMsg != "" {
		// 根据消息类型选择颜色
		msgStyle := successMsgStyle
		if strings.HasPrefix(v.successMsg, "⚠️") {
			// 警告消息使用黄色
			msgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Bold(true)
		}
		s += "\n  " + msgStyle.Render(v.successMsg) + "\n"
	}
	
	// 资源标题栏：使用 lipgloss 自适应窗口宽度
	totalCount := len(v.containers)
	showingCount := len(v.filteredContainers)
	
	// 统计各状态容器数量
	runningCount := 0
	stoppedCount := 0
	for _, c := range v.containers {
		if c.State == "running" {
			runningCount++
		} else {
			stoppedCount++
		}
	}
	
	// 构建统计信息
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
	
	// 搜索附加信息
	if showingCount != totalCount || (!v.isSearching && v.searchQuery != "") {
		filterParts := []string{}
		if showingCount != totalCount {
			filterParts = append(filterParts, fmt.Sprintf("Showing: %d", showingCount))
		}
		if !v.isSearching && v.searchQuery != "" {
			filterParts = append(filterParts, fmt.Sprintf("Search: \"%s\"", v.searchQuery))
		}
		filterInfo := searchHintStyle.Render("  [" + strings.Join(filterParts, " | ") + "]")
		statsContent += filterInfo
	}
	
	// 计算分隔线宽度（与表格宽度一致）
	lineWidth := v.width - 6
	if lineWidth < 60 {
		lineWidth = 60
	}
	line := lineStyle.Render(strings.Repeat("─", lineWidth))
	
	// 居中显示统计信息
	statsLine := lipgloss.NewStyle().Width(lineWidth).Align(lipgloss.Center).Render(statsContent)
	
	s += "\n  " + line + "\n"
	s += "  " + statsLine + "\n"
	s += "  " + line + "\n"
	
	// 加载中状态
	if v.loading {
		loadingContent := lipgloss.JoinVertical(lipgloss.Center,
			"",
			statusBarKeyStyle.Render("⏳ 正在加载容器列表..."),
			"",
			searchHintStyle.Render("请稍候，正在从 Docker 获取数据"),
			"",
		)
		s += "\n  " + stateBoxStyle.Render(loadingContent) + "\n"
		return s
	}
	
	// 错误状态 - 没有容器数据时显示阻塞式错误框（无法关闭）
	if v.errorMsg != "" && len(v.containers) == 0 {
		// 分割错误信息，支持多行显示
		errLines := []string{""}
		errText := v.errorMsg
		// 移除开头的 ❌ 符号（如果有的话，因为我们会重新添加）
		errText = strings.TrimPrefix(errText, "❌ ")
		
		// 按 80 字符换行
		maxLineLen := 70
		for len(errText) > maxLineLen {
			errLines = append(errLines, errorMsgStyle.Render(errText[:maxLineLen]))
			errText = errText[maxLineLen:]
		}
		if errText != "" {
			errLines = append(errLines, errorMsgStyle.Render(errText))
		}
		
		errLines = append(errLines,
			"",
			statusBarKeyStyle.Render("按 r 重新加载") + searchHintStyle.Render(" 或 ") + statusBarKeyStyle.Render("按 Esc 返回"),
			"",
		)
		
		errorContent := lipgloss.JoinVertical(lipgloss.Left, errLines...)
		s += "\n  " + stateBoxStyle.Width(v.width - 10).Render(errorContent) + "\n"
		return s
	}
	
	// 空状态 - 无容器
	if len(v.containers) == 0 {
		emptyContent := lipgloss.JoinVertical(lipgloss.Left,
			"",
			searchHintStyle.Render("📦 暂无容器"),
			"",
			statusBarLabelStyle.Render("💡 快速开始:"),
			"",
			statusBarKeyStyle.Render("1.") + searchHintStyle.Render(" 启动一个测试容器:"),
			searchHintStyle.Render("   docker run -d --name test nginx"),
			"",
			statusBarKeyStyle.Render("2.") + searchHintStyle.Render(" 刷新容器列表:"),
			searchHintStyle.Render("   按 r 键刷新"),
			"",
			searchHintStyle.Render("提示: 容器列表会自动刷新（事件驱动模式）"),
			"",
		)
		s += "\n  " + stateBoxStyle.Render(emptyContent) + "\n"
		return s
	}
	
	// 空状态 - 过滤后无结果
	if len(v.filteredContainers) == 0 {
		var filterHints []string
		filterHints = append(filterHints, "", searchHintStyle.Render("🔍 没有匹配的容器"), "")
		filterHints = append(filterHints, statusBarLabelStyle.Render("当前搜索条件:"))
		if v.searchQuery != "" {
			filterHints = append(filterHints, searchHintStyle.Render("   • 搜索关键字: ")+statusBarKeyStyle.Render("\""+v.searchQuery+"\""))
		}
		filterHints = append(filterHints, "", statusBarLabelStyle.Render("💡 操作提示:"))
		if v.searchQuery != "" {
			filterHints = append(filterHints, searchHintStyle.Render("   • 按 ")+statusBarKeyStyle.Render("ESC")+searchHintStyle.Render(" 清除搜索"))
		} else {
			filterHints = append(filterHints, searchHintStyle.Render("   • 按 ")+statusBarKeyStyle.Render("/")+searchHintStyle.Render(" 开始搜索"))
		}
		filterHints = append(filterHints, searchHintStyle.Render("   • 按 ")+statusBarKeyStyle.Render("r")+searchHintStyle.Render(" 刷新列表"), "")
		
		emptyFilterContent := lipgloss.JoinVertical(lipgloss.Left, filterHints...)
		s += "\n  " + stateBoxStyle.Render(emptyFilterContent) + "\n"
		return s
	}
	
	// 使用可滚动表格渲染
	if v.scrollTable != nil {
		s += v.scrollTable.View() + "\n"
	} else {
		// 回退到 bubbles/table 组件
		s += "  " + v.tableModel.View() + "\n"
	}
	
	// 添加空行填充，确保清除之前可能残留的加载提示
	// 这是为了解决终端渲染时旧内容残留的问题
	s += "\n"
	
	// 底部搜索输入栏（如果处于搜索模式）
	if v.isSearching {
		searchLine := "\n  " + strings.Repeat("─", 67) + "\n"
		searchPrompt := "  " + searchPromptStyle.Render("Search:") + " "
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		searchInput := v.searchQuery + cursor
		cancelHint := searchHintStyle.Render("[Enter=Confirm | ESC=Cancel]")
		
		// 计算填充空格
		totalWidth := 70
		usedWidth := 10 + len(v.searchQuery) + 1 + 28
		padding := ""
		if totalWidth > usedWidth {
			padding = strings.Repeat(" ", totalWidth-usedWidth)
		}
		
		s += searchLine + searchPrompt + searchInput + padding + cancelHint + "\n"
	}
	
	// 底部左下角筛选状态提示（非搜索模式时显示）
	if !v.isSearching && v.filterType != "all" {
		filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
		s += "  " + filterStyle.Render("[Filter: "+v.filterType+"]") + "  " + searchHintStyle.Render("按 ESC 清除筛选，按 f 切换") + "\n"
	}
	
	// 如果显示确认对话框，叠加在内容上
	if v.showConfirmDialog {
		s = v.overlayDialog(s)
	}
	
	// 如果显示编辑视图，叠加在内容上
	if v.editView != nil && v.editView.IsVisible() {
		s = v.overlayEditView(s)
	}
	
	// 如果显示错误弹窗，叠加在内容上
	if v.errorDialog != nil && v.errorDialog.IsVisible() {
		s = v.errorDialog.Overlay(s)
	}
	
	return s
}

// overlayDialog 将对话框叠加到现有内容上（居中显示）
func (v *ContainerListView) overlayDialog(baseContent string) string {
	// 将基础内容按行分割
	lines := strings.Split(baseContent, "\n")
	
	// 对话框尺寸
	dialogHeight := 9
	
	// 计算对话框应该插入的位置（垂直居中）
	insertLine := 0
	if len(lines) > dialogHeight {
		insertLine = (len(lines) - dialogHeight) / 2
	}
	
	// 获取对话框内容（不包含顶部填充）
	dialogContent := v.renderConfirmDialogContent()
	dialogLines := strings.Split(dialogContent, "\n")
	
	// 构建最终输出
	var result strings.Builder
	
	for i := 0; i < len(lines); i++ {
		dialogIdx := i - insertLine
		if dialogIdx >= 0 && dialogIdx < len(dialogLines) {
			// 在这个位置显示对话框行
			result.WriteString(dialogLines[dialogIdx])
		} else if i < len(lines) {
			// 显示原始内容
			result.WriteString(lines[i])
		}
		
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// renderConfirmDialogContent 渲染对话框内容（使用 lipgloss）
func (v *ContainerListView) renderConfirmDialogContent() string {
	if v.confirmContainer == nil {
		return ""
	}

	// 定义样式 - 不硬编码背景色
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(56)
	
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	cancelBtnStyle := lipgloss.NewStyle().
		Padding(0, 2)
	
	okBtnStyle := lipgloss.NewStyle().
		Padding(0, 2)
	
	// 根据选择状态设置按钮样式 - 使用 Reverse 实现选中效果
	if v.confirmSelection == 0 {
		// Cancel 被选中
		cancelBtnStyle = cancelBtnStyle.
			Reverse(true).
			Bold(true)
		okBtnStyle = okBtnStyle.
			Foreground(lipgloss.Color("245"))
	} else {
		// OK 被选中
		cancelBtnStyle = cancelBtnStyle.
			Foreground(lipgloss.Color("245"))
		okBtnStyle = okBtnStyle.
			Reverse(true).
			Bold(true)
	}
	
	// 容器名称（截断）
	containerName := v.confirmContainer.Name
	if len(containerName) > 35 {
		containerName = containerName[:32] + "..."
	}
	
	// 根据容器状态显示不同的警告
	warningText := "This action cannot be undone!"
	if v.confirmContainer.State == "running" {
		warningText = "⚠️  容器正在运行，将强制删除！"
	}
	
	// 构建对话框内容
	title := titleStyle.Render("⚠️  Delete Container: " + containerName)
	warning := warningStyle.Render(warningText)
	
	cancelBtn := cancelBtnStyle.Render("< Cancel >")
	okBtn := okBtnStyle.Render("< OK >")
	buttons := cancelBtn + "    " + okBtn
	
	// 居中按钮
	buttonsLine := lipgloss.NewStyle().Width(52).Align(lipgloss.Center).Render(buttons)
	
	content := title + "\n\n" + warning + "\n\n" + buttonsLine
	dialog := dialogStyle.Render(content)
	
	// 水平居中
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

// renderStatusBar 渲染顶部状态栏（简化版，背景由全局处理）
func (v *ContainerListView) renderStatusBar() string {
	// 确保有最小宽度
	width := v.width
	if width < 80 {
		width = 80
	}
	
	availableWidth := width - 4
	if availableWidth < 60 {
		availableWidth = 60
	}
	
	// 计算列宽：左侧标签列 + 右侧快捷键区域
	labelColWidth := 20
	shortcutsWidth := availableWidth - labelColWidth
	
	// 根据宽度决定每行显示几个快捷键
	itemsPerRow := 4
	if shortcutsWidth < 60 {
		itemsPerRow = 3
	}
	if shortcutsWidth < 45 {
		itemsPerRow = 2
	}
	
	// 计算每个快捷键项的宽度
	itemWidth := shortcutsWidth / itemsPerRow
	if itemWidth < 12 {
		itemWidth = 12
	}
	
	// 定义样式（不再单独设置背景，由全局统一处理）
	labelStyle := lipgloss.NewStyle().
		Width(labelColWidth).
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))
	
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	itemStyle := lipgloss.NewStyle().
		Width(itemWidth)
	
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	// 构建快捷键项
	makeItem := func(key, desc string) string {
		return itemStyle.Render(keyStyle.Render(key) + descStyle.Render(" "+desc))
	}
	
	var lines []string
	
	// 第一行：Docker 状态 + 基本操作
	row1Label := labelStyle.Render("📦 Containers")
	row1Keys := makeItem("<f>", "Filter") + makeItem("</>", "Search") + makeItem("<r>", "Refresh")
	lines = append(lines, "  "+row1Label+row1Keys)
	
	// 第二行：容器操作
	row2Label := labelStyle.Render("Ops:")
	row2Keys := makeItem("<t>", "Start") + makeItem("<p>", "Stop") + makeItem("<P>", "Pause") + makeItem("<R>", "Restart")
	lines = append(lines, "  "+row2Label+row2Keys)
	
	// 第三行：高级操作
	row3Label := labelStyle.Render("Advanced:")
	row3Keys := makeItem("<Ctrl+D>", "Delete") + makeItem("<e>", "Edit") + makeItem("<i>", "Inspect") + makeItem("<l>", "Logs")
	lines = append(lines, "  "+row3Label+row3Keys)
	
	// 第四行：刷新时间 + vim 提示
	refreshInfo := "-"
	if !v.lastRefreshTime.IsZero() {
		refreshInfo = formatDuration(time.Since(v.lastRefreshTime)) + " ago"
	}
	
	row4Label := labelStyle.Render("Last Refresh:")
	row4Info := hintStyle.Render(refreshInfo) + "    " + 
		hintStyle.Render("j/k=上下  Enter=详情  Esc=返回  q=退出")
	lines = append(lines, "  "+row4Label+row4Info)
	
	return "\n" + strings.Join(lines, "\n") + "\n"
}

// renderCompactStatusBar 渲染紧凑版状态栏（窄屏模式，已废弃，统一使用自适应）
func (v *ContainerListView) renderCompactStatusBar() string {
	// 现在统一使用 renderStatusBar 的自适应逻辑
	return v.renderStatusBar()
}

// containersToRows 将容器数据转换为 table.Row
func (v *ContainerListView) containersToRows(containers []docker.Container) []table.Row {
	rows := make([]table.Row, len(containers))
	
	// 定义整行颜色样式
	exitedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))    // 灰色 - 已停止
	pausedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))    // 黄色 - 暂停
	unhealthyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // 红色 - 不健康
	
	for i, c := range containers {
		// CREATED - 友好格式
		created := formatCreatedTime(c.Created)
		
		// PORTS - 如果为空显示空字符串
		ports := c.Ports
		if ports == "" {
			ports = ""
		}
		
		// 根据状态决定是否对整行应用颜色
		var rowStyle lipgloss.Style
		var needsStyle bool
		
		switch {
		case strings.Contains(strings.ToLower(c.Status), "unhealthy"):
			// 不健康 - 红色整行
			rowStyle = unhealthyStyle
			needsStyle = true
		case c.State == "paused":
			// 暂停 - 黄色整行
			rowStyle = pausedStyle
			needsStyle = true
		case c.State == "exited":
			// 已停止 - 灰色整行
			rowStyle = exitedStyle
			needsStyle = true
		default:
			// 运行中或健康 - 不应用样式
			needsStyle = false
		}
		
		// 构建行数据
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

// formatCreatedTime 格式化创建时间为友好格式（如 "11 hours ago"）
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

// formatDuration 格式化时间差（辅助函数）
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

// truncateForBox 截断字符串以适应盒子宽度
func (v *ContainerListView) truncateForBox(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// SetSize 设置视图尺寸并调整表格大小
func (v *ContainerListView) SetSize(width, height int) {
	v.width = width
	v.height = height
	
	// 调整表格高度
	tableHeight := height - 15 // 减去状态栏、统计栏、滚动指示器等
	if tableHeight < 5 {
		tableHeight = 5
	}
	v.tableModel.SetHeight(tableHeight)
	
	// 更新可滚动表格尺寸
	if v.scrollTable != nil {
		v.scrollTable.SetSize(width-4, tableHeight)
	}
	
	// 更新编辑视图宽度
	if v.editView != nil {
		v.editView.SetWidth(width)
	}
	
	// 更新错误弹窗宽度
	if v.errorDialog != nil {
		v.errorDialog.SetWidth(width)
	}
	
	// 根据实际数据内容计算最优列宽
	v.updateColumnWidths()
	
	// 更新状态框样式的宽度
	stateBoxStyle = stateBoxStyle.Width(width - 10)
}

// updateColumnWidths 根据实际数据计算并更新列宽
func (v *ContainerListView) updateColumnWidths() {
	// 计算每列内容的最大宽度
	maxID := 12       // CONTAINER ID 固定 12 位
	maxImage := 5     // IMAGE
	maxCommand := 7   // COMMAND
	maxCreated := 7   // CREATED
	maxStatus := 6    // STATUS
	maxPorts := 5     // PORTS
	maxNames := 5     // NAMES
	
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
	
	// 只有 STATUS 列需要 ANSI 转义码补偿（因为只有这列有颜色）
	// lipgloss 颜色码约 20 字符：\x1b[38;5;XXXm (11) + \x1b[0m (4) + 额外缓冲
	statusAnsiPadding := 20
	
	// 可用宽度
	availableWidth := v.width - 10
	
	// 固定列宽
	idWidth := maxID + 2
	
	// 计算需要的总宽度
	totalNeeded := idWidth + maxImage + maxCommand + maxCreated + (maxStatus + statusAnsiPadding) + maxPorts + maxNames + 14
	
	// 如果总宽度足够，使用实际内容宽度
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
		// 宽度不够，按比例分配
		flexWidth := availableWidth - idWidth - statusAnsiPadding - 6
		
		// 按内容比例分配
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
		
		// 确保最小宽度
		if imageWidth < 15 {
			imageWidth = 15
		}
		if commandWidth < 12 {
			commandWidth = 12
		}
		if createdWidth < 12 {
			createdWidth = 12
		}
		if statusWidth < 15 + statusAnsiPadding {
			statusWidth = 15 + statusAnsiPadding
		}
		if portsWidth < 20 {
			portsWidth = 20
		}
		if namesWidth < 12 {
			namesWidth = 12
		}
		
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
	
	// 更新可滚动表格的列宽和数据（NAME 在第二列）
	if v.scrollTable != nil {
		v.scrollTable.SetColumns([]TableColumn{
			{Title: "CONTAINER ID", Width: maxID + 2},
			{Title: "NAMES", Width: maxNames + 2},
			{Title: "IMAGE", Width: maxImage + 2},
			{Title: "COMMAND", Width: maxCommand + 2},
			{Title: "CREATED", Width: maxCreated + 2},
			{Title: "STATUS", Width: maxStatus + 2},
			{Title: "PORTS", Width: maxPorts + 2},
		})
		
		// 转换数据为 TableRow（NAME 在第二列）
		if len(v.filteredContainers) > 0 {
			rows := make([]TableRow, len(v.filteredContainers))
			
			// 定义整行颜色样式
			exitedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
			pausedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
			unhealthyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			
			for i, c := range v.filteredContainers {
				created := formatCreatedTime(c.Created)
				ports := c.Ports
				if ports == "" {
					ports = "-"
				}
				
				// 根据状态决定是否对整行应用颜色
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
				
				// 构建行数据
				if needsStyle {
					rows[i] = TableRow{
						rowStyle.Render(c.ShortID),
						rowStyle.Render(c.Name),
						rowStyle.Render(c.Image),
						rowStyle.Render(c.Command),
						rowStyle.Render(created),
						rowStyle.Render(c.Status),
						rowStyle.Render(ports),
					}
				} else {
					rows[i] = TableRow{
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
			// 清空表格数据
			v.scrollTable.SetRows([]TableRow{})
		}
	}
	
	// 重新渲染表格数据
	if len(v.filteredContainers) > 0 {
		rows := v.containersToRows(v.filteredContainers)
		v.tableModel.SetRows(rows)
	} else {
		v.tableModel.SetRows([]table.Row{})
	}
}

// GetSelectedContainer 获取当前选中的容器（L3.2）
func (v *ContainerListView) GetSelectedContainer() *docker.Container {
	if len(v.filteredContainers) == 0 {
		return nil
	}
	// 优先从可滚动表格获取选中索引
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
func (v *ContainerListView) IsSearching() bool {
	return v.isSearching
}

// applyFilters 应用搜索和状态过滤
func (v *ContainerListView) applyFilters() {
	v.filteredContainers = make([]docker.Container, 0)
	
	for _, container := range v.containers {
		// 应用状态过滤
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
		
		// 应用搜索过滤
		if v.searchQuery != "" {
			// 搜索容器名称、镜像名称、ID
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

// containersLoadedMsg 容器列表加载完成消息
type containersLoadedMsg struct {
	containers []docker.Container
}

// containersLoadErrorMsg 容器列表加载错误消息
type containersLoadErrorMsg struct {
	err error
}

// containerEventMsg Docker 容器事件消息（E2.1）
type containerEventMsg struct {
	event docker.ContainerEvent
}

// containerEventErrorMsg Docker 事件监听错误消息
type containerEventErrorMsg struct {
	err error
}

// containerOperationSuccessMsg 容器操作成功消息
type containerOperationSuccessMsg struct {
	operation string // 操作类型: start, stop, restart
	container string // 容器名称
}

// containerOperationErrorMsg 容器操作失败消息
type containerOperationErrorMsg struct {
	operation string // 操作类型
	container string // 容器名称
	err       error  // 错误信息
}

// containerOperationWarningMsg 容器操作警告消息（非严重错误）
type containerOperationWarningMsg struct {
	message string // 警告消息
}

// clearSuccessMessageMsg 清除成功消息
type clearSuccessMessageMsg struct{}

// containerInspectMsg 容器检查结果消息
type containerInspectMsg struct {
	containerName string
	jsonContent   string
}

// containerInspectErrorMsg 容器检查错误消息
type containerInspectErrorMsg struct {
	err error
}

// loadContainers 加载容器列表（返回 tea.Cmd）
func (v *ContainerListView) loadContainers() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// 调用 Docker 客户端获取所有容器（包括已停止的）
	containers, err := v.dockerClient.ListContainers(ctx, true)
	if err != nil {
		return containersLoadErrorMsg{err: err}
	}
	
	return containersLoadedMsg{containers: containers}
}

// watchDockerEvents 监听 Docker 容器事件（E2.2）
func (v *ContainerListView) watchDockerEvents() tea.Cmd {
	return func() tea.Msg {
		// 创建 context
		ctx := context.Background()
		
		// 启动事件监听
		eventChan, errorChan := v.dockerClient.WatchEvents(ctx)
		
		// 等待第一个事件或错误
		select {
		case event, ok := <-eventChan:
			if !ok {
				// 通道关闭
				return containerEventErrorMsg{err: fmt.Errorf("事件通道关闭")}
			}
			// 返回事件消息，并继续监听
			return containerEventMsg{event: event}
			
		case err, ok := <-errorChan:
			if !ok || err == nil {
				return nil
			}
			return containerEventErrorMsg{err: err}
		}
	}
}

// startSelectedContainer 启动选中的容器
func (v *ContainerListView) startSelectedContainer() tea.Cmd {
	container := v.GetSelectedContainer()
	if container == nil {
		return func() tea.Msg {
			return containerOperationErrorMsg{
				operation: "启动容器",
				container: "",
				err:       fmt.Errorf("请先选择一个容器"),
			}
		}
	}

	// 检查容器状态
	if container.State == "running" {
		return func() tea.Msg {
			return containerOperationWarningMsg{
				message: fmt.Sprintf("容器 %s 已在运行中", container.Name),
			}
		}
	}

	// 执行启动操作
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := v.dockerClient.StartContainer(ctx, container.ID)
		if err != nil {
			return containerOperationErrorMsg{
				operation: "启动容器",
				container: container.Name,
				err:       err,
			}
		}

		return containerOperationSuccessMsg{
			operation: "启动",
			container: container.Name,
		}
	}
}

// stopSelectedContainer 停止选中的容器
func (v *ContainerListView) stopSelectedContainer() tea.Cmd {
	container := v.GetSelectedContainer()
	if container == nil {
		return func() tea.Msg {
			return containerOperationErrorMsg{
				operation: "停止容器",
				container: "",
				err:       fmt.Errorf("请先选择一个容器"),
			}
		}
	}

	// 检查容器状态
	if container.State != "running" {
		return func() tea.Msg {
			return containerOperationWarningMsg{
				message: fmt.Sprintf("容器 %s 未在运行", container.Name),
			}
		}
	}

	// 执行停止操作（10秒超时）
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := v.dockerClient.StopContainer(ctx, container.ID, 10)
		if err != nil {
			return containerOperationErrorMsg{
				operation: "停止容器",
				container: container.Name,
				err:       err,
			}
		}

		return containerOperationSuccessMsg{
			operation: "停止",
			container: container.Name,
		}
	}
}

// restartSelectedContainer 重启选中的容器
func (v *ContainerListView) restartSelectedContainer() tea.Cmd {
	container := v.GetSelectedContainer()
	if container == nil {
		return func() tea.Msg {
			return containerOperationErrorMsg{
				operation: "重启容器",
				container: "",
				err:       fmt.Errorf("请先选择一个容器"),
			}
		}
	}

	// 执行重启操作（10秒超时）
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := v.dockerClient.RestartContainer(ctx, container.ID, 10)
		if err != nil {
			return containerOperationErrorMsg{
				operation: "重启容器",
				container: container.Name,
				err:       err,
			}
		}

		return containerOperationSuccessMsg{
			operation: "重启",
			container: container.Name,
		}
	}
}

// showRemoveConfirmDialog 显示删除确认对话框
func (v *ContainerListView) showRemoveConfirmDialog() tea.Cmd {
	container := v.GetSelectedContainer()
	if container == nil {
		return func() tea.Msg {
			return containerOperationErrorMsg{
				operation: "删除容器",
				container: "",
				err:       fmt.Errorf("请先选择一个容器"),
			}
		}
	}

	// 显示确认对话框（不管容器状态，让用户看到信息后决定）
	v.showConfirmDialog = true
	v.confirmAction = "remove"
	v.confirmContainer = container
	v.confirmSelection = 0 // 默认选中 Cancel

	return nil
}

// removeContainer 删除容器
func (v *ContainerListView) removeContainer(container *docker.Container) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 如果容器正在运行，使用强制删除
		force := container.State == "running"

		// 删除容器
		err := v.dockerClient.RemoveContainer(ctx, container.ID, force, false)
		if err != nil {
			return containerOperationErrorMsg{
				operation: "删除容器",
				container: container.Name,
				err:       err,
			}
		}

		return containerOperationSuccessMsg{
			operation: "删除",
			container: container.Name,
		}
	}
}

// clearSuccessMessageAfter 在指定时间后清除成功消息
func (v *ContainerListView) clearSuccessMessageAfter(duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(duration)
		return clearSuccessMessageMsg{}
	}
}

// togglePauseContainer 暂停/恢复选中的容器
func (v *ContainerListView) togglePauseContainer() tea.Cmd {
	container := v.GetSelectedContainer()
	if container == nil {
		return func() tea.Msg {
			return containerOperationErrorMsg{
				operation: "暂停/恢复容器",
				container: "",
				err:       fmt.Errorf("请先选择一个容器"),
			}
		}
	}

	// 检查容器状态
	if container.State != "running" && container.State != "paused" {
		return func() tea.Msg {
			return containerOperationWarningMsg{
				message: fmt.Sprintf("容器 %s 状态为 %s，只能暂停运行中的容器或恢复已暂停的容器", container.Name, container.State),
			}
		}
	}

	// 根据当前状态决定是暂停还是恢复
	isPaused := container.State == "paused"

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		var operation string

		if isPaused {
			// 恢复容器
			err = v.dockerClient.UnpauseContainer(ctx, container.ID)
			operation = "恢复"
		} else {
			// 暂停容器
			err = v.dockerClient.PauseContainer(ctx, container.ID)
			operation = "暂停"
		}

		if err != nil {
			return containerOperationErrorMsg{
				operation: operation + "容器",
				container: container.Name,
				err:       err,
			}
		}

		return containerOperationSuccessMsg{
			operation: operation,
			container: container.Name,
		}
	}
}


// showEditView 显示编辑视图
func (v *ContainerListView) showEditView() tea.Cmd {
	container := v.GetSelectedContainer()
	if container == nil {
		return func() tea.Msg {
			return containerOperationErrorMsg{
				operation: "编辑容器",
				container: "",
				err:       fmt.Errorf("请先选择一个容器"),
			}
		}
	}

	// 获取容器详情
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		details, err := v.dockerClient.ContainerDetails(ctx, container.ID)
		if err != nil {
			return containerOperationErrorMsg{
				operation: "获取容器详情",
				container: container.Name,
				err:       err,
			}
		}

		return containerEditReadyMsg{
			container: container,
			details:   details,
		}
	}
}

// inspectContainer 获取容器的原始 JSON
func (v *ContainerListView) inspectContainer() tea.Cmd {
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
			return containerInspectErrorMsg{err: err}
		}

		return containerInspectMsg{
			containerName: containerName,
			jsonContent:   jsonContent,
		}
	}
}

// containerEditReadyMsg 容器编辑准备就绪消息
type containerEditReadyMsg struct {
	container *docker.Container
	details   *docker.ContainerDetails
}

// containerUpdateSuccessMsg 容器更新成功消息
type containerUpdateSuccessMsg struct {
	container string
}

// updateContainerConfig 更新容器配置
func (v *ContainerListView) updateContainerConfig() tea.Cmd {
	if v.editView == nil {
		return nil
	}

	containerID := v.editView.GetContainerID()
	containerName := v.editView.GetContainerName()
	config := v.editView.GetConfig()

	// 隐藏编辑视图
	v.editView.Hide()

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := v.dockerClient.UpdateContainer(ctx, containerID, config)
		if err != nil {
			return containerOperationErrorMsg{
				operation: "更新容器配置",
				container: containerName,
				err:       err,
			}
		}

		return containerOperationSuccessMsg{
			operation: "更新配置",
			container: containerName,
		}
	}
}

// overlayEditView 将编辑视图叠加到现有内容上
func (v *ContainerListView) overlayEditView(baseContent string) string {
	if v.editView == nil {
		return baseContent
	}

	// 将基础内容按行分割
	lines := strings.Split(baseContent, "\n")

	// 编辑视图尺寸
	editHeight := 16

	// 计算编辑视图应该插入的位置（垂直居中）
	insertLine := 0
	if len(lines) > editHeight {
		insertLine = (len(lines) - editHeight) / 2
	}

	// 获取编辑视图内容
	editContent := v.editView.View()
	editLines := strings.Split(editContent, "\n")

	// 构建最终输出
	var result strings.Builder

	for i := 0; i < len(lines); i++ {
		editIdx := i - insertLine
		if editIdx >= 0 && editIdx < len(editLines) {
			// 在这个位置显示编辑视图行
			result.WriteString(editLines[editIdx])
		} else if i < len(lines) {
			// 显示原始内容
			result.WriteString(lines[i])
		}

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// IsEditViewVisible 返回编辑视图是否可见
func (v *ContainerListView) IsEditViewVisible() bool {
	return v.editView != nil && v.editView.IsVisible()
}

// HasError 返回是否有错误信息显示
func (v *ContainerListView) HasError() bool {
	return v.errorDialog != nil && v.errorDialog.IsVisible()
}
