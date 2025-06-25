package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

type ProjectHandler interface {
	Create(ctx context.Context, state *models.CommandState, flags models.CreateProjectFlags) (models.Project, error)
	Switch(
		ctx context.Context,
		state *models.CommandState,
		flags models.SwitchProjectFlags,
		createNew bool,
	) (models.Project, error)
	Update(ctx context.Context, state *models.CommandState, flags models.UpdateProjectFlags) error
	Delete(ctx context.Context, state *models.CommandState) error

	PreCreateUpdateValidation() (string, string, error)
}
