package ui

import "docktui/internal/docker"

// GoBackMsg 请求返回上一级视图的消息
// 当视图处理 ESC 键且没有弹窗/对话框需要关闭时，发送此消息
type GoBackMsg struct{}

// ========== 视图切换请求消息 ==========

// ViewImageDetailsMsg 请求切换到镜像详情视图
type ViewImageDetailsMsg struct {
	Image *docker.Image
}

// ViewContainerDetailsMsg 请求切换到容器详情视图
type ViewContainerDetailsMsg struct {
	ContainerID   string
	ContainerName string
}

// ViewContainerLogsMsg 请求切换到容器日志视图
type ViewContainerLogsMsg struct {
	ContainerID   string
	ContainerName string
}

// ViewNetworkDetailsMsg 请求切换到网络详情视图
type ViewNetworkDetailsMsg struct {
	Network *docker.Network
}
