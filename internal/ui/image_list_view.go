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
	"docktui/internal/task"
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
	errorMsg       string         // 错误信息（初始加载失败时使用）
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
	confirmAction     string            // 确认的操作类型: "remove", "prune", "pull"
	confirmImage      *docker.Image     // 待操作的镜像
	confirmSelection  int               // 确认对话框中的选择: 0=Cancel, 1=OK
	confirmPullRef    string            // 待拉取的镜像引用

	// 快捷键管理
	keys KeyMap

	// 拉取功能
	pullInput *PullInputView // 拉取输入框
	taskBar   *TaskBar       // 任务进度条

	// 打标签功能
	tagInput *TagInputView // 打标签输入框
	
	// 错误弹窗
	errorDialog *ErrorDialog // 错误弹窗组件

	// JSON 查看器
	jsonViewer *JSONViewer // JSON 查看器

	// 多选功能
	selectedImages map[string]bool // 已选中的镜像 ID

	// 导出功能
	exportInput *ExportInputView // 导出输入视图
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
		{Title: "SEL", Width: 3},  // 选择列
		{Title: "IMAGE ID", Width: 14},
		{Title: "REPOSITORY", Width: 35},
		{Title: "TAG", Width: 25},
		{Title: "SIZE", Width: 12},
		{Title: "CREATED", Width: 16},
	}
	scrollTable := NewScrollableTable(scrollColumns)

	return &ImageListView{
		dockerClient:   dockerClient,
		tableModel:     t,
		scrollTable:    scrollTable,
		keys:           DefaultKeyMap(),
		searchQuery:    "",
		isSearching:    false,
		filterType:     "all",
		sortBy:         "created",
		pullInput:      NewPullInputView(),
		taskBar:        NewTaskBar(),
		tagInput:       NewTagInputView(),
		errorDialog:    NewErrorDialog(),
		jsonViewer:     NewJSONViewer(),
		selectedImages: make(map[string]bool),
		exportInput:    NewExportInputView(),
	}
}

// Init 初始化镜像列表视图
func (v *ImageListView) Init() tea.Cmd {
	v.loading = true
	return v.loadImages
}

// Update 处理消息并更新视图状态
func (v *ImageListView) Update(msg tea.Msg) (View, tea.Cmd) {
	// 如果显示导出输入视图，优先处理
	if v.exportInput != nil && v.exportInput.IsVisible() {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			wasVisible := v.exportInput.IsVisible()
			v.exportInput.Update(keyMsg)
			// 如果导出输入视图刚刚关闭且有任务启动，开始监听事件
			if wasVisible && !v.exportInput.IsVisible() && v.taskBar.HasActiveTasks() {
				return v, v.taskBar.ListenForEvents()
			}
			return v, nil
		}
	}

	// 如果显示 JSON 查看器，优先处理
	if v.jsonViewer != nil && v.jsonViewer.IsVisible() {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if v.jsonViewer.Update(keyMsg) {
				return v, nil
			}
		}
	}

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
		// 镜像操作失败，显示错误弹窗
		errMsg := fmt.Sprintf("%s失败 (%s): %v", msg.operation, msg.image, msg.err)
		if v.errorDialog != nil {
			v.errorDialog.ShowError(errMsg)
		}
		v.successMsg = "" // 清除成功消息
		return v, nil

	case imageInUseErrorMsg:
		// 镜像被容器引用，提示用户是否强制删除
		v.showForceRemoveConfirmDialog(msg.image)
		return v, nil

	case clearSuccessMessageMsg:
		// 清除成功消息
		if time.Since(v.successMsgTime) >= 3*time.Second {
			v.successMsg = ""
		}
		return v, nil

	case imageInspectMsg:
		// 显示 JSON 查看器
		if v.jsonViewer != nil {
			v.jsonViewer.SetSize(v.width, v.height)
			v.jsonViewer.Show("Image Inspect: "+msg.imageName, msg.jsonContent)
		}
		return v, nil

	case imageInspectErrorMsg:
		if v.errorDialog != nil {
			v.errorDialog.ShowError(fmt.Sprintf("获取镜像信息失败: %v", msg.err))
		}
		return v, nil

	case imageExportSuccessMsg:
		v.successMsg = fmt.Sprintf("✅ 成功导出 %d 个镜像到 %s", msg.count, msg.dir)
		v.successMsgTime = time.Now()
		// 清除选择
		v.selectedImages = make(map[string]bool)
		v.updateTableData()
		return v, v.clearSuccessMessageAfter(5 * time.Second)

	case imageExportErrorMsg:
		if v.errorDialog != nil {
			v.errorDialog.ShowError(fmt.Sprintf("导出镜像失败: %v", msg.err))
		}
		return v, nil

	case imageExportProgressMsg:
		v.successMsg = fmt.Sprintf("⏳ 正在导出 [%d/%d]: %s", msg.current, msg.total, msg.name)
		v.successMsgTime = time.Now()
		return v, nil

	case TaskEventMsg:
		// 处理任务事件
		event := msg.Event
		switch event.Type {
		case task.EventCompleted:
			// 任务完成，刷新镜像列表
			v.successMsg = fmt.Sprintf("✅ %s", event.Message)
			v.successMsgTime = time.Now()
			return v, tea.Batch(
				v.loadImages,
				v.clearSuccessMessageAfter(3*time.Second),
				v.taskBar.ListenForEvents(),
			)
		case task.EventFailed:
			// 任务失败，显示错误弹窗
			errMsg := fmt.Sprintf("%s: %v", event.TaskName, event.Error)
			if v.errorDialog != nil {
				v.errorDialog.ShowError(errMsg)
			}
			return v, v.taskBar.ListenForEvents()
		case task.EventCancelled:
			// 任务已取消
			v.successMsg = fmt.Sprintf("⏹️ %s 已取消", event.TaskName)
			v.successMsgTime = time.Now()
			return v, tea.Batch(
				v.clearSuccessMessageAfter(3*time.Second),
				v.taskBar.ListenForEvents(),
			)
		case task.EventProgress, task.EventStarted:
			// 进度更新或任务开始，继续监听
			// 不需要做任何事情，View() 会自动从 TaskBar 获取最新状态
			return v, v.taskBar.ListenForEvents()
		default:
			return v, v.taskBar.ListenForEvents()
		}

	case taskTickMsg:
		// 定时刷新任务状态
		if v.taskBar.HasActiveTasks() {
			// 继续定时刷新
			return v, v.scheduleTaskTick()
		}
		return v, nil

	case tea.KeyMsg:
		// 优先处理错误弹窗
		if v.errorDialog != nil && v.errorDialog.IsVisible() {
			if v.errorDialog.Update(msg) {
				return v, nil
			}
		}
		
		// 优先处理拉取输入框
		if v.pullInput.IsVisible() {
			confirmed, handled, cmd := v.pullInput.Update(msg)
			if confirmed {
				// 用户确认拉取
				imageRef := v.pullInput.Value()
				v.pullInput.Hide()
				// 直接启动拉取任务
				v.startPullTaskSync(imageRef)
				// 同时启动事件监听和定时刷新
				return v, tea.Batch(
					v.taskBar.ListenForEvents(),
					v.scheduleTaskTick(),
				)
			}
			if handled {
				// 事件已被处理，阻止继续传播
				return v, cmd
			}
		}

		// 优先处理打标签输入框
		if v.tagInput.IsVisible() {
			confirmed, handled, cmd := v.tagInput.Update(msg)
			if confirmed {
				// 用户确认打标签
				repo, tag := v.tagInput.GetValues()
				sourceImageID := v.tagInput.sourceImageID
				v.tagInput.Hide()
				// 执行打标签操作
				return v, v.tagImage(sourceImageID, repo, tag)
			}
			if handled {
				// 事件已被处理，阻止继续传播
				return v, cmd
			}
		}

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
					pullRef := v.confirmPullRef

					// 重置对话框状态
					v.showConfirmDialog = false
					v.confirmAction = ""
					v.confirmImage = nil
					v.confirmPullRef = ""
					v.confirmSelection = 0

					// 执行操作
					if action == "remove" && image != nil {
						return v, v.removeImage(image, false) // 普通删除
					} else if action == "force_remove" && image != nil {
						return v, v.removeImage(image, true) // 强制删除
					} else if action == "prune" {
						return v, v.pruneImages()
					} else if action == "pull" && pullRef != "" {
						v.startPullTaskSync(pullRef)
						return v, v.taskBar.ListenForEvents()
					}
				} else {
					// 选择了 Cancel，取消操作
					v.showConfirmDialog = false
					v.confirmAction = ""
					v.confirmImage = nil
					v.confirmPullRef = ""
					v.confirmSelection = 0
				}
				return v, nil
			case tea.KeyEsc:
				// ESC 直接取消
				v.showConfirmDialog = false
				v.confirmAction = ""
				v.confirmImage = nil
				v.confirmPullRef = ""
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
					pullRef := v.confirmPullRef

					// 重置对话框状态
					v.showConfirmDialog = false
					v.confirmAction = ""
					v.confirmImage = nil
					v.confirmPullRef = ""
					v.confirmSelection = 0

					// 执行操作
					if action == "remove" && image != nil {
						return v, v.removeImage(image, false) // 普通删除
					} else if action == "force_remove" && image != nil {
						return v, v.removeImage(image, true) // 强制删除
					} else if action == "prune" {
						return v, v.pruneImages()
					} else if action == "pull" && pullRef != "" {
						v.startPullTaskSync(pullRef)
						return v, v.taskBar.ListenForEvents()
					}
				} else {
					// 选择了 Cancel，取消操作
					v.showConfirmDialog = false
					v.confirmAction = ""
					v.confirmImage = nil
					v.confirmPullRef = ""
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
		case "esc":
			// ESC 优先级：清除搜索 > 清除筛选 > 返回上一级
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
			// 没有搜索和筛选条件，返回上一级
			return v, func() tea.Msg { return GoBackMsg{} }
		case "f":
			// 切换筛选状态：all -> active -> dangling -> unused -> all
			switch v.filterType {
			case "all":
				v.filterType = "active"
			case "active":
				v.filterType = "dangling"
			case "dangling":
				v.filterType = "unused"
			case "unused":
				v.filterType = "all"
			default:
				v.filterType = "all"
			}
			v.applyFilters()
			v.updateColumnWidths()
			return v, nil
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
			// 查看镜像详情 - 发送消息给父视图
			image := v.GetSelectedImage()
			if image == nil {
				return v, nil
			}
			return v, func() tea.Msg {
				return ViewImageDetailsMsg{Image: image}
			}
		case "d":
			// 删除镜像
			return v, v.showRemoveConfirmDialog()
		case "p":
			// 清理悬垂镜像
			return v, v.showPruneConfirmDialog()
		case "P":
			// 拉取镜像
			v.pullInput.SetWidth(v.width)
			v.pullInput.Show()
			return v, nil
		case "T":
			// 切换任务栏展开/收起
			v.taskBar.Toggle()
			return v, nil
		case "t":
			// 打标签
			return v, v.showTagInput()
		case "i":
			// 检查镜像（显示 JSON）
			return v, v.inspectImage()
		case " ":
			// 空格键：切换选择当前镜像
			image := v.GetSelectedImage()
			if image != nil {
				if v.selectedImages[image.ID] {
					delete(v.selectedImages, image.ID)
				} else {
					v.selectedImages[image.ID] = true
				}
				v.updateTableData()
			}
			return v, nil
		case "a":
			// 全选/取消全选
			// 检查是否所有过滤后的镜像都已选中
			allSelected := true
			for _, img := range v.filteredImages {
				if !v.selectedImages[img.ID] {
					allSelected = false
					break
				}
			}
			if allSelected && len(v.filteredImages) > 0 {
				// 已全选，取消全选
				v.selectedImages = make(map[string]bool)
			} else {
				// 全选
				for _, img := range v.filteredImages {
					v.selectedImages[img.ID] = true
				}
			}
			v.updateTableData()
			return v, nil
		case "E":
			// 导出镜像
			return v, v.showExportDialog()
		case "x":
			// 取消当前任务
			if v.taskBar.HasActiveTasks() {
				v.taskBar.CancelFirstTask()
				v.successMsg = "⏹️ 正在取消任务..."
				v.successMsgTime = time.Now()
				return v, v.clearSuccessMessageAfter(2 * time.Second)
			}
			return v, nil
		}
	}

	return v, nil
}

// View 渲染镜像列表视图
func (v *ImageListView) View() string {
	// 如果显示 JSON 查看器
	if v.jsonViewer != nil && v.jsonViewer.IsVisible() {
		return v.jsonViewer.View()
	}

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

	// 底部左下角筛选状态提示（非搜索模式时显示）
	if !v.isSearching && v.filterType != "all" {
		filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
		s += "  " + filterStyle.Render("[Filter: "+v.filterType+"]") + "  " + imageSearchHintStyle.Render("按 ESC 清除筛选，按 f 切换") + "\n"
	}

	// 任务进度条（如果有活跃任务）
	if v.taskBar.HasActiveTasks() {
		v.taskBar.SetWidth(v.width)
		s += v.taskBar.View()
	}

	// 如果显示拉取输入框，叠加在内容上
	if v.pullInput.IsVisible() {
		s = v.overlayPullInput(s)
	}

	// 如果显示打标签输入框，叠加在内容上
	if v.tagInput.IsVisible() {
		s = v.overlayTagInput(s)
	}

	// 如果显示确认对话框，叠加在内容上
	if v.showConfirmDialog {
		s = v.overlayDialog(s)
	}
	
	// 如果显示错误弹窗，叠加在内容上
	if v.errorDialog != nil && v.errorDialog.IsVisible() {
		s = v.errorDialog.Overlay(s)
	}

	// 如果显示导出输入视图，叠加在内容上
	if v.exportInput != nil && v.exportInput.IsVisible() {
		s = v.overlayExportInput(s)
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

	// 更新拉取输入框和任务栏尺寸
	v.pullInput.SetWidth(width)
	v.taskBar.SetWidth(width)
	
	// 更新错误弹窗宽度
	if v.errorDialog != nil {
		v.errorDialog.SetWidth(width)
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

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)

	makeItem := func(key, desc string) string {
		return itemStyle.Render(keyStyle.Render(key) + descStyle.Render(" "+desc))
	}

	var lines []string

	// 第一行：镜像标题 + 基本操作
	row1Label := labelStyle.Render("🖼️ Images")
	row1Keys := makeItem("<f>", "Filter") + makeItem("</>", "Search") + makeItem("<r>", "Refresh")
	lines = append(lines, "  "+row1Label+row1Keys)

	// 第二行：镜像操作
	row2Label := labelStyle.Render("Ops:")
	row2Keys := makeItem("<d>", "Delete") + makeItem("<p>", "Prune") + makeItem("<P>", "Pull")
	lines = append(lines, "  "+row2Label+row2Keys)

	// 第三行：高级操作（包含多选和导出）
	row3Label := labelStyle.Render("Advanced:")
	row3Keys := makeItem("<t>", "Tag") + makeItem("<Space>", "Select") + makeItem("<a>", "All") + makeItem("<E>", "Export")
	lines = append(lines, "  "+row3Label+row3Keys)

	// 第四行：刷新时间 + vim 提示 + 选中数量
	refreshInfo := "-"
	if !v.lastRefreshTime.IsZero() {
		refreshInfo = formatDuration(time.Since(v.lastRefreshTime)) + " ago"
	}

	row4Label := labelStyle.Render("Last Refresh:")
	row4Info := hintStyle.Render(refreshInfo) + "    " +
		hintStyle.Render("j/k=上下  Enter=详情  Esc=返回  q=退出")
	
	// 如果有选中的镜像，显示选中数量
	if len(v.selectedImages) > 0 {
		row4Info += "    " + selectedStyle.Render(fmt.Sprintf("[已选: %d]", len(v.selectedImages)))
	}
	
	lines = append(lines, "  "+row4Label+row4Info)

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

// updateTableData 更新表格数据（不重新计算列宽）
func (v *ImageListView) updateTableData() {
	if v.scrollTable == nil || len(v.filteredImages) == 0 {
		return
	}

	rows := make([]TableRow, len(v.filteredImages))
	
	// 定义整行颜色样式
	danglingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	unusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	
	for i, img := range v.filteredImages {
		created := formatCreatedTime(img.Created)
		size := formatSize(img.Size)
		
		// 选择标记
		selMark := " "
		if v.selectedImages[img.ID] {
			selMark = selectedStyle.Render("✓")
		}
		
		// 根据镜像状态决定是否对整行应用颜色
		var rowStyle lipgloss.Style
		var needsStyle bool
		
		if img.Dangling {
			rowStyle = danglingStyle
			needsStyle = true
		} else if !img.InUse {
			rowStyle = unusedStyle
			needsStyle = true
		} else {
			needsStyle = false
		}
		
		// 构建行数据
		if needsStyle {
			rows[i] = TableRow{
				selMark,
				rowStyle.Render(img.ShortID),
				rowStyle.Render(img.Repository),
				rowStyle.Render(img.Tag),
				rowStyle.Render(size),
				rowStyle.Render(created),
			}
		} else {
			rows[i] = TableRow{
				selMark,
				img.ShortID,
				img.Repository,
				img.Tag,
				size,
				created,
			}
		}
	}
	v.scrollTable.SetRows(rows)
}

// 消息类型定义
type imagesLoadedMsg struct {
	images []docker.Image
}

type imagesLoadErrorMsg struct {
	err error
}

type imageInspectMsg struct {
	imageName   string
	jsonContent string
}

type imageInspectErrorMsg struct {
	err error
}

// 导出相关消息
type imageExportSuccessMsg struct {
	count    int
	dir      string
	fileSize int64
}

type imageExportErrorMsg struct {
	err error
}

type imageExportProgressMsg struct {
	current int
	total   int
	name    string
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
			{Title: "SEL", Width: 3},  // 选择列
			{Title: "IMAGE ID", Width: maxID + 2},
			{Title: "REPOSITORY", Width: maxRepository + 2},
			{Title: "TAG", Width: maxTag + 2},
			{Title: "SIZE", Width: maxSize + 2},
			{Title: "CREATED", Width: maxCreated + 2},
		})

		// 转换数据为 TableRow
		if len(v.filteredImages) > 0 {
			rows := make([]TableRow, len(v.filteredImages))
			
			// 定义整行颜色样式
			danglingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
			unusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
			selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
			
			for i, img := range v.filteredImages {
				created := formatCreatedTime(img.Created)
				size := formatSize(img.Size)
				
				// 选择标记
				selMark := " "
				if v.selectedImages[img.ID] {
					selMark = selectedStyle.Render("✓")
				}
				
				// 根据镜像状态决定是否对整行应用颜色
				var rowStyle lipgloss.Style
				var needsStyle bool
				
				if img.Dangling {
					rowStyle = danglingStyle
					needsStyle = true
				} else if !img.InUse {
					rowStyle = unusedStyle
					needsStyle = true
				} else {
					needsStyle = false
				}
				
				// 构建行数据
				if needsStyle {
					rows[i] = TableRow{
						selMark,
						rowStyle.Render(img.ShortID),
						rowStyle.Render(img.Repository),
						rowStyle.Render(img.Tag),
						rowStyle.Render(size),
						rowStyle.Render(created),
					}
				} else {
					rows[i] = TableRow{
						selMark,
						img.ShortID,
						img.Repository,
						img.Tag,
						size,
						created,
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
	if len(v.filteredImages) > 0 {
		rows := v.imagesToRows(v.filteredImages)
		v.tableModel.SetRows(rows)
	} else {
		v.tableModel.SetRows([]table.Row{})
	}
}

// imagesToRows 将镜像数据转换为 table.Row
func (v *ImageListView) imagesToRows(images []docker.Image) []table.Row {
	rows := make([]table.Row, len(images))

	// 定义整行颜色样式
	danglingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // 黄色 - 悬垂镜像
	unusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))   // 灰色 - 未使用

	for i, img := range images {
		// CREATED - 友好格式
		created := formatCreatedTime(img.Created)

		// SIZE - 友好格式
		size := formatSize(img.Size)

		// 根据镜像状态决定是否对整行应用颜色
		var rowStyle lipgloss.Style
		var needsStyle bool
		
		if img.Dangling {
			// 悬垂镜像 - 黄色整行
			rowStyle = danglingStyle
			needsStyle = true
		} else if !img.InUse {
			// 未使用镜像 - 灰色整行
			rowStyle = unusedStyle
			needsStyle = true
		} else {
			// 活跃镜像 - 不应用样式
			needsStyle = false
		}

		// 构建行数据
		if needsStyle {
			rows[i] = table.Row{
				rowStyle.Render(img.ShortID),
				rowStyle.Render(img.Repository),
				rowStyle.Render(img.Tag),
				rowStyle.Render(size),
				rowStyle.Render(created),
			}
		} else {
			rows[i] = table.Row{
				img.ShortID,
				img.Repository,
				img.Tag,
				size,
				created,
			}
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

// inspectImage 获取镜像的原始 JSON
func (v *ImageListView) inspectImage() tea.Cmd {
	image := v.GetSelectedImage()
	if image == nil {
		return nil
	}

	imageID := image.ID
	imageName := image.Repository + ":" + image.Tag

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		jsonContent, err := v.dockerClient.InspectImageRaw(ctx, imageID)
		if err != nil {
			return imageInspectErrorMsg{err: err}
		}

		return imageInspectMsg{
			imageName:   imageName,
			jsonContent: jsonContent,
		}
	}
}

// overlayPullInput 将拉取输入框叠加到现有内容上（居中显示）
func (v *ImageListView) overlayPullInput(baseContent string) string {
	return OverlayCentered(baseContent, v.pullInput.View(), v.width, v.height)
}

// overlayDialog 将对话框叠加到现有内容上（居中显示）
func (v *ImageListView) overlayDialog(baseContent string) string {
	return OverlayCentered(baseContent, v.renderConfirmDialogContent(), v.width, v.height)
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
		// 普通删除镜像对话框
		imageName := v.confirmImage.Repository + ":" + v.confirmImage.Tag
		if len(imageName) > 35 {
			imageName = imageName[:32] + "..."
		}

		title = titleStyle.Render("⚠️  删除镜像: " + imageName)
		warning = warningStyle.Render("此操作不可撤销！")
	} else if v.confirmAction == "force_remove" && v.confirmImage != nil {
		// 强制删除镜像对话框（镜像被容器引用）
		imageName := v.confirmImage.Repository + ":" + v.confirmImage.Tag
		if len(imageName) > 35 {
			imageName = imageName[:32] + "..."
		}

		title = titleStyle.Render("⚠️  强制删除镜像: " + imageName)
		warning = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render(
			"⚠️  该镜像正在被容器使用！\n") +
			warningStyle.Render("强制删除可能导致相关容器无法正常运行。\n确定要继续吗？")
	} else if v.confirmAction == "prune" {
		// 清理悬垂镜像对话框
		title = titleStyle.Render("⚠️  清理悬垂镜像")
		warning = warningStyle.Render("将删除所有无标签的悬垂镜像，释放磁盘空间")
	} else if v.confirmAction == "pull" && v.confirmPullRef != "" {
		// 拉取镜像确认对话框
		imageName := v.confirmPullRef
		if len(imageName) > 35 {
			imageName = imageName[:32] + "..."
		}

		title = titleStyle.Render("📥  拉取镜像: " + imageName)
		warning = warningStyle.Render("确认要拉取此镜像吗？")
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

// showForceRemoveConfirmDialog 显示强制删除确认对话框（镜像被容器引用时）
func (v *ImageListView) showForceRemoveConfirmDialog(image *docker.Image) {
	v.showConfirmDialog = true
	v.confirmAction = "force_remove"
	v.confirmImage = image
	v.confirmSelection = 0 // 默认选中 Cancel
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
func (v *ImageListView) removeImage(image *docker.Image, force bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 删除镜像
		err := v.dockerClient.RemoveImage(ctx, image.ID, force, false)
		if err != nil {
			// 检查是否是因为镜像被容器引用
			errStr := err.Error()
			if strings.Contains(errStr, "image is being used") || 
			   strings.Contains(errStr, "image has dependent child images") ||
			   strings.Contains(errStr, "conflict") {
				return imageInUseErrorMsg{
					image: image,
					err:   err,
				}
			}
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

// imageInUseErrorMsg 镜像被容器引用错误消息
type imageInUseErrorMsg struct {
	image *docker.Image
	err   error
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

// taskTickMsg 任务状态定时刷新消息
type taskTickMsg struct{}

// scheduleTaskTick 安排下一次任务状态刷新
func (v *ImageListView) scheduleTaskTick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return taskTickMsg{}
	})
}

// startPullTask 启动镜像拉取任务
func (v *ImageListView) startPullTask(imageRef string) tea.Cmd {
	return func() tea.Msg {
		// 创建拉取任务
		pullTask := task.NewPullTask(v.dockerClient, imageRef)
		
		// 提交到任务管理器
		manager := task.GetManager()
		manager.Submit(pullTask)

		// 显示开始消息
		v.successMsg = fmt.Sprintf("📥 开始拉取: %s", imageRef)
		v.successMsgTime = time.Now()

		return nil
	}
}

// startPullTaskSync 同步启动镜像拉取任务（不返回 tea.Cmd）
func (v *ImageListView) startPullTaskSync(imageRef string) {
	// 创建拉取任务
	pullTask := task.NewPullTask(v.dockerClient, imageRef)
	
	// 提交到任务管理器
	manager := task.GetManager()
	manager.Submit(pullTask)

	// 显示开始消息
	v.successMsg = fmt.Sprintf("📥 开始拉取: %s", imageRef)
	v.successMsgTime = time.Now()
}

// showTagInput 显示打标签输入框
func (v *ImageListView) showTagInput() tea.Cmd {
	image := v.GetSelectedImage()
	if image == nil {
		return func() tea.Msg {
			return imageOperationErrorMsg{
				operation: "打标签",
				image:     "",
				err:       fmt.Errorf("请先选择一个镜像"),
			}
		}
	}

	// 设置输入框宽度并显示
	v.tagInput.SetWidth(v.width)
	v.tagInput.Show(
		image.Repository+":"+image.Tag, // 源镜像显示名称
		image.ID,                        // 源镜像 ID
		image.Repository,                // 源镜像仓库名
		image.Tag,                       // 源镜像标签
	)

	return nil
}

// overlayTagInput 将打标签输入框叠加到现有内容上（居中显示）
func (v *ImageListView) overlayTagInput(baseContent string) string {
	return OverlayCentered(baseContent, v.tagInput.View(), v.width, v.height)
}

// tagImage 执行打标签操作
func (v *ImageListView) tagImage(sourceImageID, repository, tag string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 构建目标引用
		targetRef := repository + ":" + tag

		// 调用 Docker 客户端打标签
		err := v.dockerClient.TagImage(ctx, sourceImageID, repository, tag)
		if err != nil {
			return imageOperationErrorMsg{
				operation: "打标签",
				image:     targetRef,
				err:       err,
			}
		}

		return imageOperationSuccessMsg{
			operation: "打标签",
			image:     targetRef,
		}
	}
}

// HasError 返回是否有错误弹窗显示
func (v *ImageListView) HasError() bool {
	return v.errorDialog != nil && v.errorDialog.IsVisible()
}

// IsShowingJSONViewer 返回是否正在显示 JSON 查看器
func (v *ImageListView) IsShowingJSONViewer() bool {
	return v.jsonViewer != nil && v.jsonViewer.IsVisible()
}

// showExportDialog 显示导出对话框
func (v *ImageListView) showExportDialog() tea.Cmd {
	// 获取要导出的镜像列表
	var images []ExportImageInfo

	if len(v.selectedImages) > 0 {
		// 有多选，导出选中的镜像
		for _, img := range v.filteredImages {
			if v.selectedImages[img.ID] {
				images = append(images, ExportImageInfo{
					ID:         img.ID,
					Repository: img.Repository,
					Tag:        img.Tag,
				})
			}
		}
	} else {
		// 没有多选，导出当前选中的镜像
		img := v.GetSelectedImage()
		if img != nil {
			images = append(images, ExportImageInfo{
				ID:         img.ID,
				Repository: img.Repository,
				Tag:        img.Tag,
			})
		}
	}

	if len(images) == 0 {
		v.successMsg = "⚠️ 请先选择要导出的镜像"
		v.successMsgTime = time.Now()
		return v.clearSuccessMessageAfter(2 * time.Second)
	}

	// 显示导出输入视图
	v.exportInput.SetWidth(v.width)
	v.exportInput.Show(images)
	v.exportInput.SetCallbacks(
		func(dir string, mode ExportMode, compress bool) {
			// 确认导出 - 使用任务系统
			v.startExportTask(images, dir, mode, compress)
		},
		func() {
			// 取消
		},
	)

	return nil
}

// startExportTask 启动导出任务
func (v *ImageListView) startExportTask(images []ExportImageInfo, dir string, mode ExportMode, compress bool) {
	// 转换为 task 包的类型
	taskImages := make([]task.ExportImageInfo, len(images))
	for i, img := range images {
		taskImages[i] = task.ExportImageInfo{
			ID:         img.ID,
			Repository: img.Repository,
			Tag:        img.Tag,
		}
	}

	// 转换导出模式
	taskMode := task.ExportModeSingle
	if mode == ExportModeMultiple {
		taskMode = task.ExportModeMultiple
	}

	// 创建导出任务
	exportTask := task.NewExportTask(v.dockerClient, taskImages, dir, taskMode, compress)

	// 提交到任务管理器
	manager := task.GetManager()
	manager.Submit(exportTask)

	// 显示开始消息
	v.successMsg = fmt.Sprintf("📤 开始导出 %d 个镜像到 %s", len(images), dir)
	v.successMsgTime = time.Now()

	// 清除选择
	v.selectedImages = make(map[string]bool)
	v.updateTableData()
}

// GetSelectedCount 获取选中的镜像数量
func (v *ImageListView) GetSelectedCount() int {
	return len(v.selectedImages)
}

// IsShowingExportInput 返回是否正在显示导出输入视图
func (v *ImageListView) IsShowingExportInput() bool {
	return v.exportInput != nil && v.exportInput.IsVisible()
}

// overlayExportInput 将导出输入视图叠加到现有内容上（居中显示）
func (v *ImageListView) overlayExportInput(baseContent string) string {
	return OverlayCentered(baseContent, v.exportInput.View(), v.width, v.height)
}
