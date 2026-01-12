package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

// ErrorDialog 错误弹窗组件
type ErrorDialog struct {
	visible bool
	title   string
	message string
	width   int
}

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

// Update 处理输入，返回 handled: 事件是否已被处理
func (d *ErrorDialog) Update(msg tea.Msg) bool {
	if !d.visible {
		return false
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter":
			d.Hide()
			return true
		}
		return true
	}

	return false
}

// View 渲染错误弹窗
func (d *ErrorDialog) View() string {
	if !d.visible {
		return ""
	}

	boxWidth := d.width - 10
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 80 {
		boxWidth = 80
	}
	contentWidth := boxWidth - 6

	var msgLines []string
	msg := d.message
	msg = strings.TrimPrefix(msg, "❌ ")

	for len(msg) > contentWidth {
		msgLines = append(msgLines, msg[:contentWidth])
		msg = msg[contentWidth:]
	}
	if msg != "" {
		msgLines = append(msgLines, msg)
	}

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
	return OverlayCentered(baseContent, d.View(), d.width, 0)
}
