package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ScrollableTable 可水平滚动的表格组件
type ScrollableTable struct {
	// 列定义
	columns []TableColumn
	// 行数据
	rows []TableRow
	// 当前选中行索引
	cursor int
	// 水平滚动偏移量（字符数）
	horizontalOffset int
	// 滚动步长
	scrollStep int
	// 可见区域
	width  int
	height int
	// 是否聚焦
	focused bool
	// 样式
	styles TableStyles
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
	Header       lipgloss.Style
	Cell         lipgloss.Style
	Selected     lipgloss.Style
	Border       lipgloss.Style
	ScrollIndicator lipgloss.Style
}

// DefaultTableStyles 默认表格样式
func DefaultTableStyles() TableStyles {
	return TableStyles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(0, 1),
		Cell: lipgloss.NewStyle().
			Padding(0, 1),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Padding(0, 1),
		Border: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		ScrollIndicator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
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
	// 确保 cursor 在有效范围内
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

// getTotalWidth 计算表格总宽度
func (t *ScrollableTable) getTotalWidth() int {
	total := 0
	for _, col := range t.columns {
		total += col.Width + 2 // +2 for padding
	}
	return total
}

// View 渲染表格
func (t *ScrollableTable) View() string {
	if len(t.columns) == 0 {
		return ""
	}

	var b strings.Builder
	
	// 计算可见区域
	visibleWidth := t.width - 4 // 留出边距
	if visibleWidth < 40 {
		visibleWidth = 40
	}
	
	totalWidth := t.getTotalWidth()
	canScrollLeft := t.horizontalOffset > 0
	canScrollRight := t.horizontalOffset < totalWidth-visibleWidth
	
	// 渲染滚动指示器（顶部）
	if canScrollLeft || canScrollRight {
		scrollInfo := t.renderScrollIndicator(canScrollLeft, canScrollRight)
		b.WriteString(scrollInfo)
		b.WriteString("\n")
	}
	
	// 渲染表头
	headerLine := t.renderHeader()
	b.WriteString(t.applyHorizontalScroll(headerLine, visibleWidth))
	b.WriteString("\n")
	
	// 渲染分隔线
	separatorLine := t.renderSeparator()
	b.WriteString(t.applyHorizontalScroll(separatorLine, visibleWidth))
	b.WriteString("\n")
	
	// 计算可见行范围（垂直滚动）
	startRow := 0
	endRow := len(t.rows)
	visibleRows := t.height - 3 // 减去表头、分隔线、滚动指示器
	if visibleRows < 1 {
		visibleRows = 1
	}
	
	// 确保选中行可见
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
	
	// 渲染数据行
	for i := startRow; i < endRow; i++ {
		rowLine := t.renderRow(i)
		b.WriteString(t.applyHorizontalScroll(rowLine, visibleWidth))
		if i < endRow-1 {
			b.WriteString("\n")
		}
	}
	
	return b.String()
}

// renderScrollIndicator 渲染滚动指示器
func (t *ScrollableTable) renderScrollIndicator(canLeft, canRight bool) string {
	leftArrow := "  "
	rightArrow := "  "
	
	if canLeft {
		leftArrow = t.styles.ScrollIndicator.Render("◀ ")
	}
	if canRight {
		rightArrow = t.styles.ScrollIndicator.Render(" ▶")
	}
	
	// 计算滚动百分比
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

// renderHeader 渲染表头
func (t *ScrollableTable) renderHeader() string {
	var cells []string
	for _, col := range t.columns {
		cell := t.padOrTruncate(col.Title, col.Width)
		cells = append(cells, t.styles.Header.Render(cell))
	}
	return strings.Join(cells, "")
}

// renderSeparator 渲染分隔线
func (t *ScrollableTable) renderSeparator() string {
	var parts []string
	for _, col := range t.columns {
		parts = append(parts, strings.Repeat("─", col.Width+2))
	}
	return t.styles.Border.Render(strings.Join(parts, ""))
}

// renderRow 渲染数据行
func (t *ScrollableTable) renderRow(index int) string {
	if index < 0 || index >= len(t.rows) {
		return ""
	}
	
	row := t.rows[index]
	isSelected := index == t.cursor && t.focused
	
	var cells []string
	for i, col := range t.columns {
		cellValue := ""
		if i < len(row) {
			cellValue = row[i]
		}
		cell := t.padOrTruncate(cellValue, col.Width)
		
		if isSelected {
			cells = append(cells, t.styles.Selected.Render(cell))
		} else {
			cells = append(cells, t.styles.Cell.Render(cell))
		}
	}
	return strings.Join(cells, "")
}

// padOrTruncate 填充或截断字符串到指定宽度
func (t *ScrollableTable) padOrTruncate(s string, width int) string {
	// 计算实际显示宽度（考虑 ANSI 转义码）
	visibleLen := t.visibleLength(s)
	
	if visibleLen > width {
		// 截断（需要处理 ANSI 转义码）
		return t.truncateWithAnsi(s, width-3) + "..."
	}
	
	// 填充
	padding := width - visibleLen
	if padding > 0 {
		return s + strings.Repeat(" ", padding)
	}
	return s
}

// visibleLength 计算可见字符长度（排除 ANSI 转义码）
func (t *ScrollableTable) visibleLength(s string) int {
	// 简单实现：移除 ANSI 转义码后计算长度
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

// truncateWithAnsi 截断字符串（保留 ANSI 转义码）
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
	
	// 确保关闭所有 ANSI 样式
	result.WriteString("\x1b[0m")
	return result.String()
}

// applyHorizontalScroll 应用水平滚动
func (t *ScrollableTable) applyHorizontalScroll(line string, visibleWidth int) string {
	if t.horizontalOffset == 0 && t.visibleLength(line) <= visibleWidth {
		return "  " + line
	}
	
	// 将行转换为可见字符数组（保留 ANSI 转义码）
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
	
	// 应用水平偏移
	startIdx := t.horizontalOffset
	if startIdx > len(chars) {
		startIdx = len(chars)
	}
	endIdx := startIdx + visibleWidth
	if endIdx > len(chars) {
		endIdx = len(chars)
	}
	
	// 构建可见部分
	var result strings.Builder
	result.WriteString("  ") // 左边距
	
	lastStyle := ""
	for i := startIdx; i < endIdx; i++ {
		c := chars[i]
		if c.style != "" && c.style != lastStyle {
			result.WriteString(c.style)
			lastStyle = c.style
		}
		result.WriteRune(c.char)
	}
	
	// 重置样式
	result.WriteString("\x1b[0m")
	
	return result.String()
}
