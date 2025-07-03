package user

import (
	"context"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/common/printer"
)

const (
	RoleMember Role = "member"
	RoleAdmin  Role = "admin"
	RoleOwner  Role = "owner"
	NoneRole   Role = "none"
)

var (
	ErrFailedToSetUserRoleInOrg = eris.New("Failed to set user role in organization")
)

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

	userRole, err := h.promptForRole(ctx, flags.Role)
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

func (h *Handler) promptForRole(ctx context.Context, roleFlag string) (string, error) {
	roles := []string{
		string(RoleMember),
		string(RoleAdmin),
		string(RoleOwner),
		string(NoneRole),
	}

	roleIndex := 0
	if roleFlag != "" {
		for i, role := range roles {
			if roleFlag == role {
				roleIndex = i
			}
		}
	}

	title := "Available Roles"
	prompt := "Select a role by number"
	roleIndex, err := h.inputService.Select(ctx, title, prompt, roles, roleIndex)
	if err != nil {
		return "", eris.Wrap(err, "Failed to get role")
	}

	return roles[roleIndex], nil
}
