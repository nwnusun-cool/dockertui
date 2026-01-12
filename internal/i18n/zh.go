package i18n

var zhMessages = &Messages{
	// 通用
	Loading: "加载中...",
	Error:   "错误",
	Success: "成功",
	Confirm: "确认",
	Cancel:  "取消",
	Refresh: "刷新",
	Back:    "返回",
	Exit:    "退出",
	Help:    "帮助",
	Search:  "搜索",
	Export:  "导出",
	Select:  "选择",
	Enter:   "进入",
	Delete:  "删除",
	Create:  "创建",
	Remove:  "移除",
	Clear:   "清除",
	Filter:  "筛选",
	OK:      "确定",

	// 状态
	Connected:    "已连接",
	Disconnected: "未连接",
	Running:      "运行中",
	Stopped:      "已停止",
	Paused:       "已暂停",
	Available:    "可用",
	Unavailable:  "不可用",

	// 首页
	LocalDocker: "本地 Docker",

	// 资源
	Containers: "容器",
	Images:     "镜像",
	Networks:   "网络",
	Volumes:    "卷",
	Compose:    "Compose",

	// 日志视图
	LogsTitle:     "日志",
	Follow:        "跟随",
	FollowMode:    "跟随模式",
	Wrap:          "换行",
	Lines:         "行数",
	NoLogs:        "暂无日志",
	LoadingLogs:   "正在加载日志...",
	LoadFailed:    "加载失败",
	ExportTo:      "导出到",
	ExportSuccess: "已导出到",
	ExportFailed:  "导出失败",
	SearchNext:    "下一个",
	SearchPrev:    "上一个",
	SearchClear:   "清除",

	// 容器操作
	Start:   "启动",
	Stop:    "停止",
	Restart: "重启",
	Pause:   "暂停",
	Unpause: "恢复",
	Shell:   "终端",
	Logs:    "日志",
	Inspect: "详情",

	// 提示
	PressToConfirm: "Enter=确认",
	PressToCancel:  "ESC=取消",
	ScrollHint:     "滚动",
	JumpHint:       "首/尾",
	UnknownView:    "未知视图",

	// Shell
	EnterShell:      "进入容器 Shell",
	ShellTips:       "提示:",
	ShellExitHint:   "输入 exit 或按 Ctrl+D 退出 shell",
	ShellReturnHint: "退出后将自动返回 DockTUI",
	ShellExecFailed: "Shell 执行失败",
	UsingSDKMode:    "使用 Docker SDK 模式...",

	// 错误消息
	FeatureUnavailable:   "该功能暂不可用",
	FeatureInDevelopment: "该功能开发中...",
	VolumeInDevelopment:  "卷管理功能开发中...",
	ComposeUnavailable:   "Docker Compose 未安装或不可用",
	ViewNotInitialized:   "视图未初始化",
	SelectContainerFirst: "请先选择一个容器",
	OnlyRunningContainer: "只能在运行中的容器执行 shell",
	ViewError:            "视图错误",
	SelectFirst:          "请先选择一个项目",
	FatalError:           "致命错误",
	Warning:              "警告",

	// 加载状态
	LoadingNetworks:   "正在加载网络列表...",
	LoadingImages:     "正在加载镜像列表...",
	LoadingContainers: "正在加载容器列表...",
	PleaseWait:        "请稍候，正在从 Docker 获取数据",

	// 错误状态
	LoadFailedReason:  "加载失败",
	PossibleReasons:   "可能的原因:",
	DockerNotRunning:  "Docker 守护进程未运行",
	NetworkIssue:      "网络连接问题",
	PermissionDenied:  "权限不足",

	// 空状态
	NoNetworks:     "暂无自定义网络",
	NoImages:       "暂无镜像",
	NoContainers:   "暂无容器",
	QuickStart:     "快速开始:",
	CreateNetwork:  "创建一个网络:",
	PullImageHint:  "拉取一个镜像:",
	RefreshList:    "按 r 键刷新",
	OrPress:        "或",

	// 操作
	OperationSuccess: "成功",
	OperationFailed:  "失败",
	GetInfoFailed:    "获取信息失败",
	CreateSuccess:    "创建成功",
	DeleteSuccess:    "删除成功",
	PruneSuccess:     "清理成功",
	ExportedTo:       "已导出到",
	Exporting:        "正在导出",
	TaskCancelled:    "任务已取消",
	CancellingTask:   "正在取消任务...",
	StartPulling:     "开始拉取",

	// 确认对话框
	ConfirmDeleteNetwork: "确认删除网络",
	ConfirmPruneNetworks: "确认清理网络",
	ConfirmDeleteImage:   "删除镜像",
	ForceDeleteImage:     "强制删除镜像",
	ImageInUseWarning:    "该镜像正在被容器使用！",
	PruneDanglingImages:  "清理悬垂镜像",
	ConfirmPullImage:     "拉取镜像",
	CannotUndo:           "此操作不可撤销！",

	// 已选择
	Selected: "已选",

	// 按键提示
	UpDown:       "j/k=上下",
	EnterDetails: "Enter=详情",
	EscBack:      "Esc=返回",
	QuitApp:      "q=退出",
	Sort:         "s=排序",
}
