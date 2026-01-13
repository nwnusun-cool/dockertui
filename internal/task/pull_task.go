package task

import (
	"context"
	"fmt"

	"docktui/internal/docker"
)

// PullTask 镜像拉取任务
type PullTask struct {
	*BaseTask
	imageRef     string
	dockerClient docker.Client
	progress     docker.PullProgress
}

// PullClient 拉取客户端接口（用于类型断言）
type PullClient interface {
	PullImageWithProgress(ctx context.Context, imageRef string) (<-chan docker.PullProgress, error)
}

// NewPullTask 创建镜像拉取任务
func NewPullTask(client docker.Client, imageRef string) *PullTask {
	taskID := GenerateTaskID()
	return &PullTask{
		BaseTask:     NewBaseTask(taskID, fmt.Sprintf("Pull %s", imageRef)),
		imageRef:     imageRef,
		dockerClient: client,
	}
}

// ImageRef 返回镜像引用
func (t *PullTask) ImageRef() string {
	return t.imageRef
}

// GetProgress 获取拉取进度
func (t *PullTask) GetProgress() docker.PullProgress {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.progress
}

// Run 执行拉取任务
func (t *PullTask) Run(ctx context.Context) error {
	t.SetStatus(StatusRunning)
	t.SetMessage("Connecting...")

	// 类型断言获取带进度的拉取方法
	pullClient, ok := t.dockerClient.(*docker.LocalClient)
	if !ok {
		err := fmt.Errorf("client does not support progress pull")
		t.SetStatus(StatusFailed)
		t.SetError(err)
		return err
	}

	// 开始拉取
	progressChan, err := pullClient.PullImageWithProgress(ctx, t.imageRef)
	if err != nil {
		t.SetStatus(StatusFailed)
		t.SetError(err)
		t.SetMessage("Pull failed: " + err.Error())
		return err
	}

	// 获取任务管理器用于发送进度事件
	manager := GetManager()

	// 处理进度更新
	for progress := range progressChan {
		select {
		case <-ctx.Done():
			t.SetStatus(StatusCancelled)
			t.SetMessage("Cancelled")
			return ctx.Err()
		default:
		}

		// 更新内部进度
		t.mu.Lock()
		t.progress = progress
		t.mu.Unlock()

		// 更新任务状态
		t.SetProgress(progress.Percentage)
		t.SetMessage(progress.Message)

		// 发送进度事件
		manager.EmitProgress(t.ID(), t.Name(), progress.Percentage, progress.Message)

		// 检查错误
		if progress.Error != nil {
			t.SetStatus(StatusFailed)
			t.SetError(progress.Error)
			return progress.Error
		}

		// 检查完成
		if progress.Status == docker.PullStatusComplete {
			t.SetStatus(StatusCompleted)
			t.SetProgress(100)
			t.SetMessage("Pull completed")
			return nil
		}
	}

	// 通道关闭，检查最终状态
	if t.Status() != StatusCompleted && t.Status() != StatusFailed {
		t.SetStatus(StatusCompleted)
		t.SetProgress(100)
		t.SetMessage("Pull completed")
	}

	return t.Error()
}
