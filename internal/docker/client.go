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
	"github.com/docker/docker/api/types/image"
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

// Image 表示镜像的基本信息（用于列表视图）
type Image struct {
	ID         string            // 镜像 ID（完整）
	ShortID    string            // 镜像 ID（短，12位）
	Repository string            // 仓库名
	Tag        string            // 标签
	Size       int64             // 大小（字节）
	Created    time.Time         // 创建时间
	Digest     string            // 摘要
	Labels     map[string]string // 标签

	// 运行时状态
	InUse      bool     // 是否被容器使用
	Dangling   bool     // 是否为悬垂镜像（无标签）
	Containers []string // 使用此镜像的容器 ID 列表
}

// ContainerRef 表示容器引用信息
type ContainerRef struct {
	ID    string // 容器 ID
	Name  string // 容器名称
	State string // 容器状态: running, exited, paused 等
}

// ImageDetails 表示镜像的详细信息（用于详情视图）
type ImageDetails struct {
	ID            string            // 镜像 ID
	Repository    string            // 仓库名
	Tag           string            // 标签
	Size          int64             // 大小（字节）
	Created       time.Time         // 创建时间
	Digest        string            // 摘要
	Labels        map[string]string // 标签
	Architecture  string            // 架构
	OS            string            // 操作系统
	Author        string            // 作者
	Comment       string            // 注释
	Layers        []string          // 层 ID 列表
	History       []ImageHistory    // 镜像构建历史
	Env           []string          // 环境变量
	Cmd           []string          // 默认命令
	Entrypoint    []string          // 入口点
	WorkingDir    string            // 工作目录
	ExposedPorts  []string          // 暴露的端口
	Volumes       []string          // 卷
	User          string            // 用户
	Containers    []ContainerRef    // 使用此镜像的容器列表
}

// ImageHistory 表示镜像构建历史的一条记录
type ImageHistory struct {
	ID        string    // 层 ID（可能为 <missing>）
	Created   time.Time // 创建时间
	CreatedBy string    // 创建命令
	Size      int64     // 大小（字节）
	Comment   string    // 注释
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

	// ===== 镜像管理 =====

	// ListImages 获取镜像列表
	// showAll: true 显示所有镜像（包括悬垂镜像），false 仅显示有标签的镜像
	ListImages(ctx context.Context, showAll bool) ([]Image, error)

	// ImageDetails 获取指定镜像的详细信息
	ImageDetails(ctx context.Context, imageID string) (*ImageDetails, error)

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

// ListImages 获取镜像列表
func (c *LocalClient) ListImages(ctx context.Context, showAll bool) ([]Image, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker 客户端未初始化")
	}

	// 构建过滤器
	filterArgs := filters.NewArgs()
	if !showAll {
		// 不显示悬垂镜像
		filterArgs.Add("dangling", "false")
	}

	// 调用 Docker SDK 获取镜像列表
	images, err := c.cli.ImageList(ctx, image.ListOptions{
		All:     showAll,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("获取镜像列表失败: %w", err)
	}

	// 获取所有容器，用于判断镜像是否被使用
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %w", err)
	}

	// 构建镜像ID到容器ID的映射
	imageToContainers := make(map[string][]string)
	for _, cont := range containers {
		imageToContainers[cont.ImageID] = append(imageToContainers[cont.ImageID], cont.ID)
	}

	// 转换为内部数据结构
	result := make([]Image, 0, len(images))
	for _, img := range images {
		// 短 ID（12位）
		shortID := img.ID
		if strings.HasPrefix(shortID, "sha256:") {
			shortID = shortID[7:]
		}
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		// 仓库名和标签
		repository := "<none>"
		tag := "<none>"
		if len(img.RepoTags) > 0 {
			// 取第一个标签
			parts := strings.Split(img.RepoTags[0], ":")
			if len(parts) == 2 {
				repository = parts[0]
				tag = parts[1]
			} else {
				repository = img.RepoTags[0]
			}
		}

		// 判断是否为悬垂镜像
		dangling := repository == "<none>" && tag == "<none>"

		// 判断是否被容器使用
		containerIDs := imageToContainers[img.ID]
		inUse := len(containerIDs) > 0

		// 提取摘要
		digest := ""
		if len(img.RepoDigests) > 0 {
			digest = img.RepoDigests[0]
		}

		result = append(result, Image{
			ID:         img.ID,
			ShortID:    shortID,
			Repository: repository,
			Tag:        tag,
			Size:       img.Size,
			Created:    time.Unix(img.Created, 0),
			Digest:     digest,
			Labels:     img.Labels,
			InUse:      inUse,
			Dangling:   dangling,
			Containers: containerIDs,
		})
	}

	return result, nil
}

// ImageDetails 获取指定镜像的详细信息
func (c *LocalClient) ImageDetails(ctx context.Context, imageID string) (*ImageDetails, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker 客户端未初始化")
	}

	// 调用 Docker SDK 获取镜像详细信息
	inspectResp, _, err := c.cli.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("获取镜像详情失败: %w", err)
	}

	// 仓库名和标签
	repository := "<none>"
	tag := "<none>"
	if len(inspectResp.RepoTags) > 0 {
		parts := strings.Split(inspectResp.RepoTags[0], ":")
		if len(parts) == 2 {
			repository = parts[0]
			tag = parts[1]
		} else {
			repository = inspectResp.RepoTags[0]
		}
	}

	// 提取暴露的端口
	exposedPorts := make([]string, 0)
	if inspectResp.Config != nil && inspectResp.Config.ExposedPorts != nil {
		for port := range inspectResp.Config.ExposedPorts {
			exposedPorts = append(exposedPorts, string(port))
		}
	}

	// 提取卷
	volumes := make([]string, 0)
	if inspectResp.Config != nil && inspectResp.Config.Volumes != nil {
		for vol := range inspectResp.Config.Volumes {
			volumes = append(volumes, vol)
		}
	}

	// 获取使用此镜像的容器
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %w", err)
	}

	containerRefs := make([]ContainerRef, 0)
	for _, cont := range containers {
		if cont.ImageID == inspectResp.ID {
			// 提取容器名称（去除前导 /）
			containerName := ""
			if len(cont.Names) > 0 {
				containerName = cont.Names[0]
				if len(containerName) > 0 && containerName[0] == '/' {
					containerName = containerName[1:]
				}
			}

			containerRefs = append(containerRefs, ContainerRef{
				ID:    cont.ID,
				Name:  containerName,
				State: string(cont.State),
			})
		}
	}

	// 提取环境变量、命令、入口点等
	env := []string{}
	cmd := []string{}
	entrypoint := []string{}
	workingDir := ""
	user := ""

	if inspectResp.Config != nil {
		env = inspectResp.Config.Env
		cmd = inspectResp.Config.Cmd
		entrypoint = inspectResp.Config.Entrypoint
		workingDir = inspectResp.Config.WorkingDir
		user = inspectResp.Config.User
	}

	// 提取摘要
	digest := ""
	if len(inspectResp.RepoDigests) > 0 {
		digest = inspectResp.RepoDigests[0]
	}

	// 提取标签
	var labels map[string]string
	if inspectResp.Config != nil {
		labels = inspectResp.Config.Labels
	}

	// 提取层信息
	var layers []string
	if inspectResp.RootFS.Type != "" {
		layers = inspectResp.RootFS.Layers
	}

	// 解析创建时间
	created := time.Now()
	if inspectResp.Created != "" {
		if t, err := time.Parse(time.RFC3339Nano, inspectResp.Created); err == nil {
			created = t
		}
	}

	// 获取镜像构建历史
	historyResp, err := c.cli.ImageHistory(ctx, imageID)
	var history []ImageHistory
	if err == nil {
		for _, h := range historyResp {
			historyItem := ImageHistory{
				ID:        h.ID,
				Created:   time.Unix(h.Created, 0),
				CreatedBy: h.CreatedBy,
				Size:      h.Size,
				Comment:   h.Comment,
			}
			// 如果 ID 为空，显示为 <missing>
			if historyItem.ID == "" {
				historyItem.ID = "<missing>"
			}
			history = append(history, historyItem)
		}
	}

	return &ImageDetails{
		ID:           inspectResp.ID,
		Repository:   repository,
		Tag:          tag,
		Size:         inspectResp.Size,
		Created:      created,
		Digest:       digest,
		Labels:       labels,
		Architecture: inspectResp.Architecture,
		OS:           inspectResp.Os,
		Author:       inspectResp.Author,
		Comment:      inspectResp.Comment,
		Layers:       layers,
		History:      history,
		Env:          env,
		Cmd:          cmd,
		Entrypoint:   entrypoint,
		WorkingDir:   workingDir,
		ExposedPorts: exposedPorts,
		Volumes:      volumes,
		User:         user,
		Containers:   containerRefs,
	}, nil
}

// RemoveImage 删除镜像
func (c *LocalClient) RemoveImage(ctx context.Context, imageID string, force bool, prune bool) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	_, err := c.cli.ImageRemove(ctx, imageID, image.RemoveOptions{
		Force:         force,
		PruneChildren: prune,
	})
	if err != nil {
		return fmt.Errorf("删除镜像失败: %w", err)
	}

	return nil
}

// PruneImages 清理悬垂镜像
func (c *LocalClient) PruneImages(ctx context.Context) (int, int64, error) {
	if c == nil || c.cli == nil {
		return 0, 0, fmt.Errorf("Docker 客户端未初始化")
	}

	// 构建过滤器，只清理悬垂镜像
	filterArgs := filters.NewArgs()
	filterArgs.Add("dangling", "true")

	report, err := c.cli.ImagesPrune(ctx, filterArgs)
	if err != nil {
		return 0, 0, fmt.Errorf("清理镜像失败: %w", err)
	}

	return len(report.ImagesDeleted), int64(report.SpaceReclaimed), nil
}

// TagImage 给镜像打标签
func (c *LocalClient) TagImage(ctx context.Context, imageID string, repository string, tag string) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	// 构建完整的标签引用
	ref := repository + ":" + tag

	err := c.cli.ImageTag(ctx, imageID, ref)
	if err != nil {
		return fmt.Errorf("打标签失败: %w", err)
	}

	return nil
}

// UntagImage 删除镜像标签
func (c *LocalClient) UntagImage(ctx context.Context, imageRef string) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	// 删除标签（不强制，不删除子镜像）
	_, err := c.cli.ImageRemove(ctx, imageRef, image.RemoveOptions{
		Force:         false,
		PruneChildren: false,
	})
	if err != nil {
		return fmt.Errorf("删除标签失败: %w", err)
	}

	return nil
}

// SaveImage 导出镜像到 tar 文件
func (c *LocalClient) SaveImage(ctx context.Context, imageIDs []string) (io.ReadCloser, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker 客户端未初始化")
	}

	reader, err := c.cli.ImageSave(ctx, imageIDs)
	if err != nil {
		return nil, fmt.Errorf("导出镜像失败: %w", err)
	}

	return reader, nil
}

// LoadImage 从 tar 文件加载镜像
func (c *LocalClient) LoadImage(ctx context.Context, input io.Reader, quiet bool) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	// Docker SDK v28+ ImageLoad 不接受 quiet 参数
	resp, err := c.cli.ImageLoad(ctx, input)
	if err != nil {
		return fmt.Errorf("加载镜像失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应以确保完成
	// quiet 参数在我们的接口中保留，用于控制是否处理输出
	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
}

// PullImage 拉取镜像
func (c *LocalClient) PullImage(ctx context.Context, imageRef string) (io.ReadCloser, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker 客户端未初始化")
	}

	reader, err := c.cli.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("拉取镜像失败: %w", err)
	}

	return reader, nil
}

// PushImage 推送镜像到 registry
func (c *LocalClient) PushImage(ctx context.Context, imageRef string) (io.ReadCloser, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker 客户端未初始化")
	}

	reader, err := c.cli.ImagePush(ctx, imageRef, image.PushOptions{})
	if err != nil {
		return nil, fmt.Errorf("推送镜像失败: %w", err)
	}

	return reader, nil
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
