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
	
	width  int
	height int
	
	// 当前统计数据
	currentStats *docker.ContainerStats
	
	// 历史数据（原始数据，1秒采样）
	cpuRawData    []DataPoint
	memoryRawData []DataPoint
	
	// 当前显示的数据（根据时间粒度聚合）
	cpuHistory    []float64
	memoryHistory []float64
	
	// 时间粒度
	granularity TimeGranularity
	
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
		cpuRawData:    make([]DataPoint, 0, 1800),    // 最多保存30分钟的原始数据（1秒采样）
		memoryRawData: make([]DataPoint, 0, 1800),
		cpuHistory:    make([]float64, 0, 60),
		memoryHistory: make([]float64, 0, 60),
		granularity:   Granularity1s,                 // 默认1秒粒度（显示最近1分钟）
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
	
	case tea.KeyMsg:
		// 处理时间粒度切换
		key := msg.String()
		switch key {
		case "1":
			v.setGranularity(Granularity1s)
			return nil
		case "2":
			v.setGranularity(Granularity5s)
			return nil
		case "3":
			v.setGranularity(Granularity10s)
			return nil
		case "4":
			v.setGranularity(Granularity30s)
			return nil
		}
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
	
	// 添加到原始数据（1秒采样）
	now := time.Now()
	v.cpuRawData = append(v.cpuRawData, DataPoint{
		Timestamp: now,
		Value:     stats.CPUPercent,
	})
	
	memoryMB := float64(stats.MemoryUsage) / 1024 / 1024
	v.memoryRawData = append(v.memoryRawData, DataPoint{
		Timestamp: now,
		Value:     memoryMB,
	})
	
	// 清理过期数据（保留最近30分钟，足够支持所有粒度）
	cutoff := now.Add(-30 * time.Minute)
	v.cpuRawData = v.cleanOldData(v.cpuRawData, cutoff)
	v.memoryRawData = v.cleanOldData(v.memoryRawData, cutoff)
	
	// 根据当前粒度聚合数据
	v.aggregateData()
}

// cleanOldData 清理过期数据
func (v *StatsView) cleanOldData(data []DataPoint, cutoff time.Time) []DataPoint {
	for i, point := range data {
		if point.Timestamp.After(cutoff) {
			return data[i:]
		}
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
		interval = 1 * time.Second
		maxPoints = 60 // 1分钟
		timeRange = "1分钟"
	case Granularity5s:
		interval = 5 * time.Second
		maxPoints = 60 // 5分钟
		timeRange = "5分钟"
	case Granularity10s:
		interval = 10 * time.Second
		maxPoints = 60 // 10分钟
		timeRange = "10分钟"
	case Granularity30s:
		interval = 30 * time.Second
		maxPoints = 60 // 30分钟
		timeRange = "30分钟"
	}
	
	// 聚合 CPU 数据
	v.cpuHistory = v.aggregateDataPoints(v.cpuRawData, interval, maxPoints)
	
	// 聚合内存数据
	v.memoryHistory = v.aggregateDataPoints(v.memoryRawData, interval, maxPoints)
	
	// 更新折线图
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
	if len(data) == 0 {
		return []float64{}
	}
	
	result := make([]float64, 0, maxPoints)
	now := time.Now()
	
	// 从最早的时间点开始
	startTime := now.Add(-time.Duration(maxPoints) * interval)
	
	for i := 0; i < maxPoints; i++ {
		bucketStart := startTime.Add(time.Duration(i) * interval)
		bucketEnd := bucketStart.Add(interval)
		
		// 收集该时间段内的所有数据点
		var sum float64
		var count int
		
		for _, point := range data {
			if point.Timestamp.After(bucketStart) && point.Timestamp.Before(bucketEnd) {
				sum += point.Value
				count++
			}
		}
		
		// 计算平均值
		if count > 0 {
			result = append(result, sum/float64(count))
		} else {
			// 没有数据点，使用0或前一个值
			if len(result) > 0 {
				result = append(result, result[len(result)-1])
			} else {
				result = append(result, 0)
			}
		}
	}
	
	return result
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
	
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
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
	
	// 第一行：当前数据
	line1 := labelStyle.Render("CPU: ") + cpuText + "    " +
		labelStyle.Render("内存: ") + memText + "    " +
		labelStyle.Render("进程数: ") + valueStyle.Render(fmt.Sprintf("%d", stats.PIDs))
	
	// 第二行：时间粒度选择
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
	
	content := line1 + "\n" + line2
	
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
