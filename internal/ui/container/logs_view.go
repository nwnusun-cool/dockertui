package container

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/pkg/stdcopy"

	"docktui/internal/docker"
	"docktui/internal/ui/components"
	"docktui/internal/ui/search"
)

// LogsView 日志视图
type LogsView struct {
	dockerClient docker.Client
	
	width  int
	height int
	
	containerID   string
	containerName string
	
	logs       []string
	viewport   viewport.Model
	followMode bool
	wrapMode   bool
	showTimestamp bool
	loading    bool
	errorMsg   string
	successMsg string
	
	followCancel    context.CancelFunc
	followActive    bool
	lastRefreshTime time.Time
	lastLogTime     string
	logChan         chan string
	chanClosed      bool
	
	// 搜索相关
	searchMode   bool
	searchInput  textinput.Model
	searcher     *search.TextSearcher
	
	// 导出相关
	exportMode   bool
	exportInput  textinput.Model
	
	keys components.KeyMap
}

// NewLogsView 创建日志视图
func NewLogsView(dockerClient docker.Client) *LogsView {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		PaddingLeft(1).
		PaddingRight(1)
	
	// 搜索输入框
	ti := textinput.New()
	ti.Placeholder = ""
	ti.CharLimit = 100
	ti.Width = 30
	
	// 导出输入框
	ei := textinput.New()
	ei.Placeholder = ""
	ei.CharLimit = 200
	ei.Width = 50
	
	return &LogsView{
		dockerClient:  dockerClient,
		viewport:      vp,
		followMode:    false,
		wrapMode:      true,
		showTimestamp: false,
		keys:          components.DefaultKeyMap(),
		logChan:       make(chan string, 100),
		width:         100,
		height:        30,
		searchInput:   ti,
		searcher:      search.NewTextSearcher(),
		exportInput:   ei,
	}
}

// SetContainer 设置要查看日志的容器
func (v *LogsView) SetContainer(containerID, containerName string) {
	v.containerID = containerID
	v.containerName = containerName
}

// Init 初始化
func (v *LogsView) Init() tea.Cmd {
	if v.containerID == "" {
		return nil
	}
	v.loading = true
	v.followMode = true  // 默认开启跟随模式
	return v.loadLogs
}

// 消息类型定义
type logsLoadedMsg struct {
	logs []string
}

type logsLoadErrorMsg struct {
	err error
}

type followLogLineMsg struct {
	line string
}

type followStoppedMsg struct {
	err error
}

type followCheckMsg struct{}

type followRefreshMsg struct {
	logs []string
}

type followContinueMsg struct{}

// Update 处理消息
func (v *LogsView) Update(msg tea.Msg) (*LogsView, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case logsLoadedMsg:
		v.logs = msg.logs
		v.loading = false
		v.errorMsg = ""
		v.viewport.SetContent(v.formatLogs())
		v.viewport.GotoBottom()
		
		// 自动启动跟随模式
		if v.followMode && !v.followActive {
			if v.followCancel != nil {
				v.followCancel()
			}
			v.logChan = make(chan string, 100)
			v.chanClosed = false
			v.followActive = true
			return v, v.startStreamingLogs()
		}
		return v, nil
		
	case logsLoadErrorMsg:
		v.loading = false
		v.errorMsg = msg.err.Error()
		return v, nil
		
	case followLogLineMsg:
		if msg.line != "" {
			v.logs = append(v.logs, msg.line)
			v.lastRefreshTime = time.Now()
			if len(v.logs) > 1000 {
				v.logs = v.logs[len(v.logs)-1000:]
			}
			v.viewport.SetContent(v.formatLogs())
			v.viewport.GotoBottom()
		}
		if v.followMode && v.followActive {
			return v, v.listenForLogs()
		}
		return v, nil
		
	case followStoppedMsg:
		v.followActive = false
		if msg.err != nil {
			v.errorMsg = fmt.Sprintf("Follow stopped: %s", msg.err.Error())
		}
		return v, nil
		
	case followContinueMsg:
		if v.followMode && v.followActive {
			return v, v.listenForLogs()
		}
		return v, nil
		
	case followCheckMsg, followRefreshMsg:
		return v, nil
		
	case tea.KeyMsg:
		// 导出模式下的按键处理
		if v.exportMode {
			switch msg.String() {
			case "esc":
				v.exportMode = false
				v.exportInput.Blur()
				return v, nil
			case "enter":
				v.exportMode = false
				v.exportInput.Blur()
				filename := v.exportInput.Value()
				if filename != "" {
					if err := v.exportLogs(filename); err != nil {
						v.errorMsg = fmt.Sprintf("Export failed: %s", err.Error())
					} else {
						v.successMsg = fmt.Sprintf("Exported to: %s", filename)
					}
				}
				return v, nil
			default:
				v.exportInput, cmd = v.exportInput.Update(msg)
				return v, cmd
			}
		}
		
		// 搜索模式下的按键处理
		if v.searchMode {
			switch msg.String() {
			case "esc":
				v.searchMode = false
				v.searchInput.Blur()
				return v, nil
			case "enter":
				v.searchMode = false
				v.searchInput.Blur()
				query := v.searchInput.Value()
				if query != "" {
					v.searcher.Search(v.logs, query)
					v.viewport.SetContent(v.formatLogs())
					// 跳转到第一个匹配
					if match := v.searcher.Current(); match != nil {
						v.gotoLine(match.Line)
					}
				}
				return v, nil
			default:
				v.searchInput, cmd = v.searchInput.Update(msg)
				return v, cmd
			}
		}
		
		// 正常模式下的按键处理
		switch {
		case msg.String() == "esc":
			// 清除成功消息
			if v.successMsg != "" {
				v.successMsg = ""
				return v, nil
			}
			// 如果有搜索结果，先清除搜索
			if v.searcher.HasMatches() {
				v.searcher.Clear()
				v.searchInput.SetValue("")
				v.viewport.SetContent(v.formatLogs())
				return v, nil
			}
			return v, func() tea.Msg { return GoBackMsg{} }
		case msg.String() == "/":
			// 进入搜索模式
			v.searchMode = true
			v.searchInput.Focus()
			return v, textinput.Blink
		case msg.String() == "e", msg.String() == "S":
			// 进入导出模式
			v.exportMode = true
			v.successMsg = ""
			// 生成默认文件名
			name := strings.ReplaceAll(v.containerName, "/", "")
			defaultName := fmt.Sprintf("%s_%s.log", name, time.Now().Format("20060102_150405"))
			v.exportInput.SetValue(defaultName)
			v.exportInput.Focus()
			return v, textinput.Blink
		case msg.String() == "n":
			// 下一个匹配
			if match := v.searcher.Next(); match != nil {
				v.viewport.SetContent(v.formatLogs())
				v.gotoLine(match.Line)
			}
			return v, nil
		case msg.String() == "N":
			// 上一个匹配
			if match := v.searcher.Prev(); match != nil {
				v.viewport.SetContent(v.formatLogs())
				v.gotoLine(match.Line)
			}
			return v, nil
		case key.Matches(msg, v.keys.ToggleFollow):
			return v.toggleFollowMode()
		case key.Matches(msg, v.keys.ToggleWrap):
			v.wrapMode = !v.wrapMode
			v.viewport.SetContent(v.formatLogs())
			return v, nil
		case key.Matches(msg, v.keys.Refresh):
			v.loading = true
			v.errorMsg = ""
			if v.followActive && v.followCancel != nil {
				v.followCancel()
				v.followActive = false
				v.followMode = false
			}
			return v, v.loadLogs
		default:
			v.viewport, cmd = v.viewport.Update(msg)
			return v, cmd
		}
	}
	
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// gotoLine 跳转到指定行
func (v *LogsView) gotoLine(lineIdx int) {
	targetY := lineIdx
	if targetY < 0 {
		targetY = 0
	}
	v.viewport.SetYOffset(targetY)
}

// exportLogs 导出日志到文件
func (v *LogsView) exportLogs(filename string) error {
	content := strings.Join(v.logs, "\n")
	return os.WriteFile(filename, []byte(content), 0644)
}

// View 渲染视图
func (v *LogsView) View() string {
	var s strings.Builder
	
	s.WriteString(v.renderHeader())
	s.WriteString(v.renderStatusBar())
	
	if v.loading {
		s.WriteString(v.renderStateBox("⏳ Loading logs...", "Please wait, fetching container logs"))
		s.WriteString(v.renderKeyHints())
		return s.String()
	}
	
	if v.errorMsg != "" {
		s.WriteString(v.renderStateBox("❌ Load failed", v.truncate(v.errorMsg, 50)))
		s.WriteString(v.renderKeyHints())
		return s.String()
	}
	
	if len(v.logs) == 0 {
		s.WriteString(v.renderEmptyState())
		s.WriteString(v.renderKeyHints())
		return s.String()
	}
	
	s.WriteString("\n  " + v.viewport.View() + "\n")
	
	// 显示成功消息
	if v.successMsg != "" {
		s.WriteString(v.renderSuccessMsg())
	}
	
	// 导出模式显示输入框
	if v.exportMode {
		s.WriteString(v.renderExportBar())
	} else if v.searchMode {
		// 搜索模式下只显示搜索栏，隐藏快捷键提示
		s.WriteString(v.renderSearchBar())
	} else {
		if v.searcher.HasMatches() {
			s.WriteString(v.renderSearchBar())
		}
		s.WriteString(v.renderKeyHints())
	}
	
	return s.String()
}

// renderSearchBar 渲染搜索栏（底部风格）
func (v *LogsView) renderSearchBar() string {
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	matchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	
	availableWidth := v.width - 4
	if availableWidth < 80 {
		availableWidth = 80
	}
	
	divider := sepStyle.Render(strings.Repeat("─", availableWidth))
	
	var content string
	
	if v.searchMode {
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		content = promptStyle.Render("/") + v.searchInput.Value() + cursor +
			"  " + infoStyle.Render("[Enter=Confirm ESC=Cancel]")
	} else if v.searcher.HasMatches() {
		matchInfo := infoStyle.Render(fmt.Sprintf("[%d/%d]", v.searcher.CurrentIndex(), v.searcher.MatchCount()))
		content = promptStyle.Render("/"+v.searcher.Query()) + " " + matchStyle.Render(matchInfo) +
			"  " + hintStyle.Render("n=Next N=Prev ESC=Clear")
	}
	
	return "\n  " + divider + "\n  " + content + "\n"
}

// renderExportBar 渲染导出栏
func (v *LogsView) renderExportBar() string {
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	
	availableWidth := v.width - 4
	if availableWidth < 80 {
		availableWidth = 80
	}
	
	divider := sepStyle.Render(strings.Repeat("─", availableWidth))
	
	cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
	content := promptStyle.Render("📁 Export to: ") + v.exportInput.Value() + cursor +
		"  " + infoStyle.Render("[Enter=Confirm ESC=Cancel]")
	
	return "\n  " + divider + "\n  " + content + "\n"
}

// renderSuccessMsg 渲染成功消息
func (v *LogsView) renderSuccessMsg() string {
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	return "  " + successStyle.Render("✓ "+v.successMsg) + "\n"
}

// renderHeader 渲染标题栏
func (v *LogsView) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Padding(0, 1)
	
	title := titleStyle.Render("📜 Logs: " + v.containerName)
	
	lineWidth := v.width - 4
	if lineWidth < 60 {
		lineWidth = 60
	}
	line := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(strings.Repeat("─", lineWidth))
	
	return "\n  " + title + "\n  " + line + "\n"
}

// renderStatusBar 渲染状态栏
func (v *LogsView) renderStatusBar() string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	onStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	offStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	liveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	
	var followStatus string
	if v.followMode {
		if v.followActive {
			followStatus = liveStyle.Render("● LIVE")
		} else {
			followStatus = onStyle.Render("READY")
		}
	} else {
		followStatus = offStyle.Render("OFF")
	}
	
	var wrapStatus string
	if v.wrapMode {
		wrapStatus = onStyle.Render("ON")
	} else {
		wrapStatus = offStyle.Render("OFF")
	}
	
	sep := sepStyle.Render("  │  ")
	
	status := labelStyle.Render("Follow:") + " " + followStatus + sep +
		labelStyle.Render("Wrap:") + " " + wrapStatus + sep +
		labelStyle.Render("Lines:") + " " + valueStyle.Render(fmt.Sprintf("%d", len(v.logs)))
	
	if v.followMode && v.followActive && !v.lastRefreshTime.IsZero() {
		status += sep + offStyle.Render("Latest: "+v.lastRefreshTime.Format("15:04:05"))
	}
	
	return "\n  " + status + "\n"
}

// renderStateBox 渲染状态提示框
func (v *LogsView) renderStateBox(title, message string) string {
	boxWidth := v.width - 8
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 80 {
		boxWidth = 80
	}
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth).
		Align(lipgloss.Center)
	
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	
	content := titleStyle.Render(title) + "\n\n" + msgStyle.Render(message)
	
	return "\n  " + boxStyle.Render(content) + "\n"
}

// renderEmptyState 渲染空状态
func (v *LogsView) renderEmptyState() string {
	boxWidth := v.width - 8
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 80 {
		boxWidth = 80
	}
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth)
	
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	
	content := lipgloss.JoinVertical(lipgloss.Left,
		hintStyle.Render("📭 No logs"),
		"",
		titleStyle.Render("Possible reasons:"),
		hintStyle.Render("  • Container just started, no logs yet"),
		hintStyle.Render("  • Application not outputting to stdout/stderr"),
		hintStyle.Render("  • Logs have been cleared or rotated"),
		"",
		titleStyle.Render("Tips:"),
		hintStyle.Render("  • Press ")+keyStyle.Render("f")+hintStyle.Render(" to enable Follow mode"),
		hintStyle.Render("  • Press ")+keyStyle.Render("r")+hintStyle.Render(" to refresh"),
	)
	
	return "\n  " + boxStyle.Render(content) + "\n"
}

// renderKeyHints 渲染底部快捷键提示
func (v *LogsView) renderKeyHints() string {
	availableWidth := v.width - 4
	if availableWidth < 80 {
		availableWidth = 80
	}
	
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	
	items := []struct{ key, desc string }{
		{"j/k", "Scroll"},
		{"g/G", "Top/Bottom"},
		{"/", "Search"},
		{"e", "Export"},
		{"f", "Follow"},
		{"w", "Wrap"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	}
	
	var parts []string
	for _, item := range items {
		parts = append(parts, keyStyle.Render(item.key)+" "+descStyle.Render(item.desc))
	}
	
	sep := sepStyle.Render("  │  ")
	line := strings.Join(parts, sep)
	
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(strings.Repeat("─", availableWidth))
	
	return "\n  " + divider + "\n  " + line + "\n"
}

// formatLogs 格式化日志内容
func (v *LogsView) formatLogs() string {
	if len(v.logs) == 0 {
		return "No logs"
	}
	
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	highlightStyle := lipgloss.NewStyle().Background(lipgloss.Color("226")).Foreground(lipgloss.Color("0"))
	currentHighlightStyle := lipgloss.NewStyle().Background(lipgloss.Color("208")).Foreground(lipgloss.Color("0")).Bold(true)
	
	var formatted strings.Builder
	contentWidth := v.width - 12
	if contentWidth < 40 {
		contentWidth = 40
	}
	
	for i, line := range v.logs {
		formatted.WriteString(lineNumStyle.Render(fmt.Sprintf("%4d │ ", i+1)))
		
		// 基础样式
		var style lipgloss.Style
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "error") || strings.Contains(lineLower, "fatal") {
			style = errorStyle
		} else if strings.Contains(lineLower, "warn") {
			style = warnStyle
		} else if strings.Contains(lineLower, "info") {
			style = infoStyle
		} else {
			style = normalStyle
		}
		
		// 处理搜索高亮
		displayLine := line
		if v.searcher.HasMatches() && v.searcher.IsLineMatched(i) {
			isCurrentLine := v.searcher.IsCurrentMatchLine(i)
			displayLine = v.highlightLine(line, i, highlightStyle, currentHighlightStyle, isCurrentLine)
		} else {
			displayLine = style.Render(line)
		}
		
		if v.wrapMode && len(line) > contentWidth {
			// 换行模式下简化处理（不高亮，避免复杂度）
			for j := 0; j < len(line); j += contentWidth {
				end := j + contentWidth
				if end > len(line) {
					end = len(line)
				}
				if j == 0 {
					formatted.WriteString(style.Render(line[j:end]))
				} else {
					formatted.WriteString("\n" + lineNumStyle.Render("     │ ") + style.Render(line[j:end]))
				}
			}
		} else {
			formatted.WriteString(displayLine)
		}
		
		formatted.WriteString("\n")
	}
	
	return formatted.String()
}

// highlightLine 高亮行中的匹配文本
func (v *LogsView) highlightLine(line string, lineIdx int, hlStyle, currentHlStyle lipgloss.Style, isCurrentLine bool) string {
	matches := v.searcher.GetLineMatches(lineIdx)
	if len(matches) == 0 {
		return line
	}
	
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	
	var result strings.Builder
	lastEnd := 0
	
	for _, m := range matches {
		// 添加匹配前的文本
		if m.Column > lastEnd {
			result.WriteString(normalStyle.Render(line[lastEnd:m.Column]))
		}
		
		// 添加高亮的匹配文本
		end := m.Column + m.Length
		if end > len(line) {
			end = len(line)
		}
		
		// 当前匹配使用不同颜色
		if isCurrentLine && v.searcher.Current() != nil && 
		   v.searcher.Current().Line == lineIdx && v.searcher.Current().Column == m.Column {
			result.WriteString(currentHlStyle.Render(line[m.Column:end]))
		} else {
			result.WriteString(hlStyle.Render(line[m.Column:end]))
		}
		
		lastEnd = end
	}
	
	// 添加最后一个匹配后的文本
	if lastEnd < len(line) {
		result.WriteString(normalStyle.Render(line[lastEnd:]))
	}
	
	return result.String()
}

// truncate 截断字符串
func (v *LogsView) truncate(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// SetSize 设置视图尺寸
func (v *LogsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width - 6
	v.viewport.Height = height - 12
	if v.viewport.Height < 5 {
		v.viewport.Height = 5
	}
}

// processLogLine 处理日志行：提取时间戳并根据设置决定是否显示
func (v *LogsView) processLogLine(line string) string {
	if len(line) > 30 && line[0] >= '0' && line[0] <= '9' {
		if idx := strings.Index(line, " "); idx > 0 && idx < 35 {
			v.lastLogTime = line[:idx]
			if !v.showTimestamp && idx+1 < len(line) {
				return line[idx+1:]
			}
		}
	}
	return line
}

// loadLogs 加载容器日志
func (v *LogsView) loadLogs() tea.Msg {
	if v.containerID == "" {
		return logsLoadErrorMsg{err: fmt.Errorf("container ID is empty")}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	opts := docker.LogOptions{
		Follow:     false,
		Tail:       100,       // 只获取最近 100 行作为初始显示
		Timestamps: true,
	}
	
	logReader, err := v.dockerClient.ContainerLogs(ctx, v.containerID, opts)
	if err != nil {
		return logsLoadErrorMsg{err: err}
	}
	defer logReader.Close()
	
	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, logReader)
	if err != nil && err != io.EOF {
		return logsLoadErrorMsg{err: fmt.Errorf("failed to parse log stream: %w", err)}
	}
	
	var logs []string
	
	stdoutScanner := bufio.NewScanner(&stdout)
	buf := make([]byte, 0, 64*1024)
	stdoutScanner.Buffer(buf, 1024*1024)
	
	for stdoutScanner.Scan() {
		line := stdoutScanner.Text()
		line = v.processLogLine(line)
		logs = append(logs, line)
	}
	
	stderrScanner := bufio.NewScanner(&stderr)
	stderrScanner.Buffer(buf, 1024*1024)
	
	for stderrScanner.Scan() {
		line := stderrScanner.Text()
		line = v.processLogLine(line)
		logs = append(logs, line)
	}
	
	if err := stdoutScanner.Err(); err != nil {
		return logsLoadErrorMsg{err: fmt.Errorf("failed to read stdout: %w", err)}
	}
	
	if err := stderrScanner.Err(); err != nil {
		return logsLoadErrorMsg{err: fmt.Errorf("failed to read stderr: %w", err)}
	}
	
	return logsLoadedMsg{logs: logs}
}

// toggleFollowMode 切换 follow 模式
func (v *LogsView) toggleFollowMode() (*LogsView, tea.Cmd) {
	v.followMode = !v.followMode
	
	if v.followMode {
		if v.containerID != "" {
			if v.followCancel != nil {
				v.followCancel()
			}
			v.logChan = make(chan string, 100)
			v.chanClosed = false
			v.followActive = true
			v.viewport.GotoBottom()
			return v, v.startStreamingLogs()
		}
	} else {
		if v.followCancel != nil {
			v.followCancel()
			v.followCancel = nil
		}
		v.followActive = false
	}
	
	return v, nil
}

// startStreamingLogs 启动流式日志读取
func (v *LogsView) startStreamingLogs() tea.Cmd {
	return tea.Batch(
		v.listenForLogs(),
		v.readLogStream(),
	)
}

// listenForLogs 监听日志通道
func (v *LogsView) listenForLogs() tea.Cmd {
	return func() tea.Msg {
		select {
		case line := <-v.logChan:
			if line == "" {
				return followStoppedMsg{err: nil}
			}
			return followLogLineMsg{line: line}
		case <-time.After(100 * time.Millisecond):
			if v.followMode && v.followActive {
				return followContinueMsg{}
			}
			return followStoppedMsg{err: nil}
		}
	}
}

// readLogStream 读取日志流
func (v *LogsView) readLogStream() tea.Cmd {
	return func() tea.Msg {
		if v.containerID == "" {
			return followStoppedMsg{err: fmt.Errorf("container ID is empty")}
		}
		
		ctx, cancel := context.WithCancel(context.Background())
		v.followCancel = cancel
		v.chanClosed = false
		
		go func() {
			defer func() {
				if !v.chanClosed {
					v.chanClosed = true
					close(v.logChan)
				}
			}()
			
			opts := docker.LogOptions{
				Follow:     true,
				Tail:       0,
				Timestamps: true,
			}
			
			if v.lastLogTime != "" {
				opts.Since = v.lastLogTime
			}
			
			logReader, err := v.dockerClient.ContainerLogs(ctx, v.containerID, opts)
			if err != nil {
				return
			}
			defer logReader.Close()
			
			stdoutReader, stdoutWriter := io.Pipe()
			stderrReader, stderrWriter := io.Pipe()
			
			go func() {
				defer stdoutWriter.Close()
				defer stderrWriter.Close()
				stdcopy.StdCopy(stdoutWriter, stderrWriter, logReader)
			}()
			
			go func() {
				scanner := bufio.NewScanner(stdoutReader)
				buf := make([]byte, 0, 64*1024)
				scanner.Buffer(buf, 1024*1024)
				
				for scanner.Scan() {
					select {
					case <-ctx.Done():
						return
					default:
						line := scanner.Text()
						line = v.processLogLine(line)
						
						select {
						case v.logChan <- line:
						case <-ctx.Done():
							return
						}
					}
				}
			}()
			
			scanner := bufio.NewScanner(stderrReader)
			buf := make([]byte, 0, 64*1024)
			scanner.Buffer(buf, 1024*1024)
			
			for scanner.Scan() {
				select {
				case <-ctx.Done():
					return
				default:
					line := scanner.Text()
					line = v.processLogLine(line)
					
					select {
					case v.logChan <- line:
					case <-ctx.Done():
						return
					}
				}
			}
			
			select {
			case v.logChan <- "":
			case <-ctx.Done():
			}
		}()
		
		return nil
	}
}

// Cleanup 清理资源
func (v *LogsView) Cleanup() {
	if v.followCancel != nil {
		v.followCancel()
		v.followCancel = nil
	}
	v.followActive = false
	v.followMode = false
	v.logChan = make(chan string, 100)
	v.chanClosed = false
}
