package compose

import (
	"io"
	"time"
)

// ProjectStatus 项目状态
type ProjectStatus int

const (
	StatusUnknown ProjectStatus = iota
	StatusRunning               // 所有服务运行中
	StatusPartial               // 部分服务运行中
	StatusStopped               // 所有服务已停止
	StatusError                 // 错误状态
)

// String 返回状态的字符串表示
func (s ProjectStatus) String() string {
	switch s {
	case StatusRunning:
		return "Running"
	case StatusPartial:
		return "Partial"
	case StatusStopped:
		return "Stopped"
	case StatusError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Project 表示一个 docker-compose 项目
type Project struct {
	Name         string            // 项目名称
	Path         string            // 项目根目录绝对路径
	ComposeFiles []string          // compose 文件列表（相对路径）
	EnvFiles     []string          // 环境变量文件列表
	WorkingDir   string            // 工作目录
	Labels       map[string]string // 项目标签

	// 运行时状态
	Services    []Service     // 服务列表
	Status      ProjectStatus // 项目状态
	LastUpdated time.Time     // 最后更新时间
}

// Service 表示 compose 项目中的一个服务
type Service struct {
	Name       string   // 服务名称
	Image      string   // 镜像名称
	State      string   // 状态：running/exited/restarting/paused
	Containers []string // 容器 ID 列表
	Replicas   int      // 副本数量
	Running    int      // 运行中的副本数
	Ports      []string // 端口映射
}

// PortMapping 端口映射
type PortMapping struct {
	HostIP        string
	HostPort      int
	ContainerPort int
	Protocol      string
}

// UpOptions 启动项目的选项
type UpOptions struct {
	Detach        bool     // 后台运行（-d）
	Build         bool     // 构建镜像（--build）
	ForceRecreate bool     // 强制重建（--force-recreate）
	NoDeps        bool     // 不启动依赖服务（--no-deps）
	Services      []string // 指定服务（为空则启动所有）
	Timeout       int      // 超时时间（秒）
	Pull          string   // 拉取策略：always/missing/never
}

// DownOptions 停止项目的选项
type DownOptions struct {
	RemoveVolumes bool // 删除卷（-v）
	RemoveOrphans bool // 删除孤立容器（--remove-orphans）
	RemoveImages  string // 删除镜像：all/local（空表示不删除）
	Timeout       int  // 超时时间（秒）
}

// LogOptions 日志选项
type LogOptions struct {
	Follow     bool     // 跟踪模式（-f）
	Tail       int      // 显示最后 N 行（0 表示全部）
	Timestamps bool     // 显示时间戳（-t）
	Services   []string // 指定服务（为空则显示所有）
	Since      string   // 起始时间
	Until      string   // 结束时间
}

// BuildOptions 构建选项
type BuildOptions struct {
	NoCache  bool     // 不使用缓存（--no-cache）
	Pull     bool     // 拉取基础镜像（--pull）
	Services []string // 指定服务
}

// PullOptions 拉取选项
type PullOptions struct {
	IgnorePullFailures bool     // 忽略拉取失败
	Services           []string // 指定服务
}

// OperationResult 操作结果
type OperationResult struct {
	Success  bool          // 是否成功
	Message  string        // 消息
	Output   string        // stdout 输出
	Error    string        // stderr 输出
	ExitCode int           // 退出码
	Duration time.Duration // 耗时
}

// ErrorType 错误类型
type ErrorType int

const (
	ErrorUnknown ErrorType = iota
	ErrorConfig            // compose 文件配置错误
	ErrorNetwork           // 网络错误（端口冲突等）
	ErrorImage             // 镜像相关错误
	ErrorRuntime           // 运行时错误
	ErrorPermission        // 权限错误
	ErrorNotFound          // 命令或文件未找到
)

// ComposeError compose 操作错误
type ComposeError struct {
	Type       ErrorType
	Message    string
	Details    string
	Suggestion string
}

// Error 实现 error 接口
func (e *ComposeError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// Client docker-compose 客户端接口
type Client interface {
	// 版本信息
	Version() (string, error)
	CommandType() string // 返回 "docker compose" 或 "docker-compose"

	// 项目操作
	Up(project *Project, opts UpOptions) (*OperationResult, error)
	Down(project *Project, opts DownOptions) (*OperationResult, error)
	Start(project *Project, services []string) (*OperationResult, error)
	Stop(project *Project, services []string, timeout int) (*OperationResult, error)
	Restart(project *Project, services []string, timeout int) (*OperationResult, error)
	Pause(project *Project, services []string) (*OperationResult, error)
	Unpause(project *Project, services []string) (*OperationResult, error)

	// 信息查询
	PS(project *Project) ([]Service, error)
	Logs(project *Project, opts LogOptions) (io.ReadCloser, error)
	Config(project *Project) (string, error)

	// 镜像操作
	Build(project *Project, opts BuildOptions) (*OperationResult, error)
	Pull(project *Project, opts PullOptions) (*OperationResult, error)
}
