package project

import (
	"pkg.world.dev/world-cli/cmd/world/pkg/models"
)

// Interface guard.
var _ HandlerInterface = (*Handler)(nil)

type Handler struct {
}

type HandlerInterface interface {
	Create(ctx models.CommandContext, flags *models.CreateProjectFlags, createNew bool) (*models.Project, error)
	Switch(ctx models.CommandContext, flags *models.SwitchProjectFlags) (*models.Project, error)
	Update(ctx models.CommandContext, flags *models.UpdateProjectFlags) error
	Delete(ctx models.CommandContext) error
}
