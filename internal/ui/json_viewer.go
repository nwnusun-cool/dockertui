package ui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	// 回调
	onClose func()
}

// NewJSONViewer 创建 JSON 查看器
func NewJSONViewer() *JSONViewer {
	return &JSONViewer{
		visible: false,
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
	v.updateMaxScroll()
}

// Hide 隐藏查看器
func (v *JSONViewer) Hide() {
	v.visible = false
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

// Update 处理按键
func (v *JSONViewer) Update(msg tea.KeyMsg) bool {
	if !v.visible {
		return false
	}

	switch msg.String() {
	case "esc", "q", "i":
		v.Hide()
		if v.onClose != nil {
			v.onClose()
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
		lineNum := i + v.scrollY + 1
		line := v.lines[i+v.scrollY]

		// 水平滚动
		if v.scrollX > 0 && len(line) > v.scrollX {
			line = line[v.scrollX:]
		} else if v.scrollX > 0 {
			line = ""
		}

		// 截断过长的行
		if len(line) > visibleWidth {
			line = line[:visibleWidth-3] + "..."
		}

		// 语法高亮
		coloredLine := v.colorize(line)

		// 行号 + 内容
		lineNumStr := jsonViewerLineNumStyle.Render(strconv.Itoa(lineNum))
		s.WriteString("  " + lineNumStr + " │ " + coloredLine + "\n")
	}

	// 底部分隔线
	s.WriteString("  " + strings.Repeat("─", v.width-6) + "\n")

	// 滚动信息和快捷键提示
	scrollInfo := ""
	if v.maxScrollY > 0 {
		percent := 0
		if v.maxScrollY > 0 {
			percent = v.scrollY * 100 / v.maxScrollY
		}
		scrollInfo = jsonViewerHintStyle.Render(strconv.Itoa(percent) + "%")
	}

	hints := jsonViewerHintStyle.Render("j/k=上下  g/G=首尾  Ctrl+D/U=翻页  h/l=左右  ESC/q=关闭")
	s.WriteString("  " + hints + "  " + scrollInfo + "\n")

	return s.String()
}

// colorize 对 JSON 行进行语法高亮
func (v *JSONViewer) colorize(line string) string {
	// 简单的语法高亮
	result := line

	// 处理键名（"key":）
	inString := false
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
			inString = !inString
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
