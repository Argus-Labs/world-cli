package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

// ProjectHandler defines the interface for project-related operations.
type ProjectHandler interface {
	// Create creates a new project in the specified organization.
	Create(ctx context.Context, org models.Organization, flags models.CreateProjectFlags) (models.Project, error)
	// Switch handles project selection and switching logic.
	// If enableCreation is true, allows creating a new project during switch.
	Switch(
		ctx context.Context,
		flags models.SwitchProjectFlags,
		org models.Organization,
		enableCreation bool,
	) (models.Project, error)
	// HandleSwitch performs the actual project switching operation.
	HandleSwitch(ctx context.Context, org models.Organization) error
	// Update modifies an existing project with new information.
	Update(ctx context.Context, project models.Project, org models.Organization, flags models.UpdateProjectFlags) error
	// Delete removes a project from the organization.
	Delete(ctx context.Context, project models.Project) error

	// Utils

	// PreCreateUpdateValidation validates the current environment for project operations.
	// Returns repository path, URL, and any validation errors.
	PreCreateUpdateValidation(printError bool) (string, string, error)
	// PrintNoProjectsInOrganization displays guidance when no projects exist.
	PrintNoProjectsInOrganization()
}
