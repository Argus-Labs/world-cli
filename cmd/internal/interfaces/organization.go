package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

// OrganizationHandler defines the interface for organization-related operations.
type OrganizationHandler interface {
	// Create creates a new organization with the specified details.
	Create(
		ctx context.Context,
		flags models.CreateOrganizationFlags,
	) (models.Organization, error)

	// Switch handles organization selection and switching logic.
	Switch(
		ctx context.Context,
		flags models.SwitchOrganizationFlags,
	) (models.Organization, error)

	// MembersList lists members of an organization.
	MembersList(
		ctx context.Context,
		org models.Organization,
		flags models.MembersListFlags,
	) error

	// PromptForSwitch manages organization selection with optional creation.
	// If enableCreation is true, allows creating a new organization during selection.
	PromptForSwitch(
		ctx context.Context,
		orgs []models.Organization,
		enableCreation bool,
	) (models.Organization, error)

	// Utils

	// PrintNoOrganizations displays guidance when no organizations exist.
	PrintNoOrganizations()
}
