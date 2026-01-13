package compose

import (
	"io"
	"time"
)

// ProjectStatus represents project status
type ProjectStatus int

const (
	StatusUnknown ProjectStatus = iota
	StatusRunning               // All services running
	StatusPartial               // Some services running
	StatusStopped               // All services stopped
	StatusError                 // Error state
)

// String returns the string representation of the status
func (s ProjectStatus) String() string {
	switch s {
	case StatusRunning:
		return "Running"
	case StatusPartial:
		return "Partial"
	case StatusStopped:
		return "Stopped"
	case StatusError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Project represents a docker-compose project
type Project struct {
	Name         string            // Project name
	Path         string            // Project root directory absolute path
	ComposeFiles []string          // Compose file list (relative paths)
	EnvFiles     []string          // Environment variable file list
	WorkingDir   string            // Working directory
	Labels       map[string]string // Project labels

	// Runtime state
	Services    []Service     // Service list
	Status      ProjectStatus // Project status
	LastUpdated time.Time     // Last updated time
}

// Service represents a service in a compose project
type Service struct {
	Name       string   // Service name
	Image      string   // Image name
	State      string   // State: running/exited/restarting/paused
	Containers []string // Container ID list
	Replicas   int      // Replica count
	Running    int      // Running replica count
	Ports      []string // Port mappings
}

// PortMapping represents port mapping
type PortMapping struct {
	HostIP        string
	HostPort      int
	ContainerPort int
	Protocol      string
}

// UpOptions represents options for starting a project
type UpOptions struct {
	Detach        bool     // Run in background (-d)
	Build         bool     // Build images (--build)
	ForceRecreate bool     // Force recreate (--force-recreate)
	NoDeps        bool     // Don't start dependencies (--no-deps)
	Services      []string // Specific services (empty for all)
	Timeout       int      // Timeout in seconds
	Pull          string   // Pull policy: always/missing/never
}

// DownOptions represents options for stopping a project
type DownOptions struct {
	RemoveVolumes bool   // Remove volumes (-v)
	RemoveOrphans bool   // Remove orphan containers (--remove-orphans)
	RemoveImages  string // Remove images: all/local (empty for none)
	Timeout       int    // Timeout in seconds
}

// LogOptions represents log options
type LogOptions struct {
	Follow     bool     // Follow mode (-f)
	Tail       int      // Show last N lines (0 for all)
	Timestamps bool     // Show timestamps (-t)
	Services   []string // Specific services (empty for all)
	Since      string   // Start time
	Until      string   // End time
}

// BuildOptions represents build options
type BuildOptions struct {
	NoCache  bool     // Don't use cache (--no-cache)
	Pull     bool     // Pull base images (--pull)
	Services []string // Specific services
}

// PullOptions represents pull options
type PullOptions struct {
	IgnorePullFailures bool     // Ignore pull failures
	Services           []string // Specific services
}

// OperationResult represents operation result
type OperationResult struct {
	Success  bool          // Whether successful
	Message  string        // Message
	Output   string        // stdout output
	Error    string        // stderr output
	ExitCode int           // Exit code
	Duration time.Duration // Duration
}

// ErrorType represents error type
type ErrorType int

const (
	ErrorUnknown ErrorType = iota
	ErrorConfig            // Compose file configuration error
	ErrorNetwork           // Network error (port conflict, etc.)
	ErrorImage             // Image related error
	ErrorRuntime           // Runtime error
	ErrorPermission        // Permission error
	ErrorNotFound          // Command or file not found
)

// ComposeError represents compose operation error
type ComposeError struct {
	Type       ErrorType
	Message    string
	Details    string
	Suggestion string
}

// Error implements the error interface
func (e *ComposeError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// Client represents docker-compose client interface
type Client interface {
	// Version info
	Version() (string, error)
	CommandType() string // Returns "docker compose" or "docker-compose"

	// Project operations
	Up(project *Project, opts UpOptions) (*OperationResult, error)
	Down(project *Project, opts DownOptions) (*OperationResult, error)
	Start(project *Project, services []string) (*OperationResult, error)
	Stop(project *Project, services []string, timeout int) (*OperationResult, error)
	Restart(project *Project, services []string, timeout int) (*OperationResult, error)
	Pause(project *Project, services []string) (*OperationResult, error)
	Unpause(project *Project, services []string) (*OperationResult, error)

	// Information queries
	PS(project *Project) ([]Service, error)
	Logs(project *Project, opts LogOptions) (io.ReadCloser, error)
	Config(project *Project) (string, error)

	// Image operations
	Build(project *Project, opts BuildOptions) (*OperationResult, error)
	Pull(project *Project, opts PullOptions) (*OperationResult, error)
}
