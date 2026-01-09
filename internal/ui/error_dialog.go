package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrorDialog 错误弹窗组件
type ErrorDialog struct {
	visible  bool
	title    string
	message  string
	width    int
}

// 错误弹窗样式
var (
	errorDialogStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2)

	errorDialogTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	errorDialogMsgStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	errorDialogHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
)

// NewErrorDialog 创建错误弹窗
func NewErrorDialog() *ErrorDialog {
	return &ErrorDialog{
		visible: false,
		width:   80,
	}
}

// Show 显示错误弹窗
func (d *ErrorDialog) Show(title, message string) {
	d.visible = true
	d.title = title
	d.message = message
}

// ShowError 显示错误（简化方法）
func (d *ErrorDialog) ShowError(message string) {
	d.Show("❌ 操作失败", message)
}

// Hide 隐藏错误弹窗
func (d *ErrorDialog) Hide() {
	d.visible = false
	d.title = ""
	d.message = ""
}

// IsVisible 是否可见
func (d *ErrorDialog) IsVisible() bool {
	return d.visible
}

// SetWidth 设置宽度
func (d *ErrorDialog) SetWidth(width int) {
	d.width = width
}

// Update 处理输入
// 返回 handled: 事件是否已被处理
func (d *ErrorDialog) Update(msg tea.Msg) bool {
	if !d.visible {
		return false
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// ESC 或 Enter 关闭弹窗
		switch msg.String() {
		case "esc", "enter":
			d.Hide()
			return true
		}
		// 其他按键也拦截，防止穿透
		return true
	}

	return false
}

// View 渲染错误弹窗
func (d *ErrorDialog) View() string {
	if !d.visible {
		return ""
	}

	// 计算弹窗宽度
	boxWidth := d.width - 10
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 80 {
		boxWidth = 80
	}
	contentWidth := boxWidth - 6 // 减去边框和 padding

	// 按行分割消息
	var msgLines []string
	msg := d.message
	// 移除开头的 ❌ 符号（如果有）
	msg = strings.TrimPrefix(msg, "❌ ")

	for len(msg) > contentWidth {
		msgLines = append(msgLines, msg[:contentWidth])
		msg = msg[contentWidth:]
	}
	if msg != "" {
		msgLines = append(msgLines, msg)
	}

	// 构建内容
	var contentParts []string
	contentParts = append(contentParts, errorDialogTitleStyle.Render(d.title))
	contentParts = append(contentParts, "")
	for _, line := range msgLines {
		contentParts = append(contentParts, errorDialogMsgStyle.Render(line))
	}
	contentParts = append(contentParts, "")
	contentParts = append(contentParts, errorDialogHintStyle.Render("[Esc/Enter] 关闭"))

	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)
	return errorDialogStyle.Width(boxWidth).Render(content)
}

// Overlay 将错误弹窗叠加到基础内容上（居中显示）
func (d *ErrorDialog) Overlay(baseContent string) string {
	if !d.visible {
		return baseContent
	}

	// 将基础内容按行分割
	lines := strings.Split(baseContent, "\n")

	// 获取弹窗内容
	dialog := d.View()
	dialogLines := strings.Split(dialog, "\n")

	// 计算弹窗位置（垂直居中）
	dialogHeight := len(dialogLines)
	insertLine := 0
	if len(lines) > dialogHeight {
		insertLine = (len(lines) - dialogHeight) / 2
	}

	// 计算水平居中的 padding
	boxWidth := d.width - 10
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 80 {
		boxWidth = 80
	}
	leftPadding := 0
	if d.width > boxWidth+4 {
		leftPadding = (d.width - boxWidth - 4) / 2
	}

	// 构建最终输出
	var result strings.Builder
	for i := 0; i < len(lines); i++ {
		dialogIdx := i - insertLine
		if dialogIdx >= 0 && dialogIdx < len(dialogLines) {
			result.WriteString(strings.Repeat(" ", leftPadding))
			result.WriteString(dialogLines[dialogIdx])
		} else if i < len(lines) {
			result.WriteString(lines[i])
		}
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
