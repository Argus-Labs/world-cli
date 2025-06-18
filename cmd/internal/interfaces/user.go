package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

type UserHandler interface {
	InviteToOrganization(ctx context.Context, flags *models.InviteUserToOrganizationFlags) error
	ChangeRoleInOrganization(ctx context.Context, flags *models.ChangeUserRoleInOrganizationFlags) error
	Update(ctx context.Context, flags *models.UpdateUserFlags) error
}
