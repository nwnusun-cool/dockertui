// Package styles 定义全局统一的 UI 样式
package styles

import "github.com/charmbracelet/lipgloss"

// 颜色常量
const (
	ColorPrimary   = "220" // 黄色 - 标题、高亮
	ColorSecondary = "81"  // 蓝色 - 键名、标签
	ColorSuccess   = "82"  // 绿色 - 成功、运行中
	ColorError     = "196" // 红色 - 错误
	ColorWarning   = "214" // 橙色 - 警告
	ColorMuted     = "245" // 灰色 - 次要信息、提示
	ColorText      = "252" // 白色 - 正常文本
	ColorBorder    = "240" // 深灰 - 边框
)

// ========== 通用基础样式 ==========

// 标题样式
var (
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSecondary)).
			Bold(true)
)

// 文本样式
var (
	TextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText))

	MutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))

	KeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSecondary))

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText))

	LabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSecondary)).
			Width(16)

	HintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))
)

// 消息样式
var (
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess)).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorError)).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWarning)).
			Bold(true)
)

// ========== 状态样式 ==========

var (
	RunningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess))

	StoppedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))

	ActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess))

	InactiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))
)

// ========== 标签页样式 ==========

var (
	TabActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true).
			Underline(true)

	TabInactiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))
)

// ========== 按钮样式 ==========

var (
	ButtonActiveStyle = lipgloss.NewStyle().
				Reverse(true).
				Bold(true).
				Padding(0, 2)

	ButtonInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorMuted)).
				Padding(0, 2)
)

// ========== 边框/容器样式 ==========

var (
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorBorder)).
			Padding(0, 1)

	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorBorder)).
			Padding(1, 2)

	StateBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorBorder)).
			Padding(1, 2).
			Width(66)
)

// ========== 搜索样式 ==========

var (
	SearchPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorSecondary)).
				Bold(true)

	SearchHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))
)

// ========== 表单样式 ==========

var (
	FormTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true)

	FormLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSecondary)).
			Width(14)

	FormInputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText))

	FormInputActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorPrimary)).
				Bold(true)

	FormHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))

	FormCheckboxStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorSuccess))

	FormErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorError))
)

// ========== 状态栏样式 ==========

var (
	StatusBarLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorPrimary)).
				Bold(true)

	StatusBarKeyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorSecondary))

	StatusBarValueStyle = lipgloss.NewStyle()
)
