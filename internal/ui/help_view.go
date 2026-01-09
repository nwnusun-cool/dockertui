package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
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
	keys KeyMap
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
		keys:         DefaultKeyMap(),
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
			title: "全局快捷键",
			items: []helpItem{
				{"q / Ctrl+C", "退出程序"},
				{"?", "显示/隐藏帮助"},
				{"Esc", "返回上一级"},
				{"c", "直达容器列表"},
				{"i", "直达镜像列表"},
				{"n", "直达网络管理 (开发中)"},
				{"v", "直达卷管理 (开发中)"},
				{"o", "直达 Compose"},
			},
		},
		{
			title: "首页导航",
			items: []helpItem{
				{"↑/↓", "切换运行时/资源区域"},
				{"←/→", "选择运行时/资源"},
				{"1-5", "快速选择资源"},
				{"Enter", "进入选中的资源"},
				{"r", "刷新状态"},
			},
		},
		{
			title: "列表导航",
			items: []helpItem{
				{"j / ↓", "向下移动"},
				{"k / ↑", "向上移动"},
				{"g / Home", "跳到首行"},
				{"G / End", "跳到末尾"},
				{"/", "搜索"},
			},
		},
		{
			title: "容器操作",
			items: []helpItem{
				{"Enter", "查看容器详情"},
				{"l", "查看容器日志"},
				{"s", "选择 Shell 并进入"},
				{"t", "启动容器"},
				{"p", "停止容器"},
				{"R", "重启容器"},
			},
		},
		{
			title: "日志操作",
			items: []helpItem{
				{"f", "切换 Follow 模式"},
				{"w", "切换自动换行"},
				{"j/k", "上下滚动"},
				{"g/G", "跳到首尾"},
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
		Render("📋 快捷键详情:\n\n  " + v.help.View(v.keys))
	
	// 渲染页脚
	footer := helpFooterStyle.Render(
		"💡 提示：快捷键风格遵循 vim 习惯，降低学习成本\n" +
		"📦 项目地址：github.com/yourusername/docktui\n" +
		"📖 版本：v0.1.0\n\n" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render("按 ESC 或 b 返回"),
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
