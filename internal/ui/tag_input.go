package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TagInputView 镜像打标签输入框
type TagInputView struct {
	// 输入框
	repoInput textinput.Model // 仓库名输入框
	tagInput  textinput.Model // 标签输入框

	// 源镜像信息
	sourceImage   string // 源镜像显示名称
	sourceImageID string // 源镜像 ID

	// UI 状态
	visible    bool
	width      int
	focusIndex int // 0=仓库名, 1=标签, 2=取消, 3=确认
}

// 样式定义
var (
	tagInputBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("81")).
		Padding(1, 2)

	tagInputTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	tagInputLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Width(10)

	tagInputHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	tagInputSourceStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))
)

// NewTagInputView 创建打标签输入框
func NewTagInputView() *TagInputView {
	// 仓库名输入框
	repoInput := textinput.New()
	repoInput.Placeholder = "myrepo/image"
	repoInput.CharLimit = 128
	repoInput.Width = 40
	repoInput.Prompt = ""

	// 标签输入框
	tagInput := textinput.New()
	tagInput.Placeholder = "latest"
	tagInput.CharLimit = 64
	tagInput.Width = 40
	tagInput.Prompt = ""

	return &TagInputView{
		repoInput:  repoInput,
		tagInput:   tagInput,
		visible:    false,
		focusIndex: 0,
	}
}

// Show 显示输入框
func (v *TagInputView) Show(sourceImage, sourceImageID, sourceRepo, sourceTag string) {
	v.visible = true
	v.sourceImage = sourceImage
	v.sourceImageID = sourceImageID

	// 预填充仓库名
	v.repoInput.SetValue(sourceRepo)
	v.tagInput.SetValue("")

	// 聚焦到仓库名输入框
	v.focusIndex = 0
	v.repoInput.Focus()
	v.tagInput.Blur()
}

// Hide 隐藏输入框
func (v *TagInputView) Hide() {
	v.visible = false
	v.repoInput.Blur()
	v.tagInput.Blur()
}

// IsVisible 是否可见
func (v *TagInputView) IsVisible() bool {
	return v.visible
}

// GetValues 获取输入值
func (v *TagInputView) GetValues() (repository, tag string) {
	repository = strings.TrimSpace(v.repoInput.Value())
	tag = strings.TrimSpace(v.tagInput.Value())

	// 如果标签为空，使用 latest
	if tag == "" {
		tag = "latest"
	}

	return repository, tag
}

// SetWidth 设置宽度
func (v *TagInputView) SetWidth(width int) {
	v.width = width
	inputWidth := width - 30
	if inputWidth < 25 {
		inputWidth = 25
	}
	if inputWidth > 50 {
		inputWidth = 50
	}
	v.repoInput.Width = inputWidth
	v.tagInput.Width = inputWidth
}

// Update 处理输入
// 返回值: (confirmed bool, handled bool, cmd tea.Cmd)
func (v *TagInputView) Update(msg tea.Msg) (bool, bool, tea.Cmd) {
	if !v.visible {
		return false, false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		keyStr := msg.String()

		// Enter 键
		if msg.Type == tea.KeyEnter || keyStr == "enter" {
			// 如果焦点在确认按钮
			if v.focusIndex == 3 {
				repo, _ := v.GetValues()
				if repo != "" {
					return true, true, nil
				}
				return false, true, nil
			}

			// 如果焦点在取消按钮
			if v.focusIndex == 2 {
				v.Hide()
				return false, true, nil
			}

			// 其他情况，移动到下一个焦点
			v.nextFocus()
			return false, true, nil
		}

		// Esc 键
		if msg.Type == tea.KeyEsc || keyStr == "esc" {
			v.Hide()
			return false, true, nil
		}

		// Tab 键切换焦点
		if msg.Type == tea.KeyTab {
			v.nextFocus()
			return false, true, nil
		}

		// Shift+Tab 反向切换
		if msg.Type == tea.KeyShiftTab {
			v.prevFocus()
			return false, true, nil
		}

		// 上下键切换焦点
		if msg.Type == tea.KeyUp || keyStr == "up" {
			v.prevFocus()
			return false, true, nil
		}
		if msg.Type == tea.KeyDown || keyStr == "down" {
			v.nextFocus()
			return false, true, nil
		}

		// 左右键在按钮区域切换
		if v.focusIndex >= 2 {
			if msg.Type == tea.KeyLeft || keyStr == "left" {
				if v.focusIndex == 3 {
					v.focusIndex = 2
				}
				return false, true, nil
			}
			if msg.Type == tea.KeyRight || keyStr == "right" {
				if v.focusIndex == 2 {
					v.focusIndex = 3
				}
				return false, true, nil
			}
		}
	}

	// 传递给当前聚焦的输入框
	var cmd tea.Cmd
	if v.focusIndex == 0 {
		v.repoInput, cmd = v.repoInput.Update(msg)
	} else if v.focusIndex == 1 {
		v.tagInput, cmd = v.tagInput.Update(msg)
	}

	return false, true, cmd
}

// nextFocus 切换到下一个焦点
func (v *TagInputView) nextFocus() {
	v.focusIndex = (v.focusIndex + 1) % 4
	v.updateInputFocus()
}

// prevFocus 切换到上一个焦点
func (v *TagInputView) prevFocus() {
	v.focusIndex = (v.focusIndex + 3) % 4
	v.updateInputFocus()
}

// updateInputFocus 更新输入框焦点状态
func (v *TagInputView) updateInputFocus() {
	if v.focusIndex == 0 {
		v.repoInput.Focus()
		v.tagInput.Blur()
	} else if v.focusIndex == 1 {
		v.repoInput.Blur()
		v.tagInput.Focus()
	} else {
		v.repoInput.Blur()
		v.tagInput.Blur()
	}
}

// View 渲染输入框
func (v *TagInputView) View() string {
	if !v.visible {
		return ""
	}

	// 标题
	title := tagInputTitleStyle.Render("🏷️  给镜像打标签")

	// 源镜像信息
	sourceInfo := tagInputLabelStyle.Render("源镜像:") + " " +
		tagInputSourceStyle.Render(v.sourceImage) +
		tagInputHintStyle.Render(" ("+v.sourceImageID[:12]+")")

	// 仓库名输入行
	repoLabel := tagInputLabelStyle.Render("仓库名:")
	repoInputStyle := lipgloss.NewStyle()
	if v.focusIndex == 0 {
		repoInputStyle = repoInputStyle.Foreground(lipgloss.Color("81"))
	}
	repoLine := repoLabel + " " + repoInputStyle.Render(v.repoInput.View())

	// 标签输入行
	tagLabel := tagInputLabelStyle.Render("标  签:")
	tagInputStyle := lipgloss.NewStyle()
	if v.focusIndex == 1 {
		tagInputStyle = tagInputStyle.Foreground(lipgloss.Color("81"))
	}
	tagLine := tagLabel + " " + tagInputStyle.Render(v.tagInput.View())

	// 预览
	repo, tag := v.GetValues()
	previewText := ""
	if repo != "" {
		previewText = tagInputHintStyle.Render("预览: ") +
			tagInputSourceStyle.Render(repo+":"+tag)
	}

	// 按钮
	cancelBtnStyle := lipgloss.NewStyle().Padding(0, 2)
	okBtnStyle := lipgloss.NewStyle().Padding(0, 2)

	if v.focusIndex == 2 {
		cancelBtnStyle = cancelBtnStyle.Reverse(true).Bold(true)
	} else {
		cancelBtnStyle = cancelBtnStyle.Foreground(lipgloss.Color("245"))
	}

	if v.focusIndex == 3 {
		okBtnStyle = okBtnStyle.Reverse(true).Bold(true)
	} else {
		okBtnStyle = okBtnStyle.Foreground(lipgloss.Color("245"))
	}

	cancelBtn := cancelBtnStyle.Render("< 取消 >")
	okBtn := okBtnStyle.Render("< 确认 >")
	buttons := cancelBtn + "    " + okBtn

	// 提示
	hints := tagInputHintStyle.Render("[Tab/↑↓=切换] [Enter=确认] [Esc=取消]")

	// 组合内容
	var contentParts []string
	contentParts = append(contentParts, title, "", sourceInfo, "", repoLine, "", tagLine)

	if previewText != "" {
		contentParts = append(contentParts, "", previewText)
	}

	contentParts = append(contentParts, "", buttons, "", hints)

	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)

	// 计算框宽度
	boxWidth := v.width - 10
	if boxWidth < 55 {
		boxWidth = 55
	}
	if boxWidth > 70 {
		boxWidth = 70
	}

	box := tagInputBoxStyle.Width(boxWidth).Render(content)

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
