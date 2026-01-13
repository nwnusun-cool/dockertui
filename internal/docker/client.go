package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	sdk "github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"

	"docktui/internal/docker/image"
	"docktui/internal/docker/network"
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

// ===== 镜像类型别名（委托给 image 包）=====
// 保持向后兼容，现有代码无需修改

// Image 表示镜像的基本信息（用于列表视图）
type Image = image.Image

// ContainerRef 表示容器引用信息
type ContainerRef = image.ContainerRef

// ImageDetails 表示镜像的详细信息（用于详情视图）
type ImageDetails = image.Details

// ImageHistory 表示镜像构建历史的一条记录
type ImageHistory = image.History

// ===== 网络类型别名（委托给 network 包）=====

// Network 表示网络的基本信息（用于列表视图）
type Network = network.Network

// NetworkDetails 表示网络的详细信息（用于详情视图）
type NetworkDetails = network.Details

// NetworkIPAMConfig IP 地址管理配置
type NetworkIPAMConfig = network.IPAMConfig

// NetworkIPAMPoolConfig IP 池配置
type NetworkIPAMPoolConfig = network.IPAMPoolConfig

// NetworkContainerEndpoint 容器在网络中的端点信息
type NetworkContainerEndpoint = network.ContainerEndpoint

// NetworkCreateOptions 创建网络的选项
type NetworkCreateOptions = network.CreateOptions

// NetworkConnectOptions 连接容器到网络的选项
type NetworkConnectOptions = network.ConnectOptions

// NetworkDisconnectOptions 断开容器与网络连接的选项
type NetworkDisconnectOptions = network.DisconnectOptions

// ContainerUpdateConfig 容器更新配置
// 注意：CPU/内存限制仅在 Linux 原生 Docker 或 WSL2 后端支持
type ContainerUpdateConfig struct {
	// 重启策略
	RestartPolicy     string // no, always, on-failure, unless-stopped
	RestartMaxRetries int    // on-failure 时的最大重试次数

	// CPU 限制（仅 Linux）
	CPUShares int64 // CPU 份额（相对权重，默认 1024）
	NanoCPUs  int64 // CPU 限制（纳秒，1 CPU = 1e9）

	// 内存限制（仅 Linux）
	Memory     int64 // 内存限制（字节）
	MemorySwap int64 // 内存+交换空间限制（字节），-1 表示不限制交换空间
}

// LogOptions 日志读取选项
type LogOptions struct {
	Follow     bool   // 是否持续跟随（类似 -f）
	Tail       int    // 获取最后 N 行：>0=最后N行, 0=不获取历史, <0=全部
	Timestamps bool   // 是否显示时间戳
	Since      string // 从某个时间开始（RFC3339 格式或 Unix 时间戳）
}

// ContainerEvent 表示 Docker 容器事件
type ContainerEvent struct {
	Action      string    // 事件类型: start, stop, die, create, destroy, rename 等
	ContainerID string    // 容器 ID
	ContainerName string  // 容器名称
	Timestamp   time.Time // 事件时间
}

// ProcessInfo 表示容器内进程信息
type ProcessInfo struct {
	PID     string // 宿主机进程 ID
	PPID    string // 父进程 ID
	User    string // 用户
	CPU     string // CPU 使用率
	Memory  string // 内存使用
	VSZ     string // 虚拟内存大小
	RSS     string // 常驻内存大小
	TTY     string // 终端
	Stat    string // 进程状态
	Start   string // 启动时间
	Time    string // 运行时间
	Command string // 命令
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

	// InspectContainerRaw 获取容器的原始 JSON 数据
	InspectContainerRaw(ctx context.Context, containerID string) (string, error)

	// ContainerStats 获取容器资源使用统计
	ContainerStats(ctx context.Context, containerID string) (*ContainerStats, error)

	// ContainerTop 获取容器内进程列表（类似 docker top）
	// 只有运行中的容器才能获取进程列表
	ContainerTop(ctx context.Context, containerID string) ([]ProcessInfo, error)

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

	// UpdateContainer 更新容器配置
	// 支持修改：重启策略、CPU 限制、内存限制等
	UpdateContainer(ctx context.Context, containerID string, config ContainerUpdateConfig) error

	// ===== 镜像管理 =====

	// ListImages 获取镜像列表
	// showAll: true 显示所有镜像（包括悬垂镜像），false 仅显示有标签的镜像
	ListImages(ctx context.Context, showAll bool) ([]Image, error)

	// ImageDetails 获取指定镜像的详细信息
	ImageDetails(ctx context.Context, imageID string) (*ImageDetails, error)

	// InspectImageRaw 获取镜像的原始 JSON 数据
	InspectImageRaw(ctx context.Context, imageID string) (string, error)

	// RemoveImage 删除镜像
	// force: 是否强制删除（即使有容器使用）
	// prune: 是否删除未标记的父镜像
	RemoveImage(ctx context.Context, imageID string, force bool, prune bool) error

	// PruneImages 清理悬垂镜像（无标签的镜像）
	// 返回删除的镜像数量和释放的空间（字节）
	PruneImages(ctx context.Context) (int, int64, error)

	// TagImage 给镜像打标签
	// imageID: 源镜像 ID
	// repository: 目标仓库名（如 myrepo/myimage）
	// tag: 目标标签（如 v1.0）
	TagImage(ctx context.Context, imageID string, repository string, tag string) error

	// UntagImage 删除镜像标签
	// imageRef: 镜像引用（如 myrepo/myimage:v1.0）
	UntagImage(ctx context.Context, imageRef string) error

	// SaveImage 导出镜像到 tar 文件
	// imageIDs: 要导出的镜像 ID 列表
	// 返回 io.ReadCloser，调用方负责关闭和写入文件
	SaveImage(ctx context.Context, imageIDs []string) (io.ReadCloser, error)

	// LoadImage 从 tar 文件加载镜像
	// input: tar 文件的 io.Reader
	// quiet: 是否静默模式（不输出详细信息）
	LoadImage(ctx context.Context, input io.Reader, quiet bool) error

	// PullImage 拉取镜像
	// imageRef: 镜像引用（如 nginx:latest）
	// 返回 io.ReadCloser 用于读取拉取进度，调用方负责关闭
	PullImage(ctx context.Context, imageRef string) (io.ReadCloser, error)

	// PushImage 推送镜像到 registry
	// imageRef: 镜像引用（如 myrepo/myimage:v1.0）
	// 返回 io.ReadCloser 用于读取推送进度，调用方负责关闭
	PushImage(ctx context.Context, imageRef string) (io.ReadCloser, error)

	// ===== 网络管理 =====

	// ListNetworks 获取网络列表
	ListNetworks(ctx context.Context) ([]Network, error)

	// NetworkDetails 获取指定网络的详细信息
	NetworkDetails(ctx context.Context, networkID string) (*NetworkDetails, error)

	// CreateNetwork 创建网络
	// 返回新创建的网络 ID
	CreateNetwork(ctx context.Context, opts NetworkCreateOptions) (string, error)

	// RemoveNetwork 删除网络
	RemoveNetwork(ctx context.Context, networkID string) error

	// PruneNetworks 清理未使用的网络
	// 返回删除的网络名称列表
	PruneNetworks(ctx context.Context) ([]string, error)

	// ConnectNetwork 将容器连接到网络
	ConnectNetwork(ctx context.Context, networkID string, opts NetworkConnectOptions) error

	// DisconnectNetwork 将容器从网络断开
	DisconnectNetwork(ctx context.Context, networkID string, opts NetworkDisconnectOptions) error

	// InspectNetworkRaw 获取网络的原始 JSON 数据
	InspectNetworkRaw(ctx context.Context, networkID string) (string, error)

	// Close 关闭客户端连接，释放资源
	Close() error
}

// LocalClient 封装本地 Docker SDK 客户端实现。
type LocalClient struct {
	cli        *sdk.Client
	imageCli   *image.Client   // 镜像操作客户端
	networkCli *network.Client // 网络操作客户端
}

// GetSDKClient 返回底层的 Docker SDK 客户端
func (c *LocalClient) GetSDKClient() *sdk.Client {
	return c.cli
}

// NewLocalClientFromEnv 基于环境变量创建本地 Docker 客户端，并开启 API 版本协商。
func NewLocalClientFromEnv() (*LocalClient, error) {
	cli, err := sdk.NewClientWithOpts(
		sdk.FromEnv,
		sdk.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return &LocalClient{
		cli:        cli,
		imageCli:   image.NewClient(cli),
		networkCli: network.NewClient(cli),
	}, nil
}

// Ping 用于验证 Docker 守护进程是否可用。
func (c *LocalClient) Ping(ctx context.Context) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
	}
	_, err := c.cli.Ping(ctx)
	return err
}

// ListContainers 获取容器列表
func (c *LocalClient) ListContainers(ctx context.Context, showAll bool) ([]Container, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: showAll})
	if err != nil {
		return nil, fmt.Errorf("failed to get container list: %w", err)
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
		return nil, fmt.Errorf("Docker client not initialized")
	}

	// 调用 Docker SDK 获取容器详细信息
	inspectResp, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container details: %w", err)
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

// InspectContainerRaw 获取容器的原始 JSON 数据
func (c *LocalClient) InspectContainerRaw(ctx context.Context, containerID string) (string, error) {
	if c == nil || c.cli == nil {
		return "", fmt.Errorf("Docker client not initialized")
	}

	// 调用 Docker SDK 获取容器详细信息
	inspectResp, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to get container details: %w", err)
	}

	// 格式化为 JSON
	jsonData, err := json.MarshalIndent(inspectResp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON serialization failed: %w", err)
	}

	return string(jsonData), nil
}

// ContainerLogs 获取容器日志
func (c *LocalClient) ContainerLogs(ctx context.Context, containerID string, opts LogOptions) (io.ReadCloser, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}

	// 构建 Docker SDK 日志选项
	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.Follow,
		Timestamps: opts.Timestamps,
	}

	// 处理 Tail 参数
	// Tail > 0: 获取最后 N 行
	// Tail = 0: 不获取历史日志（通常配合 Follow 使用）
	// Tail < 0: 获取全部日志
	if opts.Tail > 0 {
		tailStr := fmt.Sprintf("%d", opts.Tail)
		logOpts.Tail = tailStr
	} else if opts.Tail < 0 {
		// 获取全部日志
		logOpts.Tail = "all"
	}
	// Tail = 0 时不设置，表示不获取历史日志

	// 处理 Since 参数（用于 Follow 模式，只获取指定时间之后的日志）
	if opts.Since != "" {
		logOpts.Since = opts.Since
	}

	// 调用 Docker SDK 获取日志流
	logReader, err := c.cli.ContainerLogs(ctx, containerID, logOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}

	return logReader, nil
}

// ExecShell 在容器中启动交互式 shell
// 注意：这是一个简化的实现，适用于基本的交互式场景
// 完整的终端支持（如终端大小调整、特殊键处理）需要额外的终端设置
// 详细实现请查看 exec.go 文件
// 实际实现在 exec.go 中的 ExecShell 方法

// ===== 镜像管理方法（委托给 image.Client）=====

// ListImages 获取镜像列表
func (c *LocalClient) ListImages(ctx context.Context, showAll bool) ([]Image, error) {
	if c == nil || c.imageCli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}
	return c.imageCli.List(ctx, showAll)
}

// ImageDetails 获取指定镜像的详细信息
func (c *LocalClient) ImageDetails(ctx context.Context, imageID string) (*ImageDetails, error) {
	if c == nil || c.imageCli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}
	return c.imageCli.GetDetails(ctx, imageID)
}

// InspectImageRaw 获取镜像的原始 JSON 数据
func (c *LocalClient) InspectImageRaw(ctx context.Context, imageID string) (string, error) {
	if c == nil || c.imageCli == nil {
		return "", fmt.Errorf("Docker client not initialized")
	}
	return c.imageCli.InspectRaw(ctx, imageID)
}

// RemoveImage 删除镜像
func (c *LocalClient) RemoveImage(ctx context.Context, imageID string, force bool, prune bool) error {
	if c == nil || c.imageCli == nil {
		return fmt.Errorf("Docker client not initialized")
	}
	return c.imageCli.Remove(ctx, imageID, force, prune)
}

// PruneImages 清理悬垂镜像
func (c *LocalClient) PruneImages(ctx context.Context) (int, int64, error) {
	if c == nil || c.imageCli == nil {
		return 0, 0, fmt.Errorf("Docker client not initialized")
	}
	return c.imageCli.Prune(ctx)
}

// TagImage 给镜像打标签
func (c *LocalClient) TagImage(ctx context.Context, imageID string, repository string, tag string) error {
	if c == nil || c.imageCli == nil {
		return fmt.Errorf("Docker client not initialized")
	}
	return c.imageCli.Tag(ctx, imageID, repository, tag)
}

// UntagImage 删除镜像标签
func (c *LocalClient) UntagImage(ctx context.Context, imageRef string) error {
	if c == nil || c.imageCli == nil {
		return fmt.Errorf("Docker client not initialized")
	}
	return c.imageCli.Untag(ctx, imageRef)
}

// SaveImage 导出镜像到 tar 文件
func (c *LocalClient) SaveImage(ctx context.Context, imageIDs []string) (io.ReadCloser, error) {
	if c == nil || c.imageCli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}
	return c.imageCli.Save(ctx, imageIDs)
}

// LoadImage 从 tar 文件加载镜像
func (c *LocalClient) LoadImage(ctx context.Context, input io.Reader, quiet bool) error {
	if c == nil || c.imageCli == nil {
		return fmt.Errorf("Docker client not initialized")
	}
	return c.imageCli.Load(ctx, input, quiet)
}

// PullImage 拉取镜像
func (c *LocalClient) PullImage(ctx context.Context, imageRef string) (io.ReadCloser, error) {
	if c == nil || c.imageCli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}
	return c.imageCli.Pull(ctx, imageRef)
}

// PushImage 推送镜像到 registry
func (c *LocalClient) PushImage(ctx context.Context, imageRef string) (io.ReadCloser, error) {
	if c == nil || c.imageCli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}
	return c.imageCli.Push(ctx, imageRef)
}

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
			errorChan <- fmt.Errorf("Docker client not initialized")
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
					errorChan <- fmt.Errorf("failed to watch Docker events: %w", err)
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
		return fmt.Errorf("Docker client not initialized")
	}

	err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

// StopContainer 停止运行中的容器
func (c *LocalClient) StopContainer(ctx context.Context, containerID string, timeout int) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
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
		return fmt.Errorf("failed to stop container: %w", err)
	}

	return nil
}

// RestartContainer 重启容器
func (c *LocalClient) RestartContainer(ctx context.Context, containerID string, timeout int) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
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
		return fmt.Errorf("failed to restart container: %w", err)
	}

	return nil
}

// RemoveContainer 删除容器
func (c *LocalClient) RemoveContainer(ctx context.Context, containerID string, force bool, removeVolumes bool) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
	}

	err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: removeVolumes,
	})
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}

// PauseContainer 暂停容器
func (c *LocalClient) PauseContainer(ctx context.Context, containerID string) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
	}

	err := c.cli.ContainerPause(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to pause container: %w", err)
	}

	return nil
}

// UnpauseContainer 恢复暂停的容器
func (c *LocalClient) UnpauseContainer(ctx context.Context, containerID string) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
	}

	err := c.cli.ContainerUnpause(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to unpause container: %w", err)
	}

	return nil
}

// UpdateContainer 更新容器配置
func (c *LocalClient) UpdateContainer(ctx context.Context, containerID string, config ContainerUpdateConfig) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
	}

	// 构建更新配置
	updateConfig := container.UpdateConfig{
		Resources: container.Resources{},
	}

	// 设置 CPU 限制
	if config.CPUShares > 0 {
		updateConfig.Resources.CPUShares = config.CPUShares
	}
	if config.NanoCPUs > 0 {
		updateConfig.Resources.NanoCPUs = config.NanoCPUs
	}

	// 设置内存限制
	// 注意：必须同时设置 MemorySwap，否则可能报错
	if config.Memory > 0 {
		updateConfig.Resources.Memory = config.Memory
		// MemorySwap = -1 表示不限制交换空间（内存 + 无限交换）
		// MemorySwap = Memory 表示禁用交换空间
		// MemorySwap = Memory * 2 表示交换空间等于内存
		if config.MemorySwap != 0 {
			updateConfig.Resources.MemorySwap = config.MemorySwap
		} else {
			// 默认设置为 -1（不限制交换空间），避免与现有 memoryswap 冲突
			updateConfig.Resources.MemorySwap = -1
		}
	}

	// 设置重启策略
	if config.RestartPolicy != "" {
		updateConfig.RestartPolicy = container.RestartPolicy{
			Name:              container.RestartPolicyMode(config.RestartPolicy),
			MaximumRetryCount: config.RestartMaxRetries,
		}
	}

	// 调用 Docker SDK 更新容器
	_, err := c.cli.ContainerUpdate(ctx, containerID, updateConfig)
	if err != nil {
		return err
	}

	return nil
}

// ===== 网络管理方法（委托给 network.Client）=====

// ListNetworks 获取网络列表
func (c *LocalClient) ListNetworks(ctx context.Context) ([]Network, error) {
	if c == nil || c.networkCli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}
	return c.networkCli.List(ctx)
}

// NetworkDetails 获取指定网络的详细信息
func (c *LocalClient) NetworkDetails(ctx context.Context, networkID string) (*NetworkDetails, error) {
	if c == nil || c.networkCli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}
	return c.networkCli.GetDetails(ctx, networkID)
}

// CreateNetwork 创建网络
func (c *LocalClient) CreateNetwork(ctx context.Context, opts NetworkCreateOptions) (string, error) {
	if c == nil || c.networkCli == nil {
		return "", fmt.Errorf("Docker client not initialized")
	}
	return c.networkCli.Create(ctx, opts)
}

// RemoveNetwork 删除网络
func (c *LocalClient) RemoveNetwork(ctx context.Context, networkID string) error {
	if c == nil || c.networkCli == nil {
		return fmt.Errorf("Docker client not initialized")
	}
	return c.networkCli.Remove(ctx, networkID)
}

// PruneNetworks 清理未使用的网络
func (c *LocalClient) PruneNetworks(ctx context.Context) ([]string, error) {
	if c == nil || c.networkCli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}
	return c.networkCli.Prune(ctx)
}

// ConnectNetwork 将容器连接到网络
func (c *LocalClient) ConnectNetwork(ctx context.Context, networkID string, opts NetworkConnectOptions) error {
	if c == nil || c.networkCli == nil {
		return fmt.Errorf("Docker client not initialized")
	}
	return c.networkCli.Connect(ctx, networkID, opts)
}

// DisconnectNetwork 将容器从网络断开
func (c *LocalClient) DisconnectNetwork(ctx context.Context, networkID string, opts NetworkDisconnectOptions) error {
	if c == nil || c.networkCli == nil {
		return fmt.Errorf("Docker client not initialized")
	}
	return c.networkCli.Disconnect(ctx, networkID, opts)
}

// InspectNetworkRaw 获取网络的原始 JSON 数据
func (c *LocalClient) InspectNetworkRaw(ctx context.Context, networkID string) (string, error) {
	if c == nil || c.networkCli == nil {
		return "", fmt.Errorf("Docker client not initialized")
	}
	return c.networkCli.InspectRaw(ctx, networkID)
}

// ContainerTop 获取容器内进程列表（类似 docker top）
func (c *LocalClient) ContainerTop(ctx context.Context, containerID string) ([]ProcessInfo, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}

	// 调用 Docker SDK 获取进程列表
	// 不传参数，使用默认的 ps 输出格式（包含宿主机 PID）
	topResult, err := c.cli.ContainerTop(ctx, containerID, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get container process list: %w", err)
	}

	// 解析结果 - 根据 Titles 动态映射字段
	// 默认输出通常是: UID, PID, PPID, C, STIME, TTY, TIME, CMD
	titleMap := make(map[string]int)
	for i, title := range topResult.Titles {
		titleMap[strings.ToUpper(title)] = i
	}

	processes := make([]ProcessInfo, 0, len(topResult.Processes))
	for _, proc := range topResult.Processes {
		p := ProcessInfo{}
		
		// PID - 这是宿主机的 PID
		if idx, ok := titleMap["PID"]; ok && idx < len(proc) {
			p.PID = proc[idx]
		}
		// PPID - 父进程 ID（宿主机）
		if idx, ok := titleMap["PPID"]; ok && idx < len(proc) {
			p.PPID = proc[idx]
		}
		// USER/UID
		if idx, ok := titleMap["USER"]; ok && idx < len(proc) {
			p.User = proc[idx]
		} else if idx, ok := titleMap["UID"]; ok && idx < len(proc) {
			p.User = proc[idx]
		}
		// CPU (C 列)
		if idx, ok := titleMap["C"]; ok && idx < len(proc) {
			p.CPU = proc[idx]
		} else if idx, ok := titleMap["%CPU"]; ok && idx < len(proc) {
			p.CPU = proc[idx]
		}
		// STIME/START
		if idx, ok := titleMap["STIME"]; ok && idx < len(proc) {
			p.Start = proc[idx]
		} else if idx, ok := titleMap["START"]; ok && idx < len(proc) {
			p.Start = proc[idx]
		}
		// TTY
		if idx, ok := titleMap["TTY"]; ok && idx < len(proc) {
			p.TTY = proc[idx]
		}
		// TIME
		if idx, ok := titleMap["TIME"]; ok && idx < len(proc) {
			p.Time = proc[idx]
		}
		// CMD/COMMAND
		if idx, ok := titleMap["CMD"]; ok && idx < len(proc) {
			p.Command = proc[idx]
		} else if idx, ok := titleMap["COMMAND"]; ok && idx < len(proc) {
			p.Command = proc[idx]
		}
		// STAT
		if idx, ok := titleMap["STAT"]; ok && idx < len(proc) {
			p.Stat = proc[idx]
		}
		
		processes = append(processes, p)
	}

	return processes, nil
}
