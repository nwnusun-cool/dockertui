package components

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
	title   string
	content string
	lines   []string
	width   int
	height  int

	scrollY    int
	scrollX    int
	maxScrollY int
	maxScrollX int

	visible bool

	searcher    *search.TextSearcher
	isSearching bool
	searchInput string

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

func (v *JSONViewer) updateMaxScroll() {
	visibleLines := v.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}

	v.maxScrollY = len(v.lines) - visibleLines
	if v.maxScrollY < 0 {
		v.maxScrollY = 0
	}

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

func (v *JSONViewer) scrollToLine(lineIdx int) {
	visibleLines := v.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}

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

	if v.isSearching {
		return v.handleSearchInput(msg)
	}

	switch msg.String() {
	case "esc", "q", "i":
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
		v.isSearching = true
		v.searchInput = ""
		return true
	case "n":
		if match := v.searcher.Next(); match != nil {
			v.scrollToLine(match.Line)
		}
		return true
	case "N":
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

func (v *JSONViewer) handleSearchInput(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyEsc:
		v.isSearching = false
		v.searchInput = ""
		return true
	case tea.KeyEnter:
		v.isSearching = false
		if v.searchInput != "" {
			v.searcher.Search(v.lines, v.searchInput)
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

	s.WriteString("\n  " + jsonViewerTitleStyle.Render("📋 "+v.title) + "\n")
	s.WriteString("  " + strings.Repeat("─", v.width-6) + "\n")

	visibleLines := v.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}

	visibleWidth := v.width - 10
	if visibleWidth < 20 {
		visibleWidth = 20
	}

	for i := 0; i < visibleLines && i+v.scrollY < len(v.lines); i++ {
		lineIdx := i + v.scrollY
		lineNum := lineIdx + 1
		line := v.lines[lineIdx]

		displayLine := line
		if v.scrollX > 0 && len(line) > v.scrollX {
			displayLine = line[v.scrollX:]
		} else if v.scrollX > 0 {
			displayLine = ""
		}

		if len(displayLine) > visibleWidth {
			displayLine = displayLine[:visibleWidth-3] + "..."
		}

		var coloredLine string
		if v.searcher.HasMatches() && v.searcher.IsLineMatched(lineIdx) {
			coloredLine = v.highlightSearchMatches(line, lineIdx, v.scrollX, visibleWidth)
		} else {
			coloredLine = v.colorize(displayLine)
		}

		lineNumStr := jsonViewerLineNumStyle.Render(strconv.Itoa(lineNum))
		s.WriteString("  " + lineNumStr + " │ " + coloredLine + "\n")
	}

	s.WriteString("  " + strings.Repeat("─", v.width-6) + "\n")
	s.WriteString(v.renderStatusBar())

	return s.String()
}

func (v *JSONViewer) renderStatusBar() string {
	var status string

	if v.isSearching {
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		status = "  " + jsonSearchPromptStyle.Render("/") + v.searchInput + cursor +
			"  " + jsonSearchInfoStyle.Render("[Enter=Confirm ESC=Cancel]") + "\n"
	} else if v.searcher.HasMatches() {
		matchInfo := jsonSearchInfoStyle.Render(
			"[" + strconv.Itoa(v.searcher.CurrentIndex()) + "/" +
				strconv.Itoa(v.searcher.MatchCount()) + "]")
		status = "  " + jsonSearchPromptStyle.Render("/"+v.searcher.Query()) + " " + matchInfo +
			"  " + jsonViewerHintStyle.Render("n=Next N=Previous ESC=Clear") + "\n"
	} else if v.searchInput != "" && !v.searcher.HasMatches() {
		status = "  " + jsonSearchNoMatchStyle.Render("Not found: "+v.searchInput) +
			"  " + jsonViewerHintStyle.Render("ESC=Clear") + "\n"
	} else {
		scrollInfo := ""
		if v.maxScrollY > 0 {
			percent := 0
			if v.maxScrollY > 0 {
				percent = v.scrollY * 100 / v.maxScrollY
			}
			scrollInfo = jsonViewerHintStyle.Render(strconv.Itoa(percent) + "%")
		}
		hints := jsonViewerHintStyle.Render("j/k=Up/Down  g/G=Top/Bottom  /=Search  n/N=Jump  ESC/q=Close")
		status = "  " + hints + "  " + scrollInfo + "\n"
	}

	return status
}

func (v *JSONViewer) highlightSearchMatches(line string, lineIdx int, scrollX int, visibleWidth int) string {
	matches := v.searcher.GetLineMatches(lineIdx)
	if len(matches) == 0 {
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

	var result strings.Builder
	currentMatch := v.searcher.Current()
	pos := 0

	for _, m := range matches {
		matchStart := m.Column - scrollX
		matchEnd := matchStart + m.Length

		if matchEnd <= 0 || matchStart >= visibleWidth {
			continue
		}

		if matchStart < 0 {
			matchStart = 0
		}
		if matchEnd > visibleWidth {
			matchEnd = visibleWidth
		}

		if matchStart > pos {
			beforeText := line[scrollX+pos : scrollX+matchStart]
			result.WriteString(v.colorize(beforeText))
		}

		matchText := line[scrollX+matchStart : scrollX+matchEnd]
		if currentMatch != nil && currentMatch.Line == lineIdx && currentMatch.Column == m.Column {
			result.WriteString(jsonSearchCurrentStyle.Render(matchText))
		} else {
			result.WriteString(jsonSearchMatchStyle.Render(matchText))
		}

		pos = matchEnd
	}

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

func (v *JSONViewer) colorize(line string) string {
	result := line
	var colored strings.Builder
	i := 0

	for i < len(line) {
		ch := line[i]

		if ch == '"' && (i == 0 || line[i-1] != '\\') {
			start := i
			i++
			for i < len(line) && !(line[i] == '"' && line[i-1] != '\\') {
				i++
			}
			if i < len(line) {
				i++
			}

			str := line[start:i]

			if i < len(line) && line[i] == ':' {
				colored.WriteString(jsonViewerKeyStyle.Render(str))
			} else {
				colored.WriteString(jsonViewerStringStyle.Render(str))
			}
			continue
		}

		if (ch >= '0' && ch <= '9') || (ch == '-' && i+1 < len(line) && line[i+1] >= '0' && line[i+1] <= '9') {
			start := i
			for i < len(line) && ((line[i] >= '0' && line[i] <= '9') || line[i] == '.' || line[i] == '-' || line[i] == 'e' || line[i] == 'E' || line[i] == '+') {
				i++
			}
			colored.WriteString(jsonViewerNumberStyle.Render(line[start:i]))
			continue
		}

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

		if i+4 <= len(line) && line[i:i+4] == "null" {
			colored.WriteString(jsonViewerNullStyle.Render("null"))
			i += 4
			continue
		}

		if ch == '{' || ch == '}' || ch == '[' || ch == ']' {
			colored.WriteString(jsonViewerBracketStyle.Render(string(ch)))
			i++
			continue
		}

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
	return v.View()
}
