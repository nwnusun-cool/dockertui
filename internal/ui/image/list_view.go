package image

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
	"docktui/internal/ui/components"
)

// ListView 镜像列表视图
type ListView struct {
	dockerClient docker.Client
	width, height int
	images, filteredImages []docker.Image
	tableModel table.Model
	scrollTable *components.ScrollableTable
	loading bool
	errorMsg, successMsg string
	successMsgTime time.Time
	searchQuery string
	isSearching bool
	filterType string // "all", "active", "dangling", "unused"
	sortBy string
	lastRefreshTime time.Time
	showConfirmDialog bool
	confirmAction string
	confirmImage *docker.Image
	confirmSelection int
	confirmPullRef string
	keys components.KeyMap
	pullInput *components.PullInputView
	taskBar *components.TaskBar
	tagInput *components.TagInputView
	errorDialog *components.ErrorDialog
	jsonViewer *components.JSONViewer
	selectedImages map[string]bool
	exportInput *components.ExportInputView
}

// NewListView 创建镜像列表视图
func NewListView(dockerClient docker.Client) *ListView {
	columns := []table.Column{
		{Title: "IMAGE ID", Width: 14},
		{Title: "REPOSITORY", Width: 30},
		{Title: "TAG", Width: 20},
		{Title: "SIZE", Width: 12},
		{Title: "CREATED", Width: 14},
	}
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t := table.New(table.WithColumns(columns), table.WithFocused(true), table.WithHeight(10))
	t.SetStyles(s)
	scrollColumns := []components.TableColumn{
		{Title: "SEL", Width: 3},
		{Title: "IMAGE ID", Width: 14},
		{Title: "REPOSITORY", Width: 35},
		{Title: "TAG", Width: 25},
		{Title: "SIZE", Width: 12},
		{Title: "CREATED", Width: 16},
	}
	return &ListView{
		dockerClient: dockerClient,
		tableModel: t,
		scrollTable: components.NewScrollableTable(scrollColumns),
		keys: components.DefaultKeyMap(),
		filterType: "all",
		sortBy: "created",
		pullInput: components.NewPullInputView(),
		taskBar: components.NewTaskBar(),
		tagInput: components.NewTagInputView(),
		errorDialog: components.NewErrorDialog(),
		jsonViewer: components.NewJSONViewer(),
		selectedImages: make(map[string]bool),
		exportInput: components.NewExportInputView(),
	}
}

// Init 初始化镜像列表视图
func (v *ListView) Init() tea.Cmd {
	v.loading = true
	return v.loadImages
}

// Update 处理消息并更新视图状态
func (v *ListView) Update(msg tea.Msg) (*ListView, tea.Cmd) {
	if v.exportInput != nil && v.exportInput.IsVisible() {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			wasVisible := v.exportInput.IsVisible()
			v.exportInput.Update(keyMsg)
			if wasVisible && !v.exportInput.IsVisible() && v.taskBar.HasActiveTasks() {
				return v, v.taskBar.ListenForEvents()
			}
			return v, nil
		}
	}
	if v.jsonViewer != nil && v.jsonViewer.IsVisible() {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if v.jsonViewer.Update(keyMsg) { return v, nil }
		}
	}
	switch msg := msg.(type) {
	case ImagesLoadedMsg:
		v.images = msg.Images
		v.loading = false
		v.errorMsg = ""
		v.lastRefreshTime = time.Now()
		v.applyFilters()
		v.updateColumnWidths()
		return v, nil
	case ImagesLoadErrorMsg:
		v.loading = false
		v.errorMsg = msg.Err.Error()
		return v, nil
	case ImageOperationSuccessMsg:
		v.successMsg = fmt.Sprintf("✅ %s succeeded: %s", msg.Operation, msg.Image)
		v.successMsgTime = time.Now()
		v.errorMsg = ""
		return v, tea.Batch(v.loadImages, v.clearSuccessMessageAfter(3*time.Second))
	case ImageOperationErrorMsg:
		if v.errorDialog != nil { v.errorDialog.ShowError(fmt.Sprintf("%s failed (%s): %v", msg.Operation, msg.Image, msg.Err)) }
		v.successMsg = ""
		return v, nil
	case ImageInUseErrorMsg:
		v.showForceRemoveConfirmDialog(msg.Image)
		return v, nil
	case ClearSuccessMessageMsg:
		if time.Since(v.successMsgTime) >= 3*time.Second { v.successMsg = "" }
		return v, nil
	case ImageInspectMsg:
		if v.jsonViewer != nil {
			v.jsonViewer.SetSize(v.width, v.height)
			v.jsonViewer.Show("Image Inspect: "+msg.ImageName, msg.JSONContent)
		}
		return v, nil
	case ImageInspectErrorMsg:
		if v.errorDialog != nil { v.errorDialog.ShowError(fmt.Sprintf("Failed to get image info: %v", msg.Err)) }
		return v, nil
	case ImageExportSuccessMsg:
		v.successMsg = fmt.Sprintf("✅ Exported %d images to %s", msg.Count, msg.Dir)
		v.successMsgTime = time.Now()
		v.selectedImages = make(map[string]bool)
		v.updateTableData()
		return v, v.clearSuccessMessageAfter(5*time.Second)
	case ImageExportErrorMsg:
		if v.errorDialog != nil { v.errorDialog.ShowError(fmt.Sprintf("Export image failed: %v", msg.Err)) }
		return v, nil
	case ImageExportProgressMsg:
		v.successMsg = fmt.Sprintf("⏳ Exporting [%d/%d]: %s", msg.Current, msg.Total, msg.Name)
		v.successMsgTime = time.Now()
		return v, nil
	case components.TaskEventMsg:
		return v.handleTaskEvent(msg)
	case TaskTickMsg:
		if v.taskBar.HasActiveTasks() { return v, v.scheduleTaskTick() }
		return v, nil
	case tea.KeyMsg:
		return v.handleKeyMsg(msg)
	}
	return v, nil
}

func (v *ListView) handleTaskEvent(msg components.TaskEventMsg) (*ListView, tea.Cmd) {
	event := msg.Event
	switch event.Type {
	case task.EventCompleted:
		v.successMsg = fmt.Sprintf("✅ %s", event.Message)
		v.successMsgTime = time.Now()
		return v, tea.Batch(v.loadImages, v.clearSuccessMessageAfter(3*time.Second), v.taskBar.ListenForEvents())
	case task.EventFailed:
		if v.errorDialog != nil { v.errorDialog.ShowError(fmt.Sprintf("%s: %v", event.TaskName, event.Error)) }
		return v, v.taskBar.ListenForEvents()
	case task.EventCancelled:
		v.successMsg = fmt.Sprintf("⏹️ %s cancelled", event.TaskName)
		v.successMsgTime = time.Now()
		return v, tea.Batch(v.clearSuccessMessageAfter(3*time.Second), v.taskBar.ListenForEvents())
	case task.EventProgress, task.EventStarted:
		return v, v.taskBar.ListenForEvents()
	default:
		return v, v.taskBar.ListenForEvents()
	}
}

func (v *ListView) handleKeyMsg(msg tea.KeyMsg) (*ListView, tea.Cmd) {
	if v.errorDialog != nil && v.errorDialog.IsVisible() {
		if v.errorDialog.Update(msg) { return v, nil }
	}
	if v.pullInput.IsVisible() {
		confirmed, handled, cmd := v.pullInput.Update(msg)
		if confirmed {
			imageRef := v.pullInput.Value()
			v.pullInput.Hide()
			v.startPullTaskSync(imageRef)
			return v, tea.Batch(v.taskBar.ListenForEvents(), v.scheduleTaskTick())
		}
		if handled { return v, cmd }
	}
	if v.tagInput.IsVisible() {
		confirmed, handled, cmd := v.tagInput.Update(msg)
		if confirmed {
			repo, tag := v.tagInput.GetValues()
			sourceImageID := v.tagInput.SourceImageID
			v.tagInput.Hide()
			return v, v.tagImage(sourceImageID, repo, tag)
		}
		if handled { return v, cmd }
	}
	if v.showConfirmDialog { return v.handleConfirmDialogKey(msg) }
	if v.isSearching { return v.handleSearchKey(msg) }
	return v.handleNormalKey(msg)
}

func (v *ListView) handleConfirmDialogKey(msg tea.KeyMsg) (*ListView, tea.Cmd) {
	switch msg.Type {
	case tea.KeyLeft, tea.KeyRight, tea.KeyTab:
		v.confirmSelection = 1 - v.confirmSelection
		return v, nil
	case tea.KeyEnter:
		return v.executeConfirmAction()
	case tea.KeyEsc:
		v.resetConfirmDialog()
		return v, nil
	case tea.KeyRunes:
		if msg.String() == "h" || msg.String() == "l" {
			v.confirmSelection = 1 - v.confirmSelection
			return v, nil
		}
	}
	if msg.String() == "enter" { return v.executeConfirmAction() }
	return v, nil
}

func (v *ListView) executeConfirmAction() (*ListView, tea.Cmd) {
	if v.confirmSelection == 1 {
		action, image, pullRef := v.confirmAction, v.confirmImage, v.confirmPullRef
		v.resetConfirmDialog()
		if action == "remove" && image != nil { return v, v.removeImage(image, false) }
		if action == "force_remove" && image != nil { return v, v.removeImage(image, true) }
		if action == "prune" { return v, v.pruneImages() }
		if action == "pull" && pullRef != "" {
			v.startPullTaskSync(pullRef)
			return v, v.taskBar.ListenForEvents()
		}
	} else { v.resetConfirmDialog() }
	return v, nil
}

func (v *ListView) resetConfirmDialog() {
	v.showConfirmDialog = false
	v.confirmAction = ""
	v.confirmImage = nil
	v.confirmPullRef = ""
	v.confirmSelection = 0
}

func (v *ListView) handleSearchKey(msg tea.KeyMsg) (*ListView, tea.Cmd) {
	switch msg.String() {
	case "enter": v.isSearching = false
	case "esc": v.isSearching = false; v.searchQuery = ""; v.applyFilters(); v.updateColumnWidths()
	case "backspace":
		if len(v.searchQuery) > 0 { v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]; v.applyFilters(); v.updateColumnWidths() }
	default:
		if len(msg.String()) == 1 { v.searchQuery += msg.String(); v.applyFilters(); v.updateColumnWidths() }
	}
	return v, nil
}

func (v *ListView) handleNormalKey(msg tea.KeyMsg) (*ListView, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if v.searchQuery != "" { v.searchQuery = ""; v.applyFilters(); v.updateColumnWidths(); return v, nil }
		if v.filterType != "all" { v.filterType = "all"; v.applyFilters(); v.updateColumnWidths(); return v, nil }
		return v, func() tea.Msg { return GoBackMsg{} }
	case "f":
		switch v.filterType {
		case "all": v.filterType = "active"
		case "active": v.filterType = "dangling"
		case "dangling": v.filterType = "unused"
		default: v.filterType = "all"
		}
		v.applyFilters(); v.updateColumnWidths()
	case "/": v.isSearching = true; v.searchQuery = ""
	case "r", "f5": v.loading = true; v.errorMsg = ""; return v, v.loadImages
	case "j", "down": if v.scrollTable != nil { v.scrollTable.MoveDown(1) }; v.tableModel.MoveDown(1)
	case "k", "up": if v.scrollTable != nil { v.scrollTable.MoveUp(1) }; v.tableModel.MoveUp(1)
	case "g": if v.scrollTable != nil { v.scrollTable.GotoTop() }; v.tableModel.GotoTop()
	case "G": if v.scrollTable != nil { v.scrollTable.GotoBottom() }; v.tableModel.GotoBottom()
	case "h", "left": if v.scrollTable != nil { v.scrollTable.ScrollLeft() }
	case "l", "right": if v.scrollTable != nil { v.scrollTable.ScrollRight() }
	case "enter":
		image := v.GetSelectedImage()
		if image == nil { return v, nil }
		return v, func() tea.Msg { return ViewImageDetailsMsg{Image: image} }
	case "d": return v, v.showRemoveConfirmDialog()
	case "p": return v, v.showPruneConfirmDialog()
	case "P": v.pullInput.SetWidth(v.width); v.pullInput.Show()
	case "T": v.taskBar.Toggle()
	case "t": return v, v.showTagInput()
	case "i": return v, v.inspectImage()
	case " ":
		image := v.GetSelectedImage()
		if image != nil {
			if v.selectedImages[image.ID] { delete(v.selectedImages, image.ID) } else { v.selectedImages[image.ID] = true }
			v.updateTableData()
		}
	case "a":
		allSelected := true
		for _, img := range v.filteredImages { if !v.selectedImages[img.ID] { allSelected = false; break } }
		if allSelected && len(v.filteredImages) > 0 { v.selectedImages = make(map[string]bool) } else { for _, img := range v.filteredImages { v.selectedImages[img.ID] = true } }
		v.updateTableData()
	case "E": return v, v.showExportDialog()
	case "x":
		if v.taskBar.HasActiveTasks() {
			v.taskBar.CancelFirstTask()
			v.successMsg = "⏹️ Cancelling task..."
			v.successMsgTime = time.Now()
			return v, v.clearSuccessMessageAfter(2*time.Second)
		}
	}
	return v, nil
}

// View 渲染镜像列表视图
func (v *ListView) View() string {
	if v.jsonViewer != nil && v.jsonViewer.IsVisible() { return v.jsonViewer.View() }
	var s string
	s += v.renderStatusBar()
	if v.successMsg != "" {
		msgStyle := SuccessMsgStyle
		if strings.HasPrefix(v.successMsg, "⚠️") { msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true) }
		s += "\n  " + msgStyle.Render(v.successMsg) + "\n"
	}
	s += v.renderStatsBar()
	if v.loading {
		loadingContent := lipgloss.JoinVertical(lipgloss.Center, "", StatusBarKeyStyle.Render("⏳ Loading image list..."), "", SearchHintStyle.Render("Please wait, fetching data from Docker"), "")
		s += "\n  " + StateBoxStyle.Render(loadingContent) + "\n"
		return s
	}
	if v.errorMsg != "" {
		errorContent := lipgloss.JoinVertical(lipgloss.Left, "", ErrorMsgStyle.Render("❌ Load failed: "+v.errorMsg), "", StatusBarLabelStyle.Render("💡 Possible reasons:"), SearchHintStyle.Render("   • Docker daemon not running"), SearchHintStyle.Render("   • Network connection issue"), SearchHintStyle.Render("   • Insufficient permissions"), "", StatusBarKeyStyle.Render("Press r to reload")+" "+SearchHintStyle.Render("or")+" "+StatusBarKeyStyle.Render("Press Esc to go back"), "")
		s += "\n  " + StateBoxStyle.Render(errorContent) + "\n"
		return s
	}
	if len(v.images) == 0 {
		emptyContent := lipgloss.JoinVertical(lipgloss.Left, "", SearchHintStyle.Render("📦 No images"), "", StatusBarLabelStyle.Render("💡 Quick start:"), "", StatusBarKeyStyle.Render("1.")+" "+SearchHintStyle.Render("Pull an image:"), SearchHintStyle.Render("   docker pull nginx"), "", StatusBarKeyStyle.Render("2.")+" "+SearchHintStyle.Render("Refresh image list:"), SearchHintStyle.Render("   Press r to refresh"), "")
		s += "\n  " + StateBoxStyle.Render(emptyContent) + "\n"
		return s
	}
	if v.scrollTable != nil { s += v.scrollTable.View() + "\n" } else { s += "  " + v.tableModel.View() + "\n" }
	if v.isSearching {
		searchLine := "\n  " + strings.Repeat("─", 67) + "\n"
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		s += searchLine + "  " + SearchPromptStyle.Render("Search:") + " " + v.searchQuery + cursor + "    " + SearchHintStyle.Render("[Enter=Confirm | ESC=Cancel]") + "\n"
	}
	if !v.isSearching && v.filterType != "all" {
		filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
		s += "  " + filterStyle.Render("[Filter: "+v.filterType+"]") + "  " + SearchHintStyle.Render("Press ESC to clear filter, press f to switch") + "\n"
	}
	if v.taskBar.HasActiveTasks() { v.taskBar.SetWidth(v.width); s += v.taskBar.View() }
	if v.pullInput.IsVisible() { s = v.overlayPullInput(s) }
	if v.tagInput.IsVisible() { s = v.overlayTagInput(s) }
	if v.showConfirmDialog { s = v.overlayDialog(s) }
	if v.errorDialog != nil && v.errorDialog.IsVisible() { s = v.errorDialog.Overlay(s) }
	if v.exportInput != nil && v.exportInput.IsVisible() { s = v.overlayExportInput(s) }
	return s
}

// SetSize 设置视图尺寸
func (v *ListView) SetSize(width, height int) {
	v.width = width; v.height = height
	tableHeight := height - 15; if tableHeight < 5 { tableHeight = 5 }
	v.tableModel.SetHeight(tableHeight)
	if v.scrollTable != nil { v.scrollTable.SetSize(width-4, tableHeight) }
	v.pullInput.SetWidth(width)
	v.taskBar.SetWidth(width)
	if v.errorDialog != nil { v.errorDialog.SetWidth(width) }
}

func (v *ListView) renderStatusBar() string {
	width := v.width; if width < 80 { width = 80 }
	availableWidth := width - 4; if availableWidth < 60 { availableWidth = 60 }
	labelColWidth := 20
	shortcutsWidth := availableWidth - labelColWidth
	itemsPerRow := 4; if shortcutsWidth < 60 { itemsPerRow = 3 }
	itemWidth := shortcutsWidth / itemsPerRow; if itemWidth < 12 { itemWidth = 12 }
	labelStyle := lipgloss.NewStyle().Width(labelColWidth).Foreground(lipgloss.Color("220")).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	descStyle := lipgloss.NewStyle()
	itemStyle := lipgloss.NewStyle().Width(itemWidth)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	makeItem := func(key, desc string) string { return itemStyle.Render(keyStyle.Render(key) + descStyle.Render(" "+desc)) }
	var lines []string
	lines = append(lines, "  "+labelStyle.Render("🖼️ Images")+makeItem("<f>", "Filter")+makeItem("</>", "Search")+makeItem("<r>", "Refresh"))
	lines = append(lines, "  "+labelStyle.Render("Ops:")+makeItem("<d>", "Delete")+makeItem("<p>", "Prune")+makeItem("<P>", "Pull"))
	lines = append(lines, "  "+labelStyle.Render("Advanced:")+makeItem("<t>", "Tag")+makeItem("<Space>", "Select")+makeItem("<a>", "All")+makeItem("<E>", "Export"))
	refreshInfo := "-"; if !v.lastRefreshTime.IsZero() { refreshInfo = FormatDuration(time.Since(v.lastRefreshTime)) + " ago" }
	row4Info := hintStyle.Render(refreshInfo) + "    " + hintStyle.Render("j/k=Up/Down  Enter=Details  Esc=Back  q=Quit")
	if len(v.selectedImages) > 0 { row4Info += "    " + selectedStyle.Render(fmt.Sprintf("[Selected: %d]", len(v.selectedImages))) }
	lines = append(lines, "  "+labelStyle.Render("Last Refresh:")+row4Info)
	return "\n" + strings.Join(lines, "\n") + "\n"
}

func (v *ListView) renderStatsBar() string {
	totalCount := len(v.images)
	showingCount := len(v.filteredImages)
	activeCount, danglingCount, unusedCount := 0, 0, 0
	for _, img := range v.images {
		if img.InUse { activeCount++ }
		if img.Dangling { danglingCount++ }
		if !img.InUse && !img.Dangling { unusedCount++ }
	}
	totalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	activeStyleColor := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	danglingStyleColor := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	unusedStyleColor := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statsContent := totalStyle.Render(fmt.Sprintf("📦 Total: %d", totalCount)) + separatorStyle.Render("  │  ") + activeStyleColor.Render(fmt.Sprintf("🟢 Active: %d", activeCount)) + separatorStyle.Render("  │  ") + danglingStyleColor.Render(fmt.Sprintf("🟡 Dangling: %d", danglingCount)) + separatorStyle.Render("  │  ") + unusedStyleColor.Render(fmt.Sprintf("🔴 Unused: %d", unusedCount))
	if showingCount != totalCount || (!v.isSearching && v.searchQuery != "") {
		filterParts := []string{}
		if showingCount != totalCount { filterParts = append(filterParts, fmt.Sprintf("Showing: %d", showingCount)) }
		if !v.isSearching && v.searchQuery != "" { filterParts = append(filterParts, fmt.Sprintf("Search: \"%s\"", v.searchQuery)) }
		statsContent += SearchHintStyle.Render("  [" + strings.Join(filterParts, " | ") + "]")
	}
	lineWidth := v.width - 6; if lineWidth < 60 { lineWidth = 60 }
	line := lineStyle.Render(strings.Repeat("─", lineWidth))
	statsLine := lipgloss.NewStyle().Width(lineWidth).Align(lipgloss.Center).Render(statsContent)
	return "\n  " + line + "\n  " + statsLine + "\n  " + line + "\n"
}

func (v *ListView) loadImages() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	images, err := v.dockerClient.ListImages(ctx, true)
	if err != nil { return ImagesLoadErrorMsg{Err: err} }
	return ImagesLoadedMsg{Images: images}
}

func (v *ListView) applyFilters() {
	v.filteredImages = make([]docker.Image, 0)
	for _, img := range v.images {
		if v.searchQuery != "" {
			query := strings.ToLower(v.searchQuery)
			if !strings.Contains(strings.ToLower(img.Repository), query) && !strings.Contains(strings.ToLower(img.Tag), query) && !strings.Contains(strings.ToLower(img.ID), query) { continue }
		}
		switch v.filterType {
		case "active": if !img.InUse { continue }
		case "dangling": if !img.Dangling { continue }
		case "unused": if img.InUse || img.Dangling { continue }
		}
		v.filteredImages = append(v.filteredImages, img)
	}
}

func (v *ListView) updateTableData() {
	if v.scrollTable == nil || len(v.filteredImages) == 0 { return }
	rows := make([]components.TableRow, len(v.filteredImages))
	danglingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	unusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	for i, img := range v.filteredImages {
		created := FormatCreatedTime(img.Created)
		size := FormatSize(img.Size)
		selMark := " "; if v.selectedImages[img.ID] { selMark = selectedStyle.Render("✓") }
		var rowStyle lipgloss.Style; var needsStyle bool
		if img.Dangling { rowStyle = danglingStyle; needsStyle = true } else if !img.InUse { rowStyle = unusedStyle; needsStyle = true }
		if needsStyle {
			rows[i] = components.TableRow{selMark, rowStyle.Render(img.ShortID), rowStyle.Render(img.Repository), rowStyle.Render(img.Tag), rowStyle.Render(size), rowStyle.Render(created)}
		} else {
			rows[i] = components.TableRow{selMark, img.ShortID, img.Repository, img.Tag, size, created}
		}
	}
	v.scrollTable.SetRows(rows)
}

func (v *ListView) updateColumnWidths() {
	maxID, maxRepository, maxTag, maxSize, maxCreated := 12, 10, 3, 4, 7
	for _, img := range v.filteredImages {
		if len(img.Repository) > maxRepository { maxRepository = len(img.Repository) }
		if len(img.Tag) > maxTag { maxTag = len(img.Tag) }
		sizeStr := FormatSize(img.Size); if len(sizeStr) > maxSize { maxSize = len(sizeStr) }
		created := FormatCreatedTime(img.Created); if len(created) > maxCreated { maxCreated = len(created) }
	}
	if v.scrollTable != nil {
		v.scrollTable.SetColumns([]components.TableColumn{
			{Title: "SEL", Width: 3},
			{Title: "IMAGE ID", Width: maxID + 2},
			{Title: "REPOSITORY", Width: maxRepository + 2},
			{Title: "TAG", Width: maxTag + 2},
			{Title: "SIZE", Width: maxSize + 2},
			{Title: "CREATED", Width: maxCreated + 2},
		})
	}
	v.updateTableData()
	if len(v.filteredImages) > 0 {
		rows := v.imagesToRows(v.filteredImages)
		v.tableModel.SetRows(rows)
	} else { v.tableModel.SetRows([]table.Row{}) }
}

func (v *ListView) imagesToRows(images []docker.Image) []table.Row {
	rows := make([]table.Row, len(images))
	danglingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	unusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	for i, img := range images {
		created := FormatCreatedTime(img.Created)
		size := FormatSize(img.Size)
		var rowStyle lipgloss.Style; var needsStyle bool
		if img.Dangling { rowStyle = danglingStyle; needsStyle = true } else if !img.InUse { rowStyle = unusedStyle; needsStyle = true }
		if needsStyle {
			rows[i] = table.Row{rowStyle.Render(img.ShortID), rowStyle.Render(img.Repository), rowStyle.Render(img.Tag), rowStyle.Render(size), rowStyle.Render(created)}
		} else {
			rows[i] = table.Row{img.ShortID, img.Repository, img.Tag, size, created}
		}
	}
	return rows
}

// GetSelectedImage 获取当前选中的镜像
func (v *ListView) GetSelectedImage() *docker.Image {
	if len(v.filteredImages) == 0 { return nil }
	var selectedIndex int
	if v.scrollTable != nil { selectedIndex = v.scrollTable.Cursor() } else { selectedIndex = v.tableModel.Cursor() }
	if selectedIndex < 0 || selectedIndex >= len(v.filteredImages) { return nil }
	return &v.filteredImages[selectedIndex]
}

func (v *ListView) inspectImage() tea.Cmd {
	image := v.GetSelectedImage()
	if image == nil { return nil }
	imageID, imageName := image.ID, image.Repository+":"+image.Tag
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		jsonContent, err := v.dockerClient.InspectImageRaw(ctx, imageID)
		if err != nil { return ImageInspectErrorMsg{Err: err} }
		return ImageInspectMsg{ImageName: imageName, JSONContent: jsonContent}
	}
}

func (v *ListView) overlayPullInput(baseContent string) string {
	return components.OverlayCentered(baseContent, v.pullInput.View(), v.width, v.height)
}

func (v *ListView) overlayTagInput(baseContent string) string {
	return components.OverlayCentered(baseContent, v.tagInput.View(), v.width, v.height)
}

func (v *ListView) overlayDialog(baseContent string) string {
	return components.OverlayCentered(baseContent, v.renderConfirmDialogContent(), v.width, v.height)
}

func (v *ListView) overlayExportInput(baseContent string) string {
	return components.OverlayCentered(baseContent, v.exportInput.View(), v.width, v.height)
}

func (v *ListView) renderConfirmDialogContent() string {
	dialogStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(1, 2).Width(56)
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	cancelBtnStyle := lipgloss.NewStyle().Padding(0, 2)
	okBtnStyle := lipgloss.NewStyle().Padding(0, 2)
	if v.confirmSelection == 0 { cancelBtnStyle = cancelBtnStyle.Reverse(true).Bold(true); okBtnStyle = okBtnStyle.Foreground(lipgloss.Color("245")) } else { cancelBtnStyle = cancelBtnStyle.Foreground(lipgloss.Color("245")); okBtnStyle = okBtnStyle.Reverse(true).Bold(true) }
	var title, warning string
	if v.confirmAction == "remove" && v.confirmImage != nil {
		imageName := v.confirmImage.Repository + ":" + v.confirmImage.Tag; if len(imageName) > 35 { imageName = imageName[:32] + "..." }
		title = titleStyle.Render("⚠️  Delete Image: " + imageName)
		warning = warningStyle.Render("This action cannot be undone!")
	} else if v.confirmAction == "force_remove" && v.confirmImage != nil {
		imageName := v.confirmImage.Repository + ":" + v.confirmImage.Tag; if len(imageName) > 35 { imageName = imageName[:32] + "..." }
		title = titleStyle.Render("⚠️  Force Delete Image: " + imageName)
		warning = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render("⚠️  This image is being used by containers!\n") + warningStyle.Render("Force deletion may cause related containers to malfunction.\nAre you sure you want to continue?")
	} else if v.confirmAction == "prune" {
		title = titleStyle.Render("⚠️  Prune Dangling Images")
		warning = warningStyle.Render("Will delete all untagged dangling images to free disk space")
	} else if v.confirmAction == "pull" && v.confirmPullRef != "" {
		imageName := v.confirmPullRef; if len(imageName) > 35 { imageName = imageName[:32] + "..." }
		title = titleStyle.Render("📥  Pull Image: " + imageName)
		warning = warningStyle.Render("Confirm to pull this image?")
	}
	buttons := lipgloss.NewStyle().Width(52).Align(lipgloss.Center).Render(cancelBtnStyle.Render("< Cancel >") + "    " + okBtnStyle.Render("< OK >"))
	content := title + "\n\n" + warning + "\n\n" + buttons
	dialog := dialogStyle.Render(content)
	if v.width > 60 {
		leftPadding := (v.width - 60) / 2
		lines := strings.Split(dialog, "\n")
		var result strings.Builder
		for i, line := range lines { result.WriteString(strings.Repeat(" ", leftPadding)); result.WriteString(line); if i < len(lines)-1 { result.WriteString("\n") } }
		return result.String()
	}
	return dialog
}

func (v *ListView) showRemoveConfirmDialog() tea.Cmd {
	image := v.GetSelectedImage()
	if image == nil { return func() tea.Msg { return ImageOperationErrorMsg{Operation: "Delete image", Image: "", Err: fmt.Errorf("please select an image first")} } }
	v.showConfirmDialog = true; v.confirmAction = "remove"; v.confirmImage = image; v.confirmSelection = 0
	return nil
}

func (v *ListView) showForceRemoveConfirmDialog(image *docker.Image) {
	v.showConfirmDialog = true; v.confirmAction = "force_remove"; v.confirmImage = image; v.confirmSelection = 0
}

func (v *ListView) showPruneConfirmDialog() tea.Cmd {
	v.showConfirmDialog = true; v.confirmAction = "prune"; v.confirmImage = nil; v.confirmSelection = 0
	return nil
}

func (v *ListView) removeImage(image *docker.Image, force bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := v.dockerClient.RemoveImage(ctx, image.ID, force, false)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "image is being used") || strings.Contains(errStr, "image has dependent child images") || strings.Contains(errStr, "conflict") {
				return ImageInUseErrorMsg{Image: image, Err: err}
			}
			return ImageOperationErrorMsg{Operation: "Delete image", Image: image.Repository + ":" + image.Tag, Err: err}
		}
		return ImageOperationSuccessMsg{Operation: "Delete", Image: image.Repository + ":" + image.Tag}
	}
}

func (v *ListView) pruneImages() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		count, spaceReclaimed, err := v.dockerClient.PruneImages(ctx)
		if err != nil { return ImageOperationErrorMsg{Operation: "Prune dangling images", Image: "", Err: err} }
		return ImageOperationSuccessMsg{Operation: "Prune dangling images", Image: fmt.Sprintf("Deleted %d images, freed %s space", count, FormatSize(spaceReclaimed))}
	}
}

func (v *ListView) clearSuccessMessageAfter(duration time.Duration) tea.Cmd {
	return func() tea.Msg { time.Sleep(duration); return ClearSuccessMessageMsg{} }
}

func (v *ListView) scheduleTaskTick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg { return TaskTickMsg{} })
}

func (v *ListView) startPullTaskSync(imageRef string) {
	pullTask := task.NewPullTask(v.dockerClient, imageRef)
	manager := task.GetManager()
	manager.Submit(pullTask)
	v.successMsg = fmt.Sprintf("📥 Start pulling: %s", imageRef)
	v.successMsgTime = time.Now()
}

func (v *ListView) showTagInput() tea.Cmd {
	image := v.GetSelectedImage()
	if image == nil { return func() tea.Msg { return ImageOperationErrorMsg{Operation: "Tag", Image: "", Err: fmt.Errorf("please select an image first")} } }
	v.tagInput.SetWidth(v.width)
	v.tagInput.Show(image.Repository+":"+image.Tag, image.ID, image.Repository, image.Tag)
	return nil
}

func (v *ListView) tagImage(sourceImageID, repository, tag string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		targetRef := repository + ":" + tag
		err := v.dockerClient.TagImage(ctx, sourceImageID, repository, tag)
		if err != nil { return ImageOperationErrorMsg{Operation: "Tag", Image: targetRef, Err: err} }
		return ImageOperationSuccessMsg{Operation: "Tag", Image: targetRef}
	}
}

func (v *ListView) showExportDialog() tea.Cmd {
	var images []components.ExportImageInfo
	if len(v.selectedImages) > 0 {
		for _, img := range v.filteredImages {
			if v.selectedImages[img.ID] { images = append(images, components.ExportImageInfo{ID: img.ID, Repository: img.Repository, Tag: img.Tag}) }
		}
	} else {
		img := v.GetSelectedImage()
		if img != nil { images = append(images, components.ExportImageInfo{ID: img.ID, Repository: img.Repository, Tag: img.Tag}) }
	}
	if len(images) == 0 {
		v.successMsg = "⚠️ Please select images to export first"
		v.successMsgTime = time.Now()
		return v.clearSuccessMessageAfter(2*time.Second)
	}
	v.exportInput.SetWidth(v.width)
	v.exportInput.Show(images)
	v.exportInput.SetCallbacks(
		func(dir string, mode components.ExportMode, compress bool) { v.startExportTask(images, dir, mode, compress) },
		func() {},
	)
	return nil
}

func (v *ListView) startExportTask(images []components.ExportImageInfo, dir string, mode components.ExportMode, compress bool) {
	taskImages := make([]task.ExportImageInfo, len(images))
	for i, img := range images { taskImages[i] = task.ExportImageInfo{ID: img.ID, Repository: img.Repository, Tag: img.Tag} }
	taskMode := task.ExportModeSingle; if mode == components.ExportModeMultiple { taskMode = task.ExportModeMultiple }
	exportTask := task.NewExportTask(v.dockerClient, taskImages, dir, taskMode, compress)
	manager := task.GetManager()
	manager.Submit(exportTask)
	v.successMsg = fmt.Sprintf("📤 Start exporting %d images to %s", len(images), dir)
	v.successMsgTime = time.Now()
	v.selectedImages = make(map[string]bool)
	v.updateTableData()
}

// HasError 返回是否有错误弹窗显示
func (v *ListView) HasError() bool { return v.errorDialog != nil && v.errorDialog.IsVisible() }

// IsShowingJSONViewer 返回是否正在显示 JSON 查看器
func (v *ListView) IsShowingJSONViewer() bool { return v.jsonViewer != nil && v.jsonViewer.IsVisible() }

// GetSelectedCount 获取选中的镜像数量
func (v *ListView) GetSelectedCount() int { return len(v.selectedImages) }

// IsShowingExportInput 返回是否正在显示导出输入视图
func (v *ListView) IsShowingExportInput() bool { return v.exportInput != nil && v.exportInput.IsVisible() }

// ShowConfirmDialog 返回是否显示确认对话框
func (v *ListView) ShowConfirmDialog() bool { return v.showConfirmDialog }

// IsShowingPullInput 返回是否正在显示拉取输入框
func (v *ListView) IsShowingPullInput() bool { return v.pullInput != nil && v.pullInput.IsVisible() }

// IsShowingTagInput 返回是否正在显示打标签输入框
func (v *ListView) IsShowingTagInput() bool { return v.tagInput != nil && v.tagInput.IsVisible() }

// IsPullInputVisible 返回拉取输入框是否可见
func (v *ListView) IsPullInputVisible() bool {
	return v.pullInput != nil && v.pullInput.IsVisible()
}

// IsTagInputVisible 返回打标签输入框是否可见
func (v *ListView) IsTagInputVisible() bool {
	return v.tagInput != nil && v.tagInput.IsVisible()
}
