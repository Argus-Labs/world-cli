// Package operations provides simplified interfaces for common Docker operations
package operations

import (
	"context"
	"io"

	"github.com/moby/moby/api/types"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/infrastructure/docker/service"
)

// Manager handles Docker operations with simplified interfaces
type Manager struct {
	client *client.Client
}

// NewManager creates a new Docker operations manager
func NewManager(cli *client.Client) *Manager {
	return &Manager{client: cli}
}

// ContainerOperation represents common container lifecycle operations
type ContainerOperation struct {
	Name       string
	Config     *types.ContainerConfig
	HostConfig *container.HostConfig
}

// ServiceOperation wraps service-specific operations
func (m *Manager) ServiceOperation(ctx context.Context, svc service.Service, op func(ContainerOperation) error) error {
	config := &types.ContainerConfig{
		Image:        svc.ContainerConfig.Image,
		Cmd:          svc.ContainerConfig.Cmd,
		Env:          svc.ContainerConfig.Env,
		ExposedPorts: svc.ContainerConfig.ExposedPorts,
		Labels:       svc.ContainerConfig.Labels,
		StopSignal:   svc.ContainerConfig.StopSignal,
		WorkingDir:   svc.ContainerConfig.WorkingDir,
	}

	operation := ContainerOperation{
		Name:       svc.Name,
		Config:     config,
		HostConfig: &svc.HostConfig,
	}
	return op(operation)
}

// StartContainer starts a container with proper error handling and validation
func (m *Manager) StartContainer(ctx context.Context, op ContainerOperation) error {
	// Check if container exists
	_, err := m.client.ContainerInspect(ctx, op.Name)
	if client.IsErrNotFound(err) {
		// Create container if it doesn't exist
		_, err = m.client.ContainerCreate(ctx, op.Config, op.HostConfig, nil, nil, op.Name)
		if err != nil {
			return eris.Wrapf(err, "failed to create container %s", op.Name)
		}
	} else if err != nil {
		return eris.Wrapf(err, "failed to inspect container %s", op.Name)
	}

	// Start the container
	err = m.client.ContainerStart(ctx, op.Name, types.ContainerStartOptions{})
	if err != nil {
		return eris.Wrapf(err, "failed to start container %s", op.Name)
	}

	return nil
}

// StopContainer stops a container with proper error handling
func (m *Manager) StopContainer(ctx context.Context, op ContainerOperation) error {
	err := m.client.ContainerStop(ctx, op.Name, nil)
	if err != nil && !client.IsErrNotFound(err) {
		return eris.Wrapf(err, "failed to stop container %s", op.Name)
	}
	return nil
}

// RemoveContainer removes a container with proper error handling
func (m *Manager) RemoveContainer(ctx context.Context, op ContainerOperation) error {
	// Stop container first
	err := m.StopContainer(ctx, op)
	if err != nil {
		return err
	}

	// Remove the container
	err = m.client.ContainerRemove(ctx, op.Name, types.ContainerRemoveOptions{})
	if err != nil && !client.IsErrNotFound(err) {
		return eris.Wrapf(err, "failed to remove container %s", op.Name)
	}
	return nil
}

// PullImage pulls a Docker image with proper error handling
func (m *Manager) PullImage(ctx context.Context, imageName string) error {
	out, err := m.client.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return eris.Wrapf(err, "failed to pull image %s", imageName)
	}
	defer out.Close()
	_, err = io.Copy(io.Discard, out)
	return err
}

// BuildImage builds a Docker image with proper error handling
func (m *Manager) BuildImage(ctx context.Context, options types.ImageBuildOptions) error {
	response, err := m.client.ImageBuild(ctx, nil, options)
	if err != nil {
		return eris.Wrapf(err, "failed to build image")
	}
	defer response.Body.Close()
	_, err = io.Copy(io.Discard, response.Body)
	return err
}
