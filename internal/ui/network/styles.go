package network

import (
	"docktui/internal/ui/styles"
)

// 网络模块样式 - 引用全局样式
var (
	// 标题
	TitleStyle = styles.TitleStyle

	// 网络类型
	DriverStyle  = styles.KeyStyle
	BuiltInStyle = styles.MutedStyle
	CustomStyle  = styles.ActiveStyle

	// 状态框
	StateBoxStyle = styles.StateBoxStyle

	// 详情视图
	DetailTitleStyle = styles.TitleStyle
	DetailLabelStyle = styles.LabelStyle
	DetailValueStyle = styles.ValueStyle
	DetailBoxStyle   = styles.BoxStyle
	DetailHintStyle  = styles.HintStyle
	DetailKeyStyle   = styles.KeyStyle

	// 标签页
	TabActiveStyle   = styles.TabActiveStyle
	TabInactiveStyle = styles.TabInactiveStyle

	// 容器状态
	ContainerRunningStyle = styles.RunningStyle
	ContainerStoppedStyle = styles.StoppedStyle

	// 表单
	FormTitleStyle       = styles.FormTitleStyle
	FormLabelStyle       = styles.FormLabelStyle
	FormInputStyle       = styles.FormInputStyle
	FormInputActiveStyle = styles.FormInputActiveStyle
	FormHintStyle        = styles.FormHintStyle
	FormCheckboxStyle    = styles.FormCheckboxStyle
	FormButtonStyle      = styles.ButtonInactiveStyle
	FormButtonActiveStyle = styles.ButtonActiveStyle
	FormErrorStyle       = styles.FormErrorStyle
)
