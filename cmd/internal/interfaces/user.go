package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

type UserHandler interface {
	InviteToOrganization(
		ctx context.Context,
		organization models.Organization,
		flags models.InviteUserToOrganizationFlags,
	) error
	ChangeRoleInOrganization(
		ctx context.Context,
		organization models.Organization,
		flags models.ChangeUserRoleInOrganizationFlags,
	) error
	Update(ctx context.Context, flags models.UpdateUserFlags) error
}
