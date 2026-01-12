package image

import "docktui/internal/docker"

// ImagesLoadedMsg 镜像列表加载完成消息
type ImagesLoadedMsg struct {
	Images []docker.Image
}

// ImagesLoadErrorMsg 镜像列表加载错误消息
type ImagesLoadErrorMsg struct {
	Err error
}

// ImageOperationSuccessMsg 镜像操作成功消息
type ImageOperationSuccessMsg struct {
	Operation string
	Image     string
}

// ImageOperationErrorMsg 镜像操作错误消息
type ImageOperationErrorMsg struct {
	Operation string
	Image     string
	Err       error
}

// ImageInUseErrorMsg 镜像被容器引用错误消息
type ImageInUseErrorMsg struct {
	Image *docker.Image
	Err   error
}

// ImageInspectMsg 镜像检查消息
type ImageInspectMsg struct {
	ImageName   string
	JSONContent string
}

// ImageInspectErrorMsg 镜像检查错误消息
type ImageInspectErrorMsg struct {
	Err error
}

// ImageExportSuccessMsg 镜像导出成功消息
type ImageExportSuccessMsg struct {
	Count    int
	Dir      string
	FileSize int64
}

// ImageExportErrorMsg 镜像导出错误消息
type ImageExportErrorMsg struct {
	Err error
}

// ImageExportProgressMsg 镜像导出进度消息
type ImageExportProgressMsg struct {
	Current int
	Total   int
	Name    string
}

// ImageDetailsLoadedMsg 镜像详情加载完成消息
type ImageDetailsLoadedMsg struct {
	Details *docker.ImageDetails
}

// ImageDetailsLoadErrorMsg 镜像详情加载错误消息
type ImageDetailsLoadErrorMsg struct {
	Err error
}

// ClearSuccessMessageMsg 清除成功消息
type ClearSuccessMessageMsg struct{}

// TaskTickMsg 任务状态定时刷新消息
type TaskTickMsg struct{}

// ViewImageDetailsMsg 查看镜像详情消息
type ViewImageDetailsMsg struct {
	Image *docker.Image
}

// GoBackMsg 返回上一级消息
type GoBackMsg struct{}
