package ui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/ui/search"
)

// JSON 查看器样式
var (
	jsonViewerTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	jsonViewerKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	jsonViewerStringStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	jsonViewerNumberStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("213"))

	jsonViewerBoolStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("208"))

	jsonViewerNullStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	jsonViewerBracketStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	jsonViewerLineNumStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(5).
		Align(lipgloss.Right)

	jsonViewerHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	// 搜索相关样式
	jsonSearchPromptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	jsonSearchMatchStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("226")).
		Foreground(lipgloss.Color("0"))

	jsonSearchCurrentStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("208")).
		Foreground(lipgloss.Color("0")).
		Bold(true)

	jsonSearchInfoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	jsonSearchNoMatchStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))
)

// JSONViewer JSON 查看器组件
type JSONViewer struct {
	title   string   // 标题
	content string   // 原始 JSON 内容
	lines   []string // 按行分割的内容
	width   int
	height  int

	// 滚动状态
	scrollY    int // 垂直滚动偏移
	scrollX    int // 水平滚动偏移
	maxScrollY int // 最大垂直滚动
	maxScrollX int // 最大水平滚动

	// 可见性
	visible bool

	// 搜索相关
	searcher    *search.TextSearcher
	isSearching bool   // 是否处于搜索输入模式
	searchInput string // 搜索输入框内容

	// 回调
	onClose func()
}

// NewJSONViewer 创建 JSON 查看器
func NewJSONViewer() *JSONViewer {
	return &JSONViewer{
		visible:  false,
		searcher: search.NewTextSearcher(),
	}
}

// Show 显示 JSON 内容
func (v *JSONViewer) Show(title, content string) {
	v.title = title
	v.content = content
	v.lines = strings.Split(content, "\n")
	v.scrollY = 0
	v.scrollX = 0
	v.visible = true
	v.isSearching = false
	v.searchInput = ""
	v.searcher.Clear()
	v.updateMaxScroll()
}

// Hide 隐藏查看器
func (v *JSONViewer) Hide() {
	v.visible = false
	v.isSearching = false
	v.searcher.Clear()
}

// IsVisible 检查是否可见
func (v *JSONViewer) IsVisible() bool {
	return v.visible
}

// SetSize 设置尺寸
func (v *JSONViewer) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.updateMaxScroll()
}

// SetOnClose 设置关闭回调
func (v *JSONViewer) SetOnClose(fn func()) {
	v.onClose = fn
}

// updateMaxScroll 更新最大滚动值
func (v *JSONViewer) updateMaxScroll() {
	// 可见行数（减去标题和底部提示）
	visibleLines := v.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}

	v.maxScrollY = len(v.lines) - visibleLines
	if v.maxScrollY < 0 {
		v.maxScrollY = 0
	}

	// 计算最大行宽
	maxLineWidth := 0
	for _, line := range v.lines {
		if len(line) > maxLineWidth {
			maxLineWidth = len(line)
		}
	}

	v.maxScrollX = maxLineWidth - (v.width - 10)
	if v.maxScrollX < 0 {
		v.maxScrollX = 0
	}
}

// scrollToLine 滚动到指定行
func (v *JSONViewer) scrollToLine(lineIdx int) {
	visibleLines := v.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}

	// 将目标行滚动到视图中央
	targetScroll := lineIdx - visibleLines/2
	if targetScroll < 0 {
		targetScroll = 0
	}
	if targetScroll > v.maxScrollY {
		targetScroll = v.maxScrollY
	}
	v.scrollY = targetScroll
}

// Update 处理按键
func (v *JSONViewer) Update(msg tea.KeyMsg) bool {
	if !v.visible {
		return false
	}

	// 搜索输入模式
	if v.isSearching {
		return v.handleSearchInput(msg)
	}

	// 普通模式
	switch msg.String() {
	case "esc", "q", "i":
		// 如果有搜索结果，先清除搜索
		if v.searcher.HasMatches() {
			v.searcher.Clear()
			v.searchInput = ""
			return true
		}
		v.Hide()
		if v.onClose != nil {
			v.onClose()
		}
		return true
	case "/":
		// 进入搜索模式
		v.isSearching = true
		v.searchInput = ""
		return true
	case "n":
		// 下一个匹配
		if match := v.searcher.Next(); match != nil {
			v.scrollToLine(match.Line)
		}
		return true
	case "N":
		// 上一个匹配
		if match := v.searcher.Prev(); match != nil {
			v.scrollToLine(match.Line)
		}
		return true
	case "j", "down":
		if v.scrollY < v.maxScrollY {
			v.scrollY++
		}
		return true
	case "k", "up":
		if v.scrollY > 0 {
			v.scrollY--
		}
		return true
	case "h", "left":
		if v.scrollX > 0 {
			v.scrollX -= 4
			if v.scrollX < 0 {
				v.scrollX = 0
			}
		}
		return true
	case "l", "right":
		if v.scrollX < v.maxScrollX {
			v.scrollX += 4
		}
		return true
	case "g":
		v.scrollY = 0
		return true
	case "G":
		v.scrollY = v.maxScrollY
		return true
	case "ctrl+d", "pgdown":
		v.scrollY += 10
		if v.scrollY > v.maxScrollY {
			v.scrollY = v.maxScrollY
		}
		return true
	case "ctrl+u", "pgup":
		v.scrollY -= 10
		if v.scrollY < 0 {
			v.scrollY = 0
		}
		return true
	}

	return false
}

// handleSearchInput 处理搜索输入模式的按键
func (v *JSONViewer) handleSearchInput(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyEsc:
		// 取消搜索
		v.isSearching = false
		v.searchInput = ""
		return true
	case tea.KeyEnter:
		// 确认搜索
		v.isSearching = false
		if v.searchInput != "" {
			v.searcher.Search(v.lines, v.searchInput)
			// 跳转到第一个匹配
			if match := v.searcher.Current(); match != nil {
				v.scrollToLine(match.Line)
			}
		}
		return true
	case tea.KeyBackspace:
		if len(v.searchInput) > 0 {
			v.searchInput = v.searchInput[:len(v.searchInput)-1]
		}
		return true
	case tea.KeyRunes:
		v.searchInput += string(msg.Runes)
		return true
	case tea.KeySpace:
		v.searchInput += " "
		return true
	}
	return true
}

// View 渲染视图
func (v *JSONViewer) View() string {
	if !v.visible {
		return ""
	}

	var s strings.Builder

	// 标题
	s.WriteString("\n  " + jsonViewerTitleStyle.Render("📋 "+v.title) + "\n")
	s.WriteString("  " + strings.Repeat("─", v.width-6) + "\n")

	// 可见行数
	visibleLines := v.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}

	// 可见宽度
	visibleWidth := v.width - 10
	if visibleWidth < 20 {
		visibleWidth = 20
	}

	// 渲染可见行
	for i := 0; i < visibleLines && i+v.scrollY < len(v.lines); i++ {
		lineIdx := i + v.scrollY
		lineNum := lineIdx + 1
		line := v.lines[lineIdx]

		// 水平滚动
		displayLine := line
		if v.scrollX > 0 && len(line) > v.scrollX {
			displayLine = line[v.scrollX:]
		} else if v.scrollX > 0 {
			displayLine = ""
		}

		// 截断过长的行
		if len(displayLine) > visibleWidth {
			displayLine = displayLine[:visibleWidth-3] + "..."
		}

		// 语法高亮（如果没有搜索匹配）
		var coloredLine string
		if v.searcher.HasMatches() && v.searcher.IsLineMatched(lineIdx) {
			// 有搜索匹配，使用搜索高亮
			coloredLine = v.highlightSearchMatches(line, lineIdx, v.scrollX, visibleWidth)
		} else {
			coloredLine = v.colorize(displayLine)
		}

		// 行号 + 内容
		lineNumStr := jsonViewerLineNumStyle.Render(strconv.Itoa(lineNum))
		s.WriteString("  " + lineNumStr + " │ " + coloredLine + "\n")
	}

	// 底部分隔线
	s.WriteString("  " + strings.Repeat("─", v.width-6) + "\n")

	// 底部状态栏
	s.WriteString(v.renderStatusBar())

	return s.String()
}

// renderStatusBar 渲染底部状态栏
func (v *JSONViewer) renderStatusBar() string {
	var status string

	if v.isSearching {
		// 搜索输入模式
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		status = "  " + jsonSearchPromptStyle.Render("/") + v.searchInput + cursor +
			"  " + jsonSearchInfoStyle.Render("[Enter=确认 ESC=取消]") + "\n"
	} else if v.searcher.HasMatches() {
		// 显示搜索结果
		matchInfo := jsonSearchInfoStyle.Render(
			"[" + strconv.Itoa(v.searcher.CurrentIndex()) + "/" +
				strconv.Itoa(v.searcher.MatchCount()) + "]")
		status = "  " + jsonSearchPromptStyle.Render("/"+v.searcher.Query()) + " " + matchInfo +
			"  " + jsonViewerHintStyle.Render("n=下一个 N=上一个 ESC=清除") + "\n"
	} else if v.searchInput != "" && !v.searcher.HasMatches() {
		// 无匹配结果
		status = "  " + jsonSearchNoMatchStyle.Render("未找到: "+v.searchInput) +
			"  " + jsonViewerHintStyle.Render("ESC=清除") + "\n"
	} else {
		// 普通模式
		scrollInfo := ""
		if v.maxScrollY > 0 {
			percent := 0
			if v.maxScrollY > 0 {
				percent = v.scrollY * 100 / v.maxScrollY
			}
			scrollInfo = jsonViewerHintStyle.Render(strconv.Itoa(percent) + "%")
		}
		hints := jsonViewerHintStyle.Render("j/k=上下  g/G=首尾  /=搜索  n/N=跳转  ESC/q=关闭")
		status = "  " + hints + "  " + scrollInfo + "\n"
	}

	return status
}

// highlightSearchMatches 高亮显示搜索匹配
func (v *JSONViewer) highlightSearchMatches(line string, lineIdx int, scrollX int, visibleWidth int) string {
	matches := v.searcher.GetLineMatches(lineIdx)
	if len(matches) == 0 {
		// 无匹配，使用普通语法高亮
		displayLine := line
		if scrollX > 0 && len(line) > scrollX {
			displayLine = line[scrollX:]
		} else if scrollX > 0 {
			displayLine = ""
		}
		if len(displayLine) > visibleWidth {
			displayLine = displayLine[:visibleWidth-3] + "..."
		}
		return v.colorize(displayLine)
	}

	// 构建高亮后的行
	var result strings.Builder
	currentMatch := v.searcher.Current()
	pos := 0

	for _, m := range matches {
		// 调整位置以适应水平滚动
		matchStart := m.Column - scrollX
		matchEnd := matchStart + m.Length

		// 跳过不可见的匹配
		if matchEnd <= 0 || matchStart >= visibleWidth {
			continue
		}

		// 调整边界
		if matchStart < 0 {
			matchStart = 0
		}
		if matchEnd > visibleWidth {
			matchEnd = visibleWidth
		}

		// 添加匹配前的文本
		if matchStart > pos {
			beforeText := line[scrollX+pos : scrollX+matchStart]
			result.WriteString(v.colorize(beforeText))
		}

		// 添加高亮的匹配文本
		matchText := line[scrollX+matchStart : scrollX+matchEnd]
		if currentMatch != nil && currentMatch.Line == lineIdx && currentMatch.Column == m.Column {
			// 当前匹配用更醒目的颜色
			result.WriteString(jsonSearchCurrentStyle.Render(matchText))
		} else {
			result.WriteString(jsonSearchMatchStyle.Render(matchText))
		}

		pos = matchEnd
	}

	// 添加剩余文本
	if pos < visibleWidth && scrollX+pos < len(line) {
		endPos := visibleWidth
		if scrollX+endPos > len(line) {
			endPos = len(line) - scrollX
		}
		if pos < endPos {
			remainingText := line[scrollX+pos : scrollX+endPos]
			result.WriteString(v.colorize(remainingText))
		}
	}

	return result.String()
}

// colorize 对 JSON 行进行语法高亮
func (v *JSONViewer) colorize(line string) string {
	// 简单的语法高亮
	result := line

	// 处理键名（"key":）
	var colored strings.Builder
	i := 0

	for i < len(line) {
		ch := line[i]

		if ch == '"' && (i == 0 || line[i-1] != '\\') {
			// 找到字符串开始
			start := i
			i++
			for i < len(line) && !(line[i] == '"' && line[i-1] != '\\') {
				i++
			}
			if i < len(line) {
				i++ // 包含结束引号
			}

			str := line[start:i]

			// 判断是键名还是值
			if i < len(line) && line[i] == ':' {
				// 键名
				colored.WriteString(jsonViewerKeyStyle.Render(str))
			} else {
				// 字符串值
				colored.WriteString(jsonViewerStringStyle.Render(str))
			}
			continue
		}

		// 数字
		if (ch >= '0' && ch <= '9') || (ch == '-' && i+1 < len(line) && line[i+1] >= '0' && line[i+1] <= '9') {
			start := i
			for i < len(line) && ((line[i] >= '0' && line[i] <= '9') || line[i] == '.' || line[i] == '-' || line[i] == 'e' || line[i] == 'E' || line[i] == '+') {
				i++
			}
			colored.WriteString(jsonViewerNumberStyle.Render(line[start:i]))
			continue
		}

		// true/false
		if i+4 <= len(line) && line[i:i+4] == "true" {
			colored.WriteString(jsonViewerBoolStyle.Render("true"))
			i += 4
			continue
		}
		if i+5 <= len(line) && line[i:i+5] == "false" {
			colored.WriteString(jsonViewerBoolStyle.Render("false"))
			i += 5
			continue
		}

		// null
		if i+4 <= len(line) && line[i:i+4] == "null" {
			colored.WriteString(jsonViewerNullStyle.Render("null"))
			i += 4
			continue
		}

		// 括号
		if ch == '{' || ch == '}' || ch == '[' || ch == ']' {
			colored.WriteString(jsonViewerBracketStyle.Render(string(ch)))
			i++
			continue
		}

		// 其他字符
		colored.WriteByte(ch)
		i++
	}

	if colored.Len() > 0 {
		result = colored.String()
	}

	return result
}

// Overlay 将 JSON 查看器叠加到基础内容上
func (v *JSONViewer) Overlay(baseContent string) string {
	if !v.visible {
		return baseContent
	}

	// 直接返回 JSON 查看器的内容（全屏显示）
	return v.View()
}
