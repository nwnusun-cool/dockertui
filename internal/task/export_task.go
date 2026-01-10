package task

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"docktui/internal/docker"
)

// ExportMode 导出模式
type ExportMode int

const (
	ExportModeSingle   ExportMode = iota // 单文件（所有镜像打包）
	ExportModeMultiple                   // 多文件（每个镜像单独导出）
)

// ExportImageInfo 导出镜像信息
type ExportImageInfo struct {
	ID         string
	Repository string
	Tag        string
}

// ExportTask 镜像导出任务
type ExportTask struct {
	*BaseTask
	dockerClient docker.Client
	images       []ExportImageInfo
	exportDir    string
	exportMode   ExportMode
	compress     bool
	
	// 导出结果
	exportedFiles []string
	totalSize     int64
}

// NewExportTask 创建镜像导出任务
func NewExportTask(client docker.Client, images []ExportImageInfo, dir string, mode ExportMode, compress bool) *ExportTask {
	taskID := GenerateTaskID()
	name := fmt.Sprintf("导出 %d 个镜像", len(images))
	if len(images) == 1 {
		imgName := images[0].Repository
		if imgName == "<none>" {
			imgName = images[0].ID[:12]
		}
		name = fmt.Sprintf("导出 %s", imgName)
	}
	
	return &ExportTask{
		BaseTask:     NewBaseTask(taskID, name),
		dockerClient: client,
		images:       images,
		exportDir:    dir,
		exportMode:   mode,
		compress:     compress,
	}
}

// Run 执行导出任务
func (t *ExportTask) Run(ctx context.Context) error {
	t.SetStatus(StatusRunning)
	t.SetMessage("准备导出...")

	// 创建导出目录
	if err := os.MkdirAll(t.exportDir, 0755); err != nil {
		t.SetStatus(StatusFailed)
		t.SetError(err)
		t.SetMessage("创建目录失败: " + err.Error())
		return err
	}

	manager := GetManager()

	if t.exportMode == ExportModeSingle {
		// 单文件模式
		return t.exportSingleFile(ctx, manager)
	}
	
	// 多文件模式
	return t.exportMultipleFiles(ctx, manager)
}

// exportSingleFile 导出为单个文件
func (t *ExportTask) exportSingleFile(ctx context.Context, manager *Manager) error {
	// 收集所有镜像 ID
	var imageIDs []string
	for _, img := range t.images {
		imageIDs = append(imageIDs, img.ID)
	}

	// 生成文件名
	filename := "images_export"
	if len(t.images) == 1 {
		filename = t.generateFilename(t.images[0])
	}

	ext := ".tar"
	if t.compress {
		ext = ".tar.gz"
	}
	filepath := t.exportDir + "/" + filename + ext

	t.SetMessage(fmt.Sprintf("正在导出到 %s...", filename+ext))
	manager.EmitProgress(t.ID(), t.Name(), 10, t.Message())

	// 调用 Docker API 导出
	reader, err := t.dockerClient.SaveImage(ctx, imageIDs)
	if err != nil {
		t.SetStatus(StatusFailed)
		t.SetError(err)
		t.SetMessage("导出失败: " + err.Error())
		return err
	}
	defer reader.Close()

	// 创建输出文件
	file, err := os.Create(filepath)
	if err != nil {
		t.SetStatus(StatusFailed)
		t.SetError(err)
		t.SetMessage("创建文件失败: " + err.Error())
		return err
	}
	defer file.Close()

	var writer io.Writer = file
	var gzWriter *gzip.Writer
	if t.compress {
		gzWriter = gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = gzWriter
	}

	t.SetProgress(30)
	t.SetMessage("正在写入文件...")
	manager.EmitProgress(t.ID(), t.Name(), 30, t.Message())

	// 写入文件
	written, err := io.Copy(writer, reader)
	if err != nil {
		t.SetStatus(StatusFailed)
		t.SetError(err)
		t.SetMessage("写入失败: " + err.Error())
		return err
	}

	t.totalSize = written
	t.exportedFiles = append(t.exportedFiles, filepath)

	t.SetStatus(StatusCompleted)
	t.SetProgress(100)
	t.SetMessage(fmt.Sprintf("导出完成: %s (%s)", filename+ext, formatBytes(written)))
	manager.EmitProgress(t.ID(), t.Name(), 100, t.Message())

	return nil
}

// exportMultipleFiles 导出为多个文件
func (t *ExportTask) exportMultipleFiles(ctx context.Context, manager *Manager) error {
	total := len(t.images)
	
	for i, img := range t.images {
		select {
		case <-ctx.Done():
			t.SetStatus(StatusCancelled)
			t.SetMessage("已取消")
			return ctx.Err()
		default:
		}

		// 计算进度
		progress := float64(i) / float64(total) * 100
		
		// 生成文件名
		filename := t.generateFilename(img)
		ext := ".tar"
		if t.compress {
			ext = ".tar.gz"
		}
		filepath := t.exportDir + "/" + filename + ext

		t.SetProgress(progress)
		t.SetMessage(fmt.Sprintf("[%d/%d] 正在导出 %s...", i+1, total, filename))
		manager.EmitProgress(t.ID(), t.Name(), progress, t.Message())

		// 调用 Docker API 导出
		reader, err := t.dockerClient.SaveImage(ctx, []string{img.ID})
		if err != nil {
			// 单个失败不中断整体，继续下一个
			continue
		}

		// 创建输出文件
		file, err := os.Create(filepath)
		if err != nil {
			reader.Close()
			continue
		}

		var writer io.Writer = file
		var gzWriter *gzip.Writer
		if t.compress {
			gzWriter = gzip.NewWriter(file)
			writer = gzWriter
		}

		// 写入文件
		written, err := io.Copy(writer, reader)
		
		// 关闭资源
		if gzWriter != nil {
			gzWriter.Close()
		}
		file.Close()
		reader.Close()

		if err == nil {
			t.totalSize += written
			t.exportedFiles = append(t.exportedFiles, filepath)
		}
	}

	t.SetStatus(StatusCompleted)
	t.SetProgress(100)
	t.SetMessage(fmt.Sprintf("导出完成: %d 个文件 (%s)", len(t.exportedFiles), formatBytes(t.totalSize)))
	manager.EmitProgress(t.ID(), t.Name(), 100, t.Message())

	return nil
}

// generateFilename 生成文件名
func (t *ExportTask) generateFilename(img ExportImageInfo) string {
	name := img.Repository
	if name == "<none>" {
		name = img.ID[:12]
	} else {
		name = strings.ReplaceAll(name, "/", "_")
		if img.Tag != "<none>" {
			name += "_" + img.Tag
		}
	}
	return name
}

// GetExportedFiles 获取导出的文件列表
func (t *ExportTask) GetExportedFiles() []string {
	return t.exportedFiles
}

// GetTotalSize 获取导出的总大小
func (t *ExportTask) GetTotalSize() int64 {
	return t.totalSize
}

// formatBytes 格式化字节大小
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
