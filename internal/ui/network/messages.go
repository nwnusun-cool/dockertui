package network

import "docktui/internal/docker"

// NetworksLoadedMsg 网络列表加载完成消息
type NetworksLoadedMsg struct {
	Networks []docker.Network
}

// NetworksLoadErrorMsg 网络列表加载错误消息
type NetworksLoadErrorMsg struct {
	Err error
}

// NetworkOperationSuccessMsg 网络操作成功消息
type NetworkOperationSuccessMsg struct {
	Operation string
	Network   string
}

// NetworkOperationErrorMsg 网络操作错误消息
type NetworkOperationErrorMsg struct {
	Operation string
	Err       error
}

// NetworkInspectMsg 网络检查消息
type NetworkInspectMsg struct {
	NetworkName string
	JSONContent string
}

// NetworkInspectErrorMsg 网络检查错误消息
type NetworkInspectErrorMsg struct {
	Err error
}

// NetworkDetailLoadedMsg 网络详情加载完成消息
type NetworkDetailLoadedMsg struct {
	Details *docker.NetworkDetails
}

// NetworkDetailLoadErrorMsg 网络详情加载错误消息
type NetworkDetailLoadErrorMsg struct {
	Err error
}

// NetworkCreateSuccessMsg 网络创建成功消息
type NetworkCreateSuccessMsg struct {
	NetworkID string
}

// NetworkCreateErrorMsg 网络创建错误消息
type NetworkCreateErrorMsg struct {
	Err error
}

// ClearSuccessMessageMsg 清除成功消息
type ClearSuccessMessageMsg struct{}

// ViewNetworkDetailsMsg 查看网络详情消息
type ViewNetworkDetailsMsg struct {
	Network *docker.Network
}

// GoBackMsg 返回上一级消息
type GoBackMsg struct{}
