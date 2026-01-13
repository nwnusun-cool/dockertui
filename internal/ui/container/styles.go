package container

import (
	"docktui/internal/ui/styles"
)

// 容器模块样式 - 引用全局样式
var (
	// 状态栏
	StatusBarLabelStyle = styles.StatusBarLabelStyle
	StatusBarValueStyle = styles.StatusBarValueStyle
	StatusBarKeyStyle   = styles.StatusBarKeyStyle

	// 标题
	TitleStyle = styles.TitleStyle

	// 过滤状态
	FilterAllStyle     = styles.TextStyle
	FilterRunningStyle = styles.SuccessStyle
	FilterExitedStyle  = styles.MutedStyle

	// 消息
	SuccessMsgStyle = styles.SuccessStyle
	ErrorMsgStyle   = styles.ErrorStyle

	// 搜索
	SearchPromptStyle = styles.SearchPromptStyle
	SearchHintStyle   = styles.SearchHintStyle

	// 对话框
	DialogStyle        = styles.DialogStyle
	DialogTitleStyle   = styles.TitleStyle
	DialogWarningStyle = styles.MutedStyle

	// 按钮
	ButtonActiveStyle   = styles.ButtonActiveStyle
	ButtonInactiveStyle = styles.ButtonInactiveStyle

	// 状态框
	StateBoxStyle = styles.StateBoxStyle

	// 详情视图
	DetailBoxStyle   = styles.BoxStyle
	DetailTitleStyle = styles.TitleStyle
	DetailHintStyle  = styles.HintStyle
)
