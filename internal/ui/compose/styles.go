package compose

import "github.com/charmbracelet/lipgloss"

// 共享样式定义
var (
	// Header 样式
	HeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1).
			Bold(true)

	// 状态样式
	StatusRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82")).
				Bold(true)

	StatusPartialStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Bold(true)

	StatusStoppedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	StatusErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

	// Footer 样式
	FooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)

	FooterKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true)

	// 消息样式
	LoadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

	EmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)

	// Tab 样式
	TabActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true).
			Underline(true)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	// Box 样式
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	BoxFocusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("81")).
			Padding(0, 1)

	// Label/Value 样式
	LabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81"))

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	// Config 面板样式
	ConfigTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Bold(true)

	ConfigContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	ConfigHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)
)
