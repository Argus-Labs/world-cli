package organization

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

//nolint:revive // TODO: implement
func (h *Handler) Switch(ctx context.Context, flags *models.SwitchOrganizationFlags,
) (models.Organization, error) {
	return models.Organization{}, nil
}
