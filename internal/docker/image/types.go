package image

import "time"

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

// Details 表示镜像的详细信息（用于详情视图）
type Details struct {
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
	History       []History         // 镜像构建历史
	Env           []string          // 环境变量
	Cmd           []string          // 默认命令
	Entrypoint    []string          // 入口点
	WorkingDir    string            // 工作目录
	ExposedPorts  []string          // 暴露的端口
	Volumes       []string          // 卷
	User          string            // 用户
	Containers    []ContainerRef    // 使用此镜像的容器列表
}

// History 表示镜像构建历史的一条记录
type History struct {
	ID        string    // 层 ID（可能为 <missing>）
	Created   time.Time // 创建时间
	CreatedBy string    // 创建命令
	Size      int64     // 大小（字节）
	Comment   string    // 注释
}
