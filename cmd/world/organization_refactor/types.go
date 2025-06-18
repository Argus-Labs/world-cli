package organization

import "pkg.world.dev/world-cli/cmd/world/pkg/models"

// Interface guard.
var _ HandlerInterface = (*Handler)(nil)

type Handler struct {
	ProjectHandler ProjectHandler
}

type HandlerInterface interface {
	Create(ctx models.CommandContext, flags *models.CreateOrganizationFlags) (*models.Organization, error)
	Switch(ctx models.CommandContext, flags *models.SwitchOrganizationFlags) (*models.Organization, error)
}

type ProjectHandler interface {
	Switch(ctx models.CommandContext, flags *models.SwitchProjectFlags) (*models.Project, error)
}
