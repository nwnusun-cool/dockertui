package compose

import composelib "docktui/internal/compose"

// GoToDetailMsg 请求切换到 Compose 项目详情视图
type GoToDetailMsg struct {
	Project *composelib.Project
}

// GoToContainerDetailMsg 跳转到容器详情的消息
type GoToContainerDetailMsg struct {
	ContainerID   string
	ContainerName string
}

// GoToContainerLogsMsg 跳转到容器日志的消息
type GoToContainerLogsMsg struct {
	ContainerID   string
	ContainerName string
}

// ExecContainerShellMsg 执行容器 Shell 的消息
type ExecContainerShellMsg struct {
	ContainerID   string
	ContainerName string
}

// GoBackMsg 请求返回上一级视图
type GoBackMsg struct{}

// 列表视图内部消息
type listScanResultMsg struct {
	projects []*composelib.Project
	err      error
}

type listOperationResultMsg struct {
	message string
	err     error
}

type listRefreshStatusMsg struct {
	projects []*composelib.Project
}

type listClearMessageMsg struct{}

// 详情视图内部消息
type detailServicesMsg struct {
	services []composelib.Service
	err      error
}

type detailConfigFilesMsg struct {
	envContent  string
	envFileName string
	ymlContent  string
	ymlFileName string
	err         error
}

type detailOperationMsg struct {
	message string
	err     error
}

// detailOperationLogMsg 操作日志消息
type detailOperationLogMsg struct {
	line string
}

// detailOperationDoneMsg 操作完成消息
type detailOperationDoneMsg struct {
	result *composelib.OperationResult
}

type detailClearMessageMsg struct{}
