package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
)

// ShellInfo Shell 信息
type ShellInfo struct {
	Path        string // Shell 路径，如 /bin/bash
	Name        string // 显示名称，如 bash
	Description string // 描述，如 "Bourne Again Shell"
	Available   bool   // 是否可用
}

// 预定义的 Shell 列表
var knownShells = []ShellInfo{
	{Path: "/bin/bash", Name: "bash", Description: "Bourne Again Shell"},
	{Path: "/bin/sh", Name: "sh", Description: "POSIX Shell"},
	{Path: "/bin/ash", Name: "ash", Description: "Alpine Shell"},
	{Path: "/bin/zsh", Name: "zsh", Description: "Z Shell"},
	{Path: "/bin/fish", Name: "fish", Description: "Friendly Interactive Shell"},
	{Path: "/bin/ksh", Name: "ksh", Description: "Korn Shell"},
}

// ShellSelector Shell 选择器组件
type ShellSelector struct {
	dockerClient docker.Client
	
	containerID   string
	containerName string
	
	shells       []ShellInfo // 可用的 Shell 列表
	selectedIdx  int         // 当前选中的索引
	loading      bool        // 是否正在加载
	errorMsg     string      // 错误信息
	
	width  int
	height int
	
	// 回调
	onSelect func(shell string) // 选择 Shell 后的回调
	onCancel func()             // 取消选择的回调
}

// NewShellSelector 创建 Shell 选择器
func NewShellSelector(dockerClient docker.Client) *ShellSelector {
	return &ShellSelector{
		dockerClient: dockerClient,
		shells:       []ShellInfo{},
		selectedIdx:  0,
		width:        60,
		height:       20,
	}
}

// SetContainer 设置容器
func (s *ShellSelector) SetContainer(containerID, containerName string) {
	s.containerID = containerID
	s.containerName = containerName
	s.shells = []ShellInfo{}
	s.selectedIdx = 0
	s.loading = true
	s.errorMsg = ""
}

// SetSize 设置尺寸
func (s *ShellSelector) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetCallbacks 设置回调函数
func (s *ShellSelector) SetCallbacks(onSelect func(string), onCancel func()) {
	s.onSelect = onSelect
	s.onCancel = onCancel
}

// Init 初始化，开始检测可用的 Shell
func (s *ShellSelector) Init() tea.Cmd {
	return s.detectShells
}

// shellsDetectedMsg Shell 检测完成消息
type shellsDetectedMsg struct {
	shells []ShellInfo
}

// shellsDetectErrorMsg Shell 检测错误消息
type shellsDetectErrorMsg struct {
	err error
}

// detectShells 检测容器中可用的 Shell
func (s *ShellSelector) detectShells() tea.Msg {
	if s.containerID == "" {
		return shellsDetectErrorMsg{err: fmt.Errorf("容器 ID 为空")}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// 获取可用的 Shell 列表
	availableShells := s.dockerClient.GetAvailableShells(ctx, s.containerID)
	
	// 构建 Shell 信息列表
	shells := make([]ShellInfo, 0)
	availableSet := make(map[string]bool)
	for _, shell := range availableShells {
		availableSet[shell] = true
	}
	
	// 按预定义顺序添加可用的 Shell
	for _, known := range knownShells {
		if availableSet[known.Path] {
			shell := known
			shell.Available = true
			shells = append(shells, shell)
		}
	}
	
	if len(shells) == 0 {
		return shellsDetectErrorMsg{err: fmt.Errorf("容器中没有可用的 Shell")}
	}
	
	return shellsDetectedMsg{shells: shells}
}

// Update 处理消息
func (s *ShellSelector) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case shellsDetectedMsg:
		s.loading = false
		s.shells = msg.shells
		s.selectedIdx = 0
		return nil
		
	case shellsDetectErrorMsg:
		s.loading = false
		s.errorMsg = msg.err.Error()
		return nil
		
	case tea.KeyMsg:
		if s.loading {
			return nil
		}
		
		switch msg.String() {
		case "up", "k":
			if s.selectedIdx > 0 {
				s.selectedIdx--
			}
		case "down", "j":
			if s.selectedIdx < len(s.shells)-1 {
				s.selectedIdx++
			}
		case "enter":
			if len(s.shells) > 0 && s.onSelect != nil {
				s.onSelect(s.shells[s.selectedIdx].Path)
			}
		case "esc", "q":
			if s.onCancel != nil {
				s.onCancel()
			}
		case "1", "2", "3", "4", "5", "6":
			// 数字快捷键选择
			idx := int(msg.String()[0] - '1')
			if idx >= 0 && idx < len(s.shells) {
				s.selectedIdx = idx
				if s.onSelect != nil {
					s.onSelect(s.shells[s.selectedIdx].Path)
				}
			}
		}
	}
	
	return nil
}

// View 渲染视图
func (s *ShellSelector) View() string {
	// 对话框样式
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(50)
	
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	var content strings.Builder
	
	// 标题
	content.WriteString(titleStyle.Render("🐚 选择 Shell"))
	content.WriteString("\n")
	content.WriteString(subtitleStyle.Render("容器: " + s.containerName))
	content.WriteString("\n\n")
	
	if s.loading {
		content.WriteString(subtitleStyle.Render("⏳ 正在检测可用的 Shell..."))
	} else if s.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		content.WriteString(errorStyle.Render("❌ " + s.errorMsg))
		content.WriteString("\n\n")
		content.WriteString(subtitleStyle.Render("按 Esc 返回"))
	} else {
		// Shell 列表
		for i, shell := range s.shells {
			var line string
			
			// 数字快捷键
			numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
			num := numStyle.Render(fmt.Sprintf("[%d]", i+1))
			
			// Shell 名称和描述
			if i == s.selectedIdx {
				// 选中状态
				selectedStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("220")).
					Bold(true)
				descStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("252"))
				
				line = fmt.Sprintf("%s %s %s %s",
					num,
					selectedStyle.Render("▶"),
					selectedStyle.Render(shell.Name),
					descStyle.Render("("+shell.Description+")"),
				)
			} else {
				// 未选中状态
				nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
				descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
				
				line = fmt.Sprintf("%s   %s %s",
					num,
					nameStyle.Render(shell.Name),
					descStyle.Render("("+shell.Description+")"),
				)
			}
			
			content.WriteString(line)
			content.WriteString("\n")
		}
		
		// 底部提示
		content.WriteString("\n")
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
		
		hints := []string{
			keyStyle.Render("↑/↓") + hintStyle.Render(" 选择"),
			keyStyle.Render("Enter") + hintStyle.Render(" 确认"),
			keyStyle.Render("1-6") + hintStyle.Render(" 快捷选择"),
			keyStyle.Render("Esc") + hintStyle.Render(" 取消"),
		}
		content.WriteString(hintStyle.Render(strings.Join(hints, "  ")))
	}
	
	dialog := dialogStyle.Render(content.String())
	
	// 居中显示
	return lipgloss.Place(
		s.width,
		s.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

// IsLoading 是否正在加载
func (s *ShellSelector) IsLoading() bool {
	return s.loading
}

// HasError 是否有错误
func (s *ShellSelector) HasError() bool {
	return s.errorMsg != ""
}

// GetSelectedShell 获取选中的 Shell
func (s *ShellSelector) GetSelectedShell() string {
	if len(s.shells) > 0 && s.selectedIdx < len(s.shells) {
		return s.shells[s.selectedIdx].Path
	}
	return ""
}
