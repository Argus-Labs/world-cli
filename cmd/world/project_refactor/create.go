package project

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

//nolint:revive // TODO: implement
func (h *Handler) Create(ctx context.Context, flags *models.CreateProjectFlags, createNew bool,
) (models.Project, error) {
	return models.Project{}, nil
}
