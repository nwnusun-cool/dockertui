package network

import "time"

// Network 表示网络的基本信息（用于列表视图）
type Network struct {
	ID         string            // 网络 ID（完整）
	ShortID    string            // 网络 ID（短，12位）
	Name       string            // 网络名称
	Driver     string            // 驱动类型: bridge, host, overlay, macvlan, none
	Scope      string            // 范围: local, swarm, global
	Internal   bool              // 是否为内部网络（不能访问外部）
	IPv6       bool              // 是否启用 IPv6
	Attachable bool              // 是否可手动附加容器
	Ingress    bool              // 是否为 Ingress 网络
	Created    time.Time         // 创建时间
	Labels     map[string]string // 标签

	// 运行时状态
	ContainerCount int  // 连接的容器数量
	IsBuiltIn      bool // 是否为内置网络（bridge, host, none）
}

// Details 表示网络的详细信息（用于详情视图）
type Details struct {
	ID         string            // 网络 ID
	Name       string            // 网络名称
	Driver     string            // 驱动类型
	Scope      string            // 范围
	Internal   bool              // 是否为内部网络
	IPv6       bool              // 是否启用 IPv6
	Attachable bool              // 是否可手动附加容器
	Ingress    bool              // 是否为 Ingress 网络
	Created    time.Time         // 创建时间
	Labels     map[string]string // 标签
	Options    map[string]string // 驱动选项

	// IPAM 配置
	IPAM IPAMConfig // IP 地址管理配置

	// 连接的容器
	Containers []ContainerEndpoint // 连接到此网络的容器列表
}

// IPAMConfig IP 地址管理配置
type IPAMConfig struct {
	Driver  string       // IPAM 驱动
	Options map[string]string // IPAM 选项
	Configs []IPAMPoolConfig  // IP 池配置列表
}

// IPAMPoolConfig IP 池配置
type IPAMPoolConfig struct {
	Subnet  string // 子网 CIDR，如 172.17.0.0/16
	IPRange string // IP 范围（可选）
	Gateway string // 网关地址
}

// ContainerEndpoint 容器在网络中的端点信息
type ContainerEndpoint struct {
	ContainerID   string // 容器 ID
	ContainerName string // 容器名称
	EndpointID    string // 端点 ID
	MacAddress    string // MAC 地址
	IPv4Address   string // IPv4 地址（含 CIDR）
	IPv6Address   string // IPv6 地址（含 CIDR）
}

// CreateOptions 创建网络的选项
type CreateOptions struct {
	Name       string            // 网络名称（必填）
	Driver     string            // 驱动类型，默认 bridge
	Internal   bool              // 是否为内部网络
	Attachable bool              // 是否可手动附加容器
	IPv6       bool              // 是否启用 IPv6
	Labels     map[string]string // 标签
	Options    map[string]string // 驱动选项

	// IPAM 配置
	Subnet  string // 子网 CIDR
	Gateway string // 网关地址
	IPRange string // IP 范围
}

// ConnectOptions 连接容器到网络的选项
type ConnectOptions struct {
	ContainerID string // 容器 ID 或名称
	IPv4Address string // 指定 IPv4 地址（可选）
	IPv6Address string // 指定 IPv6 地址（可选）
	Aliases     []string // 网络别名
}

// DisconnectOptions 断开容器与网络连接的选项
type DisconnectOptions struct {
	ContainerID string // 容器 ID 或名称
	Force       bool   // 是否强制断开
}
