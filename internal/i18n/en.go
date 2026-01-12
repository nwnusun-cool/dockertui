package i18n

var enMessages = &Messages{
	// Common
	Loading: "Loading...",
	Error:   "Error",
	Success: "Success",
	Confirm: "Confirm",
	Cancel:  "Cancel",
	Refresh: "Refresh",
	Back:    "Back",
	Exit:    "Exit",
	Help:    "Help",
	Search:  "Search",
	Export:  "Export",
	Select:  "Select",
	Enter:   "Enter",
	Delete:  "Delete",
	Create:  "Create",
	Remove:  "Remove",
	Clear:   "Clear",
	Filter:  "Filter",
	OK:      "OK",

	// Status
	Connected:    "Connected",
	Disconnected: "Disconnected",
	Running:      "Running",
	Stopped:      "Stopped",
	Paused:       "Paused",
	Available:    "Available",
	Unavailable:  "Unavailable",

	// Home
	LocalDocker: "Local Docker",

	// Resources
	Containers: "Containers",
	Images:     "Images",
	Networks:   "Networks",
	Volumes:    "Volumes",
	Compose:    "Compose",

	// Logs view
	LogsTitle:     "Logs",
	Follow:        "Follow",
	FollowMode:    "Follow Mode",
	Wrap:          "Wrap",
	Lines:         "Lines",
	NoLogs:        "No logs",
	LoadingLogs:   "Loading logs...",
	LoadFailed:    "Load failed",
	ExportTo:      "Export to",
	ExportSuccess: "Exported to",
	ExportFailed:  "Export failed",
	SearchNext:    "Next",
	SearchPrev:    "Prev",
	SearchClear:   "Clear",

	// Container operations
	Start:   "Start",
	Stop:    "Stop",
	Restart: "Restart",
	Pause:   "Pause",
	Unpause: "Unpause",
	Shell:   "Shell",
	Logs:    "Logs",
	Inspect: "Inspect",

	// Hints
	PressToConfirm: "Enter=Confirm",
	PressToCancel:  "ESC=Cancel",
	ScrollHint:     "Scroll",
	JumpHint:       "Top/Bottom",
	UnknownView:    "Unknown view",

	// Shell
	EnterShell:      "Entering container shell",
	ShellTips:       "Tips:",
	ShellExitHint:   "Type exit or press Ctrl+D to exit shell",
	ShellReturnHint: "Will return to DockTUI after exit",
	ShellExecFailed: "Shell execution failed",
	UsingSDKMode:    "Using Docker SDK mode...",

	// Error messages
	FeatureUnavailable:   "This feature is unavailable",
	FeatureInDevelopment: "This feature is in development...",
	VolumeInDevelopment:  "Volume management is in development...",
	ComposeUnavailable:   "Docker Compose is not installed or unavailable",
	ViewNotInitialized:   "View not initialized",
	SelectContainerFirst: "Please select a container first",
	OnlyRunningContainer: "Can only execute shell in running containers",
	ViewError:            "View error",
	SelectFirst:          "Please select an item first",
	FatalError:           "Fatal error",
	Warning:              "Warning",

	// Loading states
	LoadingNetworks:   "Loading networks...",
	LoadingImages:     "Loading images...",
	LoadingContainers: "Loading containers...",
	PleaseWait:        "Please wait, fetching data from Docker",

	// Error states
	LoadFailedReason:  "Load failed",
	PossibleReasons:   "Possible reasons:",
	DockerNotRunning:  "Docker daemon is not running",
	NetworkIssue:      "Network connection issue",
	PermissionDenied:  "Permission denied",

	// Empty states
	NoNetworks:     "No custom networks",
	NoImages:       "No images",
	NoContainers:   "No containers",
	QuickStart:     "Quick start:",
	CreateNetwork:  "Create a network:",
	PullImageHint:  "Pull an image:",
	RefreshList:    "Press r to refresh",
	OrPress:        "or",

	// Operations
	OperationSuccess: "succeeded",
	OperationFailed:  "failed",
	GetInfoFailed:    "Failed to get info",
	CreateSuccess:    "Created successfully",
	DeleteSuccess:    "Deleted successfully",
	PruneSuccess:     "Pruned successfully",
	ExportedTo:       "Exported to",
	Exporting:        "Exporting",
	TaskCancelled:    "Task cancelled",
	CancellingTask:   "Cancelling task...",
	StartPulling:     "Start pulling",

	// Confirm dialogs
	ConfirmDeleteNetwork: "Confirm delete network",
	ConfirmPruneNetworks: "Confirm prune networks",
	ConfirmDeleteImage:   "Delete image",
	ForceDeleteImage:     "Force delete image",
	ImageInUseWarning:    "This image is being used by containers!",
	PruneDanglingImages:  "Prune dangling images",
	ConfirmPullImage:     "Pull image",
	CannotUndo:           "This action cannot be undone!",

	// Selected
	Selected: "Selected",

	// Hints for keys
	UpDown:       "j/k=Up/Down",
	EnterDetails: "Enter=Details",
	EscBack:      "Esc=Back",
	QuitApp:      "q=Quit",
	Sort:         "s=Sort",
}
