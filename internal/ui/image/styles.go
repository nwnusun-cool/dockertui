package image

import (
	"docktui/internal/ui/styles"
)

// 镜像模块样式 - 引用全局样式
var (
	// 状态栏
	StatusBarLabelStyle = styles.StatusBarLabelStyle
	StatusBarKeyStyle   = styles.StatusBarKeyStyle

	// 标题
	TitleStyle = styles.TitleStyle

	// 镜像状态
	DanglingStyle = styles.MutedStyle
	ActiveStyle   = styles.ActiveStyle
	UnusedStyle   = styles.MutedStyle

	// 消息
	SuccessMsgStyle = styles.SuccessStyle
	ErrorMsgStyle   = styles.ErrorStyle

	// 搜索
	SearchPromptStyle = styles.SearchPromptStyle
	SearchHintStyle   = styles.SearchHintStyle

	// 状态框
	StateBoxStyle = styles.StateBoxStyle

	// 详情视图
	DetailsTitleStyle = styles.TitleStyle
	DetailsLabelStyle = styles.LabelStyle
	DetailsValueStyle = styles.ValueStyle
	DetailsBoxStyle   = styles.BoxStyle
	DetailsHintStyle  = styles.HintStyle
	DetailsKeyStyle   = styles.KeyStyle

	// 标签页
	TabActiveStyle   = styles.TabActiveStyle
	TabInactiveStyle = styles.TabInactiveStyle

	// 容器状态
	ContainerRunningStyle = styles.RunningStyle
	ContainerStoppedStyle = styles.StoppedStyle
)
