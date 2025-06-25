package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

type OrganizationHandler interface {
	Create(ctx context.Context, flags *models.CreateOrganizationFlags) (models.Organization, error)
	Switch(ctx context.Context, flags *models.SwitchOrganizationFlags) (models.Organization, error)

	PromptForOrganization(ctx context.Context, orgs []models.Organization, createNew bool) (models.Organization, error)
}
