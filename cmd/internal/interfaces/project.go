package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

type ProjectHandler interface {
	Create(ctx context.Context, flags *models.CreateProjectFlags, createNew bool) (models.Project, error)
	Switch(ctx context.Context, flags *models.SwitchProjectFlags) (models.Project, error)
	Update(ctx context.Context, flags *models.UpdateProjectFlags) error
	Delete(ctx context.Context) error
}
