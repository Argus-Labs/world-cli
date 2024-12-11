package types

import "time"

const (
	// FilePermissions
	ConfigDirPerm  = 0o755
	ConfigFilePerm = 0600
	EditorDirPerm  = 0755
	GitDirPerm     = 0o755
	TokenFilePerm  = 0644

	// Docker Constants
	DockerHeaderSize = 8
	MaxRetries      = 20
	RetryInterval   = 200 * time.Millisecond
	DefaultTimeout  = time.Second

	// UI Constants
	SpinnerSpeed     = 80 * time.Millisecond
	ContainerPadding = 2
	HeaderWidth      = 40

	// Service Health Check
	ServiceRetries    = 30
	ServiceRetryDelay = 200 * time.Millisecond
	ServiceTimeout    = time.Second
	DBRetries        = 5
	DBInterval       = 3 * time.Second
	DBTimeout        = 3 * time.Second

	// Progress Bar
	ProgressBarMax = 100

	// Stream Types
	StdoutStreamType = 1
	StderrStreamType = 2

	// Platform Check
	PlatformParts = 2
)
