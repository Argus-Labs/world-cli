package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

type ProjectHandler interface {
	Create(ctx context.Context, flags models.CreateProjectFlags) (models.Project, error)
	Switch(
		ctx context.Context,
		flags models.SwitchProjectFlags,
		enableCreation bool,
	) (models.Project, error)
	HandleSwitch(ctx context.Context) error
	Update(ctx context.Context, project models.Project, flags models.UpdateProjectFlags) error
	Delete(ctx context.Context, project models.Project) error

	PreCreateUpdateValidation() (string, string, error)
}
