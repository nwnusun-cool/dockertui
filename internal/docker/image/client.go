package image

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
	sdk "github.com/docker/docker/client"
)

// Client 镜像操作客户端
type Client struct {
	cli *sdk.Client
}

// NewClient 创建镜像客户端
func NewClient(cli *sdk.Client) *Client {
	return &Client{cli: cli}
}

// List 获取镜像列表
// showAll: true 显示所有镜像（包括悬垂镜像），false 仅显示有标签的镜像
func (c *Client) List(ctx context.Context, showAll bool) ([]Image, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}

	// 构建过滤器
	filterArgs := filters.NewArgs()
	if !showAll {
		filterArgs.Add("dangling", "false")
	}

	// 调用 Docker SDK 获取镜像列表
	images, err := c.cli.ImageList(ctx, dockerimage.ListOptions{
		All:     showAll,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get image list: %w", err)
	}

	// 获取所有容器，用于判断镜像是否被使用
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get container list: %w", err)
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

		// 判断是否被容器使用
		containerIDs := imageToContainers[img.ID]
		inUse := len(containerIDs) > 0

		// 提取摘要
		digest := ""
		if len(img.RepoDigests) > 0 {
			digest = img.RepoDigests[0]
		}

		// 如果有多个标签，为每个标签创建一个列表项
		if len(img.RepoTags) > 0 {
			for _, repoTag := range img.RepoTags {
				repository := "<none>"
				tag := "<none>"
				parts := strings.Split(repoTag, ":")
				if len(parts) == 2 {
					repository = parts[0]
					tag = parts[1]
				} else {
					repository = repoTag
				}

				dangling := repository == "<none>" && tag == "<none>"

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
		} else {
			// 没有标签的镜像（悬垂镜像）
			result = append(result, Image{
				ID:         img.ID,
				ShortID:    shortID,
				Repository: "<none>",
				Tag:        "<none>",
				Size:       img.Size,
				Created:    time.Unix(img.Created, 0),
				Digest:     digest,
				Labels:     img.Labels,
				InUse:      inUse,
				Dangling:   true,
				Containers: containerIDs,
			})
		}
	}

	return result, nil
}

// GetDetails 获取指定镜像的详细信息
func (c *Client) GetDetails(ctx context.Context, imageID string) (*Details, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}

	// 调用 Docker SDK 获取镜像详细信息
	inspectResp, _, err := c.cli.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get image details: %w", err)
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
		return nil, fmt.Errorf("failed to get container list: %w", err)
	}

	containerRefs := make([]ContainerRef, 0)
	for _, cont := range containers {
		if cont.ImageID == inspectResp.ID {
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
	var history []History
	if err == nil {
		for _, h := range historyResp {
			historyItem := History{
				ID:        h.ID,
				Created:   time.Unix(h.Created, 0),
				CreatedBy: h.CreatedBy,
				Size:      h.Size,
				Comment:   h.Comment,
			}
			if historyItem.ID == "" {
				historyItem.ID = "<missing>"
			}
			history = append(history, historyItem)
		}
	}

	return &Details{
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

// InspectRaw 获取镜像的原始 JSON 数据
func (c *Client) InspectRaw(ctx context.Context, imageID string) (string, error) {
	if c == nil || c.cli == nil {
		return "", fmt.Errorf("Docker client not initialized")
	}

	// 调用 Docker SDK 获取镜像详情
	inspectResp, _, err := c.cli.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return "", fmt.Errorf("failed to get image details: %w", err)
	}

	// 格式化为 JSON
	jsonData, err := json.MarshalIndent(inspectResp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON serialization failed: %w", err)
	}

	return string(jsonData), nil
}

// Remove 删除镜像
// force: 是否强制删除（即使有容器使用）
// prune: 是否删除未标记的父镜像
func (c *Client) Remove(ctx context.Context, imageID string, force bool, prune bool) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
	}

	_, err := c.cli.ImageRemove(ctx, imageID, dockerimage.RemoveOptions{
		Force:         force,
		PruneChildren: prune,
	})
	if err != nil {
		return fmt.Errorf("failed to remove image: %w", err)
	}

	return nil
}

// Prune 清理悬垂镜像
// 返回删除的镜像数量和释放的空间（字节）
func (c *Client) Prune(ctx context.Context) (int, int64, error) {
	if c == nil || c.cli == nil {
		return 0, 0, fmt.Errorf("Docker client not initialized")
	}

	filterArgs := filters.NewArgs()
	filterArgs.Add("dangling", "true")

	report, err := c.cli.ImagesPrune(ctx, filterArgs)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to prune images: %w", err)
	}

	return len(report.ImagesDeleted), int64(report.SpaceReclaimed), nil
}

// Tag 给镜像打标签
func (c *Client) Tag(ctx context.Context, imageID string, repository string, tag string) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
	}

	ref := repository + ":" + tag

	err := c.cli.ImageTag(ctx, imageID, ref)
	if err != nil {
		return fmt.Errorf("failed to tag image: %w", err)
	}

	return nil
}

// Untag 删除镜像标签
func (c *Client) Untag(ctx context.Context, imageRef string) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
	}

	_, err := c.cli.ImageRemove(ctx, imageRef, dockerimage.RemoveOptions{
		Force:         false,
		PruneChildren: false,
	})
	if err != nil {
		return fmt.Errorf("failed to remove tag: %w", err)
	}

	return nil
}

// Save 导出镜像到 tar 文件
// 返回 io.ReadCloser，调用方负责关闭和写入文件
func (c *Client) Save(ctx context.Context, imageIDs []string) (io.ReadCloser, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}

	reader, err := c.cli.ImageSave(ctx, imageIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to export image: %w", err)
	}

	return reader, nil
}

// Load 从 tar 文件加载镜像
func (c *Client) Load(ctx context.Context, input io.Reader, quiet bool) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
	}

	resp, err := c.cli.ImageLoad(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
}

// Pull 拉取镜像
// 返回 io.ReadCloser 用于读取拉取进度，调用方负责关闭
func (c *Client) Pull(ctx context.Context, imageRef string) (io.ReadCloser, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}

	reader, err := c.cli.ImagePull(ctx, imageRef, dockerimage.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	return reader, nil
}

// Push 推送镜像到 registry
// 返回 io.ReadCloser 用于读取推送进度，调用方负责关闭
func (c *Client) Push(ctx context.Context, imageRef string) (io.ReadCloser, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}

	reader, err := c.cli.ImagePush(ctx, imageRef, dockerimage.PushOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to push image: %w", err)
	}

	return reader, nil
}
