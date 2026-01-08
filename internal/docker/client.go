package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	sdk "github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
)

// Docker Endpoint 配置说明（Windows 环境）：
//
// 1. **本地 Docker Desktop（默认）**
//    - 不设置 DOCKER_HOST，SDK 会自动使用 Windows named pipe
//    - 默认地址：npipe:////./pipe/docker_engine
//    - 适用场景：本地开发、调试
//
// 2. **远程 Docker（TCP）**
//    - 设置 DOCKER_HOST=tcp://主机:2375  (无 TLS)
//    - 设置 DOCKER_HOST=tcp://主机:2376  (启用 TLS 时)
//    - 示例：set DOCKER_HOST=tcp://192.168.3.49:2375
//    - 适用场景：远程测试环境、多主机管理
//
// 3. **显式指定 Named Pipe**
//    - 设置 DOCKER_HOST=npipe:////./pipe/docker_engine
//    - 适用场景：明确指定本地 pipe 名称
//
// 4. **Unix Socket（Linux/macOS）**
//    - 设置 DOCKER_HOST=unix:///var/run/docker.sock
//    - Windows 下 WSL2 也可能使用该方式
//
// TLS 配置（需要时）：
//   - DOCKER_TLS_VERIFY=1
//   - DOCKER_CERT_PATH=C:\path\to\certs
//   - 证书文件：ca.pem, cert.pem, key.pem
//
// 验证方法：
//   - 运行 cmd/docker-version-demo 可以验证各种 endpoint 配置
//   - 该 demo 会打印 Docker 版本和系统信息

// Container 表示容器的基本信息（用于列表视图）
type Container struct {
	ID      string    // 容器 ID（完整）
	ShortID string    // 容器 ID（短，12位）
	Name    string    // 容器名称
	Image   string    // 镜像名称
	Command string    // 启动命令
	Created time.Time // 创建时间
	Status  string    // 状态描述，如 "Up 2 hours" 或 "Up 30 seconds (healthy)"
	State   string    // 状态: running, exited, paused 等
	Ports   string    // 端口映射
}

// ContainerDetails 表示容器的详细信息（用于详情视图）
type ContainerDetails struct {
	ID            string            // 容器 ID
	Name          string            // 容器名称
	Image         string            // 镜像名称
	State         string            // 状态
	Status        string            // 状态描述
	Created       time.Time         // 创建时间
	Ports         []PortMapping     // 端口映射
	Mounts        []MountInfo       // 挂载点
	Env           []string          // 环境变量
	Labels        map[string]string // 标签
	NetworkMode   string            // 网络模式
	RestartPolicy string            // 重启策略
}

// PortMapping 表示端口映射信息
type PortMapping struct {
	PrivatePort int    // 容器内端口
	PublicPort  int    // 宿主机端口
	Type        string // 协议类型: tcp, udp
	IP          string // 绑定 IP
}

// MountInfo 表示挂载信息
type MountInfo struct {
	Type        string // 类型: bind, volume, tmpfs
	Source      string // 源路径
	Destination string // 目标路径（容器内）
	Mode        string // 读写模式: rw, ro
}

// LogOptions 日志读取选项
type LogOptions struct {
	Follow     bool  // 是否持续跟随（类似 -f）
	Tail       int   // 获取最后 N 行，0 表示全部
	Timestamps bool  // 是否显示时间戳
	Since      string // 从某个时间开始
}

// ContainerEvent 表示 Docker 容器事件
type ContainerEvent struct {
	Action      string    // 事件类型: start, stop, die, create, destroy, rename 等
	ContainerID string    // 容器 ID
	ContainerName string  // 容器名称
	Timestamp   time.Time // 事件时间
}

// Client 抽象了 docktui 需要的 Docker 能力
// 这是一个接口，方便后续扩展（如远程 Docker、mock 测试等）
type Client interface {
	// Ping 验证 Docker 守护进程是否可用
	Ping(ctx context.Context) error

	// ListContainers 获取容器列表
	// showAll: true 显示所有容器（包括停止的），false 仅显示运行中的
	ListContainers(ctx context.Context, showAll bool) ([]Container, error)

	// ContainerDetails 获取指定容器的详细信息
	ContainerDetails(ctx context.Context, containerID string) (*ContainerDetails, error)

	// ContainerStats 获取容器资源使用统计
	ContainerStats(ctx context.Context, containerID string) (*ContainerStats, error)

	// ContainerLogs 获取容器日志
	// 返回一个 io.ReadCloser，调用方负责关闭
	ContainerLogs(ctx context.Context, containerID string, opts LogOptions) (io.ReadCloser, error)

	// ExecShell 在容器中启动交互式 shell
	// 返回错误或 nil，实际交互通过标准输入输出进行
	ExecShell(ctx context.Context, containerID string, shell string) error

	// GetAvailableShells 获取容器中所有可用的 shell 列表
	GetAvailableShells(ctx context.Context, containerID string) []string

	// WatchEvents 监听 Docker 容器事件
	// 返回事件通道和错误通道，调用方负责处理
	// context 用于控制监听的生命周期
	WatchEvents(ctx context.Context) (<-chan ContainerEvent, <-chan error)

	// StartContainer 启动已停止的容器
	StartContainer(ctx context.Context, containerID string) error

	// StopContainer 停止运行中的容器
	// timeout: 等待容器优雅停止的超时时间（秒），0 表示立即强制停止
	StopContainer(ctx context.Context, containerID string, timeout int) error

	// RestartContainer 重启容器
	// timeout: 等待容器停止的超时时间（秒）
	RestartContainer(ctx context.Context, containerID string, timeout int) error

	// RemoveContainer 删除容器
	// force: 是否强制删除（即使容器正在运行）
	// removeVolumes: 是否同时删除关联的匿名卷
	RemoveContainer(ctx context.Context, containerID string, force bool, removeVolumes bool) error

	// PauseContainer 暂停容器
	PauseContainer(ctx context.Context, containerID string) error

	// UnpauseContainer 恢复暂停的容器
	UnpauseContainer(ctx context.Context, containerID string) error

	// Close 关闭客户端连接，释放资源
	Close() error
}

// LocalClient 封装本地 Docker SDK 客户端实现。
type LocalClient struct {
	cli *sdk.Client
}

// NewLocalClientFromEnv 基于环境变量创建本地 Docker 客户端，并开启 API 版本协商。
func NewLocalClientFromEnv() (*LocalClient, error) {
	cli, err := sdk.NewClientWithOpts(
		sdk.FromEnv,
		sdk.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("创建 Docker 客户端失败: %w", err)
	}
	return &LocalClient{cli: cli}, nil
}

// Ping 用于验证 Docker 守护进程是否可用。
func (c *LocalClient) Ping(ctx context.Context) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}
	_, err := c.cli.Ping(ctx)
	return err
}

// ListContainers 获取容器列表
func (c *LocalClient) ListContainers(ctx context.Context, showAll bool) ([]Container, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker 客户端未初始化")
	}

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: showAll})
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %w", err)
	}

	result := make([]Container, 0, len(containers))
	for _, c := range containers {
		// 容器名称
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}

		// 短 ID（12位）
		shortID := c.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		
		// 命令（截断显示，和 docker ps 一样）
		command := c.Command
		if len(command) > 20 {
			command = "\"" + command[:17] + "…\""
		} else {
			command = "\"" + command + "\""
		}
		
		// 端口映射（完整格式，和 docker ps 一样）
		ports := formatPortsFull(c.Ports)

		result = append(result, Container{
			ID:      c.ID,
			ShortID: shortID,
			Name:    name,
			Image:   c.Image,
			Command: command,
			Created: time.Unix(c.Created, 0),
			Status:  c.Status,
			State:   string(c.State),
			Ports:   ports,
		})
	}

	return result, nil
}

// formatPortsFull 格式化端口映射（完整格式，和 docker ps 一样）
func formatPortsFull(ports []container.Port) string {
	if len(ports) == 0 {
		return ""
	}

	portStrs := make([]string, 0, len(ports))
	for _, p := range ports {
		if p.PublicPort > 0 {
			// 有端口映射：0.0.0.0:6379->6379/tcp
			ip := p.IP
			if ip == "" {
				ip = "0.0.0.0"
			}
			portStrs = append(portStrs, fmt.Sprintf("%s:%d->%d/%s", ip, p.PublicPort, p.PrivatePort, p.Type))
		} else {
			// 只暴露端口
			portStrs = append(portStrs, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
		}
	}

	return strings.Join(portStrs, ", ")
}

// ContainerDetails 获取指定容器的详细信息
func (c *LocalClient) ContainerDetails(ctx context.Context, containerID string) (*ContainerDetails, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker 客户端未初始化")
	}

	// 调用 Docker SDK 获取容器详细信息
	inspectResp, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("获取容器详情失败: %w", err)
	}

	// 提取容器信息（直接使用 InspectResponse）
	containerInfo := inspectResp

	// 提取容器名称（去除前导 /）
	name := containerInfo.Name
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}

	// 转换端口映射
	ports := make([]PortMapping, 0)
	if containerInfo.NetworkSettings != nil && containerInfo.NetworkSettings.Ports != nil {
		for portSpec, bindings := range containerInfo.NetworkSettings.Ports {
			for _, binding := range bindings {
				// 解析端口号，Port 格式如 "6379/tcp"
				var privatePort int
				var protocol string
				// 从 portSpec 中提取端口号和协议
				portStr := fmt.Sprintf("%v", portSpec)
				fmt.Sscanf(portStr, "%d/%s", &privatePort, &protocol)
				
				var publicPort int
				fmt.Sscanf(binding.HostPort, "%d", &publicPort)

				ports = append(ports, PortMapping{
					PrivatePort: privatePort,
					PublicPort:  publicPort,
					Type:        protocol,
					IP:          binding.HostIP,
				})
			}
		}
	}

	// 转换挂载信息
	mounts := make([]MountInfo, 0, len(containerInfo.Mounts))
	for _, m := range containerInfo.Mounts {
		mounts = append(mounts, MountInfo{
			Type:        string(m.Type),
			Source:      m.Source,
			Destination: m.Destination,
			Mode:        m.Mode,
		})
	}

	// 提取环境变量
	env := []string{}
	if containerInfo.Config != nil {
		env = containerInfo.Config.Env
	}

	// 提取标签
	labels := map[string]string{}
	if containerInfo.Config != nil && containerInfo.Config.Labels != nil {
		labels = containerInfo.Config.Labels
	}

	// 提取状态信息
	state := "unknown"
	status := ""
	if containerInfo.State != nil {
		state = string(containerInfo.State.Status)
		status = fmt.Sprintf("Started at: %v", containerInfo.State.StartedAt)
	}

	// 提取镜像信息
	image := ""
	if containerInfo.Config != nil {
		image = containerInfo.Config.Image
	}

	// 提取创建时间
	created, _ := time.Parse(time.RFC3339, containerInfo.Created)

	// 提取网络模式
	networkMode := "default"
	if containerInfo.HostConfig != nil && containerInfo.HostConfig.NetworkMode != "" {
		networkMode = string(containerInfo.HostConfig.NetworkMode)
	}

	// 提取重启策略
	restartPolicy := "no"
	if containerInfo.HostConfig != nil && containerInfo.HostConfig.RestartPolicy.Name != "" {
		restartPolicy = string(containerInfo.HostConfig.RestartPolicy.Name)
		if containerInfo.HostConfig.RestartPolicy.MaximumRetryCount > 0 {
			restartPolicy += fmt.Sprintf(":%d", containerInfo.HostConfig.RestartPolicy.MaximumRetryCount)
		}
	}

	return &ContainerDetails{
		ID:            containerInfo.ID,
		Name:          name,
		Image:         image,
		State:         state,
		Status:        status,
		Created:       created,
		Ports:         ports,
		Mounts:        mounts,
		Env:           env,
		Labels:        labels,
		NetworkMode:   networkMode,
		RestartPolicy: restartPolicy,
	}, nil
}

// ContainerLogs 获取容器日志
func (c *LocalClient) ContainerLogs(ctx context.Context, containerID string, opts LogOptions) (io.ReadCloser, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker 客户端未初始化")
	}

	// 构建 Docker SDK 日志选项
	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.Follow,
		Timestamps: opts.Timestamps,
	}

	// 处理 Tail 参数
	if opts.Tail > 0 {
		tailStr := fmt.Sprintf("%d", opts.Tail)
		logOpts.Tail = tailStr
	} else if opts.Tail == 0 {
		// 0 表示获取全部日志，不设置 Tail 即可
	}

	// 处理 Since 参数
	if opts.Since != "" {
		logOpts.Since = opts.Since
	}

	// 调用 Docker SDK 获取日志流
	logReader, err := c.cli.ContainerLogs(ctx, containerID, logOpts)
	if err != nil {
		return nil, fmt.Errorf("获取容器日志失败: %w", err)
	}

	return logReader, nil
}

// ExecShell 在容器中启动交互式 shell
// 注意：这是一个简化的实现，适用于基本的交互式场景
// 完整的终端支持（如终端大小调整、特殊键处理）需要额外的终端设置
// 详细实现请查看 exec.go 文件
// 实际实现在 exec.go 中的 ExecShell 方法

// Close 关闭 Docker 客户端连接
func (c *LocalClient) Close() error {
	if c == nil || c.cli == nil {
		return nil
	}
	return c.cli.Close()
}

// WatchEvents 监听 Docker 容器事件
func (c *LocalClient) WatchEvents(ctx context.Context) (<-chan ContainerEvent, <-chan error) {
	eventChan := make(chan ContainerEvent, 10)
	errorChan := make(chan error, 1)

	// 启动 goroutine 监听事件
	go func() {
		defer close(eventChan)
		defer close(errorChan)

		// 检查客户端是否初始化
		if c == nil || c.cli == nil {
			errorChan <- fmt.Errorf("Docker 客户端未初始化")
			return
		}

		// 设置事件过滤器，只关注容器事件
		filterArgs := filters.NewArgs()
		filterArgs.Add("type", "container")
		
		msgChan, errChan := c.cli.Events(ctx, events.ListOptions{
			Filters: filterArgs,
		})

		for {
			select {
			case <-ctx.Done():
				// Context 取消，退出监听
				return
			case err := <-errChan:
				if err != nil {
					errorChan <- fmt.Errorf("监听 Docker 事件失败: %w", err)
					return
				}
			case msg := <-msgChan:
				// 过滤容器相关事件
				if msg.Type == events.ContainerEventType {
					// 提取容器名称
					containerName := ""
					if name, ok := msg.Actor.Attributes["name"]; ok {
						containerName = name
					}

					// 只关注关键事件：start, stop, die, create, destroy, rename
					action := string(msg.Action)
					if action == "start" || action == "stop" || action == "die" || 
					   action == "create" || action == "destroy" || action == "rename" {
						event := ContainerEvent{
							Action:        action,
							ContainerID:   msg.Actor.ID,
							ContainerName: containerName,
							Timestamp:     time.Unix(msg.Time, msg.TimeNano),
						}
						select {
						case eventChan <- event:
							// 发送成功
						case <-ctx.Done():
							// Context 取消，退出
							return
						}
					}
				}
			}
		}
	}()

	return eventChan, errorChan
}

// StartContainer 启动已停止的容器
func (c *LocalClient) StartContainer(ctx context.Context, containerID string) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("启动容器失败: %w", err)
	}

	return nil
}

// StopContainer 停止运行中的容器
func (c *LocalClient) StopContainer(ctx context.Context, containerID string, timeout int) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	// 设置超时时间
	var timeoutPtr *int
	if timeout > 0 {
		timeoutPtr = &timeout
	}

	err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: timeoutPtr,
	})
	if err != nil {
		return fmt.Errorf("停止容器失败: %w", err)
	}

	return nil
}

// RestartContainer 重启容器
func (c *LocalClient) RestartContainer(ctx context.Context, containerID string, timeout int) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	// 设置超时时间
	var timeoutPtr *int
	if timeout > 0 {
		timeoutPtr = &timeout
	}

	err := c.cli.ContainerRestart(ctx, containerID, container.StopOptions{
		Timeout: timeoutPtr,
	})
	if err != nil {
		return fmt.Errorf("重启容器失败: %w", err)
	}

	return nil
}

// RemoveContainer 删除容器
func (c *LocalClient) RemoveContainer(ctx context.Context, containerID string, force bool, removeVolumes bool) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: removeVolumes,
	})
	if err != nil {
		return fmt.Errorf("删除容器失败: %w", err)
	}

	return nil
}

// PauseContainer 暂停容器
func (c *LocalClient) PauseContainer(ctx context.Context, containerID string) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	err := c.cli.ContainerPause(ctx, containerID)
	if err != nil {
		return fmt.Errorf("暂停容器失败: %w", err)
	}

	return nil
}

// UnpauseContainer 恢复暂停的容器
func (c *LocalClient) UnpauseContainer(ctx context.Context, containerID string) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	err := c.cli.ContainerUnpause(ctx, containerID)
	if err != nil {
		return fmt.Errorf("恢复容器失败: %w", err)
	}

	return nil
}
