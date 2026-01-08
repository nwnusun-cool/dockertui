package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
)

// 镜像列表视图样式定义 - 使用自适应颜色
var (
	// 状态栏样式
	imageStatusBarLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	imageStatusBarKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	// 标题栏样式
	imageTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	// 镜像状态样式
	imageDanglingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")) // 悬垂镜像 - 灰色

	imageActiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")) // 活跃镜像 - 绿色

	imageUnusedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")) // 未使用镜像 - 灰色

	// 成功/错误消息样式
	imageSuccessMsgStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)

	imageErrorMsgStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	// 搜索栏样式
	imageSearchPromptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Bold(true)

	imageSearchHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	// 加载/空状态框样式
	imageStateBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(66)
)

// ImageListView 镜像列表视图
type ImageListView struct {
	dockerClient docker.Client

	// UI 尺寸
	width  int
	height int

	// 数据状态
	images         []docker.Image // 镜像列表数据（原始）
	filteredImages []docker.Image // 过滤后的镜像列表
	tableModel     table.Model    // bubbles/table 组件（保留兼容）
	scrollTable    *ScrollableTable // 可水平滚动的表格
	loading        bool           // 是否正在加载
	errorMsg       string         // 错误信息
	successMsg     string         // 成功消息
	successMsgTime time.Time      // 成功消息显示时间

	// 搜索状态
	searchQuery string // 搜索关键字
	isSearching bool   // 是否处于搜索模式

	// 筛选状态
	filterType string // "all", "active", "dangling", "unused"

	// 排序状态
	sortBy string // "size", "created", "repository"

	// 刷新状态
	lastRefreshTime time.Time // 上次刷新时间

	// 确认对话框状态
	showConfirmDialog bool              // 是否显示确认对话框
	confirmAction     string            // 确认的操作类型: "remove", "prune"
	confirmImage      *docker.Image     // 待操作的镜像
	confirmSelection  int               // 确认对话框中的选择: 0=Cancel, 1=OK

	// 快捷键管理
	keys KeyMap
}

// NewImageListView 创建镜像列表视图
func NewImageListView(dockerClient docker.Client) *ImageListView {
	// 定义表格列
	columns := []table.Column{
		{Title: "IMAGE ID", Width: 14},
		{Title: "REPOSITORY", Width: 30},
		{Title: "TAG", Width: 20},
		{Title: "SIZE", Width: 12},
		{Title: "CREATED", Width: 14},
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

	// 创建可滚动表格
	scrollColumns := []TableColumn{
		{Title: "IMAGE ID", Width: 14},
		{Title: "REPOSITORY", Width: 35},
		{Title: "TAG", Width: 25},
		{Title: "SIZE", Width: 12},
		{Title: "CREATED", Width: 16},
	}
	scrollTable := NewScrollableTable(scrollColumns)

	return &ImageListView{
		dockerClient: dockerClient,
		tableModel:   t,
		scrollTable:  scrollTable,
		keys:         DefaultKeyMap(),
		searchQuery:  "",
		isSearching:  false,
		filterType:   "all",
		sortBy:       "created",
	}
}

// Init 初始化镜像列表视图
func (v *ImageListView) Init() tea.Cmd {
	v.loading = true
	return v.loadImages
}

// Update 处理消息并更新视图状态
func (v *ImageListView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case imagesLoadedMsg:
		// 镜像列表加载完成
		v.images = msg.images
		v.loading = false
		v.errorMsg = ""
		v.lastRefreshTime = time.Now()

		// 应用过滤和搜索
		v.applyFilters()

		// 根据数据内容更新列宽，然后渲染表格
		v.updateColumnWidths()

		return v, nil

	case imagesLoadErrorMsg:
		// 镜像列表加载失败
		v.loading = false
		v.errorMsg = msg.err.Error()
		return v, nil

	case imageOperationSuccessMsg:
		// 镜像操作成功，显示成功消息并刷新列表
		v.successMsg = fmt.Sprintf("✅ %s成功: %s", msg.operation, msg.image)
		v.successMsgTime = time.Now()
		v.errorMsg = "" // 清除错误消息
		return v, tea.Batch(
			v.loadImages,
			v.clearSuccessMessageAfter(3 * time.Second),
		)

	case imageOperationErrorMsg:
		// 镜像操作失败，显示错误信息
		v.errorMsg = fmt.Sprintf("❌ %s失败 (%s): %v", msg.operation, msg.image, msg.err)
		v.successMsg = "" // 清除成功消息
		return v, nil

	case clearSuccessMessageMsg:
		// 清除成功消息
		if time.Since(v.successMsgTime) >= 3*time.Second {
			v.successMsg = ""
		}
		return v, nil

	case tea.KeyMsg:
		// 优先处理确认对话框的按键
		if v.showConfirmDialog {
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
					image := v.confirmImage

					// 重置对话框状态
					v.showConfirmDialog = false
					v.confirmAction = ""
					v.confirmImage = nil
					v.confirmSelection = 0

					// 执行操作
					if action == "remove" && image != nil {
						return v, v.removeImage(image)
					} else if action == "prune" {
						return v, v.pruneImages()
					}
				} else {
					// 选择了 Cancel，取消操作
					v.showConfirmDialog = false
					v.confirmAction = ""
					v.confirmImage = nil
					v.confirmSelection = 0
				}
				return v, nil
			case tea.KeyEsc:
				// ESC 直接取消
				v.showConfirmDialog = false
				v.confirmAction = ""
				v.confirmImage = nil
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
			
			// 在对话框模式下，也检查字符串形式的 enter
			if msg.String() == "enter" {
				// 确认选择
				if v.confirmSelection == 1 {
					// 选择了 OK，执行操作
					action := v.confirmAction
					image := v.confirmImage

					// 重置对话框状态
					v.showConfirmDialog = false
					v.confirmAction = ""
					v.confirmImage = nil
					v.confirmSelection = 0

					// 执行操作
					if action == "remove" && image != nil {
						return v, v.removeImage(image)
					} else if action == "prune" {
						return v, v.pruneImages()
					}
				} else {
					// 选择了 Cancel，取消操作
					v.showConfirmDialog = false
					v.confirmAction = ""
					v.confirmImage = nil
					v.confirmSelection = 0
				}
				return v, nil
			}
			
			// 在对话框模式下，忽略其他按键
			return v, nil
		}

		// 处理搜索模式
		if v.isSearching {
			switch msg.String() {
			case "enter":
				v.isSearching = false
				return v, nil
			case "esc":
				v.isSearching = false
				v.searchQuery = ""
				v.applyFilters()
				v.updateColumnWidths()
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

		// 处理快捷键
		switch msg.String() {
		case "/":
			v.isSearching = true
			v.searchQuery = ""
			return v, nil
		case "r", "f5":
			v.loading = true
			v.errorMsg = ""
			return v, v.loadImages
		case "j", "down":
			if v.scrollTable != nil {
				v.scrollTable.MoveDown(1)
			}
			v.tableModel.MoveDown(1)
			return v, nil
		case "k", "up":
			if v.scrollTable != nil {
				v.scrollTable.MoveUp(1)
			}
			v.tableModel.MoveUp(1)
			return v, nil
		case "g":
			if v.scrollTable != nil {
				v.scrollTable.GotoTop()
			}
			v.tableModel.GotoTop()
			return v, nil
		case "G":
			if v.scrollTable != nil {
				v.scrollTable.GotoBottom()
			}
			v.tableModel.GotoBottom()
			return v, nil
		case "h", "left":
			if v.scrollTable != nil {
				v.scrollTable.ScrollLeft()
			}
			return v, nil
		case "l", "right":
			if v.scrollTable != nil {
				v.scrollTable.ScrollRight()
			}
			return v, nil
		case "enter":
			// 查看镜像详情
			image := v.GetSelectedImage()
			if image == nil {
				return v, nil
			}
			// TODO: 实现镜像详情视图
			v.successMsg = "⚠️ 镜像详情功能开发中..."
			v.successMsgTime = time.Now()
			return v, v.clearSuccessMessageAfter(2 * time.Second)
		case "d":
			// 删除镜像
			return v, v.showRemoveConfirmDialog()
		case "p":
			// 清理悬垂镜像
			return v, v.showPruneConfirmDialog()
		}
	}

	return v, nil
}

// View 渲染镜像列表视图
func (v *ImageListView) View() string {
	var s string

	// 顶部状态栏和操作提示
	s += v.renderStatusBar()

	// 显示成功消息（如果有）
	if v.successMsg != "" {
		msgStyle := imageSuccessMsgStyle
		if strings.HasPrefix(v.successMsg, "⚠️") {
			msgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Bold(true)
		}
		s += "\n  " + msgStyle.Render(v.successMsg) + "\n"
	}

	// 统计信息栏
	s += v.renderStatsBar()

	// 加载中状态
	if v.loading {
		loadingContent := lipgloss.JoinVertical(lipgloss.Center,
			"",
			imageStatusBarKeyStyle.Render("⏳ 正在加载镜像列表..."),
			"",
			imageSearchHintStyle.Render("请稍候，正在从 Docker 获取数据"),
			"",
		)
		s += "\n  " + imageStateBoxStyle.Render(loadingContent) + "\n"
		return s
	}

	// 错误状态
	if v.errorMsg != "" {
		errorContent := lipgloss.JoinVertical(lipgloss.Left,
			"",
			imageErrorMsgStyle.Render("❌ 加载失败: "+v.errorMsg),
			"",
			imageStatusBarLabelStyle.Render("💡 可能的原因:"),
			imageSearchHintStyle.Render("   • Docker 守护进程未运行"),
			imageSearchHintStyle.Render("   • 网络连接问题"),
			imageSearchHintStyle.Render("   • 权限不足"),
			"",
			imageStatusBarKeyStyle.Render("按 r 重新加载") + imageSearchHintStyle.Render(" 或 ") + imageStatusBarKeyStyle.Render("按 Esc 返回"),
			"",
		)
		s += "\n  " + imageStateBoxStyle.Render(errorContent) + "\n"
		return s
	}

	// 空状态
	if len(v.images) == 0 {
		emptyContent := lipgloss.JoinVertical(lipgloss.Left,
			"",
			imageSearchHintStyle.Render("📦 暂无镜像"),
			"",
			imageStatusBarLabelStyle.Render("💡 快速开始:"),
			"",
			imageStatusBarKeyStyle.Render("1.") + imageSearchHintStyle.Render(" 拉取一个镜像:"),
			imageSearchHintStyle.Render("   docker pull nginx"),
			"",
			imageStatusBarKeyStyle.Render("2.") + imageSearchHintStyle.Render(" 刷新镜像列表:"),
			imageSearchHintStyle.Render("   按 r 键刷新"),
			"",
		)
		s += "\n  " + imageStateBoxStyle.Render(emptyContent) + "\n"
		return s
	}

	// 使用可滚动表格渲染
	if v.scrollTable != nil {
		s += v.scrollTable.View() + "\n"
	} else {
		s += "  " + v.tableModel.View() + "\n"
	}

	// 底部搜索输入栏（如果处于搜索模式）
	if v.isSearching {
		searchLine := "\n  " + strings.Repeat("─", 67) + "\n"
		searchPrompt := "  " + imageSearchPromptStyle.Render("Search:") + " "
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		searchInput := v.searchQuery + cursor
		cancelHint := imageSearchHintStyle.Render("[Enter=Confirm | ESC=Cancel]")

		s += searchLine + searchPrompt + searchInput + "    " + cancelHint + "\n"
	}

	// 如果显示确认对话框，叠加在内容上
	if v.showConfirmDialog {
		s = v.overlayDialog(s)
	}

	return s
}

// SetSize 设置视图尺寸
func (v *ImageListView) SetSize(width, height int) {
	v.width = width
	v.height = height

	// 调整表格高度
	tableHeight := height - 15
	if tableHeight < 5 {
		tableHeight = 5
	}
	v.tableModel.SetHeight(tableHeight)

	// 更新可滚动表格尺寸
	if v.scrollTable != nil {
		v.scrollTable.SetSize(width-4, tableHeight)
	}
}

// renderStatusBar 渲染顶部状态栏
func (v *ImageListView) renderStatusBar() string {
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

	itemWidth := shortcutsWidth / itemsPerRow
	if itemWidth < 12 {
		itemWidth = 12
	}

	labelStyle := lipgloss.NewStyle().
		Width(labelColWidth).
		Foreground(lipgloss.Color("220")).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	descStyle := lipgloss.NewStyle()

	itemStyle := lipgloss.NewStyle().
		Width(itemWidth)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	makeItem := func(key, desc string) string {
		return itemStyle.Render(keyStyle.Render(key) + descStyle.Render(" "+desc))
	}

	var lines []string

	// 第一行：Docker 状态 + 基本操作
	row1Label := labelStyle.Render("Docker: Connected")
	row1Keys := makeItem("<a>", "Filter") + makeItem("</>", "Search") + makeItem("<r>", "Refresh")
	lines = append(lines, "  "+row1Label+row1Keys)

	// 第二行：镜像操作
	row2Label := labelStyle.Render("Ops:")
	row2Keys := makeItem("<d>", "Delete") + makeItem("<p>", "Prune") + makeItem("<i>", "Inspect") + makeItem("<e>", "Export")
	lines = append(lines, "  "+row2Label+row2Keys)

	// 第三行：高级操作
	row3Label := labelStyle.Render("Advanced:")
	row3Keys := makeItem("<t>", "Tag") + makeItem("<u>", "Untag") + makeItem("<P>", "Push") + makeItem("<p>", "Pull")
	lines = append(lines, "  "+row3Label+row3Keys)

	// 第四行：查看操作
	row4Label := labelStyle.Render("View:")
	row4Keys := makeItem("<Enter>", "Details") + makeItem("<c>", "Containers") + makeItem("<Esc>", "Back") + makeItem("<q>", "Quit")
	lines = append(lines, "  "+row4Label+row4Keys)

	// 第五行：版本 + 刷新时间 + vim 提示
	versionInfo := "v0.1.0"
	refreshInfo := "-"
	if !v.lastRefreshTime.IsZero() {
		refreshInfo = formatDuration(time.Since(v.lastRefreshTime)) + " ago"
	}

	row5Label := labelStyle.Render("Version: " + versionInfo)
	row5Info := hintStyle.Render("Last Refresh: "+refreshInfo) + "    " +
		hintStyle.Render("(vim): j/k=上下  h/l=左右滚动  Enter=选择  Esc=返回  q=退出")
	lines = append(lines, "  "+row5Label+row5Info)

	return "\n" + strings.Join(lines, "\n") + "\n"
}

// renderStatsBar 渲染统计信息栏
func (v *ImageListView) renderStatsBar() string {
	totalCount := len(v.images)
	showingCount := len(v.filteredImages)

	// 统计各状态镜像数量
	activeCount := 0
	danglingCount := 0
	unusedCount := 0

	for _, img := range v.images {
		if img.InUse {
			activeCount++
		}
		if img.Dangling {
			danglingCount++
		}
		if !img.InUse && !img.Dangling {
			unusedCount++
		}
	}

	// 构建统计信息
	totalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	activeStyleColor := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	danglingStyleColor := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	unusedStyleColor := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	statsContent := totalStyle.Render(fmt.Sprintf("📦 Total: %d", totalCount)) +
		separatorStyle.Render("  │  ") +
		activeStyleColor.Render(fmt.Sprintf("🟢 Active: %d", activeCount)) +
		separatorStyle.Render("  │  ") +
		danglingStyleColor.Render(fmt.Sprintf("🟡 Dangling: %d", danglingCount)) +
		separatorStyle.Render("  │  ") +
		unusedStyleColor.Render(fmt.Sprintf("🔴 Unused: %d", unusedCount))

	// 搜索附加信息
	if showingCount != totalCount || (!v.isSearching && v.searchQuery != "") {
		filterParts := []string{}
		if showingCount != totalCount {
			filterParts = append(filterParts, fmt.Sprintf("Showing: %d", showingCount))
		}
		if !v.isSearching && v.searchQuery != "" {
			filterParts = append(filterParts, fmt.Sprintf("Search: \"%s\"", v.searchQuery))
		}
		filterInfo := imageSearchHintStyle.Render("  [" + strings.Join(filterParts, " | ") + "]")
		statsContent += filterInfo
	}

	lineWidth := v.width - 6
	if lineWidth < 60 {
		lineWidth = 60
	}
	line := lineStyle.Render(strings.Repeat("─", lineWidth))

	statsLine := lipgloss.NewStyle().Width(lineWidth).Align(lipgloss.Center).Render(statsContent)

	return "\n  " + line + "\n" +
		"  " + statsLine + "\n" +
		"  " + line + "\n"
}

// loadImages 加载镜像列表
func (v *ImageListView) loadImages() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 调用 Docker 客户端获取所有镜像（包括悬垂镜像）
	images, err := v.dockerClient.ListImages(ctx, true)
	if err != nil {
		return imagesLoadErrorMsg{err: err}
	}

	return imagesLoadedMsg{images: images}
}

// applyFilters 应用过滤和搜索
func (v *ImageListView) applyFilters() {
	v.filteredImages = make([]docker.Image, 0)

	for _, img := range v.images {
		// 应用搜索过滤
		if v.searchQuery != "" {
			query := strings.ToLower(v.searchQuery)
			if !strings.Contains(strings.ToLower(img.Repository), query) &&
				!strings.Contains(strings.ToLower(img.Tag), query) &&
				!strings.Contains(strings.ToLower(img.ID), query) {
				continue
			}
		}

		// 应用类型过滤
		switch v.filterType {
		case "active":
			if !img.InUse {
				continue
			}
		case "dangling":
			if !img.Dangling {
				continue
			}
		case "unused":
			if img.InUse || img.Dangling {
				continue
			}
		}

		v.filteredImages = append(v.filteredImages, img)
	}
}

// 消息类型定义
type imagesLoadedMsg struct {
	images []docker.Image
}

type imagesLoadErrorMsg struct {
	err error
}

// updateColumnWidths 根据实际数据计算并更新列宽
func (v *ImageListView) updateColumnWidths() {
	// 计算每列内容的最大宽度
	maxID := 12         // IMAGE ID 固定 12 位
	maxRepository := 10 // REPOSITORY
	maxTag := 3         // TAG
	maxSize := 4        // SIZE
	maxCreated := 7     // CREATED

	for _, img := range v.filteredImages {
		if len(img.Repository) > maxRepository {
			maxRepository = len(img.Repository)
		}
		if len(img.Tag) > maxTag {
			maxTag = len(img.Tag)
		}
		sizeStr := formatSize(img.Size)
		if len(sizeStr) > maxSize {
			maxSize = len(sizeStr)
		}
		created := formatCreatedTime(img.Created)
		if len(created) > maxCreated {
			maxCreated = len(created)
		}
	}

	// 可用宽度
	availableWidth := v.width - 10

	// 固定列宽
	idWidth := maxID + 2

	// 计算需要的总宽度
	totalNeeded := idWidth + maxRepository + maxTag + maxSize + maxCreated + 10

	// 如果总宽度足够，使用实际内容宽度
	if totalNeeded <= availableWidth {
		v.tableModel.SetColumns([]table.Column{
			{Title: "IMAGE ID", Width: idWidth},
			{Title: "REPOSITORY", Width: maxRepository + 2},
			{Title: "TAG", Width: maxTag + 2},
			{Title: "SIZE", Width: maxSize + 2},
			{Title: "CREATED", Width: maxCreated + 2},
		})
	} else {
		// 宽度不够，按比例分配
		flexWidth := availableWidth - idWidth - 6

		// 按内容比例分配
		totalVar := maxRepository + maxTag + maxSize + maxCreated
		if totalVar == 0 {
			totalVar = 1
		}

		repoWidth := flexWidth * maxRepository / totalVar
		tagWidth := flexWidth * maxTag / totalVar
		sizeWidth := flexWidth * maxSize / totalVar
		createdWidth := flexWidth * maxCreated / totalVar

		// 确保最小宽度
		if repoWidth < 20 {
			repoWidth = 20
		}
		if tagWidth < 10 {
			tagWidth = 10
		}
		if sizeWidth < 8 {
			sizeWidth = 8
		}
		if createdWidth < 12 {
			createdWidth = 12
		}

		v.tableModel.SetColumns([]table.Column{
			{Title: "IMAGE ID", Width: idWidth},
			{Title: "REPOSITORY", Width: repoWidth},
			{Title: "TAG", Width: tagWidth},
			{Title: "SIZE", Width: sizeWidth},
			{Title: "CREATED", Width: createdWidth},
		})
	}

	// 更新可滚动表格的列宽和数据
	if v.scrollTable != nil {
		v.scrollTable.SetColumns([]TableColumn{
			{Title: "IMAGE ID", Width: maxID + 2},
			{Title: "REPOSITORY", Width: maxRepository + 2},
			{Title: "TAG", Width: maxTag + 2},
			{Title: "SIZE", Width: maxSize + 2},
			{Title: "CREATED", Width: maxCreated + 2},
		})

		// 转换数据为 TableRow
		if len(v.filteredImages) > 0 {
			rows := make([]TableRow, len(v.filteredImages))
			for i, img := range v.filteredImages {
				created := formatCreatedTime(img.Created)
				size := formatSize(img.Size)
				rows[i] = TableRow{
					img.ShortID,
					img.Repository,
					img.Tag,
					size,
					created,
				}
			}
			v.scrollTable.SetRows(rows)
		}
	}

	// 重新渲染表格数据
	if len(v.filteredImages) > 0 {
		rows := v.imagesToRows(v.filteredImages)
		v.tableModel.SetRows(rows)
	}
}

// imagesToRows 将镜像数据转换为 table.Row
func (v *ImageListView) imagesToRows(images []docker.Image) []table.Row {
	rows := make([]table.Row, len(images))

	for i, img := range images {
		// CREATED - 友好格式
		created := formatCreatedTime(img.Created)

		// SIZE - 友好格式
		size := formatSize(img.Size)

		// 根据镜像状态应用样式
		var styledRepo, styledTag string
		if img.Dangling {
			// 悬垂镜像 - 灰色
			styledRepo = imageDanglingStyle.Render(img.Repository)
			styledTag = imageDanglingStyle.Render(img.Tag)
		} else if img.InUse {
			// 活跃镜像 - 绿色
			styledRepo = imageActiveStyle.Render(img.Repository)
			styledTag = imageActiveStyle.Render(img.Tag)
		} else {
			// 未使用镜像 - 灰色
			styledRepo = imageUnusedStyle.Render(img.Repository)
			styledTag = imageUnusedStyle.Render(img.Tag)
		}

		rows[i] = table.Row{
			img.ShortID,
			styledRepo,
			styledTag,
			size,
			created,
		}
	}

	return rows
}

// formatSize 格式化文件大小为友好格式
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}


// GetSelectedImage 获取当前选中的镜像
func (v *ImageListView) GetSelectedImage() *docker.Image {
	if len(v.filteredImages) == 0 {
		return nil
	}
	// 优先从可滚动表格获取选中索引
	var selectedIndex int
	if v.scrollTable != nil {
		selectedIndex = v.scrollTable.Cursor()
	} else {
		selectedIndex = v.tableModel.Cursor()
	}
	if selectedIndex < 0 || selectedIndex >= len(v.filteredImages) {
		return nil
	}
	return &v.filteredImages[selectedIndex]
}

// overlayDialog 将对话框叠加到现有内容上（居中显示）
func (v *ImageListView) overlayDialog(baseContent string) string {
	// 将基础内容按行分割
	lines := strings.Split(baseContent, "\n")

	// 对话框尺寸
	dialogHeight := 9

	// 计算对话框应该插入的位置（垂直居中）
	insertLine := 0
	if len(lines) > dialogHeight {
		insertLine = (len(lines) - dialogHeight) / 2
	}

	// 获取对话框内容
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

// renderConfirmDialogContent 渲染对话框内容
func (v *ImageListView) renderConfirmDialogContent() string {
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

	var title, warning string

	if v.confirmAction == "remove" && v.confirmImage != nil {
		// 删除镜像对话框
		imageName := v.confirmImage.Repository + ":" + v.confirmImage.Tag
		if len(imageName) > 35 {
			imageName = imageName[:32] + "..."
		}

		title = titleStyle.Render("⚠️  Delete Image: " + imageName)
		if v.confirmImage.InUse {
			warning = warningStyle.Render("⚠️  镜像正在被容器使用，将强制删除！")
		} else {
			warning = warningStyle.Render("This action cannot be undone!")
		}
	} else if v.confirmAction == "prune" {
		// 清理悬垂镜像对话框
		title = titleStyle.Render("⚠️  Prune Dangling Images")
		warning = warningStyle.Render("将删除所有无标签的悬垂镜像，释放磁盘空间")
	}

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

// showRemoveConfirmDialog 显示删除确认对话框
func (v *ImageListView) showRemoveConfirmDialog() tea.Cmd {
	image := v.GetSelectedImage()
	if image == nil {
		return func() tea.Msg {
			return imageOperationErrorMsg{
				operation: "删除镜像",
				image:     "",
				err:       fmt.Errorf("请先选择一个镜像"),
			}
		}
	}

	// 显示确认对话框
	v.showConfirmDialog = true
	v.confirmAction = "remove"
	v.confirmImage = image
	v.confirmSelection = 0 // 默认选中 Cancel

	return nil
}

// showPruneConfirmDialog 显示清理悬垂镜像确认对话框
func (v *ImageListView) showPruneConfirmDialog() tea.Cmd {
	// 显示确认对话框
	v.showConfirmDialog = true
	v.confirmAction = "prune"
	v.confirmImage = nil
	v.confirmSelection = 0 // 默认选中 Cancel

	return nil
}

// removeImage 删除镜像
func (v *ImageListView) removeImage(image *docker.Image) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 如果镜像正在被使用，使用强制删除
		force := image.InUse

		// 删除镜像
		err := v.dockerClient.RemoveImage(ctx, image.ID, force, false)
		if err != nil {
			return imageOperationErrorMsg{
				operation: "删除镜像",
				image:     image.Repository + ":" + image.Tag,
				err:       err,
			}
		}

		return imageOperationSuccessMsg{
			operation: "删除",
			image:     image.Repository + ":" + image.Tag,
		}
	}
}

// pruneImages 清理悬垂镜像
func (v *ImageListView) pruneImages() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// 清理悬垂镜像
		count, spaceReclaimed, err := v.dockerClient.PruneImages(ctx)
		if err != nil {
			return imageOperationErrorMsg{
				operation: "清理悬垂镜像",
				image:     "",
				err:       err,
			}
		}

		return imageOperationSuccessMsg{
			operation: "清理悬垂镜像",
			image:     fmt.Sprintf("删除了 %d 个镜像，释放 %s 空间", count, formatSize(spaceReclaimed)),
		}
	}
}

// clearSuccessMessageAfter 在指定时间后清除成功消息
func (v *ImageListView) clearSuccessMessageAfter(duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(duration)
		return clearSuccessMessageMsg{}
	}
}

// 消息类型定义
type imageOperationSuccessMsg struct {
	operation string // 操作类型: 删除, 清理悬垂镜像
	image     string // 镜像名称或描述
}

type imageOperationErrorMsg struct {
	operation string // 操作类型
	image     string // 镜像名称
	err       error  // 错误信息
}

// clearSuccessMessageMsg 已在 container_list_view.go 中定义，这里复用

