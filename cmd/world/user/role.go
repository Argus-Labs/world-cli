package user

import (
	"context"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/common/printer"
)

var (
	ErrFailedToSetUserRoleInOrg = eris.New("Failed to set user role in organization")
)

var roles = []string{
	"member",
	"admin",
	"owner",
}

func (h *Handler) ChangeRoleInOrganization(
	ctx context.Context,
	organization models.Organization,
	flags models.ChangeUserRoleInOrganizationFlags,
) error {
	printer.NewLine(1)
	printer.Headerln("  Update User Role in Organization  ")
	userEmail, err := h.inputService.Prompt(ctx, "Enter user email to update", flags.Email)
	if err != nil {
		return eris.Wrap(err, "Failed to get user email")
	}

	if userEmail == "" {
		return eris.New("User email cannot be empty")
	}

	if flags.Role == "" {
		flags.Role = roles[0]
	}

	userRole, err := h.inputService.Prompt(ctx, "Enter user role to update", flags.Role)
	if err != nil {
		return eris.Wrap(err, "Failed to get user role")
	}

	// Send request
	err = h.apiClient.UpdateUserRoleInOrganization(ctx, organization.ID, userEmail, userRole)
	if err != nil {
		printer.Errorf("Failed to set role in organization: %s\n", err)
		return eris.Wrap(err, ErrFailedToSetUserRoleInOrg.Error())
	}

	printer.NewLine(1)
	printer.Successf("Successfully updated role for user %s!\n", userEmail)
	printer.Infof("New role: %s\n", userRole)
	return nil
}
