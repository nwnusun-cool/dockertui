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

// StatsView 资源监控视图组件
type StatsView struct {
	dockerClient docker.Client
	containerID  string
	
	width  int
	height int
	
	// 当前统计数据
	currentStats *docker.ContainerStats
	
	// 历史数据（用于折线图）
	cpuHistory    []float64
	memoryHistory []float64
	
	// 折线图组件
	cpuChart    *Sparkline
	memoryChart *Sparkline
	
	// 状态
	loading  bool
	errorMsg string
	active   bool // 是否激活（用于控制定时刷新）
	
	// 上次网络/磁盘数据（用于计算速率）
	lastNetworkRx  uint64
	lastNetworkTx  uint64
	lastBlockRead  uint64
	lastBlockWrite uint64
	lastStatsTime  time.Time
	
	// 计算出的速率
	networkRxRate  float64 // bytes/s
	networkTxRate  float64
	blockReadRate  float64
	blockWriteRate float64
}

// NewStatsView 创建资源监控视图
func NewStatsView(dockerClient docker.Client) *StatsView {
	return &StatsView{
		dockerClient:  dockerClient,
		cpuHistory:    make([]float64, 0, 60),
		memoryHistory: make([]float64, 0, 60),
		cpuChart:      NewSparkline("CPU 使用率", 60, 8),
		memoryChart:   NewSparkline("内存使用", 60, 8),
	}
}

// SetContainer 设置容器
func (v *StatsView) SetContainer(containerID string) {
	v.containerID = containerID
	v.cpuHistory = make([]float64, 0, 60)
	v.memoryHistory = make([]float64, 0, 60)
	v.currentStats = nil
	v.lastStatsTime = time.Time{}
}

// SetSize 设置尺寸
func (v *StatsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	
	// 更新折线图尺寸
	chartWidth := (width - 12) / 2
	if chartWidth < 30 {
		chartWidth = 30
	}
	chartHeight := (height - 10) / 2
	if chartHeight < 6 {
		chartHeight = 6
	}
	
	v.cpuChart.Width = chartWidth
	v.cpuChart.Height = chartHeight
	v.memoryChart.Width = chartWidth
	v.memoryChart.Height = chartHeight
}

// Start 开始监控
func (v *StatsView) Start() tea.Cmd {
	v.active = true
	v.loading = true
	return v.fetchStats
}

// Stop 停止监控
func (v *StatsView) Stop() {
	v.active = false
}

// Update 处理消息
func (v *StatsView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case statsLoadedMsg:
		v.loading = false
		v.errorMsg = ""
		v.updateStats(msg.stats)
		
		// 继续定时刷新
		if v.active {
			return v.scheduleRefresh()
		}
		return nil
		
	case statsErrorMsg:
		v.loading = false
		v.errorMsg = msg.err.Error()
		
		// 即使出错也继续尝试
		if v.active {
			return v.scheduleRefresh()
		}
		return nil
		
	case statsRefreshMsg:
		if v.active {
			return v.fetchStats
		}
		return nil
	}
	
	return nil
}

// updateStats 更新统计数据
func (v *StatsView) updateStats(stats *docker.ContainerStats) {
	if stats == nil {
		return
	}
	
	// 计算速率
	if !v.lastStatsTime.IsZero() {
		elapsed := time.Since(v.lastStatsTime).Seconds()
		if elapsed > 0 {
			v.networkRxRate = float64(stats.NetworkRx-v.lastNetworkRx) / elapsed
			v.networkTxRate = float64(stats.NetworkTx-v.lastNetworkTx) / elapsed
			v.blockReadRate = float64(stats.BlockRead-v.lastBlockRead) / elapsed
			v.blockWriteRate = float64(stats.BlockWrite-v.lastBlockWrite) / elapsed
		}
	}
	
	// 保存当前值用于下次计算
	v.lastNetworkRx = stats.NetworkRx
	v.lastNetworkTx = stats.NetworkTx
	v.lastBlockRead = stats.BlockRead
	v.lastBlockWrite = stats.BlockWrite
	v.lastStatsTime = time.Now()
	
	// 更新当前数据
	v.currentStats = stats
	
	// 添加到历史数据
	v.cpuHistory = append(v.cpuHistory, stats.CPUPercent)
	if len(v.cpuHistory) > 60 {
		v.cpuHistory = v.cpuHistory[1:]
	}
	
	// 内存转换为 MB
	memoryMB := float64(stats.MemoryUsage) / 1024 / 1024
	v.memoryHistory = append(v.memoryHistory, memoryMB)
	if len(v.memoryHistory) > 60 {
		v.memoryHistory = v.memoryHistory[1:]
	}
	
	// 更新折线图数据
	v.cpuChart.SetData(v.cpuHistory)
	v.cpuChart.Max = 100
	v.cpuChart.Unit = "%"
	v.cpuChart.Color = "82" // 绿色
	
	v.memoryChart.SetData(v.memoryHistory)
	v.memoryChart.Max = float64(stats.MemoryLimit) / 1024 / 1024
	v.memoryChart.Unit = "MB"
	v.memoryChart.Color = "81" // 青色
}

// Render 渲染视图
func (v *StatsView) Render() string {
	if v.loading && v.currentStats == nil {
		return v.renderLoading()
	}
	
	if v.errorMsg != "" && v.currentStats == nil {
		return v.renderError()
	}
	
	if v.currentStats == nil {
		return v.renderEmpty()
	}
	
	var s strings.Builder
	
	// 顶部摘要
	s.WriteString(v.renderSummary())
	s.WriteString("\n")
	
	// 折线图区域
	s.WriteString(v.renderCharts())
	s.WriteString("\n")
	
	// 底部 I/O 信息
	s.WriteString(v.renderIOInfo())
	
	return s.String()
}

// renderSummary 渲染顶部摘要
func (v *StatsView) renderSummary() string {
	stats := v.currentStats
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 2).
		Width(v.width - 8)
	
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	cpuColor := "82"  // 绿色
	if stats.CPUPercent > 80 {
		cpuColor = "196" // 红色
	} else if stats.CPUPercent > 50 {
		cpuColor = "220" // 黄色
	}
	
	memColor := "82"
	if stats.MemoryPercent > 80 {
		memColor = "196"
	} else if stats.MemoryPercent > 50 {
		memColor = "220"
	}
	
	cpuStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cpuColor)).Bold(true)
	memStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(memColor)).Bold(true)
	
	cpuText := cpuStyle.Render(fmt.Sprintf("%.1f%%", stats.CPUPercent))
	memUsed := formatBytes(stats.MemoryUsage)
	memLimit := formatBytes(stats.MemoryLimit)
	memText := memStyle.Render(fmt.Sprintf("%s / %s (%.1f%%)", memUsed, memLimit, stats.MemoryPercent))
	
	content := labelStyle.Render("CPU: ") + cpuText + "    " +
		labelStyle.Render("内存: ") + memText + "    " +
		labelStyle.Render("进程数: ") + valueStyle.Render(fmt.Sprintf("%d", stats.PIDs))
	
	return "\n  " + boxStyle.Render(content)
}

// renderCharts 渲染折线图
func (v *StatsView) renderCharts() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)
	
	// 计算每个图表的宽度
	chartWidth := (v.width - 16) / 2
	if chartWidth < 30 {
		chartWidth = 30
	}
	
	v.cpuChart.Width = chartWidth
	v.memoryChart.Width = chartWidth
	
	cpuBox := boxStyle.Width(chartWidth + 4).Render(v.cpuChart.Render())
	memBox := boxStyle.Width(chartWidth + 4).Render(v.memoryChart.Render())
	
	// 水平排列两个图表
	return "  " + lipgloss.JoinHorizontal(lipgloss.Top, cpuBox, "  ", memBox)
}

// renderIOInfo 渲染 I/O 信息
func (v *StatsView) renderIOInfo() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 2).
		Width(v.width - 8)
	
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)
	
	rxStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	txStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	
	netRx := rxStyle.Render("↓ " + formatBytesRate(v.networkRxRate))
	netTx := txStyle.Render("↑ " + formatBytesRate(v.networkTxRate))
	
	blockR := rxStyle.Render("R " + formatBytes(v.currentStats.BlockRead))
	blockW := txStyle.Render("W " + formatBytes(v.currentStats.BlockWrite))
	
	content := labelStyle.Render("网络 I/O: ") + netRx + "  " + netTx + "    " +
		labelStyle.Render("磁盘 I/O: ") + blockR + "  " + blockW
	
	return "  " + boxStyle.Render(content)
}

// renderLoading 渲染加载状态
func (v *StatsView) renderLoading() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Align(lipgloss.Center).
		Width(v.width - 8)
	
	return "\n" + style.Render("⏳ 正在获取资源数据...")
}

// renderError 渲染错误状态
func (v *StatsView) renderError() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Align(lipgloss.Center).
		Width(v.width - 8)
	
	return "\n" + style.Render("❌ " + v.errorMsg)
}

// renderEmpty 渲染空状态
func (v *StatsView) renderEmpty() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Align(lipgloss.Center).
		Width(v.width - 8)
	
	return "\n" + style.Render("📊 等待数据...")
}

// 消息类型
type statsLoadedMsg struct {
	stats *docker.ContainerStats
}

type statsErrorMsg struct {
	err error
}

type statsRefreshMsg struct{}

// fetchStats 获取统计数据
func (v *StatsView) fetchStats() tea.Msg {
	if v.containerID == "" {
		return statsErrorMsg{err: fmt.Errorf("容器 ID 为空")}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	stats, err := v.dockerClient.ContainerStats(ctx, v.containerID)
	if err != nil {
		return statsErrorMsg{err: err}
	}
	
	return statsLoadedMsg{stats: stats}
}

// scheduleRefresh 安排下次刷新
func (v *StatsView) scheduleRefresh() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return statsRefreshMsg{}
	})
}

// formatBytes 格式化字节数
func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// formatBytesRate 格式化字节速率
func formatBytesRate(bytesPerSec float64) string {
	const (
		KB = 1024
		MB = KB * 1024
	)
	
	switch {
	case bytesPerSec >= MB:
		return fmt.Sprintf("%.1fMB/s", bytesPerSec/MB)
	case bytesPerSec >= KB:
		return fmt.Sprintf("%.1fKB/s", bytesPerSec/KB)
	default:
		return fmt.Sprintf("%.0fB/s", bytesPerSec)
	}
}
