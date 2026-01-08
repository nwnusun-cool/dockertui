package docker

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

// ContainerStats 容器资源使用统计
type ContainerStats struct {
	Timestamp   time.Time // 采集时间
	CPUPercent  float64   // CPU 使用率 (%)
	MemoryUsage uint64    // 内存使用量 (bytes)
	MemoryLimit uint64    // 内存限制 (bytes)
	MemoryPercent float64 // 内存使用率 (%)
	NetworkRx   uint64    // 网络接收 (bytes)
	NetworkTx   uint64    // 网络发送 (bytes)
	BlockRead   uint64    // 磁盘读取 (bytes)
	BlockWrite  uint64    // 磁盘写入 (bytes)
	PIDs        uint64    // 进程数
}

// ContainerStats 获取容器资源使用统计（单次）
func (c *LocalClient) ContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	if c == nil || c.cli == nil {
		return nil, nil
	}

	// 获取容器统计信息（非流式，只获取一次）
	resp, err := c.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析 JSON
	var statsJSON statsJSON
	if err := json.Unmarshal(body, &statsJSON); err != nil {
		return nil, err
	}

	// 计算 CPU 使用率
	cpuPercent := calculateCPUPercent(&statsJSON)

	// 计算内存使用率
	memoryPercent := 0.0
	if statsJSON.MemoryStats.Limit > 0 {
		memoryPercent = float64(statsJSON.MemoryStats.Usage) / float64(statsJSON.MemoryStats.Limit) * 100
	}

	// 计算网络 I/O
	var networkRx, networkTx uint64
	for _, net := range statsJSON.Networks {
		networkRx += net.RxBytes
		networkTx += net.TxBytes
	}

	// 计算磁盘 I/O
	var blockRead, blockWrite uint64
	for _, bio := range statsJSON.BlkioStats.IoServiceBytesRecursive {
		switch bio.Op {
		case "Read", "read":
			blockRead += bio.Value
		case "Write", "write":
			blockWrite += bio.Value
		}
	}

	return &ContainerStats{
		Timestamp:     time.Now(),
		CPUPercent:    cpuPercent,
		MemoryUsage:   statsJSON.MemoryStats.Usage,
		MemoryLimit:   statsJSON.MemoryStats.Limit,
		MemoryPercent: memoryPercent,
		NetworkRx:     networkRx,
		NetworkTx:     networkTx,
		BlockRead:     blockRead,
		BlockWrite:    blockWrite,
		PIDs:          statsJSON.PidsStats.Current,
	}, nil
}

// statsJSON Docker stats API 返回的 JSON 结构
type statsJSON struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     uint64 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	BlkioStats struct {
		IoServiceBytesRecursive []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
	} `json:"blkio_stats"`
	PidsStats struct {
		Current uint64 `json:"current"`
	} `json:"pids_stats"`
}

// calculateCPUPercent 计算 CPU 使用率
func calculateCPUPercent(stats *statsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent := (cpuDelta / systemDelta) * float64(stats.CPUStats.OnlineCPUs) * 100.0
		return cpuPercent
	}
	return 0.0
}
