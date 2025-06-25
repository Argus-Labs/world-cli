package project

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

//nolint:revive // TODO: implement
func (h *Handler) Switch(ctx context.Context, flags *models.SwitchProjectFlags,
) (models.Project, error) {
	return models.Project{}, nil
}
