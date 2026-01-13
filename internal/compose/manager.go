package compose

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	sdk "github.com/docker/docker/client"
)

// Manager 整合 Docker API 和 CLI 的 Compose 管理器
type Manager struct {
	dockerCli *sdk.Client
	cliClient Client    // CLI 客户端（用于 up/down/build 等复杂操作）
	discovery *Discovery // 项目发现器
	mu        sync.RWMutex
}

// NewManager 创建 Compose 管理器
func NewManager(dockerCli *sdk.Client) (*Manager, error) {
	// 创建 CLI 客户端
	cliClient, err := NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize compose CLI: %w", err)
	}

	return &Manager{
		dockerCli: dockerCli,
		cliClient: cliClient,
		discovery: NewDiscovery(dockerCli),
	}, nil
}

// ===== 项目发现（Docker API）=====

// DiscoverProjects 发现所有 Compose 项目
func (m *Manager) DiscoverProjects(ctx context.Context) ([]*Project, error) {
	return m.discovery.DiscoverProjects(ctx)
}

// GetProject 获取指定项目
func (m *Manager) GetProject(ctx context.Context, projectName string) (*Project, error) {
	return m.discovery.GetProject(ctx, projectName)
}

// RefreshProject 刷新项目状态
func (m *Manager) RefreshProject(ctx context.Context, projectName string) (*Project, error) {
	return m.discovery.RefreshProject(ctx, projectName)
}

// InvalidateCache 使缓存失效
func (m *Manager) InvalidateCache() {
	m.discovery.InvalidateCache()
}

// ===== 生命周期管理（CLI）=====

// Up 启动项目（使用 CLI）
func (m *Manager) Up(ctx context.Context, project *Project, opts UpOptions) (*OperationResult, error) {
	result, err := m.cliClient.Up(project, opts)
	if err == nil {
		m.discovery.InvalidateCache()
	}
	return result, err
}

// Down 停止项目（使用 CLI）
func (m *Manager) Down(ctx context.Context, project *Project, opts DownOptions) (*OperationResult, error) {
	result, err := m.cliClient.Down(project, opts)
	if err == nil {
		m.discovery.InvalidateCache()
	}
	return result, err
}

// Build 构建镜像（使用 CLI）
func (m *Manager) Build(ctx context.Context, project *Project, opts BuildOptions) (*OperationResult, error) {
	return m.cliClient.Build(project, opts)
}

// Pull 拉取镜像（使用 CLI）
func (m *Manager) Pull(ctx context.Context, project *Project, opts PullOptions) (*OperationResult, error) {
	return m.cliClient.Pull(project, opts)
}

// ===== 简单操作（Docker API，更快更可控）=====

// StartService 启动服务的所有容器
func (m *Manager) StartService(ctx context.Context, projectName, serviceName string) error {
	containers, err := m.discovery.GetServiceContainers(ctx, projectName, serviceName)
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		return fmt.Errorf("service %s has no containers", serviceName)
	}

	for _, c := range containers {
		if c.State != "running" {
			if err := m.dockerCli.ContainerStart(ctx, c.ID, container.StartOptions{}); err != nil {
				return fmt.Errorf("failed to start container %s: %w", c.ID[:12], err)
			}
		}
	}

	m.discovery.InvalidateCache()
	return nil
}

// StopService 停止服务的所有容器
func (m *Manager) StopService(ctx context.Context, projectName, serviceName string, timeout int) error {
	containers, err := m.discovery.GetServiceContainers(ctx, projectName, serviceName)
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		return fmt.Errorf("service %s has no containers", serviceName)
	}

	var timeoutPtr *int
	if timeout > 0 {
		timeoutPtr = &timeout
	}

	for _, c := range containers {
		if c.State == "running" {
			if err := m.dockerCli.ContainerStop(ctx, c.ID, container.StopOptions{Timeout: timeoutPtr}); err != nil {
				return fmt.Errorf("failed to stop container %s: %w", c.ID[:12], err)
			}
		}
	}

	m.discovery.InvalidateCache()
	return nil
}

// RestartService 重启服务的所有容器
func (m *Manager) RestartService(ctx context.Context, projectName, serviceName string, timeout int) error {
	containers, err := m.discovery.GetServiceContainers(ctx, projectName, serviceName)
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		return fmt.Errorf("service %s has no containers", serviceName)
	}

	var timeoutPtr *int
	if timeout > 0 {
		timeoutPtr = &timeout
	}

	for _, c := range containers {
		if err := m.dockerCli.ContainerRestart(ctx, c.ID, container.StopOptions{Timeout: timeoutPtr}); err != nil {
			return fmt.Errorf("failed to restart container %s: %w", c.ID[:12], err)
		}
	}

	m.discovery.InvalidateCache()
	return nil
}

// PauseService 暂停服务的所有容器
func (m *Manager) PauseService(ctx context.Context, projectName, serviceName string) error {
	containers, err := m.discovery.GetServiceContainers(ctx, projectName, serviceName)
	if err != nil {
		return err
	}

	for _, c := range containers {
		if c.State == "running" {
			if err := m.dockerCli.ContainerPause(ctx, c.ID); err != nil {
				return fmt.Errorf("failed to pause container %s: %w", c.ID[:12], err)
			}
		}
	}

	m.discovery.InvalidateCache()
	return nil
}

// UnpauseService 恢复服务的所有容器
func (m *Manager) UnpauseService(ctx context.Context, projectName, serviceName string) error {
	containers, err := m.discovery.GetServiceContainers(ctx, projectName, serviceName)
	if err != nil {
		return err
	}

	for _, c := range containers {
		if c.State == "paused" {
			if err := m.dockerCli.ContainerUnpause(ctx, c.ID); err != nil {
				return fmt.Errorf("failed to unpause container %s: %w", c.ID[:12], err)
			}
		}
	}

	m.discovery.InvalidateCache()
	return nil
}

// ===== 项目级别操作（Docker API）=====

// StartProject 启动项目的所有容器
func (m *Manager) StartProject(ctx context.Context, projectName string) error {
	containers, err := m.discovery.GetProjectContainers(ctx, projectName)
	if err != nil {
		return err
	}

	for _, c := range containers {
		if c.State != "running" {
			if err := m.dockerCli.ContainerStart(ctx, c.ID, container.StartOptions{}); err != nil {
				return fmt.Errorf("failed to start container %s: %w", c.ID[:12], err)
			}
		}
	}

	m.discovery.InvalidateCache()
	return nil
}

// StopProject 停止项目的所有容器
func (m *Manager) StopProject(ctx context.Context, projectName string, timeout int) error {
	containers, err := m.discovery.GetProjectContainers(ctx, projectName)
	if err != nil {
		return err
	}

	var timeoutPtr *int
	if timeout > 0 {
		timeoutPtr = &timeout
	}

	for _, c := range containers {
		if c.State == "running" {
			if err := m.dockerCli.ContainerStop(ctx, c.ID, container.StopOptions{Timeout: timeoutPtr}); err != nil {
				return fmt.Errorf("failed to stop container %s: %w", c.ID[:12], err)
			}
		}
	}

	m.discovery.InvalidateCache()
	return nil
}

// RestartProject 重启项目的所有容器
func (m *Manager) RestartProject(ctx context.Context, projectName string, timeout int) error {
	containers, err := m.discovery.GetProjectContainers(ctx, projectName)
	if err != nil {
		return err
	}

	var timeoutPtr *int
	if timeout > 0 {
		timeoutPtr = &timeout
	}

	for _, c := range containers {
		if err := m.dockerCli.ContainerRestart(ctx, c.ID, container.StopOptions{Timeout: timeoutPtr}); err != nil {
			return fmt.Errorf("failed to restart container %s: %w", c.ID[:12], err)
		}
	}

	m.discovery.InvalidateCache()
	return nil
}

// ===== 日志（Docker API）=====

// ServiceLogs 获取服务日志（流式）
func (m *Manager) ServiceLogs(ctx context.Context, projectName, serviceName string, opts LogOptions) (io.ReadCloser, error) {
	containers, err := m.discovery.GetServiceContainers(ctx, projectName, serviceName)
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("service %s has no containers", serviceName)
	}

	// 获取第一个容器的日志
	// TODO: 支持多容器日志合并
	return m.dockerCli.ContainerLogs(ctx, containers[0].ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.Follow,
		Timestamps: opts.Timestamps,
		Tail:       formatTail(opts.Tail),
		Since:      opts.Since,
		Until:      opts.Until,
	})
}

// ProjectLogs 获取项目日志（使用 CLI，支持多服务）
func (m *Manager) ProjectLogs(project *Project, opts LogOptions) (io.ReadCloser, error) {
	return m.cliClient.Logs(project, opts)
}

// ===== 配置查询（CLI）=====

// GetConfig 获取合并后的配置
func (m *Manager) GetConfig(project *Project) (string, error) {
	return m.cliClient.Config(project)
}

// GetPS 获取服务状态
func (m *Manager) GetPS(project *Project) ([]Service, error) {
	return m.cliClient.PS(project)
}

// ===== 工具方法 =====

// GetVersion 获取 compose 版本
func (m *Manager) GetVersion() (string, error) {
	return m.cliClient.Version()
}

// GetCommandType 获取命令类型
func (m *Manager) GetCommandType() string {
	return m.cliClient.CommandType()
}

// formatTail 格式化 tail 参数
func formatTail(tail int) string {
	if tail <= 0 {
		return "all"
	}
	return fmt.Sprintf("%d", tail)
}

// ===== 统计信息 =====

// ServiceStats 服务统计信息
type ServiceStats struct {
	ServiceName string
	Containers  []ContainerStats
}

// ContainerStats 容器统计信息
type ContainerStats struct {
	ContainerID   string
	ContainerName string
	CPUPercent    float64
	MemoryUsage   uint64
	MemoryLimit   uint64
	MemoryPercent float64
	NetworkRx     uint64
	NetworkTx     uint64
	BlockRead     uint64
	BlockWrite    uint64
	PIDs          uint64
}

// GetServiceStats 获取服务的资源统计
func (m *Manager) GetServiceStats(ctx context.Context, projectName, serviceName string) (*ServiceStats, error) {
	containers, err := m.discovery.GetServiceContainers(ctx, projectName, serviceName)
	if err != nil {
		return nil, err
	}

	stats := &ServiceStats{
		ServiceName: serviceName,
		Containers:  make([]ContainerStats, 0, len(containers)),
	}

	for _, c := range containers {
		if c.State != "running" {
			continue
		}

		// 获取容器统计
		resp, err := m.dockerCli.ContainerStats(ctx, c.ID, false)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		// 解析统计数据
		cs := parseContainerStats(c.ID, c.Names, resp.Body)
		if cs != nil {
			stats.Containers = append(stats.Containers, *cs)
		}
	}

	return stats, nil
}

// parseContainerStats 解析容器统计数据
func parseContainerStats(containerID string, names []string, body io.Reader) *ContainerStats {
	// 简化实现，实际需要解析 JSON 流
	name := ""
	if len(names) > 0 {
		name = names[0]
		if len(name) > 0 && name[0] == '/' {
			name = name[1:]
		}
	}

	return &ContainerStats{
		ContainerID:   containerID,
		ContainerName: name,
	}
}

// ===== 事件监听 =====

// ProjectEvent 项目事件
type ProjectEvent struct {
	ProjectName   string
	ServiceName   string
	ContainerID   string
	ContainerName string
	Action        string // start, stop, die, create, destroy
	Timestamp     time.Time
}

// WatchProjectEvents 监听项目事件
func (m *Manager) WatchProjectEvents(ctx context.Context, projectName string) (<-chan ProjectEvent, <-chan error) {
	eventChan := make(chan ProjectEvent, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(eventChan)
		defer close(errChan)

		// 使用 Docker 事件 API
		msgChan, errMsgChan := m.dockerCli.Events(ctx, events.ListOptions{})

		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errMsgChan:
				if err != nil {
					errChan <- err
				}
				return
			case msg := <-msgChan:
				// 检查是否是目标项目的事件
				if msg.Actor.Attributes[LabelProject] != projectName {
					continue
				}

				event := ProjectEvent{
					ProjectName:   projectName,
					ServiceName:   msg.Actor.Attributes[LabelService],
					ContainerID:   msg.Actor.ID,
					ContainerName: msg.Actor.Attributes["name"],
					Action:        string(msg.Action),
					Timestamp:     time.Unix(msg.Time, msg.TimeNano),
				}

				select {
				case eventChan <- event:
				case <-ctx.Done():
					return
				}

				// 事件发生后使缓存失效
				m.discovery.InvalidateCache()
			}
		}
	}()

	return eventChan, errChan
}
