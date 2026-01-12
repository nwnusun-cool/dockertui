package network

import "github.com/charmbracelet/lipgloss"

// 网络列表视图样式
var (
	TitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	DriverStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	BuiltInStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	CustomStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	StateBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(66)
)

// 网络详情视图样式
var (
	DetailTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	DetailLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Width(16)

	DetailValueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	DetailBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	TabActiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true).
		Underline(true)

	TabInactiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	DetailHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	DetailKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	ContainerRunningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	ContainerStoppedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
)

// 创建网络表单样式
var (
	FormTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	FormLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Width(14)

	FormInputStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	FormInputActiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	FormHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	FormCheckboxStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	FormButtonStyle = lipgloss.NewStyle().
		Padding(0, 2)

	FormButtonActiveStyle = lipgloss.NewStyle().
		Padding(0, 2).
		Reverse(true).
		Bold(true)

	FormErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))
)
