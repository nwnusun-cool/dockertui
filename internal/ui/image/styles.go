package image

import "github.com/charmbracelet/lipgloss"

// 镜像列表视图样式
var (
	StatusBarLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	StatusBarKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	TitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	DanglingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	ActiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	UnusedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	SuccessMsgStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)

	ErrorMsgStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	SearchPromptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Bold(true)

	SearchHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	StateBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(66)
)

// 镜像详情视图样式
var (
	DetailsTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	DetailsLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Width(16)

	DetailsValueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	DetailsBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	TabActiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true).
		Underline(true)

	TabInactiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	DetailsHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	DetailsKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	ContainerRunningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	ContainerStoppedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
)
