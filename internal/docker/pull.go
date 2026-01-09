package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"

	dockerimage "github.com/docker/docker/api/types/image"
)

// pullEvent Docker 拉取事件的 JSON 结构
type pullEvent struct {
	Status         string `json:"status"`
	ID             string `json:"id"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
	Error string `json:"error"`
}

// PullImageWithProgress 带进度的镜像拉取
// 返回进度通道，调用方通过通道接收进度更新
func (c *LocalClient) PullImageWithProgress(ctx context.Context, imageRef string) (<-chan PullProgress, error) {
	if c == nil || c.cli == nil {
		return nil, ErrClientNotInitialized
	}

	progressChan := make(chan PullProgress, 10)

	go func() {
		defer close(progressChan)

		// 发送初始状态
		progress := NewPullProgress(imageRef)
		progress.Message = "正在连接..."
		progressChan <- *progress

		// 开始拉取
		reader, err := c.cli.ImagePull(ctx, imageRef, dockerimage.PullOptions{})
		if err != nil {
			progress.Status = PullStatusError
			progress.Error = err
			progress.Message = "拉取失败: " + err.Error()
			progressChan <- *progress
			return
		}
		defer reader.Close()

		// 解析进度流
		c.parsePullProgress(ctx, reader, progress, progressChan)
	}()

	return progressChan, nil
}

// parsePullProgress 解析 Docker 拉取输出流
func (c *LocalClient) parsePullProgress(ctx context.Context, reader io.Reader, progress *PullProgress, out chan<- PullProgress) {
	scanner := bufio.NewScanner(reader)
	// 增大缓冲区以处理长行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			progress.Status = PullStatusError
			progress.Error = ctx.Err()
			progress.Message = "操作已取消"
			out <- *progress
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		var event pullEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		// 处理错误
		if event.Error != "" {
			progress.Status = PullStatusError
			progress.Error = &pullError{message: event.Error}
			progress.Message = event.Error
			out <- *progress
			return
		}

		// 解析状态
		c.handlePullEvent(&event, progress)

		// 发送进度更新
		out <- *progress
	}

	// 检查扫描错误
	if err := scanner.Err(); err != nil {
		progress.Status = PullStatusError
		progress.Error = err
		progress.Message = "读取进度失败: " + err.Error()
		out <- *progress
		return
	}

	// 拉取完成
	if progress.Status != PullStatusError {
		progress.Status = PullStatusComplete
		progress.Percentage = 100
		progress.Message = "拉取完成"
		out <- *progress
	}
}

// handlePullEvent 处理单个拉取事件
func (c *LocalClient) handlePullEvent(event *pullEvent, progress *PullProgress) {
	status := strings.ToLower(event.Status)

	// 更新消息
	if event.ID != "" {
		progress.Message = event.ID + ": " + event.Status
	} else {
		progress.Message = event.Status
	}

	// 无层 ID 的全局状态
	if event.ID == "" {
		if strings.Contains(status, "pulling from") || strings.Contains(status, "pulling") {
			progress.Status = PullStatusPending
		} else if strings.Contains(status, "digest") || strings.Contains(status, "status") {
			// 最终状态消息
			if strings.Contains(status, "downloaded") || strings.Contains(status, "up to date") {
				progress.Status = PullStatusComplete
				progress.Percentage = 100
			}
		}
		return
	}

	// 有层 ID 的状态
	layerStatus := PullStatusPending
	current := event.ProgressDetail.Current
	total := event.ProgressDetail.Total
	
	switch {
	case strings.Contains(status, "downloading"):
		layerStatus = PullStatusDownloading
		progress.Status = PullStatusDownloading
	case strings.Contains(status, "extracting"):
		layerStatus = PullStatusExtracting
		progress.Status = PullStatusExtracting
	case strings.Contains(status, "pull complete"):
		layerStatus = PullStatusComplete
		// 对于完成的层，如果没有 total，从已有层信息获取或设置为 current
		if layer, exists := progress.Layers[event.ID]; exists && layer.Total > 0 {
			total = layer.Total
			current = total
		} else if current > 0 {
			total = current
		}
	case strings.Contains(status, "already exists"):
		layerStatus = PullStatusComplete
		// 已存在的层，设置一个虚拟大小表示完成
		current = 1
		total = 1
	case strings.Contains(status, "waiting"):
		layerStatus = PullStatusPending
	case strings.Contains(status, "verifying"):
		layerStatus = PullStatusComplete
	}

	// 更新层进度
	progress.UpdateLayer(
		event.ID,
		layerStatus,
		current,
		total,
	)
}

// pullError 拉取错误
type pullError struct {
	message string
}

func (e *pullError) Error() string {
	return e.message
}

// ErrClientNotInitialized 客户端未初始化错误
var ErrClientNotInitialized = &pullError{message: "Docker 客户端未初始化"}
