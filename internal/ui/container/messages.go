package container

import "docktui/internal/docker"

// ========== 列表视图消息 ==========

// ContainersLoadedMsg 容器列表加载完成消息
type ContainersLoadedMsg struct {
	Containers []docker.Container
}

// ContainersLoadErrorMsg 容器列表加载错误消息
type ContainersLoadErrorMsg struct {
	Err error
}

// ContainerEventMsg Docker 容器事件消息
type ContainerEventMsg struct {
	Event docker.ContainerEvent
}

// ContainerEventErrorMsg Docker 事件监听错误消息
type ContainerEventErrorMsg struct {
	Err error
}

// ContainerOperationSuccessMsg 容器操作成功消息
type ContainerOperationSuccessMsg struct {
	Operation string // 操作类型: start, stop, restart
	Container string // 容器名称
}

// ContainerOperationErrorMsg 容器操作失败消息
type ContainerOperationErrorMsg struct {
	Operation string // 操作类型
	Container string // 容器名称
	Err       error
}

// ContainerOperationWarningMsg 容器操作警告消息
type ContainerOperationWarningMsg struct {
	Message string
}

// ContainerBatchOperationMsg 容器批量操作结果消息
type ContainerBatchOperationMsg struct {
	Operation    string
	SuccessCount int
	FailedCount  int
	FailedNames  []string
	Err          error
}

// ClearSuccessMessageMsg 清除成功消息
type ClearSuccessMessageMsg struct{}

// ContainerInspectMsg 容器检查结果消息
type ContainerInspectMsg struct {
	ContainerName string
	JSONContent   string
}

// ContainerInspectErrorMsg 容器检查错误消息
type ContainerInspectErrorMsg struct {
	Err error
}

// ContainerEditReadyMsg 容器编辑准备就绪消息
type ContainerEditReadyMsg struct {
	Container *docker.Container
	Details   *docker.ContainerDetails
}

// ContainerUpdateSuccessMsg 容器更新成功消息
type ContainerUpdateSuccessMsg struct {
	Container string
}

// ========== 详情视图消息 ==========

// DetailsLoadedMsg 详情加载完成消息
type DetailsLoadedMsg struct {
	Details *docker.ContainerDetails
}

// DetailsLoadErrorMsg 详情加载错误消息
type DetailsLoadErrorMsg struct {
	Err error
}

// ========== 视图切换消息 ==========

// GoBackMsg 请求返回上一级视图
type GoBackMsg struct{}

// ViewDetailsMsg 请求切换到容器详情视图
type ViewDetailsMsg struct {
	ContainerID   string
	ContainerName string
}

// ViewLogsMsg 请求切换到容器日志视图
type ViewLogsMsg struct {
	ContainerID   string
	ContainerName string
}
