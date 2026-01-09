package ui

// GoBackMsg 请求返回上一级视图的消息
// 当视图处理 ESC 键且没有弹窗/对话框需要关闭时，发送此消息
type GoBackMsg struct{}
