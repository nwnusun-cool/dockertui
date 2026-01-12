package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// 主题颜色定义
var (
	ThemeTextMuted   = lipgloss.Color("245")
	ThemeBorderColor = lipgloss.Color("240")
	ThemeHighlight   = lipgloss.Color("81")
	ThemeSuccess     = lipgloss.Color("82")
	ThemeWarning     = lipgloss.Color("220")
	ThemeError       = lipgloss.Color("196")
	ThemeTitleColor  = lipgloss.Color("220")
	ThemeKeyColor    = lipgloss.Color("81")
)

// ScrollableTable 可水平滚动的表格组件
type ScrollableTable struct {
	columns          []TableColumn
	rows             []TableRow
	cursor           int
	horizontalOffset int
	scrollStep       int
	width            int
	height           int
	focused          bool
	styles           TableStyles
}

// TableColumn 表格列定义
type TableColumn struct {
	Title string
	Width int
}

// TableRow 表格行数据
type TableRow []string

// TableStyles 表格样式
type TableStyles struct {
	Header          lipgloss.Style
	Cell            lipgloss.Style
	Selected        lipgloss.Style
	Border          lipgloss.Style
	ScrollIndicator lipgloss.Style
}

// DefaultTableStyles 默认表格样式
func DefaultTableStyles() TableStyles {
	return TableStyles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(ThemeTitleColor),
		Cell:     lipgloss.NewStyle(),
		Selected: lipgloss.NewStyle().Reverse(true).Bold(true),
		Border: lipgloss.NewStyle().
			Foreground(ThemeBorderColor),
		ScrollIndicator: lipgloss.NewStyle().
			Foreground(ThemeHighlight).
			Bold(true),
	}
}

// NewScrollableTable 创建可滚动表格
func NewScrollableTable(columns []TableColumn) *ScrollableTable {
	return &ScrollableTable{
		columns:          columns,
		rows:             []TableRow{},
		cursor:           0,
		horizontalOffset: 0,
		scrollStep:       20,
		width:            80,
		height:           10,
		focused:          true,
		styles:           DefaultTableStyles(),
	}
}

// SetColumns 设置列定义
func (t *ScrollableTable) SetColumns(columns []TableColumn) {
	t.columns = columns
}

// SetRows 设置行数据
func (t *ScrollableTable) SetRows(rows []TableRow) {
	t.rows = rows
	if t.cursor >= len(rows) {
		t.cursor = len(rows) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

// SetSize 设置可见区域大小
func (t *ScrollableTable) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// SetHeight 设置高度
func (t *ScrollableTable) SetHeight(height int) {
	t.height = height
}

// SetFocused 设置聚焦状态
func (t *ScrollableTable) SetFocused(focused bool) {
	t.focused = focused
}

// SetStyles 设置样式
func (t *ScrollableTable) SetStyles(styles TableStyles) {
	t.styles = styles
}

// Cursor 获取当前选中行索引
func (t *ScrollableTable) Cursor() int {
	return t.cursor
}

// MoveUp 向上移动
func (t *ScrollableTable) MoveUp(n int) {
	t.cursor -= n
	if t.cursor < 0 {
		t.cursor = 0
	}
}

// MoveDown 向下移动
func (t *ScrollableTable) MoveDown(n int) {
	t.cursor += n
	if t.cursor >= len(t.rows) {
		t.cursor = len(t.rows) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

// GotoTop 跳转到顶部
func (t *ScrollableTable) GotoTop() {
	t.cursor = 0
}

// GotoBottom 跳转到底部
func (t *ScrollableTable) GotoBottom() {
	if len(t.rows) > 0 {
		t.cursor = len(t.rows) - 1
	}
}

// ScrollLeft 向左滚动
func (t *ScrollableTable) ScrollLeft() {
	t.horizontalOffset -= t.scrollStep
	if t.horizontalOffset < 0 {
		t.horizontalOffset = 0
	}
}

// ScrollRight 向右滚动
func (t *ScrollableTable) ScrollRight() {
	maxOffset := t.getTotalWidth() - t.width + 10
	if maxOffset < 0 {
		maxOffset = 0
	}
	t.horizontalOffset += t.scrollStep
	if t.horizontalOffset > maxOffset {
		t.horizontalOffset = maxOffset
	}
}

// HorizontalOffset 获取水平偏移量
func (t *ScrollableTable) HorizontalOffset() int {
	return t.horizontalOffset
}

func (t *ScrollableTable) getTotalWidth() int {
	total := 0
	for _, col := range t.columns {
		total += col.Width + 2
	}
	return total
}

// View 渲染表格
func (t *ScrollableTable) View() string {
	if len(t.columns) == 0 {
		return ""
	}

	var b strings.Builder

	visibleWidth := t.width - 4
	if visibleWidth < 40 {
		visibleWidth = 40
	}

	totalWidth := t.getTotalWidth()
	canScrollLeft := t.horizontalOffset > 0
	canScrollRight := t.horizontalOffset < totalWidth-visibleWidth

	if canScrollLeft || canScrollRight {
		scrollInfo := t.renderScrollIndicator(canScrollLeft, canScrollRight)
		b.WriteString(scrollInfo)
		b.WriteString("\n")
	}

	headerLine := t.renderHeader()
	b.WriteString(t.applyHorizontalScroll(headerLine, visibleWidth))
	b.WriteString("\n")

	separatorLine := t.renderSeparator()
	b.WriteString(t.applyHorizontalScroll(separatorLine, visibleWidth))
	b.WriteString("\n")

	startRow := 0
	endRow := len(t.rows)
	visibleRows := t.height - 3
	if visibleRows < 1 {
		visibleRows = 1
	}

	if t.cursor < startRow {
		startRow = t.cursor
	}
	if t.cursor >= startRow+visibleRows {
		startRow = t.cursor - visibleRows + 1
	}
	endRow = startRow + visibleRows
	if endRow > len(t.rows) {
		endRow = len(t.rows)
	}

	for i := startRow; i < endRow; i++ {
		rowLine := t.renderRow(i)
		b.WriteString(t.applyHorizontalScroll(rowLine, visibleWidth))
		if i < endRow-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (t *ScrollableTable) renderScrollIndicator(canLeft, canRight bool) string {
	leftArrow := "  "
	rightArrow := "  "

	if canLeft {
		leftArrow = t.styles.ScrollIndicator.Render("◀ ")
	}
	if canRight {
		rightArrow = t.styles.ScrollIndicator.Render(" ▶")
	}

	totalWidth := t.getTotalWidth()
	visibleWidth := t.width - 4
	if visibleWidth < 40 {
		visibleWidth = 40
	}

	scrollPercent := 0
	if totalWidth > visibleWidth {
		scrollPercent = t.horizontalOffset * 100 / (totalWidth - visibleWidth)
		if scrollPercent > 100 {
			scrollPercent = 100
		}
	}

	hint := t.styles.ScrollIndicator.Render("← → 水平滚动")
	position := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
		strings.Repeat(" ", 10) + "Scroll: " + string(rune('0'+scrollPercent/10)) + string(rune('0'+scrollPercent%10)) + "%",
	)

	return "  " + leftArrow + hint + position + rightArrow
}

func (t *ScrollableTable) renderHeader() string {
	var headerContent strings.Builder
	for _, col := range t.columns {
		cell := t.padOrTruncate(col.Title, col.Width)
		headerContent.WriteString(" ")
		headerContent.WriteString(cell)
		headerContent.WriteString(" ")
	}
	return t.styles.Header.Render(headerContent.String())
}

func (t *ScrollableTable) renderSeparator() string {
	var parts []string
	for _, col := range t.columns {
		parts = append(parts, strings.Repeat("─", col.Width+2))
	}
	return t.styles.Border.Render(strings.Join(parts, ""))
}

func (t *ScrollableTable) renderRow(index int) string {
	if index < 0 || index >= len(t.rows) {
		return ""
	}

	row := t.rows[index]
	isSelected := index == t.cursor && t.focused

	var rowContent strings.Builder
	for i, col := range t.columns {
		cellValue := ""
		if i < len(row) {
			cellValue = row[i]
		}
		cell := t.padOrTruncate(cellValue, col.Width)
		rowContent.WriteString(" ")
		rowContent.WriteString(cell)
		rowContent.WriteString(" ")
	}

	if isSelected {
		return t.styles.Selected.Render(rowContent.String())
	}
	return t.styles.Cell.Render(rowContent.String())
}

func (t *ScrollableTable) padOrTruncate(s string, width int) string {
	visibleLen := t.visibleLength(s)

	if visibleLen > width {
		return t.truncateWithAnsi(s, width-3) + "..."
	}

	padding := width - visibleLen
	if padding > 0 {
		return s + strings.Repeat(" ", padding)
	}
	return s
}

func (t *ScrollableTable) visibleLength(s string) int {
	inEscape := false
	length := 0
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		length++
	}
	return length
}

func (t *ScrollableTable) truncateWithAnsi(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	var result strings.Builder
	inEscape := false
	visibleCount := 0

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		if visibleCount >= maxLen {
			break
		}
		result.WriteRune(r)
		visibleCount++
	}

	result.WriteString("\x1b[0m")
	return result.String()
}

func (t *ScrollableTable) applyHorizontalScroll(line string, visibleWidth int) string {
	type charWithStyle struct {
		char  rune
		style string
	}

	var chars []charWithStyle
	var currentStyle strings.Builder
	inEscape := false

	for _, r := range line {
		if r == '\x1b' {
			inEscape = true
			currentStyle.WriteRune(r)
			continue
		}
		if inEscape {
			currentStyle.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		chars = append(chars, charWithStyle{
			char:  r,
			style: currentStyle.String(),
		})
		currentStyle.Reset()
	}

	startIdx := t.horizontalOffset
	if startIdx > len(chars) {
		startIdx = len(chars)
	}
	endIdx := startIdx + visibleWidth
	if endIdx > len(chars) {
		endIdx = len(chars)
	}

	var result strings.Builder
	result.WriteString("  ")

	lastStyle := ""
	actualWidth := 0
	for i := startIdx; i < endIdx; i++ {
		c := chars[i]
		if c.style != "" && c.style != lastStyle {
			result.WriteString(c.style)
			lastStyle = c.style
		}
		result.WriteRune(c.char)
		actualWidth++
	}

	if actualWidth < visibleWidth {
		padding := visibleWidth - actualWidth
		result.WriteString(strings.Repeat(" ", padding))
	}

	result.WriteString("\x1b[0m")

	return result.String()
}
