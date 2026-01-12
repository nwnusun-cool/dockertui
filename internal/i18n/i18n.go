package i18n

import (
	"os"
	"strings"
)

// Language type
type Language string

const (
	EN Language = "en"
	ZH Language = "zh"
)

var current = EN

// Messages all text messages
type Messages struct {
	// Common
	Loading   string
	Error     string
	Success   string
	Confirm   string
	Cancel    string
	Refresh   string
	Back      string
	Exit      string
	Help      string
	Search    string
	Export    string
	Select    string
	Enter     string
	Delete    string
	Create    string
	Remove    string
	Clear     string
	Filter    string
	OK        string
	
	// Status
	Connected    string
	Disconnected string
	Running      string
	Stopped      string
	Paused       string
	Available    string
	Unavailable  string
	
	// Home
	LocalDocker  string
	
	// Resources
	Containers string
	Images     string
	Networks   string
	Volumes    string
	Compose    string
	
	// Logs view
	LogsTitle      string
	Follow         string
	FollowMode     string
	Wrap           string
	Lines          string
	NoLogs         string
	LoadingLogs    string
	LoadFailed     string
	ExportTo       string
	ExportSuccess  string
	ExportFailed   string
	SearchNext     string
	SearchPrev     string
	SearchClear    string
	
	// Container operations
	Start     string
	Stop      string
	Restart   string
	Pause     string
	Unpause   string
	Shell     string
	Logs      string
	Inspect   string
	
	// List view
	LoadingList    string
	NoData         string
	PressToReload  string
	
	// Network
	NetworkName         string
	Subnet              string
	Gateway             string
	Internal            string
	CannotDeleteBuiltIn string
	NetworkInUse        string
	ConfirmDelete       string
	ConfirmPrune        string
	
	// Image
	PullImage        string
	TagImage         string
	ExportImage      string
	PruneImages      string
	NoDanglingImages string
	
	// Time
	JustNow    string
	MinutesAgo string
	HoursAgo   string
	DaysAgo    string
	
	// Hints
	PressToConfirm string
	PressToCancel  string
	ScrollHint     string
	JumpHint       string
	UnknownView    string
	
	// Shell
	EnterShell       string
	ShellTips        string
	ShellExitHint    string
	ShellReturnHint  string
	ShellExecFailed  string
	UsingSDKMode     string
	
	// Error messages
	FeatureUnavailable    string
	FeatureInDevelopment  string
	VolumeInDevelopment   string
	ComposeUnavailable    string
	ViewNotInitialized    string
	SelectContainerFirst  string
	OnlyRunningContainer  string
	ViewError             string
	SelectFirst           string
	FatalError            string
	Warning               string
	
	// Loading states
	LoadingNetworks   string
	LoadingImages     string
	LoadingContainers string
	PleaseWait        string
	
	// Error states
	LoadFailedReason  string
	PossibleReasons   string
	DockerNotRunning  string
	NetworkIssue      string
	PermissionDenied  string
	
	// Empty states
	NoNetworks        string
	NoImages          string
	NoContainers      string
	QuickStart        string
	CreateNetwork     string
	PullImageHint     string
	RefreshList       string
	OrPress           string
	
	// Operations
	OperationSuccess  string
	OperationFailed   string
	GetInfoFailed     string
	CreateSuccess     string
	DeleteSuccess     string
	PruneSuccess      string
	ExportedTo        string
	Exporting         string
	TaskCancelled     string
	CancellingTask    string
	StartPulling      string
	
	// Confirm dialogs
	ConfirmDeleteNetwork string
	ConfirmPruneNetworks string
	ConfirmDeleteImage   string
	ForceDeleteImage     string
	ImageInUseWarning    string
	PruneDanglingImages  string
	ConfirmPullImage     string
	CannotUndo           string
	
	// Selected
	Selected string
	
	// Hints for keys
	UpDown       string
	EnterDetails string
	EscBack      string
	QuitApp      string
	Sort         string
}

var messages = map[Language]*Messages{
	EN: enMessages,
	ZH: zhMessages,
}

// SetLanguage set current language
func SetLanguage(lang Language) {
	if _, ok := messages[lang]; ok {
		current = lang
	}
}

// GetLanguage get current language
func GetLanguage() Language {
	return current
}

// ToggleLanguage toggle between EN and ZH
func ToggleLanguage() Language {
	if current == EN {
		current = ZH
	} else {
		current = EN
	}
	return current
}

// GetLanguageDisplay get display name for current language
func GetLanguageDisplay() string {
	if current == ZH {
		return "中文"
	}
	return "EN"
}

// T get translated text
func T(key string) string {
	m := messages[current]
	if m == nil {
		m = messages[EN]
	}
	
	switch key {
	// Common
	case "loading":
		return m.Loading
	case "error":
		return m.Error
	case "success":
		return m.Success
	case "confirm":
		return m.Confirm
	case "cancel":
		return m.Cancel
	case "refresh":
		return m.Refresh
	case "back":
		return m.Back
	case "exit":
		return m.Exit
	case "help":
		return m.Help
	case "search":
		return m.Search
	case "export":
		return m.Export
	case "select":
		return m.Select
	case "enter":
		return m.Enter
	case "delete":
		return m.Delete
	case "create":
		return m.Create
	case "remove":
		return m.Remove
	case "clear":
		return m.Clear
	case "filter":
		return m.Filter
	case "ok":
		return m.OK
	
	// Status
	case "connected":
		return m.Connected
	case "disconnected":
		return m.Disconnected
	case "running":
		return m.Running
	case "stopped":
		return m.Stopped
	case "paused":
		return m.Paused
	case "available":
		return m.Available
	case "unavailable":
		return m.Unavailable
	
	// Home
	case "local_docker":
		return m.LocalDocker
	
	// Resources
	case "containers":
		return m.Containers
	case "images":
		return m.Images
	case "networks":
		return m.Networks
	case "volumes":
		return m.Volumes
	case "compose":
		return m.Compose
	
	// Logs
	case "logs_title":
		return m.LogsTitle
	case "follow":
		return m.Follow
	case "follow_mode":
		return m.FollowMode
	case "wrap":
		return m.Wrap
	case "lines":
		return m.Lines
	case "no_logs":
		return m.NoLogs
	case "loading_logs":
		return m.LoadingLogs
	case "load_failed":
		return m.LoadFailed
	case "export_to":
		return m.ExportTo
	case "export_success":
		return m.ExportSuccess
	case "export_failed":
		return m.ExportFailed
	case "search_next":
		return m.SearchNext
	case "search_prev":
		return m.SearchPrev
	case "search_clear":
		return m.SearchClear
	
	// Container operations
	case "start":
		return m.Start
	case "stop":
		return m.Stop
	case "restart":
		return m.Restart
	case "pause":
		return m.Pause
	case "unpause":
		return m.Unpause
	case "shell":
		return m.Shell
	case "logs":
		return m.Logs
	case "inspect":
		return m.Inspect
	
	// Hints
	case "press_to_confirm":
		return m.PressToConfirm
	case "press_to_cancel":
		return m.PressToCancel
	case "scroll_hint":
		return m.ScrollHint
	case "jump_hint":
		return m.JumpHint
	case "unknown_view":
		return m.UnknownView
	
	// Shell
	case "enter_shell":
		return m.EnterShell
	case "shell_tips":
		return m.ShellTips
	case "shell_exit_hint":
		return m.ShellExitHint
	case "shell_return_hint":
		return m.ShellReturnHint
	case "shell_exec_failed":
		return m.ShellExecFailed
	case "using_sdk_mode":
		return m.UsingSDKMode
	
	// Error messages
	case "feature_unavailable":
		return m.FeatureUnavailable
	case "feature_in_development":
		return m.FeatureInDevelopment
	case "volume_in_development":
		return m.VolumeInDevelopment
	case "compose_unavailable":
		return m.ComposeUnavailable
	case "view_not_initialized":
		return m.ViewNotInitialized
	case "select_container_first":
		return m.SelectContainerFirst
	case "only_running_container":
		return m.OnlyRunningContainer
	case "view_error":
		return m.ViewError
	case "select_first":
		return m.SelectFirst
	case "fatal_error":
		return m.FatalError
	case "warning":
		return m.Warning
	
	// Loading states
	case "loading_networks":
		return m.LoadingNetworks
	case "loading_images":
		return m.LoadingImages
	case "loading_containers":
		return m.LoadingContainers
	case "please_wait":
		return m.PleaseWait
	
	// Error states
	case "load_failed_reason":
		return m.LoadFailedReason
	case "possible_reasons":
		return m.PossibleReasons
	case "docker_not_running":
		return m.DockerNotRunning
	case "network_issue":
		return m.NetworkIssue
	case "permission_denied":
		return m.PermissionDenied
	
	// Empty states
	case "no_networks":
		return m.NoNetworks
	case "no_images":
		return m.NoImages
	case "no_containers":
		return m.NoContainers
	case "quick_start":
		return m.QuickStart
	case "create_network":
		return m.CreateNetwork
	case "pull_image_hint":
		return m.PullImageHint
	case "refresh_list":
		return m.RefreshList
	case "or_press":
		return m.OrPress
	
	// Operations
	case "operation_success":
		return m.OperationSuccess
	case "operation_failed":
		return m.OperationFailed
	case "get_info_failed":
		return m.GetInfoFailed
	case "create_success":
		return m.CreateSuccess
	case "delete_success":
		return m.DeleteSuccess
	case "prune_success":
		return m.PruneSuccess
	case "exported_to":
		return m.ExportedTo
	case "exporting":
		return m.Exporting
	case "task_cancelled":
		return m.TaskCancelled
	case "cancelling_task":
		return m.CancellingTask
	case "start_pulling":
		return m.StartPulling
	
	// Confirm dialogs
	case "confirm_delete_network":
		return m.ConfirmDeleteNetwork
	case "confirm_prune_networks":
		return m.ConfirmPruneNetworks
	case "confirm_delete_image":
		return m.ConfirmDeleteImage
	case "force_delete_image":
		return m.ForceDeleteImage
	case "image_in_use_warning":
		return m.ImageInUseWarning
	case "prune_dangling_images":
		return m.PruneDanglingImages
	case "confirm_pull_image":
		return m.ConfirmPullImage
	case "cannot_undo":
		return m.CannotUndo
	
	// Selected
	case "selected":
		return m.Selected
	
	// Hints for keys
	case "up_down":
		return m.UpDown
	case "enter_details":
		return m.EnterDetails
	case "esc_back":
		return m.EscBack
	case "quit_app":
		return m.QuitApp
	case "sort":
		return m.Sort
	
	default:
		return key
	}
}

// DetectLanguage detect language from environment variables
func DetectLanguage() Language {
	lang := os.Getenv("LANG")
	if lang == "" {
		lang = os.Getenv("LANGUAGE")
	}
	if lang == "" {
		lang = os.Getenv("LC_ALL")
	}
	
	lang = strings.ToLower(lang)
	if strings.HasPrefix(lang, "zh") {
		return ZH
	}
	return EN
}

// Init initialize i18n, auto-detect language
func Init() {
	SetLanguage(DetectLanguage())
}
