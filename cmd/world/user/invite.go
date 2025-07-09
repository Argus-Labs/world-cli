package user

import (
	"context"
	"strings"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
	"pkg.world.dev/world-cli/cmd/world/internal/utils/validate"
	"pkg.world.dev/world-cli/common/printer"
)

var ErrOrganizationInviteFailed = eris.New("Organization email invite failed, but invite is still created in CLI.")

func (h *Handler) InviteToOrganization(
	ctx context.Context,
	organization models.Organization,
	flags models.InviteUserToOrganizationFlags,
) error {
	printer.NewLine(1)
	printer.Headerln("   Invite User to Organization   ")

	userEmail, err := h.inputService.Prompt(ctx, "Enter user email to invite", flags.Email)
	if err != nil {
		return eris.Wrap(err, "Failed to get user email")
	}

	if err := validate.Email(userEmail); err != nil {
		return eris.Wrap(err, "Invalid email format")
	}

	userRole, err := h.promptForRole(ctx, flags.Role)
	if err != nil {
		return eris.Wrap(err, "Failed to get user role")
	}

	err = h.apiClient.InviteUserToOrganization(ctx, organization.ID, userEmail, userRole)
	if err != nil {
		if strings.Contains(err.Error(), ErrOrganizationInviteFailed.Error()) {
			printer.Successln("Invite created successfully, can be accepted in the World Forge.")
			printer.Errorf("Email failed to send to user: %s\n", err)
			printer.NewLine(1)
		}
		return eris.Wrap(err, "Failed to invite user to organization")
	}

	printer.NewLine(1)
	printer.Successf("Successfully invited user %s to organization!\n", userEmail)
	printer.Infof("  Assigned role: %s\n", userRole)
	return nil
}
