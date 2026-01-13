package components

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
)

// ProcessesView 进程列表视图组件
type ProcessesView struct {
	dockerClient  docker.Client
	containerID   string
	width, height int
	processes     []docker.ProcessInfo
	loading       bool
	errorMsg      string
	active        bool
}

// NewProcessesView 创建进程列表视图
func NewProcessesView(dockerClient docker.Client) *ProcessesView {
	return &ProcessesView{
		dockerClient: dockerClient,
		processes:    make([]docker.ProcessInfo, 0),
	}
}

// SetContainer 设置容器
func (v *ProcessesView) SetContainer(containerID string) {
	v.containerID = containerID
	v.processes = make([]docker.ProcessInfo, 0)
	v.errorMsg = ""
}

// SetSize 设置尺寸
func (v *ProcessesView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Start 开始监控
func (v *ProcessesView) Start() tea.Cmd {
	v.active = true
	v.loading = true
	return v.fetchProcesses
}

// Stop 停止监控
func (v *ProcessesView) Stop() {
	v.active = false
}

// ProcessesLoadedMsg 进程数据加载完成消息
type ProcessesLoadedMsg struct {
	Processes []docker.ProcessInfo
}

// ProcessesErrorMsg 进程数据加载错误消息
type ProcessesErrorMsg struct {
	Err error
}

// ProcessesRefreshMsg 进程数据刷新消息
type ProcessesRefreshMsg struct{}

// Update 处理消息
func (v *ProcessesView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case ProcessesLoadedMsg:
		v.loading = false
		v.errorMsg = ""
		v.processes = msg.Processes
		if v.active {
			return v.scheduleRefresh()
		}
		return nil
	case ProcessesErrorMsg:
		v.loading = false
		v.errorMsg = msg.Err.Error()
		if v.active {
			return v.scheduleRefresh()
		}
		return nil
	case ProcessesRefreshMsg:
		if v.active {
			return v.fetchProcesses
		}
		return nil
	}
	return nil
}

// Render 渲染视图
func (v *ProcessesView) Render() string {
	if v.loading && len(v.processes) == 0 {
		return v.renderLoading()
	}
	if v.errorMsg != "" && len(v.processes) == 0 {
		return v.renderError()
	}
	if len(v.processes) == 0 {
		return v.renderEmpty()
	}
	return v.renderProcessTable()
}

// renderProcessTable 渲染进程表格
func (v *ProcessesView) renderProcessTable() string {
	boxWidth := v.width - 6
	if boxWidth < 80 {
		boxWidth = 80
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	pidStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	var lines []string

	// 表头 - 明确标注 PID 是宿主机的
	header := fmt.Sprintf("%-10s %-10s %-10s %-6s %-10s %s",
		"HOST PID", "HOST PPID", "USER", "CPU", "TIME", "COMMAND")
	lines = append(lines, headerStyle.Render(header))
	lines = append(lines, hintStyle.Render(strings.Repeat("─", boxWidth-8)))

	// 进程行
	for _, p := range v.processes {
		// PID 用高亮显示
		pidStr := pidStyle.Render(fmt.Sprintf("%-10s", p.PID))
		ppidStr := fmt.Sprintf("%-10s", p.PPID)
		line := pidStr + valueStyle.Render(fmt.Sprintf("%-10s %-10s %-6s %-10s %s",
			ppidStr, truncateStr(p.User, 10), p.CPU, p.Time, p.Command))
		lines = append(lines, line)
	}

	// 刷新提示
	lines = append(lines, "")
	lines = append(lines, hintStyle.Render(fmt.Sprintf("Total %d processes | PID/PPID are host process IDs | Auto-refresh every second", len(v.processes))))

	return "\n" + WrapInBox(fmt.Sprintf("Process List (%d)", len(v.processes)), strings.Join(lines, "\n"), boxWidth)
}

func (v *ProcessesView) renderLoading() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Align(lipgloss.Center).
		Width(v.width - 8)
	return "\n" + style.Render("⏳ Fetching process list...")
}

func (v *ProcessesView) renderError() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Align(lipgloss.Center).
		Width(v.width - 8)
	return "\n" + style.Render("❌ " + v.errorMsg)
}

func (v *ProcessesView) renderEmpty() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Align(lipgloss.Center).
		Width(v.width - 8)
	return "\n" + style.Render("📋 Waiting for data...")
}

// fetchProcesses 获取进程列表
func (v *ProcessesView) fetchProcesses() tea.Msg {
	if v.containerID == "" {
		return ProcessesErrorMsg{Err: fmt.Errorf("container ID is empty")}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	processes, err := v.dockerClient.ContainerTop(ctx, v.containerID)
	if err != nil {
		return ProcessesErrorMsg{Err: err}
	}

	return ProcessesLoadedMsg{Processes: processes}
}

// scheduleRefresh 安排下次刷新（每秒刷新）
func (v *ProcessesView) scheduleRefresh() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return ProcessesRefreshMsg{}
	})
}

// truncateStr 截断字符串
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
