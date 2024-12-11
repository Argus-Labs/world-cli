package types

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
)

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
	types.ContainerRemoveOptions
}

// VolumeCreateOptions wraps Docker volume create options
type VolumeCreateOptions struct {
	volume.CreateOptions
}

// VolumeListOptions wraps Docker volume list options
type VolumeListOptions struct {
	volume.ListOptions
}
