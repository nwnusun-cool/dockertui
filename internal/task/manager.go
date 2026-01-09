package task

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventType 事件类型
type EventType int

const (
	EventStarted EventType = iota
	EventProgress
	EventCompleted
	EventFailed
	EventCancelled
)

// Event 任务事件
type Event struct {
	TaskID   string
	TaskName string
	Type     EventType
	Progress float64
	Message  string
	Error    error
	Time     time.Time
}

// Manager 后台任务管理器
type Manager struct {
	tasks      map[string]Task
	mu         sync.RWMutex
	eventChan  chan Event
	subscribers []chan Event
	subMu      sync.RWMutex
}

var (
	globalManager *Manager
	once          sync.Once
)

// GetManager 获取全局任务管理器（单例）
func GetManager() *Manager {
	once.Do(func() {
		globalManager = &Manager{
			tasks:       make(map[string]Task),
			eventChan:   make(chan Event, 100),
			subscribers: make([]chan Event, 0),
		}
		go globalManager.dispatchEvents()
	})
	return globalManager
}

// dispatchEvents 分发事件到所有订阅者
func (m *Manager) dispatchEvents() {
	for event := range m.eventChan {
		m.subMu.RLock()
		for _, sub := range m.subscribers {
			select {
			case sub <- event:
			default:
				// 订阅者通道已满，跳过
			}
		}
		m.subMu.RUnlock()
	}
}

// Submit 提交任务
func (m *Manager) Submit(task Task) string {
	m.mu.Lock()
	m.tasks[task.ID()] = task
	m.mu.Unlock()

	// 发送开始事件
	m.emitEvent(Event{
		TaskID:   task.ID(),
		TaskName: task.Name(),
		Type:     EventStarted,
		Message:  "任务已提交",
		Time:     time.Now(),
	})

	// 启动任务
	go m.runTask(task)

	return task.ID()
}

// runTask 运行任务
func (m *Manager) runTask(task Task) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// 设置取消函数（如果任务支持）
	if bt, ok := task.(*PullTask); ok {
		bt.BaseTask.SetCancelFunc(cancel)
	}

	err := task.Run(ctx)

	// 发送完成/失败事件
	if err != nil {
		if task.Status() == StatusCancelled {
			m.emitEvent(Event{
				TaskID:   task.ID(),
				TaskName: task.Name(),
				Type:     EventCancelled,
				Message:  "任务已取消",
				Time:     time.Now(),
			})
		} else {
			m.emitEvent(Event{
				TaskID:   task.ID(),
				TaskName: task.Name(),
				Type:     EventFailed,
				Error:    err,
				Message:  err.Error(),
				Time:     time.Now(),
			})
		}
	} else {
		m.emitEvent(Event{
			TaskID:   task.ID(),
			TaskName: task.Name(),
			Type:     EventCompleted,
			Progress: 100,
			Message:  "任务完成",
			Time:     time.Now(),
		})
	}

	cancel() // 清理 context
}

// emitEvent 发送事件
func (m *Manager) emitEvent(event Event) {
	select {
	case m.eventChan <- event:
	default:
		// 事件通道已满，跳过
	}
}

// EmitProgress 发送进度事件（供任务调用）
func (m *Manager) EmitProgress(taskID, taskName string, progress float64, message string) {
	m.emitEvent(Event{
		TaskID:   taskID,
		TaskName: taskName,
		Type:     EventProgress,
		Progress: progress,
		Message:  message,
		Time:     time.Now(),
	})
}

// Cancel 取消任务
func (m *Manager) Cancel(taskID string) error {
	m.mu.RLock()
	task, exists := m.tasks[taskID]
	m.mu.RUnlock()

	if !exists {
		return nil
	}

	task.Cancel()
	return nil
}

// GetTask 获取任务
func (m *Manager) GetTask(taskID string) Task {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tasks[taskID]
}

// ListActiveTasks 列出活跃任务
func (m *Manager) ListActiveTasks() []Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	active := make([]Task, 0)
	for _, task := range m.tasks {
		status := task.Status()
		if status == StatusPending || status == StatusRunning {
			active = append(active, task)
		}
	}
	return active
}

// ListAllTasks 列出所有任务
func (m *Manager) ListAllTasks() []Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	all := make([]Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		all = append(all, task)
	}
	return all
}

// Subscribe 订阅事件
func (m *Manager) Subscribe() <-chan Event {
	ch := make(chan Event, 50)
	m.subMu.Lock()
	m.subscribers = append(m.subscribers, ch)
	m.subMu.Unlock()
	return ch
}

// Unsubscribe 取消订阅
func (m *Manager) Unsubscribe(ch <-chan Event) {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	for i, sub := range m.subscribers {
		if sub == ch {
			m.subscribers = append(m.subscribers[:i], m.subscribers[i+1:]...)
			close(sub)
			break
		}
	}
}

// CleanupCompleted 清理已完成的任务
func (m *Manager) CleanupCompleted() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, task := range m.tasks {
		status := task.Status()
		if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
			delete(m.tasks, id)
		}
	}
}

// GenerateTaskID 生成任务 ID
func GenerateTaskID() string {
	return uuid.New().String()[:8]
}
