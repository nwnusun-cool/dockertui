package network

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
	"docktui/internal/ui/components"
)

// SortField 网络排序字段
type SortField int

const (
	SortByName SortField = iota
	SortByDriver
	SortByCreated
	SortByContainers
)

// ListView 网络列表视图
type ListView struct {
	dockerClient docker.Client
	width, height int
	networks, filteredNetworks []docker.Network
	scrollTable *components.ScrollableTable
	loading bool
	errorMsg, successMsg string
	successMsgTime time.Time
	searchQuery string
	isSearching bool
	filterDriver string
	filterDriverIndex int
	showFilterMenu bool
	sortField SortField
	sortAscending bool
	lastRefreshTime time.Time
	showConfirmDialog bool
	confirmAction string
	confirmNetwork *docker.Network
	confirmSelection int
	createView *CreateView
	showCreateView bool
	jsonViewer *components.JSONViewer
	errorDialog *components.ErrorDialog
}

// NewListView 创建网络列表视图
func NewListView(dockerClient docker.Client) *ListView {
	columns := []components.TableColumn{
		{Title: "NETWORK ID", Width: 14},
		{Title: "NAME", Width: 25},
		{Title: "DRIVER", Width: 12},
		{Title: "SCOPE", Width: 10},
		{Title: "CONTAINERS", Width: 12},
		{Title: "CREATED", Width: 16},
	}
	return &ListView{
		dockerClient: dockerClient,
		scrollTable: components.NewScrollableTable(columns),
		filterDriver: "all",
		sortField: SortByName,
		sortAscending: true,
		errorDialog: components.NewErrorDialog(),
		createView: NewCreateView(dockerClient),
		jsonViewer: components.NewJSONViewer(),
	}
}

// Init 初始化网络列表视图
func (v *ListView) Init() tea.Cmd {
	v.loading = true
	return v.loadNetworks
}

// Update 处理消息并更新视图状态
func (v *ListView) Update(msg tea.Msg) (*ListView, tea.Cmd) {
	if v.jsonViewer != nil && v.jsonViewer.IsVisible() {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if v.jsonViewer.Update(keyMsg) { return v, nil }
		}
	}
	if v.showCreateView && v.createView != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			_, cmd := v.createView.Update(msg)
			return v, cmd
		case NetworkCreateSuccessMsg:
			v.showCreateView = false
			v.successMsg = fmt.Sprintf("✅ 网络创建成功: %s", msg.NetworkID[:12])
			v.successMsgTime = time.Now()
			v.createView.Reset()
			return v, tea.Batch(v.loadNetworks, v.clearSuccessMessageAfter(3*time.Second))
		case NetworkCreateErrorMsg:
			v.createView.Update(msg)
			return v, nil
		}
	}
	switch msg := msg.(type) {
	case NetworksLoadedMsg:
		v.networks = msg.Networks
		v.loading = false
		v.errorMsg = ""
		v.lastRefreshTime = time.Now()
		v.applyFilters()
		v.updateTableData()
		return v, nil
	case NetworksLoadErrorMsg:
		v.loading = false
		v.errorMsg = msg.Err.Error()
		return v, nil
	case NetworkOperationSuccessMsg:
		v.successMsg = fmt.Sprintf("✅ %s成功: %s", msg.Operation, msg.Network)
		v.successMsgTime = time.Now()
		v.errorMsg = ""
		return v, tea.Batch(v.loadNetworks, v.clearSuccessMessageAfter(3*time.Second))
	case NetworkOperationErrorMsg:
		if v.errorDialog != nil { v.errorDialog.ShowError(fmt.Sprintf("%s失败: %v", msg.Operation, msg.Err)) }
		return v, nil
	case ClearSuccessMessageMsg:
		if time.Since(v.successMsgTime) >= 3*time.Second { v.successMsg = "" }
		return v, nil
	case NetworkInspectMsg:
		if v.jsonViewer != nil {
			v.jsonViewer.SetSize(v.width, v.height)
			v.jsonViewer.Show("Network Inspect: "+msg.NetworkName, msg.JSONContent)
		}
		return v, nil
	case NetworkInspectErrorMsg:
		if v.errorDialog != nil { v.errorDialog.ShowError(fmt.Sprintf("获取网络信息失败: %v", msg.Err)) }
		return v, nil
	case tea.KeyMsg:
		if v.errorDialog != nil && v.errorDialog.IsVisible() { if v.errorDialog.Update(msg) { return v, nil } }
		if v.showConfirmDialog { return v.handleConfirmDialogKey(msg) }
		if v.showFilterMenu { return v.handleFilterMenuKey(msg) }
		if v.isSearching { return v.handleSearchKey(msg) }
		return v.handleNormalKey(msg)
	}
	return v, nil
}

func (v *ListView) handleConfirmDialogKey(msg tea.KeyMsg) (*ListView, tea.Cmd) {
	switch msg.String() {
	case "left", "right", "tab", "h", "l": v.confirmSelection = 1 - v.confirmSelection
	case "enter":
		if v.confirmSelection == 1 {
			action, network := v.confirmAction, v.confirmNetwork
			v.resetConfirmDialog()
			if action == "remove" && network != nil { return v, v.removeNetwork(network) }
			if action == "prune" { return v, v.pruneNetworks() }
		} else { v.resetConfirmDialog() }
	case "esc": v.resetConfirmDialog()
	}
	return v, nil
}

func (v *ListView) handleSearchKey(msg tea.KeyMsg) (*ListView, tea.Cmd) {
	switch msg.String() {
	case "enter": v.isSearching = false
	case "esc": v.isSearching = false; v.searchQuery = ""; v.applyFilters(); v.updateTableData()
	case "backspace":
		if len(v.searchQuery) > 0 { v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]; v.applyFilters(); v.updateTableData() }
	default:
		if len(msg.String()) == 1 { v.searchQuery += msg.String(); v.applyFilters(); v.updateTableData() }
	}
	return v, nil
}

func (v *ListView) handleFilterMenuKey(msg tea.KeyMsg) (*ListView, tea.Cmd) {
	filterOptions := []string{"all", "bridge", "host", "overlay", "macvlan", "none"}
	switch msg.String() {
	case "esc": v.showFilterMenu = false
	case "enter":
		v.filterDriver = filterOptions[v.filterDriverIndex]
		v.showFilterMenu = false
		v.applyFilters(); v.updateTableData()
	case "j", "down": if v.filterDriverIndex < len(filterOptions)-1 { v.filterDriverIndex++ }
	case "k", "up": if v.filterDriverIndex > 0 { v.filterDriverIndex-- }
	case "1", "2", "3", "4", "5", "6":
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < len(filterOptions) {
			v.filterDriverIndex = idx
			v.filterDriver = filterOptions[idx]
			v.showFilterMenu = false
			v.applyFilters(); v.updateTableData()
		}
	}
	return v, nil
}

func (v *ListView) cycleSortField() {
	if v.sortField == SortByContainers { v.sortField = SortByName; v.sortAscending = true } else { v.sortField++; v.sortAscending = true }
}

func (v *ListView) handleNormalKey(msg tea.KeyMsg) (*ListView, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if v.searchQuery != "" { v.searchQuery = ""; v.applyFilters(); v.updateTableData(); return v, nil }
		if v.filterDriver != "all" { v.filterDriver = "all"; v.filterDriverIndex = 0; v.applyFilters(); v.updateTableData(); return v, nil }
		return v, func() tea.Msg { return GoBackMsg{} }
	case "/": v.isSearching = true; v.searchQuery = ""
	case "r", "f5": v.loading = true; v.errorMsg = ""; return v, v.loadNetworks
	case "j", "down": if v.scrollTable != nil { v.scrollTable.MoveDown(1) }
	case "k", "up": if v.scrollTable != nil { v.scrollTable.MoveUp(1) }
	case "g": if v.scrollTable != nil { v.scrollTable.GotoTop() }
	case "G": if v.scrollTable != nil { v.scrollTable.GotoBottom() }
	case "h", "left": if v.scrollTable != nil { v.scrollTable.ScrollLeft() }
	case "l", "right": if v.scrollTable != nil { v.scrollTable.ScrollRight() }
	case "d": return v, v.showRemoveConfirmDialog()
	case "p": return v, v.showPruneConfirmDialog()
	case "c":
		v.showCreateView = true
		v.createView.Reset()
		v.createView.SetCallbacks(func(networkID string) {}, func() { v.showCreateView = false })
		return v, nil
	case "f": v.showFilterMenu = true; return v, nil
	case "i": return v, v.inspectNetwork()
	case "enter":
		network := v.GetSelectedNetwork()
		if network == nil { return v, nil }
		return v, func() tea.Msg { return ViewNetworkDetailsMsg{Network: network} }
	case "s": v.cycleSortField(); v.applyFilters(); v.updateTableData(); return v, nil
	case "1": v.filterDriver = "all"; v.applyFilters(); v.updateTableData()
	case "2": v.filterDriver = "bridge"; v.applyFilters(); v.updateTableData()
	case "3": v.filterDriver = "host"; v.applyFilters(); v.updateTableData()
	case "4": v.filterDriver = "overlay"; v.applyFilters(); v.updateTableData()
	case "5": v.filterDriver = "none"; v.applyFilters(); v.updateTableData()
	}
	return v, nil
}

// View 渲染网络列表视图
func (v *ListView) View() string {
	if v.jsonViewer != nil && v.jsonViewer.IsVisible() { return v.jsonViewer.View() }
	if v.showCreateView && v.createView != nil { return v.createView.View() }
	var s string
	s += v.renderStatusBar()
	if v.successMsg != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
		s += "\n  " + msgStyle.Render(v.successMsg) + "\n"
	}
	s += v.renderStatsBar()
	if v.loading {
		loadingContent := lipgloss.JoinVertical(lipgloss.Center, "", DriverStyle.Render("⏳ 正在加载网络列表..."), "", BuiltInStyle.Render("请稍候，正在从 Docker 获取数据"), "")
		s += "\n  " + StateBoxStyle.Render(loadingContent) + "\n"
		return s
	}
	if v.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		errorContent := lipgloss.JoinVertical(lipgloss.Left, "", errorStyle.Render("❌ 加载失败: "+v.errorMsg), "", TitleStyle.Render("💡 可能的原因:"), BuiltInStyle.Render("   • Docker 守护进程未运行"), BuiltInStyle.Render("   • 网络连接问题"), "", DriverStyle.Render("按 r 重新加载"), "")
		s += "\n  " + StateBoxStyle.Render(errorContent) + "\n"
		return s
	}
	if len(v.networks) == 0 {
		emptyContent := lipgloss.JoinVertical(lipgloss.Left, "", BuiltInStyle.Render("🌐 暂无自定义网络"), "", TitleStyle.Render("💡 快速开始:"), "", DriverStyle.Render("1.")+BuiltInStyle.Render(" 创建一个网络:"), BuiltInStyle.Render("   docker network create my-network"), "", DriverStyle.Render("2.")+BuiltInStyle.Render(" 或按 c 键创建网络"), "", DriverStyle.Render("3.")+BuiltInStyle.Render(" 刷新网络列表:"), BuiltInStyle.Render("   按 r 键刷新"), "")
		s += "\n  " + StateBoxStyle.Render(emptyContent) + "\n"
		return s
	}
	if v.scrollTable != nil { s += v.scrollTable.View() + "\n" }
	if v.isSearching {
		searchLine := "\n  " + strings.Repeat("─", 67) + "\n"
		searchPrompt := "  " + DriverStyle.Render("Search:") + " "
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		cancelHint := BuiltInStyle.Render("[Enter=Confirm | ESC=Cancel]")
		s += searchLine + searchPrompt + v.searchQuery + cursor + "    " + cancelHint + "\n"
	}
	if v.showFilterMenu { s = v.overlayFilterMenu(s) }
	if v.showConfirmDialog { s = v.overlayDialog(s) }
	if v.errorDialog != nil && v.errorDialog.IsVisible() { s = v.errorDialog.Overlay(s) }
	return s
}

// SetSize 设置视图尺寸
func (v *ListView) SetSize(width, height int) {
	v.width = width; v.height = height
	tableHeight := height - 15; if tableHeight < 5 { tableHeight = 5 }
	if v.scrollTable != nil { v.scrollTable.SetSize(width-4, tableHeight) }
	if v.errorDialog != nil { v.errorDialog.SetWidth(width) }
}

func (v *ListView) renderStatusBar() string {
	width := v.width; if width < 80 { width = 80 }
	labelStyle := lipgloss.NewStyle().Width(20).Foreground(lipgloss.Color("220")).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	descStyle := lipgloss.NewStyle()
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	itemWidth := 18
	itemStyle := lipgloss.NewStyle().Width(itemWidth)
	makeItem := func(key, desc string) string { return itemStyle.Render(keyStyle.Render(key) + descStyle.Render(" "+desc)) }
	var lines []string
	lines = append(lines, "  "+labelStyle.Render("🌐 Networks")+makeItem("</>", "Search")+makeItem("<r>", "Refresh")+makeItem("<d>", "Delete"))
	lines = append(lines, "  "+labelStyle.Render("Ops:")+makeItem("<c>", "Create")+makeItem("<p>", "Prune")+makeItem("<f>", "Filter")+makeItem("<i>", "Inspect"))
	refreshInfo := "-"
	if !v.lastRefreshTime.IsZero() { refreshInfo = formatDuration(time.Since(v.lastRefreshTime)) + " ago" }
	filterInfo := ""; if v.filterDriver != "all" { filterInfo = " [Filter: " + v.filterDriver + "]" }
	sortNames := []string{"Name", "Driver", "Created", "Containers"}
	sortInfo := " [Sort: " + sortNames[v.sortField] + "]"
	lines = append(lines, "  "+labelStyle.Render("Last Refresh:")+hintStyle.Render(refreshInfo+filterInfo+sortInfo)+"    "+hintStyle.Render("j/k=上下  Enter=详情  s=排序  Esc=返回  q=退出"))
	return "\n" + strings.Join(lines, "\n") + "\n"
}

func (v *ListView) renderStatsBar() string {
	totalCount := len(v.networks)
	showingCount := len(v.filteredNetworks)
	bridgeCount, hostCount, overlayCount := 0, 0, 0
	for _, net := range v.networks {
		switch net.Driver {
		case "bridge": bridgeCount++
		case "host": hostCount++
		case "overlay": overlayCount++
		}
	}
	totalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	bridgeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	hostStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	overlayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statsContent := totalStyle.Render(fmt.Sprintf("🌐 Total: %d", totalCount)) + separatorStyle.Render("  │  ") + bridgeStyle.Render(fmt.Sprintf("🔗 Bridge: %d", bridgeCount)) + separatorStyle.Render("  │  ") + hostStyle.Render(fmt.Sprintf("🖥️ Host: %d", hostCount)) + separatorStyle.Render("  │  ") + overlayStyle.Render(fmt.Sprintf("☁️ Overlay: %d", overlayCount))
	if showingCount != totalCount { statsContent += BuiltInStyle.Render(fmt.Sprintf("  [Showing: %d]", showingCount)) }
	lineWidth := v.width - 6; if lineWidth < 60 { lineWidth = 60 }
	line := lineStyle.Render(strings.Repeat("─", lineWidth))
	statsLine := lipgloss.NewStyle().Width(lineWidth).Align(lipgloss.Center).Render(statsContent)
	return "\n  " + line + "\n" + "  " + statsLine + "\n" + "  " + line + "\n"
}

func (v *ListView) loadNetworks() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	networks, err := v.dockerClient.ListNetworks(ctx)
	if err != nil { return NetworksLoadErrorMsg{Err: err} }
	return NetworksLoadedMsg{Networks: networks}
}

func (v *ListView) applyFilters() {
	v.filteredNetworks = make([]docker.Network, 0)
	for _, net := range v.networks {
		if v.searchQuery != "" {
			query := strings.ToLower(v.searchQuery)
			if !strings.Contains(strings.ToLower(net.Name), query) && !strings.Contains(strings.ToLower(net.ID), query) && !strings.Contains(strings.ToLower(net.Driver), query) { continue }
		}
		if v.filterDriver != "all" && net.Driver != v.filterDriver { continue }
		v.filteredNetworks = append(v.filteredNetworks, net)
	}
	v.sortNetworks()
}

func (v *ListView) sortNetworks() {
	if len(v.filteredNetworks) <= 1 { return }
	n := len(v.filteredNetworks)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			shouldSwap := false
			switch v.sortField {
			case SortByName:
				if v.sortAscending { shouldSwap = v.filteredNetworks[j].Name > v.filteredNetworks[j+1].Name } else { shouldSwap = v.filteredNetworks[j].Name < v.filteredNetworks[j+1].Name }
			case SortByDriver:
				if v.sortAscending { shouldSwap = v.filteredNetworks[j].Driver > v.filteredNetworks[j+1].Driver } else { shouldSwap = v.filteredNetworks[j].Driver < v.filteredNetworks[j+1].Driver }
			case SortByCreated:
				if v.sortAscending { shouldSwap = v.filteredNetworks[j].Created.After(v.filteredNetworks[j+1].Created) } else { shouldSwap = v.filteredNetworks[j].Created.Before(v.filteredNetworks[j+1].Created) }
			case SortByContainers:
				if v.sortAscending { shouldSwap = v.filteredNetworks[j].ContainerCount > v.filteredNetworks[j+1].ContainerCount } else { shouldSwap = v.filteredNetworks[j].ContainerCount < v.filteredNetworks[j+1].ContainerCount }
			}
			if shouldSwap { v.filteredNetworks[j], v.filteredNetworks[j+1] = v.filteredNetworks[j+1], v.filteredNetworks[j] }
		}
	}
}

func (v *ListView) updateTableData() {
	if v.scrollTable == nil || len(v.filteredNetworks) == 0 { return }
	rows := make([]components.TableRow, len(v.filteredNetworks))
	for i, net := range v.filteredNetworks {
		created := formatCreatedTime(net.Created)
		containers := fmt.Sprintf("%d", net.ContainerCount)
		var nameStyled, driverStyled string
		if net.IsBuiltIn { nameStyled = BuiltInStyle.Render(net.Name); driverStyled = BuiltInStyle.Render(net.Driver) } else { nameStyled = CustomStyle.Render(net.Name); driverStyled = DriverStyle.Render(net.Driver) }
		rows[i] = components.TableRow{net.ShortID, nameStyled, driverStyled, net.Scope, containers, created}
	}
	v.scrollTable.SetRows(rows)
}

// GetSelectedNetwork 获取当前选中的网络
func (v *ListView) GetSelectedNetwork() *docker.Network {
	if len(v.filteredNetworks) == 0 || v.scrollTable == nil { return nil }
	idx := v.scrollTable.Cursor()
	if idx < 0 || idx >= len(v.filteredNetworks) { return nil }
	return &v.filteredNetworks[idx]
}

func (v *ListView) showRemoveConfirmDialog() tea.Cmd {
	network := v.GetSelectedNetwork()
	if network == nil { return nil }
	if network.IsBuiltIn { if v.errorDialog != nil { v.errorDialog.ShowError("无法删除内置网络: " + network.Name) }; return nil }
	if network.ContainerCount > 0 { if v.errorDialog != nil { v.errorDialog.ShowError(fmt.Sprintf("网络 %s 仍有 %d 个容器连接，请先断开连接", network.Name, network.ContainerCount)) }; return nil }
	v.showConfirmDialog = true; v.confirmAction = "remove"; v.confirmNetwork = network; v.confirmSelection = 0
	return nil
}

func (v *ListView) showPruneConfirmDialog() tea.Cmd {
	v.showConfirmDialog = true; v.confirmAction = "prune"; v.confirmNetwork = nil; v.confirmSelection = 0
	return nil
}

func (v *ListView) resetConfirmDialog() {
	v.showConfirmDialog = false; v.confirmAction = ""; v.confirmNetwork = nil; v.confirmSelection = 0
}

func (v *ListView) removeNetwork(network *docker.Network) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := v.dockerClient.RemoveNetwork(ctx, network.ID)
		if err != nil { return NetworkOperationErrorMsg{Operation: "删除网络", Err: err} }
		return NetworkOperationSuccessMsg{Operation: "删除网络", Network: network.Name}
	}
}

func (v *ListView) pruneNetworks() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		deleted, err := v.dockerClient.PruneNetworks(ctx)
		if err != nil { return NetworkOperationErrorMsg{Operation: "清理网络", Err: err} }
		if len(deleted) == 0 { return NetworkOperationSuccessMsg{Operation: "清理网络", Network: "无未使用的网络"} }
		return NetworkOperationSuccessMsg{Operation: "清理网络", Network: fmt.Sprintf("已删除 %d 个网络", len(deleted))}
	}
}

func (v *ListView) clearSuccessMessageAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return ClearSuccessMessageMsg{} })
}

func (v *ListView) inspectNetwork() tea.Cmd {
	network := v.GetSelectedNetwork()
	if network == nil { return nil }
	networkID, networkName := network.ID, network.Name
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		jsonContent, err := v.dockerClient.InspectNetworkRaw(ctx, networkID)
		if err != nil { return NetworkInspectErrorMsg{Err: err} }
		return NetworkInspectMsg{NetworkName: networkName, JSONContent: jsonContent}
	}
}

func (v *ListView) overlayDialog(baseContent string) string {
	return components.OverlayCentered(baseContent, v.renderConfirmDialogContent(), v.width, v.height)
}

func (v *ListView) renderConfirmDialogContent() string {
	dialogStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(1, 2).Width(56)
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	cancelBtnStyle := lipgloss.NewStyle().Padding(0, 2)
	okBtnStyle := lipgloss.NewStyle().Padding(0, 2)
	if v.confirmSelection == 0 { cancelBtnStyle = cancelBtnStyle.Reverse(true).Bold(true); okBtnStyle = okBtnStyle.Foreground(lipgloss.Color("245")) } else { cancelBtnStyle = cancelBtnStyle.Foreground(lipgloss.Color("245")); okBtnStyle = okBtnStyle.Reverse(true).Bold(true) }
	var title, warning string
	if v.confirmAction == "remove" && v.confirmNetwork != nil { title = "🗑️  确认删除网络"; warning = fmt.Sprintf("确定要删除网络 \"%s\" 吗？", v.confirmNetwork.Name) } else if v.confirmAction == "prune" { title = "🧹  确认清理网络"; warning = "确定要清理所有未使用的网络吗？\n此操作不可撤销。" }
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, cancelBtnStyle.Render("[ Cancel ]"), "    ", okBtnStyle.Render("[   OK   ]"))
	content := lipgloss.JoinVertical(lipgloss.Center, "", titleStyle.Render(title), "", warningStyle.Render(warning), "", buttons, "")
	leftPadding := (v.width - 60) / 2; if leftPadding < 0 { leftPadding = 0 }
	return strings.Repeat(" ", leftPadding) + dialogStyle.Render(content)
}

// HasError 检查是否有错误弹窗显示
func (v *ListView) HasError() bool { return v.errorDialog != nil && v.errorDialog.IsVisible() }

// IsShowingJSONViewer 返回是否正在显示 JSON 查看器
func (v *ListView) IsShowingJSONViewer() bool { return v.jsonViewer != nil && v.jsonViewer.IsVisible() }

func (v *ListView) overlayFilterMenu(baseContent string) string {
	return components.OverlayCentered(baseContent, v.renderFilterMenuContent(), v.width, v.height)
}

func (v *ListView) renderFilterMenuContent() string {
	menuStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(1, 2).Width(40)
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true).Reverse(true)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	filterOptions := []struct{ key, label, value string }{{"1", "All", "all"}, {"2", "Bridge", "bridge"}, {"3", "Host", "host"}, {"4", "Overlay", "overlay"}, {"5", "Macvlan", "macvlan"}, {"6", "None", "none"}}
	var items []string
	for i, opt := range filterOptions {
		prefix := "  "; style := itemStyle
		if i == v.filterDriverIndex { prefix = "▶ "; style = selectedStyle }
		items = append(items, prefix+style.Render(fmt.Sprintf("[%s] %s", opt.key, opt.label)))
	}
	content := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("🔍 Filter by Driver"), "", strings.Join(items, "\n"), "", hintStyle.Render("j/k=上下  Enter=确认  Esc=取消"))
	leftPadding := (v.width - 44) / 2; if leftPadding < 0 { leftPadding = 0 }
	return strings.Repeat(" ", leftPadding) + menuStyle.Render(content)
}

// IsShowingCreateView 检查是否正在显示创建视图
func (v *ListView) IsShowingCreateView() bool { return v.showCreateView }

func formatDuration(d time.Duration) string {
	if d < time.Minute { return fmt.Sprintf("%ds", int(d.Seconds())) }
	if d < time.Hour { return fmt.Sprintf("%dm", int(d.Minutes())) }
	return fmt.Sprintf("%dh", int(d.Hours()))
}

// ShowConfirmDialog 返回是否显示确认对话框
func (v *ListView) ShowConfirmDialog() bool { return v.showConfirmDialog }

// ShowFilterMenu 返回是否显示筛选菜单
func (v *ListView) ShowFilterMenu() bool { return v.showFilterMenu }
