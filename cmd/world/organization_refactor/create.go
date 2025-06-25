package organization

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

//nolint:revive // TODO: implement
func (h *Handler) Create(ctx context.Context, flags *models.CreateOrganizationFlags,
) (models.Organization, error) {
	return models.Organization{}, nil
}
