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

// 网络列表视图样式定义
var (
	networkTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	networkDriverStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	networkBuiltInStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	networkCustomStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	networkStateBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(66)
)

// NetworkSortField 网络排序字段
type NetworkSortField int

const (
	SortByNetworkName NetworkSortField = iota
	SortByNetworkDriver
	SortByNetworkCreated
	SortByNetworkContainers
)

// NetworkListView 网络列表视图
type NetworkListView struct {
	dockerClient docker.Client

	// UI 尺寸
	width  int
	height int

	// 数据状态
	networks         []docker.Network // 网络列表数据（原始）
	filteredNetworks []docker.Network // 过滤后的网络列表
	scrollTable      *ScrollableTable // 可水平滚动的表格
	loading          bool             // 是否正在加载
	errorMsg         string           // 错误信息
	successMsg       string           // 成功消息
	successMsgTime   time.Time        // 成功消息显示时间

	// 搜索状态
	searchQuery string // 搜索关键字
	isSearching bool   // 是否处于搜索模式

	// 筛选状态
	filterDriver      string           // 按驱动筛选: "all", "bridge", "host", "overlay", "macvlan", "none"
	filterDriverIndex int              // 筛选驱动索引
	showFilterMenu    bool             // 是否显示筛选菜单

	// 排序状态
	sortField     NetworkSortField // 排序字段
	sortAscending bool             // 是否升序

	// 刷新状态
	lastRefreshTime time.Time // 上次刷新时间

	// 确认对话框状态
	showConfirmDialog bool            // 是否显示确认对话框
	confirmAction     string          // 确认的操作类型: "remove", "prune"
	confirmNetwork    *docker.Network // 待操作的网络
	confirmSelection  int             // 确认对话框中的选择: 0=Cancel, 1=OK

	// 创建网络视图
	createView     *NetworkCreateView // 创建网络视图
	showCreateView bool               // 是否显示创建视图

	// JSON 查看器
	jsonViewer *JSONViewer // JSON 查看器

	// 错误弹窗
	errorDialog *ErrorDialog
}

// NewNetworkListView 创建网络列表视图
func NewNetworkListView(dockerClient docker.Client) *NetworkListView {
	// 创建可滚动表格
	columns := []TableColumn{
		{Title: "NETWORK ID", Width: 14},
		{Title: "NAME", Width: 25},
		{Title: "DRIVER", Width: 12},
		{Title: "SCOPE", Width: 10},
		{Title: "CONTAINERS", Width: 12},
		{Title: "CREATED", Width: 16},
	}
	scrollTable := NewScrollableTable(columns)

	// 创建网络创建视图
	createView := NewNetworkCreateView(dockerClient)

	return &NetworkListView{
		dockerClient:  dockerClient,
		scrollTable:   scrollTable,
		filterDriver:  "all",
		sortField:     SortByNetworkName,
		sortAscending: true,
		errorDialog:   NewErrorDialog(),
		createView:    createView,
		jsonViewer:    NewJSONViewer(),
	}
}

// Init 初始化网络列表视图
func (v *NetworkListView) Init() tea.Cmd {
	v.loading = true
	return v.loadNetworks
}

// Update 处理消息并更新视图状态
func (v *NetworkListView) Update(msg tea.Msg) (View, tea.Cmd) {
	// 如果显示 JSON 查看器，优先处理
	if v.jsonViewer != nil && v.jsonViewer.IsVisible() {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if v.jsonViewer.Update(keyMsg) {
				return v, nil
			}
		}
	}

	// 如果显示创建视图，优先处理
	if v.showCreateView && v.createView != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			_, cmd := v.createView.Update(msg)
			return v, cmd
		case networkCreateSuccessMsg:
			v.showCreateView = false
			v.successMsg = fmt.Sprintf("✅ 网络创建成功: %s", msg.networkID[:12])
			v.successMsgTime = time.Now()
			v.createView.Reset()
			return v, tea.Batch(
				v.loadNetworks,
				v.clearSuccessMessageAfter(3*time.Second),
			)
		case networkCreateErrorMsg:
			v.createView.Update(msg)
			return v, nil
		}
	}

	switch msg := msg.(type) {
	case networksLoadedMsg:
		v.networks = msg.networks
		v.loading = false
		v.errorMsg = ""
		v.lastRefreshTime = time.Now()
		v.applyFilters()
		v.updateTableData()
		return v, nil

	case networksLoadErrorMsg:
		v.loading = false
		v.errorMsg = msg.err.Error()
		return v, nil

	case networkOperationSuccessMsg:
		v.successMsg = fmt.Sprintf("✅ %s成功: %s", msg.operation, msg.network)
		v.successMsgTime = time.Now()
		v.errorMsg = ""
		return v, tea.Batch(
			v.loadNetworks,
			v.clearSuccessMessageAfter(3*time.Second),
		)

	case networkOperationErrorMsg:
		if v.errorDialog != nil {
			v.errorDialog.ShowError(fmt.Sprintf("%s失败: %v", msg.operation, msg.err))
		}
		return v, nil

	case clearSuccessMessageMsg:
		if time.Since(v.successMsgTime) >= 3*time.Second {
			v.successMsg = ""
		}
		return v, nil

	case networkInspectMsg:
		// 显示 JSON 查看器
		if v.jsonViewer != nil {
			v.jsonViewer.SetSize(v.width, v.height)
			v.jsonViewer.Show("Network Inspect: "+msg.networkName, msg.jsonContent)
		}
		return v, nil

	case networkInspectErrorMsg:
		if v.errorDialog != nil {
			v.errorDialog.ShowError(fmt.Sprintf("获取网络信息失败: %v", msg.err))
		}
		return v, nil

	case tea.KeyMsg:
		// 优先处理错误弹窗
		if v.errorDialog != nil && v.errorDialog.IsVisible() {
			if v.errorDialog.Update(msg) {
				return v, nil
			}
		}

		// 处理确认对话框
		if v.showConfirmDialog {
			return v.handleConfirmDialogKey(msg)
		}

		// 处理筛选菜单
		if v.showFilterMenu {
			return v.handleFilterMenuKey(msg)
		}

		// 处理搜索模式
		if v.isSearching {
			return v.handleSearchKey(msg)
		}

		// 处理普通按键
		return v.handleNormalKey(msg)
	}

	return v, nil
}

// handleConfirmDialogKey 处理确认对话框的按键
func (v *NetworkListView) handleConfirmDialogKey(msg tea.KeyMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "left", "right", "tab", "h", "l":
		v.confirmSelection = 1 - v.confirmSelection
	case "enter":
		if v.confirmSelection == 1 {
			action := v.confirmAction
			network := v.confirmNetwork
			v.resetConfirmDialog()

			if action == "remove" && network != nil {
				return v, v.removeNetwork(network)
			} else if action == "prune" {
				return v, v.pruneNetworks()
			}
		} else {
			v.resetConfirmDialog()
		}
	case "esc":
		v.resetConfirmDialog()
	}
	return v, nil
}

// handleSearchKey 处理搜索模式的按键
func (v *NetworkListView) handleSearchKey(msg tea.KeyMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "enter":
		v.isSearching = false
	case "esc":
		v.isSearching = false
		v.searchQuery = ""
		v.applyFilters()
		v.updateTableData()
	case "backspace":
		if len(v.searchQuery) > 0 {
			v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]
			v.applyFilters()
			v.updateTableData()
		}
	default:
		if len(msg.String()) == 1 {
			v.searchQuery += msg.String()
			v.applyFilters()
			v.updateTableData()
		}
	}
	return v, nil
}

// handleFilterMenuKey 处理筛选菜单的按键
func (v *NetworkListView) handleFilterMenuKey(msg tea.KeyMsg) (View, tea.Cmd) {
	filterOptions := []string{"all", "bridge", "host", "overlay", "macvlan", "none"}

	switch msg.String() {
	case "esc":
		v.showFilterMenu = false
	case "enter":
		v.filterDriver = filterOptions[v.filterDriverIndex]
		v.showFilterMenu = false
		v.applyFilters()
		v.updateTableData()
	case "j", "down":
		if v.filterDriverIndex < len(filterOptions)-1 {
			v.filterDriverIndex++
		}
	case "k", "up":
		if v.filterDriverIndex > 0 {
			v.filterDriverIndex--
		}
	case "1", "2", "3", "4", "5", "6":
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < len(filterOptions) {
			v.filterDriverIndex = idx
			v.filterDriver = filterOptions[idx]
			v.showFilterMenu = false
			v.applyFilters()
			v.updateTableData()
		}
	}
	return v, nil
}

// cycleSortField 切换排序字段
func (v *NetworkListView) cycleSortField() {
	if v.sortField == SortByNetworkContainers {
		v.sortField = SortByNetworkName
		v.sortAscending = true
	} else {
		v.sortField++
		v.sortAscending = true
	}
}

// handleNormalKey 处理普通按键
func (v *NetworkListView) handleNormalKey(msg tea.KeyMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// ESC 优先级：清除搜索 > 清除筛选 > 返回上一级
		if v.searchQuery != "" {
			v.searchQuery = ""
			v.applyFilters()
			v.updateTableData()
			return v, nil
		}
		if v.filterDriver != "all" {
			v.filterDriver = "all"
			v.filterDriverIndex = 0
			v.applyFilters()
			v.updateTableData()
			return v, nil
		}
		// 没有搜索和筛选条件，返回上一级
		return v, func() tea.Msg { return GoBackMsg{} }
	case "/":
		v.isSearching = true
		v.searchQuery = ""
	case "r", "f5":
		v.loading = true
		v.errorMsg = ""
		return v, v.loadNetworks
	case "j", "down":
		if v.scrollTable != nil {
			v.scrollTable.MoveDown(1)
		}
	case "k", "up":
		if v.scrollTable != nil {
			v.scrollTable.MoveUp(1)
		}
	case "g":
		if v.scrollTable != nil {
			v.scrollTable.GotoTop()
		}
	case "G":
		if v.scrollTable != nil {
			v.scrollTable.GotoBottom()
		}
	case "h", "left":
		if v.scrollTable != nil {
			v.scrollTable.ScrollLeft()
		}
	case "l", "right":
		if v.scrollTable != nil {
			v.scrollTable.ScrollRight()
		}
	case "d":
		return v, v.showRemoveConfirmDialog()
	case "p":
		return v, v.showPruneConfirmDialog()
	case "c":
		// 创建网络
		v.showCreateView = true
		v.createView.Reset()
		v.createView.SetCallbacks(
			func(networkID string) {
				// 创建成功回调在 Update 中处理
			},
			func() {
				// 取消回调
				v.showCreateView = false
			},
		)
		return v, nil
	case "f":
		// 显示筛选菜单
		v.showFilterMenu = true
		return v, nil
	case "i":
		// 检查网络（显示 JSON）
		return v, v.inspectNetwork()
	case "enter":
		// 查看网络详情 - 发送消息给父视图
		network := v.GetSelectedNetwork()
		if network == nil {
			return v, nil
		}
		return v, func() tea.Msg {
			return ViewNetworkDetailsMsg{Network: network}
		}
	case "s":
		// 切换排序
		v.cycleSortField()
		v.applyFilters()
		v.updateTableData()
		return v, nil
	case "1":
		v.filterDriver = "all"
		v.applyFilters()
		v.updateTableData()
	case "2":
		v.filterDriver = "bridge"
		v.applyFilters()
		v.updateTableData()
	case "3":
		v.filterDriver = "host"
		v.applyFilters()
		v.updateTableData()
	case "4":
		v.filterDriver = "overlay"
		v.applyFilters()
		v.updateTableData()
	case "5":
		v.filterDriver = "none"
		v.applyFilters()
		v.updateTableData()
	}
	return v, nil
}

// View 渲染网络列表视图
func (v *NetworkListView) View() string {
	// 如果显示 JSON 查看器
	if v.jsonViewer != nil && v.jsonViewer.IsVisible() {
		return v.jsonViewer.View()
	}

	// 如果显示创建视图
	if v.showCreateView && v.createView != nil {
		return v.createView.View()
	}

	var s string

	// 顶部状态栏
	s += v.renderStatusBar()

	// 成功消息
	if v.successMsg != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
		s += "\n  " + msgStyle.Render(v.successMsg) + "\n"
	}

	// 统计信息栏
	s += v.renderStatsBar()

	// 加载中状态
	if v.loading {
		loadingContent := lipgloss.JoinVertical(lipgloss.Center,
			"",
			networkDriverStyle.Render("⏳ 正在加载网络列表..."),
			"",
			networkBuiltInStyle.Render("请稍候，正在从 Docker 获取数据"),
			"",
		)
		s += "\n  " + networkStateBoxStyle.Render(loadingContent) + "\n"
		return s
	}

	// 错误状态
	if v.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		errorContent := lipgloss.JoinVertical(lipgloss.Left,
			"",
			errorStyle.Render("❌ 加载失败: "+v.errorMsg),
			"",
			networkTitleStyle.Render("💡 可能的原因:"),
			networkBuiltInStyle.Render("   • Docker 守护进程未运行"),
			networkBuiltInStyle.Render("   • 网络连接问题"),
			"",
			networkDriverStyle.Render("按 r 重新加载"),
			"",
		)
		s += "\n  " + networkStateBoxStyle.Render(errorContent) + "\n"
		return s
	}

	// 空状态
	if len(v.networks) == 0 {
		emptyContent := lipgloss.JoinVertical(lipgloss.Left,
			"",
			networkBuiltInStyle.Render("🌐 暂无自定义网络"),
			"",
			networkTitleStyle.Render("💡 快速开始:"),
			"",
			networkDriverStyle.Render("1.") + networkBuiltInStyle.Render(" 创建一个网络:"),
			networkBuiltInStyle.Render("   docker network create my-network"),
			"",
			networkDriverStyle.Render("2.") + networkBuiltInStyle.Render(" 或按 c 键创建网络"),
			"",
			networkDriverStyle.Render("3.") + networkBuiltInStyle.Render(" 刷新网络列表:"),
			networkBuiltInStyle.Render("   按 r 键刷新"),
			"",
		)
		s += "\n  " + networkStateBoxStyle.Render(emptyContent) + "\n"
		return s
	}

	// 表格
	if v.scrollTable != nil {
		s += v.scrollTable.View() + "\n"
	}

	// 搜索栏
	if v.isSearching {
		searchLine := "\n  " + strings.Repeat("─", 67) + "\n"
		searchPrompt := "  " + networkDriverStyle.Render("Search:") + " "
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		searchInput := v.searchQuery + cursor
		cancelHint := networkBuiltInStyle.Render("[Enter=Confirm | ESC=Cancel]")
		s += searchLine + searchPrompt + searchInput + "    " + cancelHint + "\n"
	}

	// 筛选菜单
	if v.showFilterMenu {
		s = v.overlayFilterMenu(s)
	}

	// 确认对话框
	if v.showConfirmDialog {
		s = v.overlayDialog(s)
	}

	// 错误弹窗
	if v.errorDialog != nil && v.errorDialog.IsVisible() {
		s = v.errorDialog.Overlay(s)
	}

	return s
}

// SetSize 设置视图尺寸
func (v *NetworkListView) SetSize(width, height int) {
	v.width = width
	v.height = height

	tableHeight := height - 15
	if tableHeight < 5 {
		tableHeight = 5
	}

	if v.scrollTable != nil {
		v.scrollTable.SetSize(width-4, tableHeight)
	}

	if v.errorDialog != nil {
		v.errorDialog.SetWidth(width)
	}
}

// renderStatusBar 渲染顶部状态栏
func (v *NetworkListView) renderStatusBar() string {
	width := v.width
	if width < 80 {
		width = 80
	}

	labelStyle := lipgloss.NewStyle().
		Width(20).
		Foreground(lipgloss.Color("220")).
		Bold(true)

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	descStyle := lipgloss.NewStyle()
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	itemWidth := 18
	itemStyle := lipgloss.NewStyle().Width(itemWidth)

	makeItem := func(key, desc string) string {
		return itemStyle.Render(keyStyle.Render(key) + descStyle.Render(" "+desc))
	}

	var lines []string

	// 第一行
	row1Label := labelStyle.Render("🌐 Networks")
	row1Keys := makeItem("</>", "Search") + makeItem("<r>", "Refresh") + makeItem("<d>", "Delete")
	lines = append(lines, "  "+row1Label+row1Keys)

	// 第二行
	row2Label := labelStyle.Render("Ops:")
	row2Keys := makeItem("<c>", "Create") + makeItem("<p>", "Prune") + makeItem("<f>", "Filter") + makeItem("<i>", "Inspect")
	lines = append(lines, "  "+row2Label+row2Keys)

	// 第三行
	refreshInfo := "-"
	if !v.lastRefreshTime.IsZero() {
		refreshInfo = formatDuration(time.Since(v.lastRefreshTime)) + " ago"
	}

	// 显示当前筛选和排序状态
	filterInfo := ""
	if v.filterDriver != "all" {
		filterInfo = " [Filter: " + v.filterDriver + "]"
	}

	sortNames := []string{"Name", "Driver", "Created", "Containers"}
	sortInfo := " [Sort: " + sortNames[v.sortField] + "]"

	row3Label := labelStyle.Render("Last Refresh:")
	row3Info := hintStyle.Render(refreshInfo+filterInfo+sortInfo) + "    " +
		hintStyle.Render("j/k=上下  Enter=详情  s=排序  Esc=返回  q=退出")
	lines = append(lines, "  "+row3Label+row3Info)

	return "\n" + strings.Join(lines, "\n") + "\n"
}

// renderStatsBar 渲染统计信息栏
func (v *NetworkListView) renderStatsBar() string {
	totalCount := len(v.networks)
	showingCount := len(v.filteredNetworks)

	// 统计各类型网络数量
	bridgeCount := 0
	hostCount := 0
	overlayCount := 0
	otherCount := 0

	for _, net := range v.networks {
		switch net.Driver {
		case "bridge":
			bridgeCount++
		case "host":
			hostCount++
		case "overlay":
			overlayCount++
		default:
			otherCount++
		}
	}

	totalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	bridgeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	hostStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	overlayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	statsContent := totalStyle.Render(fmt.Sprintf("🌐 Total: %d", totalCount)) +
		separatorStyle.Render("  │  ") +
		bridgeStyle.Render(fmt.Sprintf("🔗 Bridge: %d", bridgeCount)) +
		separatorStyle.Render("  │  ") +
		hostStyle.Render(fmt.Sprintf("🖥️ Host: %d", hostCount)) +
		separatorStyle.Render("  │  ") +
		overlayStyle.Render(fmt.Sprintf("☁️ Overlay: %d", overlayCount))

	if showingCount != totalCount {
		filterInfo := networkBuiltInStyle.Render(fmt.Sprintf("  [Showing: %d]", showingCount))
		statsContent += filterInfo
	}

	lineWidth := v.width - 6
	if lineWidth < 60 {
		lineWidth = 60
	}
	line := lineStyle.Render(strings.Repeat("─", lineWidth))
	statsLine := lipgloss.NewStyle().Width(lineWidth).Align(lipgloss.Center).Render(statsContent)

	return "\n  " + line + "\n" + "  " + statsLine + "\n" + "  " + line + "\n"
}

// loadNetworks 加载网络列表
func (v *NetworkListView) loadNetworks() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	networks, err := v.dockerClient.ListNetworks(ctx)
	if err != nil {
		return networksLoadErrorMsg{err: err}
	}

	return networksLoadedMsg{networks: networks}
}

// applyFilters 应用过滤和搜索
func (v *NetworkListView) applyFilters() {
	v.filteredNetworks = make([]docker.Network, 0)

	for _, net := range v.networks {
		// 搜索过滤
		if v.searchQuery != "" {
			query := strings.ToLower(v.searchQuery)
			if !strings.Contains(strings.ToLower(net.Name), query) &&
				!strings.Contains(strings.ToLower(net.ID), query) &&
				!strings.Contains(strings.ToLower(net.Driver), query) {
				continue
			}
		}

		// 驱动过滤
		if v.filterDriver != "all" && net.Driver != v.filterDriver {
			continue
		}

		v.filteredNetworks = append(v.filteredNetworks, net)
	}

	// 应用排序
	v.sortNetworks()
}

// sortNetworks 对网络列表排序
func (v *NetworkListView) sortNetworks() {
	if len(v.filteredNetworks) <= 1 {
		return
	}

	// 使用简单的冒泡排序
	n := len(v.filteredNetworks)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			shouldSwap := false

			switch v.sortField {
			case SortByNetworkName:
				if v.sortAscending {
					shouldSwap = v.filteredNetworks[j].Name > v.filteredNetworks[j+1].Name
				} else {
					shouldSwap = v.filteredNetworks[j].Name < v.filteredNetworks[j+1].Name
				}
			case SortByNetworkDriver:
				if v.sortAscending {
					shouldSwap = v.filteredNetworks[j].Driver > v.filteredNetworks[j+1].Driver
				} else {
					shouldSwap = v.filteredNetworks[j].Driver < v.filteredNetworks[j+1].Driver
				}
			case SortByNetworkCreated:
				if v.sortAscending {
					shouldSwap = v.filteredNetworks[j].Created.After(v.filteredNetworks[j+1].Created)
				} else {
					shouldSwap = v.filteredNetworks[j].Created.Before(v.filteredNetworks[j+1].Created)
				}
			case SortByNetworkContainers:
				if v.sortAscending {
					shouldSwap = v.filteredNetworks[j].ContainerCount > v.filteredNetworks[j+1].ContainerCount
				} else {
					shouldSwap = v.filteredNetworks[j].ContainerCount < v.filteredNetworks[j+1].ContainerCount
				}
			}

			if shouldSwap {
				v.filteredNetworks[j], v.filteredNetworks[j+1] = v.filteredNetworks[j+1], v.filteredNetworks[j]
			}
		}
	}
}

// updateTableData 更新表格数据
func (v *NetworkListView) updateTableData() {
	if v.scrollTable == nil || len(v.filteredNetworks) == 0 {
		return
	}

	rows := make([]TableRow, len(v.filteredNetworks))

	for i, net := range v.filteredNetworks {
		created := formatCreatedTime(net.Created)
		containers := fmt.Sprintf("%d", net.ContainerCount)

		// 根据网络类型设置颜色
		var nameStyled, driverStyled string
		if net.IsBuiltIn {
			nameStyled = networkBuiltInStyle.Render(net.Name)
			driverStyled = networkBuiltInStyle.Render(net.Driver)
		} else {
			nameStyled = networkCustomStyle.Render(net.Name)
			driverStyled = networkDriverStyle.Render(net.Driver)
		}

		rows[i] = TableRow{
			net.ShortID,
			nameStyled,
			driverStyled,
			net.Scope,
			containers,
			created,
		}
	}

	v.scrollTable.SetRows(rows)
}

// GetSelectedNetwork 获取当前选中的网络
func (v *NetworkListView) GetSelectedNetwork() *docker.Network {
	if len(v.filteredNetworks) == 0 || v.scrollTable == nil {
		return nil
	}
	idx := v.scrollTable.Cursor()
	if idx < 0 || idx >= len(v.filteredNetworks) {
		return nil
	}
	return &v.filteredNetworks[idx]
}

// showRemoveConfirmDialog 显示删除确认对话框
func (v *NetworkListView) showRemoveConfirmDialog() tea.Cmd {
	network := v.GetSelectedNetwork()
	if network == nil {
		return nil
	}

	if network.IsBuiltIn {
		if v.errorDialog != nil {
			v.errorDialog.ShowError("无法删除内置网络: " + network.Name)
		}
		return nil
	}

	if network.ContainerCount > 0 {
		if v.errorDialog != nil {
			v.errorDialog.ShowError(fmt.Sprintf("网络 %s 仍有 %d 个容器连接，请先断开连接", network.Name, network.ContainerCount))
		}
		return nil
	}

	v.showConfirmDialog = true
	v.confirmAction = "remove"
	v.confirmNetwork = network
	v.confirmSelection = 0
	return nil
}

// showPruneConfirmDialog 显示清理确认对话框
func (v *NetworkListView) showPruneConfirmDialog() tea.Cmd {
	v.showConfirmDialog = true
	v.confirmAction = "prune"
	v.confirmNetwork = nil
	v.confirmSelection = 0
	return nil
}

// resetConfirmDialog 重置确认对话框状态
func (v *NetworkListView) resetConfirmDialog() {
	v.showConfirmDialog = false
	v.confirmAction = ""
	v.confirmNetwork = nil
	v.confirmSelection = 0
}

// removeNetwork 删除网络
func (v *NetworkListView) removeNetwork(network *docker.Network) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := v.dockerClient.RemoveNetwork(ctx, network.ID)
		if err != nil {
			return networkOperationErrorMsg{operation: "删除网络", err: err}
		}

		return networkOperationSuccessMsg{operation: "删除网络", network: network.Name}
	}
}

// pruneNetworks 清理未使用的网络
func (v *NetworkListView) pruneNetworks() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		deleted, err := v.dockerClient.PruneNetworks(ctx)
		if err != nil {
			return networkOperationErrorMsg{operation: "清理网络", err: err}
		}

		if len(deleted) == 0 {
			return networkOperationSuccessMsg{operation: "清理网络", network: "无未使用的网络"}
		}

		return networkOperationSuccessMsg{
			operation: "清理网络",
			network:   fmt.Sprintf("已删除 %d 个网络", len(deleted)),
		}
	}
}

// clearSuccessMessageAfter 延迟清除成功消息
func (v *NetworkListView) clearSuccessMessageAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearSuccessMessageMsg{}
	})
}

// inspectNetwork 获取网络的原始 JSON
func (v *NetworkListView) inspectNetwork() tea.Cmd {
	network := v.GetSelectedNetwork()
	if network == nil {
		return nil
	}

	networkID := network.ID
	networkName := network.Name

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		jsonContent, err := v.dockerClient.InspectNetworkRaw(ctx, networkID)
		if err != nil {
			return networkInspectErrorMsg{err: err}
		}

		return networkInspectMsg{
			networkName: networkName,
			jsonContent: jsonContent,
		}
	}
}

// overlayDialog 叠加对话框
func (v *NetworkListView) overlayDialog(baseContent string) string {
	return OverlayCentered(baseContent, v.renderConfirmDialogContent(), v.width, v.height)
}

// renderConfirmDialogContent 渲染确认对话框内容
func (v *NetworkListView) renderConfirmDialogContent() string {
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

	cancelBtnStyle := lipgloss.NewStyle().Padding(0, 2)
	okBtnStyle := lipgloss.NewStyle().Padding(0, 2)

	if v.confirmSelection == 0 {
		cancelBtnStyle = cancelBtnStyle.Reverse(true).Bold(true)
		okBtnStyle = okBtnStyle.Foreground(lipgloss.Color("245"))
	} else {
		cancelBtnStyle = cancelBtnStyle.Foreground(lipgloss.Color("245"))
		okBtnStyle = okBtnStyle.Reverse(true).Bold(true)
	}

	var title, warning string

	if v.confirmAction == "remove" && v.confirmNetwork != nil {
		title = "🗑️  确认删除网络"
		warning = fmt.Sprintf("确定要删除网络 \"%s\" 吗？", v.confirmNetwork.Name)
	} else if v.confirmAction == "prune" {
		title = "🧹  确认清理网络"
		warning = "确定要清理所有未使用的网络吗？\n此操作不可撤销。"
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		cancelBtnStyle.Render("[ Cancel ]"),
		"    ",
		okBtnStyle.Render("[   OK   ]"),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		titleStyle.Render(title),
		"",
		warningStyle.Render(warning),
		"",
		buttons,
		"",
	)

	// 居中显示
	leftPadding := (v.width - 60) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	return strings.Repeat(" ", leftPadding) + dialogStyle.Render(content)
}

// HasError 检查是否有错误弹窗显示
func (v *NetworkListView) HasError() bool {
	return v.errorDialog != nil && v.errorDialog.IsVisible()
}

// IsShowingJSONViewer 返回是否正在显示 JSON 查看器
func (v *NetworkListView) IsShowingJSONViewer() bool {
	return v.jsonViewer != nil && v.jsonViewer.IsVisible()
}

// overlayFilterMenu 叠加筛选菜单
func (v *NetworkListView) overlayFilterMenu(baseContent string) string {
	return OverlayCentered(baseContent, v.renderFilterMenuContent(), v.width, v.height)
}

// renderFilterMenuContent 渲染筛选菜单内容
func (v *NetworkListView) renderFilterMenuContent() string {
	menuStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(40)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true).
		Reverse(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	filterOptions := []struct {
		key   string
		label string
		value string
	}{
		{"1", "All", "all"},
		{"2", "Bridge", "bridge"},
		{"3", "Host", "host"},
		{"4", "Overlay", "overlay"},
		{"5", "Macvlan", "macvlan"},
		{"6", "None", "none"},
	}

	var items []string
	for i, opt := range filterOptions {
		prefix := "  "
		style := itemStyle
		if i == v.filterDriverIndex {
			prefix = "▶ "
			style = selectedStyle
		}
		items = append(items, prefix+style.Render(fmt.Sprintf("[%s] %s", opt.key, opt.label)))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("🔍 Filter by Driver"),
		"",
		strings.Join(items, "\n"),
		"",
		hintStyle.Render("j/k=上下  Enter=确认  Esc=取消"),
	)

	// 居中显示
	leftPadding := (v.width - 44) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	return strings.Repeat(" ", leftPadding) + menuStyle.Render(content)
}

// IsShowingCreateView 检查是否正在显示创建视图
func (v *NetworkListView) IsShowingCreateView() bool {
	return v.showCreateView
}

// 消息类型定义
type networksLoadedMsg struct {
	networks []docker.Network
}

type networksLoadErrorMsg struct {
	err error
}

type networkOperationSuccessMsg struct {
	operation string
	network   string
}

type networkOperationErrorMsg struct {
	operation string
	err       error
}

type networkInspectMsg struct {
	networkName string
	jsonContent string
}

type networkInspectErrorMsg struct {
	err error
}
