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

type detailClearMessageMsg struct{}
