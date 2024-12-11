package commands

import (
	"context"
	"fmt"

	"pkg.world.dev/world-cli/infrastructure/docker"
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
func (c *DockerCommand) Execute(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("docker client is not initialized")
	}
	return nil
}
