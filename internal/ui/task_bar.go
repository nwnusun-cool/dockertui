package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/task"
)

// TaskBar 底部任务进度条
type TaskBar struct {
	manager  *task.Manager
	expanded bool
	width    int
	events   <-chan task.Event
}

// 任务栏样式
var (
	taskBarLineStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	taskBarIconStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	taskBarNameStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	taskBarProgressStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	taskBarHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	taskBarErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	taskBarSuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	taskBarCancelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	taskBarBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)
)

// NewTaskBar 创建任务栏
func NewTaskBar() *TaskBar {
	manager := task.GetManager()
	return &TaskBar{
		manager:  manager,
		expanded: false,
		events:   manager.Subscribe(),
	}
}

// SetWidth 设置宽度
func (t *TaskBar) SetWidth(width int) {
	t.width = width
}

// Toggle 切换展开/收起
func (t *TaskBar) Toggle() {
	t.expanded = !t.expanded
}

// IsExpanded 是否展开
func (t *TaskBar) IsExpanded() bool {
	return t.expanded
}

// HasActiveTasks 是否有活跃任务
func (t *TaskBar) HasActiveTasks() bool {
	return len(t.manager.ListActiveTasks()) > 0
}

// CancelFirstTask 取消第一个活跃任务
func (t *TaskBar) CancelFirstTask() bool {
	tasks := t.manager.ListActiveTasks()
	if len(tasks) == 0 {
		return false
	}
	t.manager.Cancel(tasks[0].ID())
	return true
}

// CancelAllTasks 取消所有活跃任务
func (t *TaskBar) CancelAllTasks() int {
	tasks := t.manager.ListActiveTasks()
	for _, tsk := range tasks {
		t.manager.Cancel(tsk.ID())
	}
	return len(tasks)
}

// Update 处理消息
func (t *TaskBar) Update(msg tea.Msg) tea.Cmd {
	// 任务栏不直接处理按键，由父视图处理
	return nil
}

// View 渲染任务栏
func (t *TaskBar) View() string {
	tasks := t.manager.ListActiveTasks()
	if len(tasks) == 0 {
		return ""
	}

	width := t.width - 4
	if width < 60 {
		width = 60
	}

	if t.expanded {
		return t.renderExpanded(tasks, width)
	}
	return t.renderCollapsed(tasks, width)
}

// renderCollapsed 渲染收起状态
func (t *TaskBar) renderCollapsed(tasks []task.Task, width int) string {
	if len(tasks) == 0 {
		return ""
	}

	// 显示第一个任务的进度
	firstTask := tasks[0]
	progress := firstTask.Progress()
	message := firstTask.Message()

	// 进度条
	barWidth := 20
	filled := int(progress / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// 任务名称（截断）
	name := firstTask.Name()
	if len(name) > 25 {
		name = name[:22] + "..."
	}

	// 状态图标
	icon := "📥"
	if firstTask.Status() == task.StatusCompleted {
		icon = "✅"
	} else if firstTask.Status() == task.StatusFailed {
		icon = "❌"
	}

	// 组合
	line := fmt.Sprintf("%s %s %.0f%% [%s]",
		taskBarIconStyle.Render(icon),
		taskBarNameStyle.Render(name),
		progress,
		taskBarProgressStyle.Render(bar),
	)

	// 任务计数
	if len(tasks) > 1 {
		line += taskBarHintStyle.Render(fmt.Sprintf("  任务: %d", len(tasks)))
	}

	// 消息（如果有空间）
	if message != "" && len(line)+len(message) < width-10 {
		line += "  " + taskBarHintStyle.Render(message)
	}

	// 展开提示
	line += "  " + taskBarHintStyle.Render("[T=展开]") + " " + taskBarCancelStyle.Render("[x=取消]")

	// 分隔线
	separator := taskBarLineStyle.Render(strings.Repeat("─", width))

	return "\n" + separator + "\n  " + line
}

// renderExpanded 渲染展开状态
func (t *TaskBar) renderExpanded(tasks []task.Task, width int) string {
	var lines []string

	// 标题
	title := taskBarIconStyle.Render(fmt.Sprintf("后台任务 (%d)", len(tasks))) +
		"  " + taskBarHintStyle.Render("[T=收起]") + " " + taskBarCancelStyle.Render("[x=取消]")
	lines = append(lines, title)

	// 分隔线
	innerWidth := width - 6
	if innerWidth < 50 {
		innerWidth = 50
	}
	lines = append(lines, taskBarLineStyle.Render(strings.Repeat("─", innerWidth)))

	// 每个任务
	for _, tsk := range tasks {
		lines = append(lines, t.renderTaskDetail(tsk, innerWidth))
	}

	// 使用边框包裹
	content := strings.Join(lines, "\n")
	box := taskBarBoxStyle.Width(width - 2).Render(content)

	return "\n" + box
}

// renderTaskDetail 渲染单个任务详情
func (t *TaskBar) renderTaskDetail(tsk task.Task, width int) string {
	progress := tsk.Progress()
	message := tsk.Message()
	status := tsk.Status()

	// 状态图标
	icon := "📥"
	switch status {
	case task.StatusCompleted:
		icon = "✅"
	case task.StatusFailed:
		icon = "❌"
	case task.StatusCancelled:
		icon = "⏹️"
	case task.StatusPending:
		icon = "⏳"
	}

	// 进度条
	barWidth := 25
	filled := int(progress / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// 任务名称
	name := tsk.Name()
	if len(name) > 20 {
		name = name[:17] + "..."
	}

	// 主行
	line := fmt.Sprintf("%s %-20s %5.1f%% [%s]",
		taskBarIconStyle.Render(icon),
		taskBarNameStyle.Render(name),
		progress,
		taskBarProgressStyle.Render(bar),
	)

	// 消息
	if message != "" {
		msgStyle := taskBarHintStyle
		if status == task.StatusFailed {
			msgStyle = taskBarErrorStyle
		}
		line += "\n   └─ " + msgStyle.Render(truncateString(message, width-10))
	}

	return line
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// TaskEventMsg 任务事件消息（用于 Bubble Tea）
type TaskEventMsg struct {
	Event task.Event
}

// ListenForEvents 监听任务事件（返回 tea.Cmd）
func (t *TaskBar) ListenForEvents() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-t.events
		if !ok {
			return nil
		}
		return TaskEventMsg{Event: event}
	}
}
