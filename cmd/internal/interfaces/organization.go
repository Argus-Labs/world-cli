package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

type OrganizationHandler interface {
	Create(
		ctx context.Context,
		state *models.CommandState,
		flags models.CreateOrganizationFlags,
	) (models.Organization, error)
	Switch(
		ctx context.Context,
		state *models.CommandState,
		flags models.SwitchOrganizationFlags,
	) (models.Organization, error)
	PromptForSwitch(
		ctx context.Context,
		state *models.CommandState,
		orgs []models.Organization,
		createNew bool,
	) (models.Organization, error)
}
