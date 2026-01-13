package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
	"docktui/internal/ui/components"
)

var (
	// 帮助视图样式（借鉴 k9s）
	helpTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("11")). // 黄色
		MarginLeft(2).
		MarginTop(1)

	helpTableStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginLeft(2)

	helpHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	helpKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")). // 青色
		Bold(true)

	helpDescStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	helpFooterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginLeft(2).
		MarginTop(1)
)

// HelpView 帮助面板视图（使用 bubbles/help + lipgloss 组件）
type HelpView struct {
	dockerClient docker.Client
	
	// UI 尺寸
	width  int
	height int
	
	// 帮助组件
	help help.Model
	keys components.KeyMap
}

// helpSection 帮助章节
type helpSection struct {
	title string
	items []helpItem
}

// helpItem 帮助项
type helpItem struct {
	key  string
	desc string
}

// NewHelpView 创建帮助视图
func NewHelpView(dockerClient docker.Client) *HelpView {
	h := help.New()
	h.ShowAll = true // 显示完整帮助
	
	return &HelpView{
		dockerClient: dockerClient,
		help:         h,
		keys:         components.DefaultKeyMap(),
	}
}

// Init 初始化帮助视图
func (v *HelpView) Init() tea.Cmd {
	return nil
}

// Update 处理消息并更新视图状态
func (v *HelpView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" || msg.String() == "?" {
			// ESC 或 ? 返回上一级
			return v, func() tea.Msg { return GoBackMsg{} }
		}
	}
	return v, nil
}

// View 渲染帮助面板（借鉴 k9s 风格）
func (v *HelpView) View() string {
	// 定义帮助章节
	sections := []helpSection{
		{
			title: "Global Shortcuts",
			items: []helpItem{
				{"q / Ctrl+C", "Quit"},
				{"?", "Show/Hide Help"},
				{"Esc", "Go Back"},
				{"c", "Go to Containers"},
				{"i", "Go to Images"},
				{"n", "Go to Networks (WIP)"},
				{"v", "Go to Volumes (WIP)"},
				{"o", "Go to Compose"},
			},
		},
		{
			title: "Home Navigation",
			items: []helpItem{
				{"↑/↓", "Switch Runtime/Resource"},
				{"←/→", "Select Runtime/Resource"},
				{"1-5", "Quick Select Resource"},
				{"Enter", "Enter Selected"},
				{"r", "Refresh"},
			},
		},
		{
			title: "List Navigation",
			items: []helpItem{
				{"j / ↓", "Move Down"},
				{"k / ↑", "Move Up"},
				{"g / Home", "Go to Top"},
				{"G / End", "Go to Bottom"},
				{"/", "Search"},
			},
		},
		{
			title: "Container Operations",
			items: []helpItem{
				{"Enter", "View Details"},
				{"l", "View Logs"},
				{"s", "Select Shell"},
				{"t", "Start Container"},
				{"p", "Stop Container"},
				{"R", "Restart Container"},
			},
		},
		{
			title: "Log Operations",
			items: []helpItem{
				{"f", "Toggle Follow Mode"},
				{"w", "Toggle Word Wrap"},
				{"j/k", "Scroll Up/Down"},
				{"g/G", "Go to Top/Bottom"},
			},
		},
	}
	
	// 渲染标题
	title := helpTitleStyle.Render("🆘 DockTUI Help (K9s Style)")
	
	// 渲染帮助表格
	var content strings.Builder
	
	for _, section := range sections {
		// 章节标题
		content.WriteString(helpHeaderStyle.Render("  " + section.title))
		content.WriteString("\n")
		
		// 章节内容
		for _, item := range section.items {
			key := helpKeyStyle.Render(item.key)
			desc := helpDescStyle.Render(item.desc)
			content.WriteString("    " + key)
			
			// 对齐描述（简单实现）
			padding := 20 - lipgloss.Width(item.key)
			if padding < 2 {
				padding = 2
			}
			content.WriteString(strings.Repeat(" ", padding))
			content.WriteString(desc)
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}
	
	table := helpTableStyle.Render(content.String())
	
	// 使用 bubbles/help 组件渲染快捷键详情
	helpDetail := lipgloss.NewStyle().
		MarginLeft(2).
		MarginTop(1).
		Render("📋 Shortcut Details:\n\n  " + v.help.View(v.keys))
	
	// 渲染页脚
	footer := helpFooterStyle.Render(
		"💡 Tip: Shortcuts follow vim conventions\n" +
		"📦 Repository: github.com/yourusername/docktui\n" +
		"📖 Version: v0.1.0\n\n" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render("Press ESC or b to go back"),
	)
	
	// 组合所有部分
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		table,
		helpDetail,
		footer,
		"",
	)
}

// SetSize 设置视图尺寸
func (v *HelpView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.help.Width = width - 4
}
