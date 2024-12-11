package commands

import (
	"context"
	"errors"

	"pkg.world.dev/world-cli/infrastructure/docker"
	"pkg.world.dev/world-cli/infrastructure/docker/service"
)

const (
	// DockerServiceDA is the name of the DA service container
	DockerServiceDA = service.Name("celestia-da")
	// DockerServiceEVM is the name of the EVM service container
	DockerServiceEVM = service.Name("evm")
)

// DockerCommand represents a command to be executed in a Docker container
type DockerCommand struct {
	client *docker.Client
}

// NewDockerCommand creates a new DockerCommand instance
func NewDockerCommand(client *docker.Client) *DockerCommand {
	return &DockerCommand{
		client: client,
	}
}

// Execute runs the Docker command
func (c *DockerCommand) Execute(_ context.Context) error {
	if c.client == nil {
		return errors.New("docker client is not initialized")
	}
	return nil
}
