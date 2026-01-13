package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	exportInputTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	exportInputLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	exportInputValueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	exportInputHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	exportInputBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)

	exportInputSelectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	exportInputCursorStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("240"))
)

// ExportMode 导出模式
type ExportMode int

const (
	ExportModeSingle   ExportMode = iota
	ExportModeMultiple
)

// ExportInputView 导出输入视图
type ExportInputView struct {
	visible    bool
	width      int
	height     int
	exportDir  string
	exportMode ExportMode
	compress   bool
	images     []ExportImageInfo
	isEditing  bool
	cursorPos  int
	focusField int
	onConfirm  func(dir string, mode ExportMode, compress bool)
	onCancel   func()
}

// ExportImageInfo 导出镜像信息
type ExportImageInfo struct {
	ID         string
	Repository string
	Tag        string
}

// NewExportInputView 创建导出输入视图
func NewExportInputView() *ExportInputView {
	return &ExportInputView{
		exportDir:  "./exports",
		exportMode: ExportModeMultiple,
		compress:   true,
		focusField: 0,
	}
}

// Show 显示导出输入视图
func (v *ExportInputView) Show(images []ExportImageInfo) {
	v.visible = true
	v.images = images
	v.isEditing = false
	v.focusField = 0
	v.cursorPos = len(v.exportDir)
}

// Hide 隐藏视图
func (v *ExportInputView) Hide() {
	v.visible = false
}

// IsVisible 检查是否可见
func (v *ExportInputView) IsVisible() bool {
	return v.visible
}

// SetSize 设置尺寸
func (v *ExportInputView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetWidth 设置宽度
func (v *ExportInputView) SetWidth(width int) {
	v.width = width
}

// SetCallbacks 设置回调
func (v *ExportInputView) SetCallbacks(onConfirm func(string, ExportMode, bool), onCancel func()) {
	v.onConfirm = onConfirm
	v.onCancel = onCancel
}

// Update 处理按键
func (v *ExportInputView) Update(msg tea.KeyMsg) bool {
	if !v.visible {
		return false
	}

	if v.isEditing {
		switch msg.Type {
		case tea.KeyEsc:
			v.isEditing = false
			return true
		case tea.KeyEnter:
			v.isEditing = false
			return true
		case tea.KeyBackspace:
			if v.cursorPos > 0 {
				v.exportDir = v.exportDir[:v.cursorPos-1] + v.exportDir[v.cursorPos:]
				v.cursorPos--
			}
			return true
		case tea.KeyLeft:
			if v.cursorPos > 0 {
				v.cursorPos--
			}
			return true
		case tea.KeyRight:
			if v.cursorPos < len(v.exportDir) {
				v.cursorPos++
			}
			return true
		case tea.KeyRunes:
			v.exportDir = v.exportDir[:v.cursorPos] + string(msg.Runes) + v.exportDir[v.cursorPos:]
			v.cursorPos += len(msg.Runes)
			return true
		}
		return true
	}

	switch msg.String() {
	case "esc", "q":
		v.Hide()
		if v.onCancel != nil {
			v.onCancel()
		}
		return true
	case "enter":
		if v.focusField == 0 {
			v.isEditing = true
			v.cursorPos = len(v.exportDir)
		} else if v.focusField == 1 {
			if v.exportMode == ExportModeSingle {
				v.exportMode = ExportModeMultiple
			} else {
				v.exportMode = ExportModeSingle
			}
		} else if v.focusField == 2 {
			v.compress = !v.compress
		} else if v.focusField == 3 {
			v.Hide()
			if v.onConfirm != nil {
				v.onConfirm(v.exportDir, v.exportMode, v.compress)
			}
		} else if v.focusField == 4 {
			v.Hide()
			if v.onCancel != nil {
				v.onCancel()
			}
		}
		return true
	case "tab", "j", "down":
		v.focusField = (v.focusField + 1) % 5
		return true
	case "shift+tab", "k", "up":
		v.focusField = (v.focusField + 4) % 5
		return true
	case "space":
		if v.focusField == 1 {
			if v.exportMode == ExportModeSingle {
				v.exportMode = ExportModeMultiple
			} else {
				v.exportMode = ExportModeSingle
			}
		} else if v.focusField == 2 {
			v.compress = !v.compress
		}
		return true
	}

	return true
}

// View 渲染视图
func (v *ExportInputView) View() string {
	if !v.visible {
		return ""
	}

	var s strings.Builder

	title := exportInputTitleStyle.Render("📦 Export Images")
	s.WriteString(title + "\n\n")

	s.WriteString(exportInputLabelStyle.Render("Images to export:") + "\n")
	for i, img := range v.images {
		if i >= 5 {
			s.WriteString(exportInputHintStyle.Render("  ... and "+string(rune('0'+len(v.images)-5))+" more images") + "\n")
			break
		}
		name := img.Repository + ":" + img.Tag
		if img.Repository == "<none>" {
			name = img.ID[:12]
		}
		s.WriteString("  • " + exportInputValueStyle.Render(name) + "\n")
	}
	s.WriteString("\n")

	dirLabel := exportInputLabelStyle.Render("Export directory:")
	dirValue := v.exportDir
	if v.isEditing {
		before := dirValue[:v.cursorPos]
		after := dirValue[v.cursorPos:]
		cursor := exportInputCursorStyle.Render(" ")
		dirValue = before + cursor + after
	}
	if v.focusField == 0 {
		dirLabel = exportInputSelectedStyle.Render("▶ Export directory:")
	}
	s.WriteString(dirLabel + " " + dirValue)
	if v.focusField == 0 && !v.isEditing {
		s.WriteString(" " + exportInputHintStyle.Render("[Enter to edit]"))
	}
	s.WriteString("\n\n")

	modeLabel := exportInputLabelStyle.Render("Export mode:")
	modeValue := "Multiple files (each image separately)"
	if v.exportMode == ExportModeSingle {
		modeValue = "Single file (all images bundled)"
	}
	if v.focusField == 1 {
		modeLabel = exportInputSelectedStyle.Render("▶ Export mode:")
	}
	s.WriteString(modeLabel + " " + exportInputValueStyle.Render(modeValue))
	if v.focusField == 1 {
		s.WriteString(" " + exportInputHintStyle.Render("[Enter/Space to toggle]"))
	}
	s.WriteString("\n\n")

	compressLabel := exportInputLabelStyle.Render("Gzip compress:")
	compressValue := "No"
	if v.compress {
		compressValue = "Yes"
	}
	if v.focusField == 2 {
		compressLabel = exportInputSelectedStyle.Render("▶ Gzip compress:")
	}
	s.WriteString(compressLabel + " " + exportInputValueStyle.Render(compressValue))
	if v.focusField == 2 {
		s.WriteString(" " + exportInputHintStyle.Render("[Enter/Space to toggle]"))
	}
	s.WriteString("\n\n")

	confirmBtn := "[Confirm Export]"
	cancelBtn := "[Cancel]"
	if v.focusField == 3 {
		confirmBtn = exportInputSelectedStyle.Render("▶ [Confirm Export]")
	} else {
		confirmBtn = exportInputLabelStyle.Render(confirmBtn)
	}
	if v.focusField == 4 {
		cancelBtn = exportInputSelectedStyle.Render("▶ [Cancel]")
	} else {
		cancelBtn = exportInputLabelStyle.Render(cancelBtn)
	}
	s.WriteString("  " + confirmBtn + "    " + cancelBtn + "\n\n")

	s.WriteString(exportInputHintStyle.Render("Tab/j/k=Switch  Enter=Confirm  ESC=Cancel"))

	boxWidth := 60
	if v.width > 0 && v.width < boxWidth+10 {
		boxWidth = v.width - 10
	}
	content := exportInputBoxStyle.Width(boxWidth).Render(s.String())

	return content
}

// GetExportDir 获取导出目录
func (v *ExportInputView) GetExportDir() string {
	return v.exportDir
}

// GetExportMode 获取导出模式
func (v *ExportInputView) GetExportMode() ExportMode {
	return v.exportMode
}

// GetCompress 获取是否压缩
func (v *ExportInputView) GetCompress() bool {
	return v.compress
}
