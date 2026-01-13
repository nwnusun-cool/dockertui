package compose

import (
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// OperationLogView 操作日志视图组件
type OperationLogView struct {
	width  int
	height int

	visible   bool
	title     string
	logs      []string
	maxLines  int
	scrollPos int
	
	// 操作状态
	running   bool
	success   bool
	errorMsg  string
	startTime time.Time
	endTime   time.Time

	mu sync.Mutex
}

// NewOperationLogView 创建操作日志视图
func NewOperationLogView() *OperationLogView {
	return &OperationLogView{
		maxLines: 500,
		logs:     make([]string, 0),
	}
}

// 样式定义
var (
	logBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	logTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	logContentStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	logSuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)

	logErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	logRunningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Bold(true)

	logHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
)

// Show 显示日志视图
func (v *OperationLogView) Show(title string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.visible = true
	v.title = title
	v.logs = make([]string, 0)
	v.scrollPos = 0
	v.running = true
	v.success = false
	v.errorMsg = ""
	v.startTime = time.Now()
}

// Hide 隐藏日志视图
func (v *OperationLogView) Hide() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.visible = false
}

// IsVisible 是否可见
func (v *OperationLogView) IsVisible() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.visible
}

// IsRunning 是否正在运行
func (v *OperationLogView) IsRunning() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.running
}

// AppendLog 追加日志行
func (v *OperationLogView) AppendLog(line string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.logs = append(v.logs, line)
	
	// 限制最大行数
	if len(v.logs) > v.maxLines {
		v.logs = v.logs[len(v.logs)-v.maxLines:]
	}
	
	// 自动滚动到底部
	v.scrollToBottom()
}

// SetComplete 设置操作完成
func (v *OperationLogView) SetComplete(success bool, errMsg string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.running = false
	v.success = success
	v.errorMsg = errMsg
	v.endTime = time.Now()
}

// SetSize 设置尺寸
func (v *OperationLogView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Update 处理按键
func (v *OperationLogView) Update(msg tea.KeyMsg) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.visible {
		return false
	}

	switch msg.String() {
	case "esc", "q":
		// 只有操作完成后才能关闭
		if !v.running {
			v.visible = false
			return true
		}
		return true
	case "j", "down":
		v.scrollDown(1)
		return true
	case "k", "up":
		v.scrollUp(1)
		return true
	case "g":
		v.scrollPos = 0
		return true
	case "G":
		v.scrollToBottom()
		return true
	case "ctrl+d", "pgdown":
		v.scrollDown(10)
		return true
	case "ctrl+u", "pgup":
		v.scrollUp(10)
		return true
	}

	return true
}

func (v *OperationLogView) scrollUp(n int) {
	v.scrollPos -= n
	if v.scrollPos < 0 {
		v.scrollPos = 0
	}
}

func (v *OperationLogView) scrollDown(n int) {
	maxScroll := v.getMaxScroll()
	v.scrollPos += n
	if v.scrollPos > maxScroll {
		v.scrollPos = maxScroll
	}
}

func (v *OperationLogView) scrollToBottom() {
	v.scrollPos = v.getMaxScroll()
}

func (v *OperationLogView) getMaxScroll() int {
	visibleLines := v.getVisibleLines()
	maxScroll := len(v.logs) - visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func (v *OperationLogView) getVisibleLines() int {
	// 减去边框、标题、状态栏、提示的高度
	lines := v.height - 8
	if lines < 5 {
		lines = 5
	}
	return lines
}

// View 渲染视图
func (v *OperationLogView) View() string {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.visible {
		return ""
	}

	boxWidth := v.width - 4
	if boxWidth < 60 {
		boxWidth = 60
	}
	if boxWidth > v.width-4 {
		boxWidth = v.width - 4
	}

	// 标题
	var statusIcon string
	var statusStyle lipgloss.Style
	if v.running {
		statusIcon = "⏳"
		statusStyle = logRunningStyle
	} else if v.success {
		statusIcon = "✅"
		statusStyle = logSuccessStyle
	} else {
		statusIcon = "❌"
		statusStyle = logErrorStyle
	}

	title := logTitleStyle.Render(statusIcon + " " + v.title)

	// 状态信息
	var statusLine string
	if v.running {
		elapsed := time.Since(v.startTime).Round(time.Second)
		statusLine = statusStyle.Render("Running... ") + logHintStyle.Render(elapsed.String())
	} else {
		duration := v.endTime.Sub(v.startTime).Round(time.Second)
		if v.success {
			statusLine = statusStyle.Render("Completed ") + logHintStyle.Render("("+duration.String()+")")
		} else {
			statusLine = statusStyle.Render("Failed ") + logHintStyle.Render("("+duration.String()+")")
			if v.errorMsg != "" {
				statusLine += "\n" + logErrorStyle.Render(v.errorMsg)
			}
		}
	}

	// 日志内容
	visibleLines := v.getVisibleLines()
	contentWidth := boxWidth - 4

	var logLines []string
	startIdx := v.scrollPos
	endIdx := startIdx + visibleLines
	if endIdx > len(v.logs) {
		endIdx = len(v.logs)
	}
	if startIdx < len(v.logs) {
		logLines = v.logs[startIdx:endIdx]
	}

	// 截断过长的行
	for i, line := range logLines {
		if len(line) > contentWidth {
			logLines[i] = line[:contentWidth-3] + "..."
		}
	}

	// 填充空行
	for len(logLines) < visibleLines {
		logLines = append(logLines, "")
	}

	logContent := strings.Join(logLines, "\n")

	// 滚动指示器
	var scrollInfo string
	if len(v.logs) > visibleLines {
		scrollInfo = logHintStyle.Render(
			" [" + string(rune('0'+startIdx+1)) + "-" + 
			string(rune('0'+endIdx)) + "/" + 
			string(rune('0'+len(v.logs))) + "]",
		)
		// 使用 fmt.Sprintf 更准确
		scrollInfo = logHintStyle.Render(
			" [Lines: " + itoa(startIdx+1) + "-" + itoa(endIdx) + "/" + itoa(len(v.logs)) + "]",
		)
	}

	// 提示
	var hint string
	if v.running {
		hint = logHintStyle.Render("j/k=Scroll  g/G=Top/Bottom")
	} else {
		hint = logHintStyle.Render("j/k=Scroll  g/G=Top/Bottom  Esc/q=Close")
	}

	// 组装
	content := logContentStyle.Render(logContent)
	box := logBoxStyle.Width(boxWidth).Render(content)

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		" "+title+scrollInfo,
		" "+statusLine,
		box,
		" "+hint,
	)
}

// Overlay 将日志视图叠加到基础内容上
func (v *OperationLogView) Overlay(baseContent string) string {
	if !v.IsVisible() {
		return baseContent
	}

	logView := v.View()
	
	// 计算居中位置
	baseLines := strings.Split(baseContent, "\n")
	logLines := strings.Split(logView, "\n")

	// 垂直居中
	startY := (len(baseLines) - len(logLines)) / 2
	if startY < 0 {
		startY = 0
	}

	// 水平居中
	logWidth := 0
	for _, line := range logLines {
		if len(line) > logWidth {
			logWidth = len(line)
		}
	}
	startX := (v.width - logWidth) / 2
	if startX < 0 {
		startX = 0
	}

	// 叠加
	result := make([]string, len(baseLines))
	copy(result, baseLines)

	for i, logLine := range logLines {
		targetY := startY + i
		if targetY >= 0 && targetY < len(result) {
			// 在该行插入日志内容
			baseLine := result[targetY]
			padding := strings.Repeat(" ", startX)
			
			// 确保基础行足够长
			for len(baseLine) < startX {
				baseLine += " "
			}
			
			// 替换部分内容
			if startX < len(baseLine) {
				result[targetY] = baseLine[:startX] + logLine
			} else {
				result[targetY] = padding + logLine
			}
		}
	}

	return strings.Join(result, "\n")
}

// itoa 简单的整数转字符串
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	
	negative := n < 0
	if negative {
		n = -n
	}
	
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	
	return string(digits)
}
