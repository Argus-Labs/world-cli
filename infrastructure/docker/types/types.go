package types

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	globalconfig "pkg.world.dev/world-cli/config"
)

// BuildkitSupport is a flag to check if buildkit is supported
var BuildkitSupport bool

const (
	// START represents starting a container/service
	START ProcessType = iota
	// STOP represents stopping a container/service
	STOP
	// REMOVE represents removing a container/service
	REMOVE
	// CREATE represents creating a container/service
	CREATE
)

var (
	// ProcessName maps process types to their string representations
	ProcessName = map[ProcessType]string{
		START:  "start",
		STOP:   "stop",
		REMOVE: "remove",
		// CREATE represents creating a container/service
		CREATE: "create",
	}

	// ProcessInitName maps process types to their initialization string representations
	ProcessInitName = map[ProcessType]string{
		START:  "starting",
		STOP:   "stopping",
		REMOVE: "removing",
		CREATE: "creating",
	}

	// ProcessFinishName maps process types to their completion string representations
	ProcessFinishName = map[ProcessType]string{
		START:  "started",
		STOP:   "stopped",
		REMOVE: "removed",
		CREATE: "created",
	}
)

// ProcessType represents different Docker operations
type ProcessType int

// Name represents the name of a Docker service
type Name string

// Builder is a function type that creates a Service configuration
type Builder func(cfg *globalconfig.Config) Service

// Service is a configuration for a docker container
// It contains the name of the container and a function to get the container and host config
type Service struct {
	Name string
	container.Config
	container.HostConfig
	network.NetworkingConfig
	ocispec.Platform

	// Dependencies are other services that need to be pull before this service
	Dependencies []Service
	// Dockerfile is the content of the Dockerfile
	Dockerfile string
	// BuildTarget is the target build of the Dockerfile e.g. builder or runtime
	BuildTarget string
}

// ContainerOperation represents a Docker container operation
type ContainerOperation struct {
	Name       string
	Image      string
	Env        []string
	Ports      []string
	Volumes    []string
	Network    string
	Command    []string
	WorkingDir string
}

// ImageOperation represents a Docker image operation
type ImageOperation struct {
	Name string
	Tag  string
}

// VolumeOperation represents a Docker volume operation
type VolumeOperation struct {
	Name string
}

// NetworkOperation represents a Docker network operation
type NetworkOperation struct {
	Name string
}

// Manager handles Docker operations
type Manager interface {
	// Container operations
	CreateContainer(ctx context.Context, op ContainerOperation) error
	StartContainer(ctx context.Context, op ContainerOperation) error
	StopContainer(ctx context.Context, op ContainerOperation) error
	RemoveContainer(ctx context.Context, op ContainerOperation) error
	ContainerExists(ctx context.Context, name string) (bool, error)

	// Image operations
	PullImage(ctx context.Context, op ImageOperation) error
	ImageExists(ctx context.Context, name string) (bool, error)

	// Volume operations
	CreateVolume(ctx context.Context, op VolumeOperation) error
	RemoveVolume(ctx context.Context, op VolumeOperation) error
	VolumeExists(ctx context.Context, name string) (bool, error)

	// Network operations
	CreateNetwork(ctx context.Context, op NetworkOperation) error
	RemoveNetwork(ctx context.Context, op NetworkOperation) error
	NetworkExists(ctx context.Context, name string) (bool, error)
}

// ContainerConfig wraps Docker container configuration
type ContainerConfig struct {
	*container.Config
	HostConfig       *container.HostConfig
	NetworkingConfig *network.NetworkingConfig
}

// StopOptions wraps Docker container stop options
type StopOptions struct {
	*container.StopOptions
}

// RemoveOptions wraps Docker container remove options
type RemoveOptions struct {
	Force   bool
	Volumes bool
}

// VolumeCreateOptions wraps Docker volume create options
type VolumeCreateOptions struct {
	volume.CreateOptions
}

// VolumeListOptions wraps Docker volume list options
type VolumeListOptions struct {
	volume.ListOptions
}
