package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/world/internal/models"
)

// UserHandler defines the interface for user-related operations.
type UserHandler interface {
	// InviteToOrganization invites a user to join an organization with a specific role.
	InviteToOrganization(
		ctx context.Context,
		organization models.Organization,
		flags models.InviteUserToOrganizationFlags,
	) error

	// ChangeRoleInOrganization updates a user's role within an organization.
	ChangeRoleInOrganization(
		ctx context.Context,
		organization models.Organization,
		flags models.ChangeUserRoleInOrganizationFlags,
	) error

	// Update modifies the current user's profile information.
	Update(ctx context.Context, flags models.UpdateUserFlags) error
}
