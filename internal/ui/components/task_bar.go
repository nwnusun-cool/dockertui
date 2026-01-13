package components

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

func (t *TaskBar) renderCollapsed(tasks []task.Task, width int) string {
	if len(tasks) == 0 {
		return ""
	}

	firstTask := tasks[0]
	progress := firstTask.Progress()
	message := firstTask.Message()

	barWidth := 20
	filled := int(progress / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	name := firstTask.Name()
	if len(name) > 25 {
		name = name[:22] + "..."
	}

	icon := "📥"
	if firstTask.Status() == task.StatusCompleted {
		icon = "✅"
	} else if firstTask.Status() == task.StatusFailed {
		icon = "❌"
	}

	line := fmt.Sprintf("%s %s %.0f%% [%s]",
		taskBarIconStyle.Render(icon),
		taskBarNameStyle.Render(name),
		progress,
		taskBarProgressStyle.Render(bar),
	)

	if len(tasks) > 1 {
		line += taskBarHintStyle.Render(fmt.Sprintf("  Tasks: %d", len(tasks)))
	}

	if message != "" && len(line)+len(message) < width-10 {
		line += "  " + taskBarHintStyle.Render(message)
	}

	line += "  " + taskBarHintStyle.Render("[T=Expand]") + " " + taskBarCancelStyle.Render("[x=Cancel]")

	separator := taskBarLineStyle.Render(strings.Repeat("─", width))

	return "\n" + separator + "\n  " + line
}

func (t *TaskBar) renderExpanded(tasks []task.Task, width int) string {
	var lines []string

	title := taskBarIconStyle.Render(fmt.Sprintf("Background Tasks (%d)", len(tasks))) +
		"  " + taskBarHintStyle.Render("[T=Collapse]") + " " + taskBarCancelStyle.Render("[x=Cancel]")
	lines = append(lines, title)

	innerWidth := width - 6
	if innerWidth < 50 {
		innerWidth = 50
	}
	lines = append(lines, taskBarLineStyle.Render(strings.Repeat("─", innerWidth)))

	for _, tsk := range tasks {
		lines = append(lines, t.renderTaskDetail(tsk, innerWidth))
	}

	content := strings.Join(lines, "\n")
	box := taskBarBoxStyle.Width(width - 2).Render(content)

	return "\n" + box
}

func (t *TaskBar) renderTaskDetail(tsk task.Task, width int) string {
	progress := tsk.Progress()
	message := tsk.Message()
	status := tsk.Status()

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

	barWidth := 25
	filled := int(progress / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	name := tsk.Name()
	if len(name) > 20 {
		name = name[:17] + "..."
	}

	line := fmt.Sprintf("%s %-20s %5.1f%% [%s]",
		taskBarIconStyle.Render(icon),
		taskBarNameStyle.Render(name),
		progress,
		taskBarProgressStyle.Render(bar),
	)

	if message != "" {
		msgStyle := taskBarHintStyle
		if status == task.StatusFailed {
			msgStyle = taskBarErrorStyle
		}
		line += "\n   └─ " + msgStyle.Render(TruncateString(message, width-10))
	}

	return line
}

// TruncateString 截断字符串
func TruncateString(s string, maxLen int) string {
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
