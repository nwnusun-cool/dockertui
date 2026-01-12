package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TagInputView 镜像打标签输入框
type TagInputView struct {
	repoInput     textinput.Model
	tagInput      textinput.Model
	sourceImage   string
	SourceImageID string
	visible       bool
	width         int
	focusIndex    int
}

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
	repoInput := textinput.New()
	repoInput.Placeholder = "myrepo/image"
	repoInput.CharLimit = 128
	repoInput.Width = 40
	repoInput.Prompt = ""

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
	v.SourceImageID = sourceImageID
	v.repoInput.SetValue(sourceRepo)
	v.tagInput.SetValue("")
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
func (v *TagInputView) Update(msg tea.Msg) (bool, bool, tea.Cmd) {
	if !v.visible {
		return false, false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		keyStr := msg.String()

		if msg.Type == tea.KeyEnter || keyStr == "enter" {
			if v.focusIndex == 3 {
				repo, _ := v.GetValues()
				if repo != "" {
					return true, true, nil
				}
				return false, true, nil
			}
			if v.focusIndex == 2 {
				v.Hide()
				return false, true, nil
			}
			v.nextFocus()
			return false, true, nil
		}

		if msg.Type == tea.KeyEsc || keyStr == "esc" {
			v.Hide()
			return false, true, nil
		}

		if msg.Type == tea.KeyTab {
			v.nextFocus()
			return false, true, nil
		}

		if msg.Type == tea.KeyShiftTab {
			v.prevFocus()
			return false, true, nil
		}

		if msg.Type == tea.KeyUp || keyStr == "up" {
			v.prevFocus()
			return false, true, nil
		}
		if msg.Type == tea.KeyDown || keyStr == "down" {
			v.nextFocus()
			return false, true, nil
		}

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

	var cmd tea.Cmd
	if v.focusIndex == 0 {
		v.repoInput, cmd = v.repoInput.Update(msg)
	} else if v.focusIndex == 1 {
		v.tagInput, cmd = v.tagInput.Update(msg)
	}

	return false, true, cmd
}

func (v *TagInputView) nextFocus() {
	v.focusIndex = (v.focusIndex + 1) % 4
	v.updateInputFocus()
}

func (v *TagInputView) prevFocus() {
	v.focusIndex = (v.focusIndex + 3) % 4
	v.updateInputFocus()
}

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

	title := tagInputTitleStyle.Render("🏷️  给镜像打标签")

	sourceInfo := tagInputLabelStyle.Render("源镜像:") + " " +
		tagInputSourceStyle.Render(v.sourceImage) +
		tagInputHintStyle.Render(" ("+v.SourceImageID[:12]+")")

	repoLabel := tagInputLabelStyle.Render("仓库名:")
	repoInputStyle := lipgloss.NewStyle()
	if v.focusIndex == 0 {
		repoInputStyle = repoInputStyle.Foreground(lipgloss.Color("81"))
	}
	repoLine := repoLabel + " " + repoInputStyle.Render(v.repoInput.View())

	tagLabel := tagInputLabelStyle.Render("标  签:")
	tagInputStyle := lipgloss.NewStyle()
	if v.focusIndex == 1 {
		tagInputStyle = tagInputStyle.Foreground(lipgloss.Color("81"))
	}
	tagLine := tagLabel + " " + tagInputStyle.Render(v.tagInput.View())

	repo, tag := v.GetValues()
	previewText := ""
	if repo != "" {
		previewText = tagInputHintStyle.Render("预览: ") +
			tagInputSourceStyle.Render(repo+":"+tag)
	}

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

	hints := tagInputHintStyle.Render("[Tab/↑↓=切换] [Enter=确认] [Esc=取消]")

	var contentParts []string
	contentParts = append(contentParts, title, "", sourceInfo, "", repoLine, "", tagLine)

	if previewText != "" {
		contentParts = append(contentParts, "", previewText)
	}

	contentParts = append(contentParts, "", buttons, "", hints)

	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)

	boxWidth := v.width - 10
	if boxWidth < 55 {
		boxWidth = 55
	}
	if boxWidth > 70 {
		boxWidth = 70
	}

	box := tagInputBoxStyle.Width(boxWidth).Render(content)

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
