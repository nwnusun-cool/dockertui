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

// TimeGranularity 时间粒度
type TimeGranularity int

const (
	Granularity1s   TimeGranularity = iota // 1秒（最近1分钟，60个点）
	Granularity5s                          // 5秒（最近5分钟，60个点）
	Granularity10s                         // 10秒（最近10分钟，60个点）
	Granularity30s                         // 30秒（最近30分钟，60个点）
)

// DataPoint 数据点
type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

// StatsView 资源监控视图组件
type StatsView struct {
	dockerClient docker.Client
	containerID  string
	width, height int
	currentStats *docker.ContainerStats
	cpuRawData, memoryRawData []DataPoint
	cpuHistory, memoryHistory []float64
	granularity TimeGranularity
	cpuChart, memoryChart *Sparkline
	loading  bool
	errorMsg string
	active   bool
	lastNetworkRx, lastNetworkTx uint64
	lastBlockRead, lastBlockWrite uint64
	lastStatsTime time.Time
	networkRxRate, networkTxRate float64
	blockReadRate, blockWriteRate float64
}

// NewStatsView 创建资源监控视图
func NewStatsView(dockerClient docker.Client) *StatsView {
	return &StatsView{
		dockerClient:  dockerClient,
		cpuRawData:    make([]DataPoint, 0, 1800),
		memoryRawData: make([]DataPoint, 0, 1800),
		cpuHistory:    make([]float64, 0, 60),
		memoryHistory: make([]float64, 0, 60),
		granularity:   Granularity1s,
		cpuChart:      NewSparkline("CPU 使用率", 60, 8),
		memoryChart:   NewSparkline("内存使用", 60, 8),
	}
}

// SetContainer 设置容器
func (v *StatsView) SetContainer(containerID string) {
	v.containerID = containerID
	v.cpuRawData = make([]DataPoint, 0, 1800)
	v.memoryRawData = make([]DataPoint, 0, 1800)
	v.cpuHistory = make([]float64, 0, 60)
	v.memoryHistory = make([]float64, 0, 60)
	v.currentStats = nil
	v.lastStatsTime = time.Time{}
	v.granularity = Granularity1s
}

// SetSize 设置尺寸
func (v *StatsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	chartWidth := (width - 12) / 2
	if chartWidth < 30 { chartWidth = 30 }
	chartHeight := (height - 10) / 2
	if chartHeight < 6 { chartHeight = 6 }
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
func (v *StatsView) Stop() { v.active = false }

// StatsLoadedMsg 统计数据加载完成消息
type StatsLoadedMsg struct { Stats *docker.ContainerStats }

// StatsErrorMsg 统计数据加载错误消息
type StatsErrorMsg struct { Err error }

// StatsRefreshMsg 统计数据刷新消息
type StatsRefreshMsg struct{}

// Update 处理消息
func (v *StatsView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case StatsLoadedMsg:
		v.loading = false
		v.errorMsg = ""
		v.updateStats(msg.Stats)
		if v.active { return v.scheduleRefresh() }
		return nil
	case StatsErrorMsg:
		v.loading = false
		v.errorMsg = msg.Err.Error()
		if v.active { return v.scheduleRefresh() }
		return nil
	case StatsRefreshMsg:
		if v.active { return v.fetchStats }
		return nil
	case tea.KeyMsg:
		switch msg.String() {
		case "1": v.setGranularity(Granularity1s)
		case "2": v.setGranularity(Granularity5s)
		case "3": v.setGranularity(Granularity10s)
		case "4": v.setGranularity(Granularity30s)
		}
	}
	return nil
}

// updateStats 更新统计数据
func (v *StatsView) updateStats(stats *docker.ContainerStats) {
	if stats == nil { return }
	if !v.lastStatsTime.IsZero() {
		elapsed := time.Since(v.lastStatsTime).Seconds()
		if elapsed > 0 {
			v.networkRxRate = float64(stats.NetworkRx-v.lastNetworkRx) / elapsed
			v.networkTxRate = float64(stats.NetworkTx-v.lastNetworkTx) / elapsed
			v.blockReadRate = float64(stats.BlockRead-v.lastBlockRead) / elapsed
			v.blockWriteRate = float64(stats.BlockWrite-v.lastBlockWrite) / elapsed
		}
	}
	v.lastNetworkRx = stats.NetworkRx
	v.lastNetworkTx = stats.NetworkTx
	v.lastBlockRead = stats.BlockRead
	v.lastBlockWrite = stats.BlockWrite
	v.lastStatsTime = time.Now()
	v.currentStats = stats
	now := time.Now()
	v.cpuRawData = append(v.cpuRawData, DataPoint{Timestamp: now, Value: stats.CPUPercent})
	memoryMB := float64(stats.MemoryUsage) / 1024 / 1024
	v.memoryRawData = append(v.memoryRawData, DataPoint{Timestamp: now, Value: memoryMB})
	cutoff := now.Add(-30 * time.Minute)
	v.cpuRawData = v.cleanOldData(v.cpuRawData, cutoff)
	v.memoryRawData = v.cleanOldData(v.memoryRawData, cutoff)
	v.aggregateData()
}

// cleanOldData 清理过期数据
func (v *StatsView) cleanOldData(data []DataPoint, cutoff time.Time) []DataPoint {
	for i, point := range data {
		if point.Timestamp.After(cutoff) { return data[i:] }
	}
	return []DataPoint{}
}

// setGranularity 设置时间粒度
func (v *StatsView) setGranularity(g TimeGranularity) {
	v.granularity = g
	v.aggregateData()
}

// aggregateData 根据时间粒度聚合数据
func (v *StatsView) aggregateData() {
	var interval time.Duration
	var maxPoints int
	var timeRange string
	switch v.granularity {
	case Granularity1s:
		interval, maxPoints, timeRange = 1*time.Second, 60, "1分钟"
	case Granularity5s:
		interval, maxPoints, timeRange = 5*time.Second, 60, "5分钟"
	case Granularity10s:
		interval, maxPoints, timeRange = 10*time.Second, 60, "10分钟"
	case Granularity30s:
		interval, maxPoints, timeRange = 30*time.Second, 60, "30分钟"
	}
	v.cpuHistory = v.aggregateDataPoints(v.cpuRawData, interval, maxPoints)
	v.memoryHistory = v.aggregateDataPoints(v.memoryRawData, interval, maxPoints)
	v.cpuChart.SetData(v.cpuHistory)
	v.cpuChart.Max = 100
	v.cpuChart.Unit = "%"
	v.cpuChart.Color = "82"
	v.cpuChart.Title = fmt.Sprintf("CPU 使用率 (最近%s)", timeRange)
	if v.currentStats != nil {
		v.memoryChart.SetData(v.memoryHistory)
		v.memoryChart.Max = float64(v.currentStats.MemoryLimit) / 1024 / 1024
		v.memoryChart.Unit = "MB"
		v.memoryChart.Color = "81"
		v.memoryChart.Title = fmt.Sprintf("内存使用 (最近%s)", timeRange)
	}
}

// aggregateDataPoints 聚合数据点
func (v *StatsView) aggregateDataPoints(data []DataPoint, interval time.Duration, maxPoints int) []float64 {
	if len(data) == 0 { return []float64{} }
	result := make([]float64, 0, maxPoints)
	now := time.Now()
	startTime := now.Add(-time.Duration(maxPoints) * interval)
	for i := 0; i < maxPoints; i++ {
		bucketStart := startTime.Add(time.Duration(i) * interval)
		bucketEnd := bucketStart.Add(interval)
		var sum float64
		var count int
		for _, point := range data {
			if point.Timestamp.After(bucketStart) && point.Timestamp.Before(bucketEnd) {
				sum += point.Value
				count++
			}
		}
		if count > 0 {
			result = append(result, sum/float64(count))
		} else if len(result) > 0 {
			result = append(result, result[len(result)-1])
		} else {
			result = append(result, 0)
		}
	}
	return result
}

// Render 渲染视图
func (v *StatsView) Render() string {
	if v.loading && v.currentStats == nil { return v.renderLoading() }
	if v.errorMsg != "" && v.currentStats == nil { return v.renderError() }
	if v.currentStats == nil { return v.renderEmpty() }
	var s strings.Builder
	s.WriteString(v.renderSummary())
	s.WriteString("\n")
	s.WriteString(v.renderCharts())
	s.WriteString("\n")
	s.WriteString(v.renderIOInfo())
	return s.String()
}

// renderSummary 渲染顶部摘要
func (v *StatsView) renderSummary() string {
	stats := v.currentStats
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 2).Width(v.width - 8)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	cpuColor := "82"
	if stats.CPUPercent > 80 { cpuColor = "196" } else if stats.CPUPercent > 50 { cpuColor = "220" }
	memColor := "82"
	if stats.MemoryPercent > 80 { memColor = "196" } else if stats.MemoryPercent > 50 { memColor = "220" }
	cpuStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cpuColor)).Bold(true)
	memStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(memColor)).Bold(true)
	cpuText := cpuStyle.Render(fmt.Sprintf("%.1f%%", stats.CPUPercent))
	memUsed := FormatBytes(stats.MemoryUsage)
	memLimit := FormatBytes(stats.MemoryLimit)
	memText := memStyle.Render(fmt.Sprintf("%s / %s (%.1f%%)", memUsed, memLimit, stats.MemoryPercent))
	line1 := labelStyle.Render("CPU: ") + cpuText + "    " + labelStyle.Render("内存: ") + memText + "    " + labelStyle.Render("进程数: ") + valueStyle.Render(fmt.Sprintf("%d", stats.PIDs))
	granularityNames := []string{"1秒", "5秒", "10秒", "30秒"}
	var granularityHints []string
	for i, name := range granularityNames {
		if TimeGranularity(i) == v.granularity {
			granularityHints = append(granularityHints, labelStyle.Render(fmt.Sprintf("[%d] %s", i+1, name)))
		} else {
			granularityHints = append(granularityHints, hintStyle.Render(fmt.Sprintf("[%d] %s", i+1, name)))
		}
	}
	line2 := hintStyle.Render("时间粒度: ") + strings.Join(granularityHints, "  ")
	return "\n  " + boxStyle.Render(line1 + "\n" + line2)
}

// renderCharts 渲染折线图
func (v *StatsView) renderCharts() string {
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(1, 2)
	chartWidth := (v.width - 16) / 2
	if chartWidth < 30 { chartWidth = 30 }
	v.cpuChart.Width = chartWidth
	v.memoryChart.Width = chartWidth
	cpuBox := boxStyle.Width(chartWidth + 4).Render(v.cpuChart.Render())
	memBox := boxStyle.Width(chartWidth + 4).Render(v.memoryChart.Render())
	return "  " + lipgloss.JoinHorizontal(lipgloss.Top, cpuBox, "  ", memBox)
}

// renderIOInfo 渲染 I/O 信息
func (v *StatsView) renderIOInfo() string {
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 2).Width(v.width - 8)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	rxStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	txStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	netRx := rxStyle.Render("↓ " + FormatBytesRate(v.networkRxRate))
	netTx := txStyle.Render("↑ " + FormatBytesRate(v.networkTxRate))
	blockR := rxStyle.Render("R " + FormatBytes(v.currentStats.BlockRead))
	blockW := txStyle.Render("W " + FormatBytes(v.currentStats.BlockWrite))
	content := labelStyle.Render("网络 I/O: ") + netRx + "  " + netTx + "    " + labelStyle.Render("磁盘 I/O: ") + blockR + "  " + blockW
	return "  " + boxStyle.Render(content)
}

func (v *StatsView) renderLoading() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Align(lipgloss.Center).Width(v.width - 8)
	return "\n" + style.Render("⏳ 正在获取资源数据...")
}

func (v *StatsView) renderError() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Align(lipgloss.Center).Width(v.width - 8)
	return "\n" + style.Render("❌ " + v.errorMsg)
}

func (v *StatsView) renderEmpty() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Align(lipgloss.Center).Width(v.width - 8)
	return "\n" + style.Render("📊 等待数据...")
}

// fetchStats 获取统计数据
func (v *StatsView) fetchStats() tea.Msg {
	if v.containerID == "" { return StatsErrorMsg{Err: fmt.Errorf("容器 ID 为空")} }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stats, err := v.dockerClient.ContainerStats(ctx, v.containerID)
	if err != nil { return StatsErrorMsg{Err: err} }
	return StatsLoadedMsg{Stats: stats}
}

// scheduleRefresh 安排下次刷新
func (v *StatsView) scheduleRefresh() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return StatsRefreshMsg{} })
}

// FormatBytes 格式化字节数
func FormatBytes(bytes uint64) string {
	const (KB, MB, GB = 1024, 1024 * 1024, 1024 * 1024 * 1024)
	switch {
	case bytes >= GB: return fmt.Sprintf("%.1fGB", float64(bytes)/float64(GB))
	case bytes >= MB: return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	case bytes >= KB: return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default: return fmt.Sprintf("%dB", bytes)
	}
}

// FormatBytesRate 格式化字节速率
func FormatBytesRate(bytesPerSec float64) string {
	const (KB, MB = 1024.0, 1024.0 * 1024.0)
	switch {
	case bytesPerSec >= MB: return fmt.Sprintf("%.1fMB/s", bytesPerSec/MB)
	case bytesPerSec >= KB: return fmt.Sprintf("%.1fKB/s", bytesPerSec/KB)
	default: return fmt.Sprintf("%.0fB/s", bytesPerSec)
	}
}
