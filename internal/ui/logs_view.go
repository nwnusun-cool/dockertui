package ui

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/pkg/stdcopy"

	"docktui/internal/docker"
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
	showTimestamp bool // 是否显示 Docker 时间戳
	loading    bool
	errorMsg   string
	
	followCancel    context.CancelFunc
	followActive    bool
	lastRefreshTime time.Time
	lastLogTime     string // 最后一条日志的时间戳，用于 Follow 模式
	logChan         chan string
	chanClosed      bool
	
	keys KeyMap
}

// NewLogsView 创建日志视图
func NewLogsView(dockerClient docker.Client) *LogsView {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		PaddingLeft(1).
		PaddingRight(1)
	
	return &LogsView{
		dockerClient:  dockerClient,
		viewport:      vp,
		followMode:    false,
		wrapMode:      true,
		showTimestamp: false, // 默认不显示 Docker 时间戳
		keys:          DefaultKeyMap(),
		logChan:       make(chan string, 100),
		width:         100,
		height:        30,
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
	return v.loadLogs
}

// Update 处理消息
func (v *LogsView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case logsLoadedMsg:
		v.logs = msg.logs
		v.loading = false
		v.errorMsg = ""
		v.viewport.SetContent(v.formatLogs())
		if v.followMode {
			v.viewport.GotoBottom()
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
			v.errorMsg = fmt.Sprintf("Follow 停止: %s", msg.err.Error())
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
		switch {
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

// View 渲染视图
func (v *LogsView) View() string {
	var s strings.Builder
	
	s.WriteString(v.renderHeader())
	s.WriteString(v.renderStatusBar())
	
	if v.loading {
		s.WriteString(v.renderStateBox("⏳ 正在加载日志...", "请稍候，正在获取容器日志"))
		s.WriteString(v.renderKeyHints())
		return s.String()
	}
	
	if v.errorMsg != "" {
		s.WriteString(v.renderStateBox("❌ 加载失败", v.truncate(v.errorMsg, 50)))
		s.WriteString(v.renderKeyHints())
		return s.String()
	}
	
	if len(v.logs) == 0 {
		s.WriteString(v.renderEmptyState())
		s.WriteString(v.renderKeyHints())
		return s.String()
	}
	
	s.WriteString("\n  " + v.viewport.View() + "\n")
	s.WriteString(v.renderKeyHints())
	
	return s.String()
}

// renderHeader 渲染标题栏
func (v *LogsView) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Padding(0, 1)
	
	title := titleStyle.Render("📜 日志: " + v.containerName)
	
	// 分隔线
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
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	onStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)
	
	offStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	liveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	
	// Follow 状态
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
	
	// Wrap 状态
	var wrapStatus string
	if v.wrapMode {
		wrapStatus = onStyle.Render("ON")
	} else {
		wrapStatus = offStyle.Render("OFF")
	}
	
	// 构建状态栏
	sep := sepStyle.Render("  │  ")
	
	status := labelStyle.Render("Follow:") + " " + followStatus + sep +
		labelStyle.Render("Wrap:") + " " + wrapStatus + sep +
		labelStyle.Render("Lines:") + " " + valueStyle.Render(fmt.Sprintf("%d", len(v.logs)))
	
	// 显示加载的日志范围
	if len(v.logs) > 0 {
		status += sep + offStyle.Render("显示最近 1000 行")
	}
	
	// 实时信息
	if v.followMode && v.followActive && !v.lastRefreshTime.IsZero() {
		status += sep + offStyle.Render("最新: "+v.lastRefreshTime.Format("15:04:05"))
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
	
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
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
	
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))
	
	content := lipgloss.JoinVertical(lipgloss.Left,
		hintStyle.Render("📭 暂无日志"),
		"",
		titleStyle.Render("可能的情况:"),
		hintStyle.Render("  • 容器刚启动，还没有产生日志"),
		hintStyle.Render("  • 应用程序没有输出到 stdout/stderr"),
		hintStyle.Render("  • 日志已被清空或轮转"),
		"",
		titleStyle.Render("操作提示:"),
		hintStyle.Render("  • 按 ")+keyStyle.Render("f")+hintStyle.Render(" 开启 Follow 模式"),
		hintStyle.Render("  • 按 ")+keyStyle.Render("r")+hintStyle.Render(" 刷新日志"),
	)
	
	return "\n  " + boxStyle.Render(content) + "\n"
}

// renderKeyHints 渲染底部快捷键提示
func (v *LogsView) renderKeyHints() string {
	availableWidth := v.width - 4
	if availableWidth < 80 {
		availableWidth = 80
	}
	
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))
	
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	
	items := []struct{ key, desc string }{
		{"j/k", "滚动"},
		{"g/G", "首/尾"},
		{"f", "Follow"},
		{"w", "换行"},
		{"r", "刷新"},
		{"Esc", "返回"},
		{"q", "退出"},
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
		return "暂无日志"
	}
	
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	
	var formatted strings.Builder
	contentWidth := v.width - 12
	if contentWidth < 40 {
		contentWidth = 40
	}
	
	for i, line := range v.logs {
		// 行号
		formatted.WriteString(lineNumStyle.Render(fmt.Sprintf("%4d │ ", i+1)))
		
		// 根据内容选择样式
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
		
		// 自动换行处理
		if v.wrapMode && len(line) > contentWidth {
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
			formatted.WriteString(style.Render(line))
		}
		
		formatted.WriteString("\n")
	}
	
	return formatted.String()
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

// processLogLine 处理日志行：提取时间戳并根据设置决定是否显示
func (v *LogsView) processLogLine(line string) string {
	// 提取时间戳（如果有）
	// 格式：2024-01-08T12:34:56.789012345Z actual log content
	if len(line) > 30 && line[0] >= '0' && line[0] <= '9' {
		if idx := strings.Index(line, " "); idx > 0 && idx < 35 {
			// 保存时间戳用于 Follow 功能
			v.lastLogTime = line[:idx]
			
			// 如果不显示时间戳，去掉它
			if !v.showTimestamp && idx+1 < len(line) {
				return line[idx+1:]
			}
		}
	}
	
	return line
}

// loadLogs 加载容器日志（初始加载，获取最近的日志）
func (v *LogsView) loadLogs() tea.Msg {
	if v.containerID == "" {
		return logsLoadErrorMsg{err: fmt.Errorf("容器 ID 为空")}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// 初始加载：获取最后 1000 行日志
	opts := docker.LogOptions{
		Follow:     false,
		Tail:       1000,
		Timestamps: true,  // 保留时间戳用于 Follow 功能
	}
	
	logReader, err := v.dockerClient.ContainerLogs(ctx, v.containerID, opts)
	if err != nil {
		return logsLoadErrorMsg{err: err}
	}
	defer logReader.Close()
	
	// 使用 stdcopy 解析 Docker 多路复用流
	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, logReader)
	if err != nil && err != io.EOF {
		return logsLoadErrorMsg{err: fmt.Errorf("解析日志流失败: %w", err)}
	}
	
	// 合并 stdout 和 stderr
	var logs []string
	
	// 处理 stdout
	stdoutScanner := bufio.NewScanner(&stdout)
	// 设置更大的缓冲区以处理长行
	buf := make([]byte, 0, 64*1024)
	stdoutScanner.Buffer(buf, 1024*1024) // 最大 1MB 的行
	
	for stdoutScanner.Scan() {
		line := stdoutScanner.Text()
		line = v.processLogLine(line)
		logs = append(logs, line)
	}
	
	// 处理 stderr
	stderrScanner := bufio.NewScanner(&stderr)
	stderrScanner.Buffer(buf, 1024*1024)
	
	for stderrScanner.Scan() {
		line := stderrScanner.Text()
		line = v.processLogLine(line)
		logs = append(logs, line)
	}
	
	if err := stdoutScanner.Err(); err != nil {
		return logsLoadErrorMsg{err: fmt.Errorf("读取 stdout 失败: %w", err)}
	}
	
	if err := stderrScanner.Err(); err != nil {
		return logsLoadErrorMsg{err: fmt.Errorf("读取 stderr 失败: %w", err)}
	}
	
	return logsLoadedMsg{logs: logs}
}

// toggleFollowMode 切换 follow 模式
func (v *LogsView) toggleFollowMode() (View, tea.Cmd) {
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

// readLogStream 读取日志流（Follow 模式：只获取新日志）
func (v *LogsView) readLogStream() tea.Cmd {
	return func() tea.Msg {
		if v.containerID == "" {
			return followStoppedMsg{err: fmt.Errorf("容器 ID 为空")}
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
				Tail:       0,        // 不获取历史日志
				Timestamps: true,     // 保留时间戳用于 Follow 功能
			}
			
			// 如果有最后一条日志的时间戳，使用 Since 参数只获取新日志
			if v.lastLogTime != "" {
				opts.Since = v.lastLogTime
			}
			
			logReader, err := v.dockerClient.ContainerLogs(ctx, v.containerID, opts)
			if err != nil {
				return
			}
			defer logReader.Close()
			
			// 使用 pipe 来处理 stdcopy
			stdoutReader, stdoutWriter := io.Pipe()
			stderrReader, stderrWriter := io.Pipe()
			
			// 启动 stdcopy 解析
			go func() {
				defer stdoutWriter.Close()
				defer stderrWriter.Close()
				stdcopy.StdCopy(stdoutWriter, stderrWriter, logReader)
			}()
			
			// 读取 stdout
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
			
			// 读取 stderr
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
