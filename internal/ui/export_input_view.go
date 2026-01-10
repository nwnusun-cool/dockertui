package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// 导出输入视图样式
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
	ExportModeSingle   ExportMode = iota // 单文件（所有镜像打包）
	ExportModeMultiple                   // 多文件（每个镜像单独导出）
)

// ExportInputView 导出输入视图
type ExportInputView struct {
	visible bool
	width   int
	height  int

	// 导出配置
	exportDir  string     // 导出目录
	exportMode ExportMode // 导出模式
	compress   bool       // 是否压缩

	// 待导出的镜像
	images []ExportImageInfo

	// 输入状态
	isEditing   bool // 是否正在编辑目录
	cursorPos   int  // 光标位置
	focusField  int  // 当前焦点字段: 0=目录, 1=模式, 2=压缩, 3=确认, 4=取消

	// 回调
	onConfirm func(dir string, mode ExportMode, compress bool)
	onCancel  func()
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

// SetWidth 设置宽度（兼容旧接口）
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

	// 编辑模式
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

	// 普通模式
	switch msg.String() {
	case "esc", "q":
		v.Hide()
		if v.onCancel != nil {
			v.onCancel()
		}
		return true
	case "enter":
		if v.focusField == 0 {
			// 进入编辑目录模式
			v.isEditing = true
			v.cursorPos = len(v.exportDir)
		} else if v.focusField == 1 {
			// 切换导出模式
			if v.exportMode == ExportModeSingle {
				v.exportMode = ExportModeMultiple
			} else {
				v.exportMode = ExportModeSingle
			}
		} else if v.focusField == 2 {
			// 切换压缩
			v.compress = !v.compress
		} else if v.focusField == 3 {
			// 确认导出
			v.Hide()
			if v.onConfirm != nil {
				v.onConfirm(v.exportDir, v.exportMode, v.compress)
			}
		} else if v.focusField == 4 {
			// 取消
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
			// 切换导出模式
			if v.exportMode == ExportModeSingle {
				v.exportMode = ExportModeMultiple
			} else {
				v.exportMode = ExportModeSingle
			}
		} else if v.focusField == 2 {
			// 切换压缩
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

	// 标题
	title := exportInputTitleStyle.Render("📦 导出镜像")
	s.WriteString(title + "\n\n")

	// 显示待导出的镜像列表
	s.WriteString(exportInputLabelStyle.Render("待导出镜像:") + "\n")
	for i, img := range v.images {
		if i >= 5 {
			s.WriteString(exportInputHintStyle.Render("  ... 还有 "+string(rune('0'+len(v.images)-5))+" 个镜像") + "\n")
			break
		}
		name := img.Repository + ":" + img.Tag
		if img.Repository == "<none>" {
			name = img.ID[:12]
		}
		s.WriteString("  • " + exportInputValueStyle.Render(name) + "\n")
	}
	s.WriteString("\n")

	// 导出目录
	dirLabel := exportInputLabelStyle.Render("导出目录:")
	dirValue := v.exportDir
	if v.isEditing {
		// 显示光标
		before := dirValue[:v.cursorPos]
		after := dirValue[v.cursorPos:]
		cursor := exportInputCursorStyle.Render(" ")
		dirValue = before + cursor + after
	}
	if v.focusField == 0 {
		dirLabel = exportInputSelectedStyle.Render("▶ 导出目录:")
	}
	s.WriteString(dirLabel + " " + dirValue)
	if v.focusField == 0 && !v.isEditing {
		s.WriteString(" " + exportInputHintStyle.Render("[Enter 编辑]"))
	}
	s.WriteString("\n\n")

	// 导出模式
	modeLabel := exportInputLabelStyle.Render("导出模式:")
	modeValue := "多文件（每个镜像单独导出）"
	if v.exportMode == ExportModeSingle {
		modeValue = "单文件（所有镜像打包）"
	}
	if v.focusField == 1 {
		modeLabel = exportInputSelectedStyle.Render("▶ 导出模式:")
	}
	s.WriteString(modeLabel + " " + exportInputValueStyle.Render(modeValue))
	if v.focusField == 1 {
		s.WriteString(" " + exportInputHintStyle.Render("[Enter/Space 切换]"))
	}
	s.WriteString("\n\n")

	// 压缩选项
	compressLabel := exportInputLabelStyle.Render("Gzip 压缩:")
	compressValue := "否"
	if v.compress {
		compressValue = "是"
	}
	if v.focusField == 2 {
		compressLabel = exportInputSelectedStyle.Render("▶ Gzip 压缩:")
	}
	s.WriteString(compressLabel + " " + exportInputValueStyle.Render(compressValue))
	if v.focusField == 2 {
		s.WriteString(" " + exportInputHintStyle.Render("[Enter/Space 切换]"))
	}
	s.WriteString("\n\n")

	// 按钮
	confirmBtn := "[确认导出]"
	cancelBtn := "[取消]"
	if v.focusField == 3 {
		confirmBtn = exportInputSelectedStyle.Render("▶ [确认导出]")
	} else {
		confirmBtn = exportInputLabelStyle.Render(confirmBtn)
	}
	if v.focusField == 4 {
		cancelBtn = exportInputSelectedStyle.Render("▶ [取消]")
	} else {
		cancelBtn = exportInputLabelStyle.Render(cancelBtn)
	}
	s.WriteString("  " + confirmBtn + "    " + cancelBtn + "\n\n")

	// 提示
	s.WriteString(exportInputHintStyle.Render("Tab/j/k=切换  Enter=确认  ESC=取消"))

	// 包装在框中
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
