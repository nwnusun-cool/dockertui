package container

import "github.com/charmbracelet/lipgloss"

// 容器列表视图样式定义 - 使用自适应颜色，不硬编码背景色
var (
	// 状态栏样式
	StatusBarLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	StatusBarValueStyle = lipgloss.NewStyle()
	
	StatusBarKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))
	
	// 标题栏样式
	TitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	// 过滤状态样式
	FilterAllStyle = lipgloss.NewStyle()
	
	FilterRunningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)
	
	FilterExitedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	// 成功/错误消息样式
	SuccessMsgStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)
	
	ErrorMsgStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	
	// 搜索栏样式
	SearchPromptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Bold(true)
	
	SearchHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	// 对话框样式 - 使用边框区分，不设置背景
	DialogStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)
	
	DialogTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	DialogWarningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	// 按钮样式 - 使用 Reverse 实现选中效果
	ButtonActiveStyle = lipgloss.NewStyle().
		Reverse(true).
		Bold(true).
		Padding(0, 2)
	
	ButtonInactiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 2)
	
	// 加载/空状态框样式
	StateBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(66)
)
