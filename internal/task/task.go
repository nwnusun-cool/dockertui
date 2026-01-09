package task

import (
	"context"
	"sync"
	"time"
)

// Status 任务状态
type Status int

const (
	StatusPending Status = iota
	StatusRunning
	StatusCompleted
	StatusFailed
	StatusCancelled
)

// String 返回状态字符串
func (s Status) String() string {
	switch s {
	case StatusPending:
		return "Pending"
	case StatusRunning:
		return "Running"
	case StatusCompleted:
		return "Completed"
	case StatusFailed:
		return "Failed"
	case StatusCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// Task 任务接口
type Task interface {
	// ID 返回任务唯一标识
	ID() string
	// Name 返回任务名称（用于显示）
	Name() string
	// Status 返回当前状态
	Status() Status
	// Progress 返回进度百分比 (0-100)
	Progress() float64
	// Message 返回当前状态消息
	Message() string
	// Error 返回错误信息（如果有）
	Error() error
	// Run 执行任务
	Run(ctx context.Context) error
	// Cancel 取消任务
	Cancel()
}

// BaseTask 任务基础实现
type BaseTask struct {
	id        string
	name      string
	status    Status
	progress  float64
	message   string
	err       error
	startTime time.Time
	endTime   time.Time
	cancelFn  context.CancelFunc
	mu        sync.RWMutex
}

// NewBaseTask 创建基础任务
func NewBaseTask(id, name string) *BaseTask {
	return &BaseTask{
		id:     id,
		name:   name,
		status: StatusPending,
	}
}

// ID 返回任务 ID
func (t *BaseTask) ID() string {
	return t.id
}

// Name 返回任务名称
func (t *BaseTask) Name() string {
	return t.name
}

// Status 返回任务状态
func (t *BaseTask) Status() Status {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// SetStatus 设置任务状态
func (t *BaseTask) SetStatus(status Status) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = status
	if status == StatusRunning && t.startTime.IsZero() {
		t.startTime = time.Now()
	}
	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
		t.endTime = time.Now()
	}
}

// Progress 返回进度
func (t *BaseTask) Progress() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.progress
}

// SetProgress 设置进度
func (t *BaseTask) SetProgress(progress float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.progress = progress
}

// Message 返回消息
func (t *BaseTask) Message() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.message
}

// SetMessage 设置消息
func (t *BaseTask) SetMessage(message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.message = message
}

// Error 返回错误
func (t *BaseTask) Error() error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.err
}

// SetError 设置错误
func (t *BaseTask) SetError(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.err = err
}

// SetCancelFunc 设置取消函数
func (t *BaseTask) SetCancelFunc(fn context.CancelFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancelFn = fn
}

// Cancel 取消任务
func (t *BaseTask) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancelFn != nil {
		t.cancelFn()
	}
	t.status = StatusCancelled
}

// Run 基础实现（子类需要覆盖）
func (t *BaseTask) Run(ctx context.Context) error {
	return nil
}

// Duration 返回任务执行时长
func (t *BaseTask) Duration() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.startTime.IsZero() {
		return 0
	}
	if t.endTime.IsZero() {
		return time.Since(t.startTime)
	}
	return t.endTime.Sub(t.startTime)
}
