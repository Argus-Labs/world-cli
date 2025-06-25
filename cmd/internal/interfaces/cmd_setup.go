package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

// CommandSetupController defines the interface for command setup.
type CommandSetupController interface {
	// SetupCommand performs the setup flow and returns the established state
	SetupCommandState(ctx context.Context, req models.SetupRequest) (models.CommandState, error)
}
