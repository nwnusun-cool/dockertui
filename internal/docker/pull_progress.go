package docker

import "time"

// PullStatus 拉取状态枚举
type PullStatus int

const (
	PullStatusPending PullStatus = iota
	PullStatusDownloading
	PullStatusExtracting
	PullStatusComplete
	PullStatusError
)

// String 返回状态的字符串表示
func (s PullStatus) String() string {
	switch s {
	case PullStatusPending:
		return "Pending"
	case PullStatusDownloading:
		return "Downloading"
	case PullStatusExtracting:
		return "Extracting"
	case PullStatusComplete:
		return "Complete"
	case PullStatusError:
		return "Error"
	default:
		return "Unknown"
	}
}

// LayerProgress 单层进度
type LayerProgress struct {
	ID         string     // 层 ID (短)
	Status     PullStatus // 状态
	Current    int64      // 已下载字节
	Total      int64      // 总字节
	Percentage float64    // 百分比
}

// PullProgress 整体拉取进度
type PullProgress struct {
	ImageRef   string                    // 镜像引用
	Status     PullStatus                // 整体状态
	Layers     map[string]*LayerProgress // 各层进度
	TotalSize  int64                     // 总大小
	Downloaded int64                     // 已下载
	Percentage float64                   // 整体百分比
	Speed      float64                   // 下载速度 (bytes/s)
	Error      error                     // 错误信息
	StartTime  time.Time                 // 开始时间
	Message    string                    // 当前状态消息
}

// NewPullProgress 创建新的拉取进度
func NewPullProgress(imageRef string) *PullProgress {
	return &PullProgress{
		ImageRef:  imageRef,
		Status:    PullStatusPending,
		Layers:    make(map[string]*LayerProgress),
		StartTime: time.Now(),
	}
}

// UpdateLayer 更新层进度
func (p *PullProgress) UpdateLayer(id string, status PullStatus, current, total int64) {
	layer, exists := p.Layers[id]
	if !exists {
		layer = &LayerProgress{ID: id}
		p.Layers[id] = layer
	}

	layer.Status = status
	layer.Current = current
	layer.Total = total

	if total > 0 {
		layer.Percentage = float64(current) / float64(total) * 100
	}

	// 重新计算整体进度
	p.recalculateTotal()
}

// recalculateTotal 重新计算整体进度
func (p *PullProgress) recalculateTotal() {
	var totalSize, downloaded int64
	var completedLayers, totalLayers int
	allComplete := true
	hasDownloading := false
	hasExtracting := false

	for _, layer := range p.Layers {
		totalLayers++
		
		// 只统计有大小信息的层
		if layer.Total > 0 {
			totalSize += layer.Total
			downloaded += layer.Current
		}

		if layer.Status == PullStatusComplete {
			completedLayers++
		} else {
			allComplete = false
		}
		
		if layer.Status == PullStatusDownloading {
			hasDownloading = true
		}
		if layer.Status == PullStatusExtracting {
			hasExtracting = true
		}
	}

	p.TotalSize = totalSize
	p.Downloaded = downloaded

	// 计算百分比：优先使用字节进度，否则使用层数进度
	if totalSize > 0 {
		p.Percentage = float64(downloaded) / float64(totalSize) * 100
	} else if totalLayers > 0 {
		// 如果没有字节信息，使用层数计算进度
		p.Percentage = float64(completedLayers) / float64(totalLayers) * 100
	}

	// 更新整体状态
	if allComplete && totalLayers > 0 {
		p.Status = PullStatusComplete
		p.Percentage = 100
	} else if hasExtracting {
		p.Status = PullStatusExtracting
	} else if hasDownloading {
		p.Status = PullStatusDownloading
	}

	// 计算速度
	elapsed := time.Since(p.StartTime).Seconds()
	if elapsed > 0 && downloaded > 0 {
		p.Speed = float64(downloaded) / elapsed
	}
}
