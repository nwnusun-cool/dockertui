package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PullInputView 镜像拉取输入框
type PullInputView struct {
	input     textinput.Model
	visible   bool
	width     int
	selection int // 0=取消, 1=确认
}

// 输入框样式
var (
	pullInputBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("81")).
		Padding(1, 2)

	pullInputTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	pullInputLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	pullInputHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
)

// NewPullInputView 创建拉取输入框
func NewPullInputView() *PullInputView {
	ti := textinput.New()
	ti.Placeholder = "nginx:latest"
	ti.CharLimit = 128
	ti.Width = 40
	ti.Prompt = ""

	return &PullInputView{
		input:   ti,
		visible: false,
	}
}

// Show 显示输入框
func (v *PullInputView) Show() {
	v.visible = true
	v.selection = 0 // 默认选中取消
	v.input.SetValue("")
	v.input.Focus()
}

// Hide 隐藏输入框
func (v *PullInputView) Hide() {
	v.visible = false
	v.selection = 0
	v.input.Blur()
}

// IsVisible 是否可见
func (v *PullInputView) IsVisible() bool {
	return v.visible
}

// Value 获取输入值
func (v *PullInputView) Value() string {
	return strings.TrimSpace(v.input.Value())
}

// SetWidth 设置宽度
func (v *PullInputView) SetWidth(width int) {
	v.width = width
	inputWidth := width - 20
	if inputWidth < 30 {
		inputWidth = 30
	}
	if inputWidth > 60 {
		inputWidth = 60
	}
	v.input.Width = inputWidth
}

// Update 处理输入
// 返回值: (confirmed bool, handled bool, cmd tea.Cmd)
// confirmed: 是否确认拉取
// handled: 事件是否已被处理（用于阻止事件传播）
func (v *PullInputView) Update(msg tea.Msg) (bool, bool, tea.Cmd) {
	if !v.visible {
		return false, false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		keyStr := msg.String()
		
		// 处理 Enter 键（同时检查类型和字符串）
		if msg.Type == tea.KeyEnter || keyStr == "enter" {
			// 如果有输入内容，直接确认拉取（无论按钮选择状态）
			if v.Value() != "" {
				return true, true, nil
			}
			// 没有输入内容，根据按钮选择执行
			if v.selection == 1 {
				// 确认按钮被选中但没有输入，不执行任何操作
				return false, true, nil
			}
			// 取消
			v.Hide()
			return false, true, nil
		}
		
		// 处理 Esc 键
		if msg.Type == tea.KeyEsc || keyStr == "esc" {
			v.Hide()
			return false, true, nil
		}
		
		// 处理 Tab 键切换按钮
		if msg.Type == tea.KeyTab {
			v.selection = 1 - v.selection
			return false, true, nil
		}
		
		// 处理上下方向键切换按钮（上下键不影响文本输入）
		if msg.Type == tea.KeyUp || keyStr == "up" {
			v.selection = 0
			return false, true, nil
		}
		if msg.Type == tea.KeyDown || keyStr == "down" {
			v.selection = 1
			return false, true, nil
		}
	}

	// 其他按键传递给输入框
	var cmd tea.Cmd
	v.input, cmd = v.input.Update(msg)
	return false, true, cmd
}

// View 渲染输入框
func (v *PullInputView) View() string {
	if !v.visible {
		return ""
	}

	// 标题
	title := pullInputTitleStyle.Render("📥 拉取镜像")

	// 输入行
	label := pullInputLabelStyle.Render("镜像名称: ")
	inputLine := label + v.input.View()

	// 按钮样式
	cancelBtnStyle := lipgloss.NewStyle().Padding(0, 2)
	okBtnStyle := lipgloss.NewStyle().Padding(0, 2)

	if v.selection == 0 {
		// 取消被选中
		cancelBtnStyle = cancelBtnStyle.Reverse(true).Bold(true)
		okBtnStyle = okBtnStyle.Foreground(lipgloss.Color("245"))
	} else {
		// 确认被选中
		cancelBtnStyle = cancelBtnStyle.Foreground(lipgloss.Color("245"))
		okBtnStyle = okBtnStyle.Reverse(true).Bold(true)
	}

	cancelBtn := cancelBtnStyle.Render("< 取消 >")
	okBtn := okBtnStyle.Render("< 确认 >")
	buttons := cancelBtn + "    " + okBtn

	// 提示
	hints := pullInputHintStyle.Render("[↑/↓/Tab=切换按钮] [Enter=确认] [Esc=取消]")

	// 组合内容
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		inputLine,
		"",
		buttons,
		"",
		hints,
	)

	// 计算框宽度
	boxWidth := v.width - 10
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 70 {
		boxWidth = 70
	}

	box := pullInputBoxStyle.Width(boxWidth).Render(content)

	// 居中
	if v.width > boxWidth+10 {
		leftPadding := (v.width - boxWidth - 4) / 2
		lines := strings.Split(box, "\n")
		for i, line := range lines {
			lines[i] = strings.Repeat(" ", leftPadding) + line
		}
		return strings.Join(lines, "\n")
	}

	return box
}
