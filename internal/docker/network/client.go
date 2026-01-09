package network

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	sdk "github.com/docker/docker/client"
)

// Client 网络操作客户端
type Client struct {
	cli *sdk.Client
}

// NewClient 创建网络客户端
func NewClient(cli *sdk.Client) *Client {
	return &Client{cli: cli}
}

// builtInNetworks 内置网络名称
var builtInNetworks = map[string]bool{
	"bridge": true,
	"host":   true,
	"none":   true,
}

// List 获取网络列表
func (c *Client) List(ctx context.Context) ([]Network, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker 客户端未初始化")
	}

	// 调用 Docker SDK 获取网络列表
	networks, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取网络列表失败: %w", err)
	}

	result := make([]Network, 0, len(networks))
	for _, net := range networks {
		// 短 ID（12位）
		shortID := net.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		// 解析创建时间
		created := time.Time{}
		if net.Created != (time.Time{}) {
			created = net.Created
		}

		// 统计连接的容器数量
		containerCount := len(net.Containers)

		// 判断是否为内置网络
		isBuiltIn := builtInNetworks[net.Name]

		result = append(result, Network{
			ID:             net.ID,
			ShortID:        shortID,
			Name:           net.Name,
			Driver:         net.Driver,
			Scope:          net.Scope,
			Internal:       net.Internal,
			IPv6:           net.EnableIPv6,
			Attachable:     net.Attachable,
			Ingress:        net.Ingress,
			Created:        created,
			Labels:         net.Labels,
			ContainerCount: containerCount,
			IsBuiltIn:      isBuiltIn,
		})
	}

	return result, nil
}

// GetDetails 获取网络详细信息
func (c *Client) GetDetails(ctx context.Context, networkID string) (*Details, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker 客户端未初始化")
	}

	// 调用 Docker SDK 获取网络详情
	net, err := c.cli.NetworkInspect(ctx, networkID, network.InspectOptions{
		Verbose: true,
	})
	if err != nil {
		return nil, fmt.Errorf("获取网络详情失败: %w", err)
	}

	// 解析 IPAM 配置
	ipamConfig := IPAMConfig{
		Driver:  net.IPAM.Driver,
		Options: net.IPAM.Options,
		Configs: make([]IPAMPoolConfig, 0, len(net.IPAM.Config)),
	}
	for _, cfg := range net.IPAM.Config {
		ipamConfig.Configs = append(ipamConfig.Configs, IPAMPoolConfig{
			Subnet:  cfg.Subnet,
			IPRange: cfg.IPRange,
			Gateway: cfg.Gateway,
		})
	}

	// 解析连接的容器
	containers := make([]ContainerEndpoint, 0, len(net.Containers))
	for containerID, endpoint := range net.Containers {
		containers = append(containers, ContainerEndpoint{
			ContainerID:   containerID,
			ContainerName: strings.TrimPrefix(endpoint.Name, "/"),
			EndpointID:    endpoint.EndpointID,
			MacAddress:    endpoint.MacAddress,
			IPv4Address:   endpoint.IPv4Address,
			IPv6Address:   endpoint.IPv6Address,
		})
	}

	// 解析创建时间
	created := time.Time{}
	if net.Created != (time.Time{}) {
		created = net.Created
	}

	return &Details{
		ID:         net.ID,
		Name:       net.Name,
		Driver:     net.Driver,
		Scope:      net.Scope,
		Internal:   net.Internal,
		IPv6:       net.EnableIPv6,
		Attachable: net.Attachable,
		Ingress:    net.Ingress,
		Created:    created,
		Labels:     net.Labels,
		Options:    net.Options,
		IPAM:       ipamConfig,
		Containers: containers,
	}, nil
}

// Create 创建网络
func (c *Client) Create(ctx context.Context, opts CreateOptions) (string, error) {
	if c == nil || c.cli == nil {
		return "", fmt.Errorf("Docker 客户端未初始化")
	}

	if opts.Name == "" {
		return "", fmt.Errorf("网络名称不能为空")
	}

	// 默认驱动
	driver := opts.Driver
	if driver == "" {
		driver = "bridge"
	}

	// 构建 IPAM 配置
	var ipamConfig *network.IPAM
	if opts.Subnet != "" || opts.Gateway != "" {
		ipamConfig = &network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{
				{
					Subnet:  opts.Subnet,
					Gateway: opts.Gateway,
					IPRange: opts.IPRange,
				},
			},
		}
	}

	// 创建网络
	resp, err := c.cli.NetworkCreate(ctx, opts.Name, network.CreateOptions{
		Driver:     driver,
		Internal:   opts.Internal,
		Attachable: opts.Attachable,
		EnableIPv6: &opts.IPv6,
		Labels:     opts.Labels,
		Options:    opts.Options,
		IPAM:       ipamConfig,
	})
	if err != nil {
		return "", fmt.Errorf("创建网络失败: %w", err)
	}

	return resp.ID, nil
}

// Remove 删除网络
func (c *Client) Remove(ctx context.Context, networkID string) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	err := c.cli.NetworkRemove(ctx, networkID)
	if err != nil {
		return fmt.Errorf("删除网络失败: %w", err)
	}

	return nil
}

// Prune 清理未使用的网络
// 返回删除的网络名称列表
func (c *Client) Prune(ctx context.Context) ([]string, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker 客户端未初始化")
	}

	report, err := c.cli.NetworksPrune(ctx, filters.Args{})
	if err != nil {
		return nil, fmt.Errorf("清理网络失败: %w", err)
	}

	return report.NetworksDeleted, nil
}

// Connect 将容器连接到网络
func (c *Client) Connect(ctx context.Context, networkID string, opts ConnectOptions) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	if opts.ContainerID == "" {
		return fmt.Errorf("容器 ID 不能为空")
	}

	// 构建端点配置
	endpointConfig := &network.EndpointSettings{
		Aliases: opts.Aliases,
	}

	// 如果指定了 IP 地址
	if opts.IPv4Address != "" || opts.IPv6Address != "" {
		endpointConfig.IPAMConfig = &network.EndpointIPAMConfig{
			IPv4Address: opts.IPv4Address,
			IPv6Address: opts.IPv6Address,
		}
	}

	err := c.cli.NetworkConnect(ctx, networkID, opts.ContainerID, endpointConfig)
	if err != nil {
		return fmt.Errorf("连接容器到网络失败: %w", err)
	}

	return nil
}

// Disconnect 将容器从网络断开
func (c *Client) Disconnect(ctx context.Context, networkID string, opts DisconnectOptions) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker 客户端未初始化")
	}

	if opts.ContainerID == "" {
		return fmt.Errorf("容器 ID 不能为空")
	}

	err := c.cli.NetworkDisconnect(ctx, networkID, opts.ContainerID, opts.Force)
	if err != nil {
		return fmt.Errorf("断开容器与网络连接失败: %w", err)
	}

	return nil
}

// InspectRaw 获取网络的原始 JSON 数据
func (c *Client) InspectRaw(ctx context.Context, networkID string) (string, error) {
	if c == nil || c.cli == nil {
		return "", fmt.Errorf("Docker 客户端未初始化")
	}

	// 调用 Docker SDK 获取网络详情
	net, err := c.cli.NetworkInspect(ctx, networkID, network.InspectOptions{
		Verbose: true,
	})
	if err != nil {
		return "", fmt.Errorf("获取网络详情失败: %w", err)
	}

	// 格式化为 JSON
	jsonData, err := json.MarshalIndent(net, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON 序列化失败: %w", err)
	}

	return string(jsonData), nil
}
