package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

// CommandSetupController defines the interface for command setup operations.
type CommandSetupController interface {
	// SetupCommandState performs the setup flow for commands and returns the established state.
	// Handles login, organization, and project setup based on requirements.
	SetupCommandState(ctx context.Context, req models.SetupRequest) (models.CommandState, error)
}
