package docker

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

// ContainerStats represents container resource usage statistics
type ContainerStats struct {
	Timestamp   time.Time // Collection time
	CPUPercent  float64   // CPU usage (%)
	MemoryUsage uint64    // Memory usage (bytes)
	MemoryLimit uint64    // Memory limit (bytes)
	MemoryPercent float64 // Memory usage percentage (%)
	NetworkRx   uint64    // Network received (bytes)
	NetworkTx   uint64    // Network transmitted (bytes)
	BlockRead   uint64    // Disk read (bytes)
	BlockWrite  uint64    // Disk write (bytes)
	PIDs        uint64    // Process count
}

// ContainerStats gets container resource usage statistics (single snapshot)
func (c *LocalClient) ContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	if c == nil || c.cli == nil {
		return nil, nil
	}

	// Get container stats (non-streaming, single snapshot)
	resp, err := c.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var statsJSON statsJSON
	if err := json.Unmarshal(body, &statsJSON); err != nil {
		return nil, err
	}

	// Calculate CPU usage
	cpuPercent := calculateCPUPercent(&statsJSON)

	// Calculate memory usage percentage
	memoryPercent := 0.0
	if statsJSON.MemoryStats.Limit > 0 {
		memoryPercent = float64(statsJSON.MemoryStats.Usage) / float64(statsJSON.MemoryStats.Limit) * 100
	}

	// Calculate network I/O
	var networkRx, networkTx uint64
	for _, net := range statsJSON.Networks {
		networkRx += net.RxBytes
		networkTx += net.TxBytes
	}

	// Calculate disk I/O
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

// statsJSON represents the JSON structure returned by Docker stats API
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

// calculateCPUPercent calculates CPU usage percentage
func calculateCPUPercent(stats *statsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent := (cpuDelta / systemDelta) * float64(stats.CPUStats.OnlineCPUs) * 100.0
		return cpuPercent
	}
	return 0.0
}
