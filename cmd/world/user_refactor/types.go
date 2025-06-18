package user

import "pkg.world.dev/world-cli/cmd/world/pkg/models"

// Interface guard.
var _ HandlerInterface = (*Handler)(nil)

type Handler struct {
}

type HandlerInterface interface {
	InviteToOrganization(ctx models.CommandContext, flags *models.InviteUserToOrganizationFlags) error
	ChangeRoleInOrganization(ctx models.CommandContext, flags *models.ChangeUserRoleInOrganizationFlags) error
	Update(ctx models.CommandContext, flags *models.UpdateUserFlags) error
}
