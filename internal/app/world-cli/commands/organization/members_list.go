package organization

import (
	"context"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
	"pkg.world.dev/world-cli/internal/pkg/printer"
)

func (h *Handler) MembersList(ctx context.Context, org models.Organization, flags models.MembersListFlags) error {
	members, err := h.apiClient.GetOrganizationMembers(ctx, org.ID)
	if err != nil {
		return eris.Wrap(err, "MembersList: Failed to get organization members")
	}

	// Create a map to group members by role
	membersByRole := make(map[models.Role][]models.OrganizationMember)

	for _, member := range members {
		// Use the member's role as the key, or "RoleNone" if role is unknown
		role := member.Role
		if _, ok := models.RolesMap[role]; !ok {
			role = models.RoleNone
		}

		// Append the member to the list for this role
		membersByRole[role] = append(membersByRole[role], member)
	}

	if len(membersByRole) == 0 {
		printer.Infof("No members found for organization: %s [%s]", org.Name, org.Slug)
		return nil
	}

	// Define the order we want to display roles
	roleOrder := []models.Role{
		models.RoleOwner,
		models.RoleAdmin,
		models.RoleMember,
		models.RoleNone,
	}

	for _, role := range roleOrder {
		if !flags.IncludeRemoved && role == models.RoleNone {
			// Skip if the role is RoleNone and the removed list flag is not set
			continue
		}

		// Only show roles that have members
		if membersInRole, exists := membersByRole[role]; exists {
			printer.NewLine(1)
			printer.Headerf("  %s  ", role)
			printer.NewLine(1)

			for _, member := range membersInRole {
				printer.Infof("%s - %s", member.User.Name, member.User.Email)
				printer.NewLine(1)
			}
		}
	}

	return nil
}
