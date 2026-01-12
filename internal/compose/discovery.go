package compose

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	sdk "github.com/docker/docker/client"
)

// Compose 标签常量
const (
	LabelProject           = "com.docker.compose.project"
	LabelProjectWorkingDir = "com.docker.compose.project.working_dir"
	LabelProjectConfigFiles = "com.docker.compose.project.config_files"
	LabelService           = "com.docker.compose.service"
	LabelContainerNumber   = "com.docker.compose.container-number"
	LabelVersion           = "com.docker.compose.version"
)

// Discovery 基于 Docker API 的 Compose 项目发现器
type Discovery struct {
	dockerCli *sdk.Client
	mu        sync.RWMutex
	cache     map[string]*Project // 项目缓存，key 为项目名
	cacheTTL  time.Duration
	lastScan  time.Time
}

// NewDiscovery 创建项目发现器
func NewDiscovery(dockerCli *sdk.Client) *Discovery {
	return &Discovery{
		dockerCli: dockerCli,
		cache:     make(map[string]*Project),
		cacheTTL:  30 * time.Second,
	}
}

// DiscoverProjects 发现所有 Compose 项目（通过容器标签）
func (d *Discovery) DiscoverProjects(ctx context.Context) ([]*Project, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 检查缓存是否有效
	if time.Since(d.lastScan) < d.cacheTTL && len(d.cache) > 0 {
		return d.cacheToSlice(), nil
	}

	// 获取所有容器（包括停止的）
	containers, err := d.dockerCli.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", LabelProject),
		),
	})
	if err != nil {
		return nil, err
	}

	// 按项目分组
	projectMap := make(map[string]*Project)
	serviceMap := make(map[string]map[string]*Service) // projectName -> serviceName -> Service

	for _, c := range containers {
		projectName := c.Labels[LabelProject]
		if projectName == "" {
			continue
		}

		// 获取或创建项目
		project, exists := projectMap[projectName]
		if !exists {
			project = &Project{
				Name:         projectName,
				WorkingDir:   c.Labels[LabelProjectWorkingDir],
				Path:         c.Labels[LabelProjectWorkingDir],
				ComposeFiles: parseConfigFiles(c.Labels[LabelProjectConfigFiles]),
				Labels:       make(map[string]string),
				Services:     []Service{},
				LastUpdated:  time.Now(),
			}
			projectMap[projectName] = project
			serviceMap[projectName] = make(map[string]*Service)
		}

		// 获取服务名
		serviceName := c.Labels[LabelService]
		if serviceName == "" {
			continue
		}

		// 获取或创建服务
		svc, exists := serviceMap[projectName][serviceName]
		if !exists {
			svc = &Service{
				Name:       serviceName,
				Image:      c.Image,
				Containers: []string{},
				Replicas:   0,
				Running:    0,
			}
			serviceMap[projectName][serviceName] = svc
		}

		// 添加容器
		svc.Containers = append(svc.Containers, c.ID)
		svc.Replicas++

		// 更新运行状态
		if c.State == "running" {
			svc.Running++
		}

		// 更新端口信息
		if len(c.Ports) > 0 && len(svc.Ports) == 0 {
			svc.Ports = formatPorts(c.Ports)
		}
	}

	// 组装项目和服务
	for projectName, project := range projectMap {
		services := serviceMap[projectName]
		for _, svc := range services {
			// 计算服务状态
			if svc.Running == 0 {
				svc.State = "exited"
			} else if svc.Running < svc.Replicas {
				svc.State = "partial"
			} else {
				svc.State = "running"
			}
			project.Services = append(project.Services, *svc)
		}

		// 计算项目状态
		project.Status = calculateProjectStatus(project.Services)
	}

	// 更新缓存
	d.cache = projectMap
	d.lastScan = time.Now()

	return d.cacheToSlice(), nil
}

// GetProject 获取指定项目
func (d *Discovery) GetProject(ctx context.Context, projectName string) (*Project, error) {
	projects, err := d.DiscoverProjects(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		if p.Name == projectName {
			return p, nil
		}
	}

	return nil, nil
}

// RefreshProject 刷新单个项目状态
func (d *Discovery) RefreshProject(ctx context.Context, projectName string) (*Project, error) {
	d.mu.Lock()
	d.lastScan = time.Time{} // 强制刷新
	d.mu.Unlock()

	return d.GetProject(ctx, projectName)
}

// GetProjectContainers 获取项目的所有容器
func (d *Discovery) GetProjectContainers(ctx context.Context, projectName string) ([]types.Container, error) {
	containers, err := d.dockerCli.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", LabelProject+"="+projectName),
		),
	})
	if err != nil {
		return nil, err
	}
	return containers, nil
}

// GetServiceContainers 获取服务的所有容器
func (d *Discovery) GetServiceContainers(ctx context.Context, projectName, serviceName string) ([]types.Container, error) {
	containers, err := d.dockerCli.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", LabelProject+"="+projectName),
			filters.Arg("label", LabelService+"="+serviceName),
		),
	})
	if err != nil {
		return nil, err
	}
	return containers, nil
}

// InvalidateCache 使缓存失效
func (d *Discovery) InvalidateCache() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastScan = time.Time{}
}

// cacheToSlice 将缓存转换为切片
func (d *Discovery) cacheToSlice() []*Project {
	projects := make([]*Project, 0, len(d.cache))
	for _, p := range d.cache {
		projects = append(projects, p)
	}
	return projects
}

// parseConfigFiles 解析配置文件列表
func parseConfigFiles(configStr string) []string {
	if configStr == "" {
		return nil
	}
	// 配置文件可能以逗号或分号分隔
	files := strings.Split(configStr, ",")
	result := make([]string, 0, len(files))
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f != "" {
			result = append(result, f)
		}
	}
	return result
}

// formatPorts 格式化端口列表
func formatPorts(ports []types.Port) []string {
	result := make([]string, 0, len(ports))
	for _, p := range ports {
		if p.PublicPort > 0 {
			result = append(result, formatPort(p))
		}
	}
	return result
}

// formatPort 格式化单个端口
func formatPort(p types.Port) string {
	if p.PublicPort > 0 {
		ip := p.IP
		if ip == "" {
			ip = "0.0.0.0"
		}
		return strings.Join([]string{
			ip, ":", itoa(int(p.PublicPort)), "->", itoa(int(p.PrivatePort)), "/", p.Type,
		}, "")
	}
	return strings.Join([]string{itoa(int(p.PrivatePort)), "/", p.Type}, "")
}

// itoa 简单的整数转字符串
func itoa(i int) string {
	return strconv.Itoa(i)
}
