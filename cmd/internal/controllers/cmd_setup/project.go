package cmdsetup

import (
	"context"
	"fmt"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/common/printer"
)

var (
	ErrProjectSelectionCanceled = eris.New("Project selection canceled")
	ErrProjectCreationCanceled  = eris.New("Project creation canceled")
)

/////////////////////
// Need Project    //
/////////////////////

func (c *Controller) handleNeedProjectData(ctx context.Context, result *models.CommandState, cfg *config.Config) error {
	projects, err := c.apiClient.GetProjects(ctx, cfg.OrganizationID)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	switch len(projects) {
	case 0: // No projects found
		return c.handleNeedProjectCaseNoProjects(ctx, result, cfg)
	case 1: // One project found
		return c.handleNeedProjectCaseOneProject(ctx, result, cfg, projects)
	default: // Multiple projects found
		return c.handleNeedProjectCaseMultipleProjects(ctx, result, cfg)
	}
}

func (c *Controller) handleNeedProjectCaseNoProjects(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
) error {
	for {
		// Must be in a valid directory to create a project
		_, _, err := c.projectHandler.PreCreateUpdateValidation()
		if err != nil {
			// TODO: printNoProjectsInOrganization()
			return ErrProjectCreationCanceled
		}

		choice, err := c.inputService.Prompt(ctx, "Would you like to create a new project? (Y/n)", "Y")
		if err != nil {
			return eris.Wrap(err, "failed to get input")
		}

		switch choice {
		case "Y":
			proj, err := c.projectHandler.Create(ctx, models.CreateProjectFlags{})
			if err != nil {
				return eris.Wrap(err, "Flow failed to create project in no-projects case")
			}
			c.updateProject(cfg, &proj, result)
			return nil
		case "n":
			return ErrProjectCreationCanceled
		default:
			printer.NewLine(1)
			printer.Infoln("Please select capital 'Y' or lowercase 'n'")
		}
	}
}

func (c *Controller) handleNeedProjectCaseOneProject(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
	projects []models.Project,
) error {
	printer.NewLine(1)

	inRepoRoot := true
	prompt := fmt.Sprintf("Select project: %s [%s]? (Y/n/c to create new)", projects[0].Name, projects[0].Slug)
	_, _, err := c.projectHandler.PreCreateUpdateValidation()
	if err != nil {
		inRepoRoot = false
		prompt = fmt.Sprintf("Select project: %s [%s]? (Y/n)", projects[0].Name, projects[0].Slug)
	}

	for {
		choice, err := c.inputService.Prompt(ctx, prompt, "Y")
		if err != nil {
			return eris.Wrap(err, "failed to get input")
		}

		switch choice {
		case "Y":
			c.updateProject(cfg, &projects[0], result)
			return nil
		case "n":
			return ErrProjectSelectionCanceled
		case "c":
			if inRepoRoot {
				proj, err := c.projectHandler.Create(ctx, models.CreateProjectFlags{})
				if err != nil {
					return eris.Wrap(err, "Flow failed to create project in one-project case")
				}
				c.updateProject(cfg, &proj, result)
				return nil
			}
			printer.Infoln("Please select capital 'Y' or lowercase 'n'/'c'")
		default:
			if inRepoRoot {
				printer.Infoln("Please select capital 'Y' or lowercase 'n'/'c'")
			} else {
				printer.Infoln("Please select capital 'Y' or lowercase 'n'")
			}
			printer.NewLine(1)
		}
	}
}

func (c *Controller) handleNeedProjectCaseMultipleProjects(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
) error {
	proj, err := c.projectHandler.Switch(ctx, models.SwitchProjectFlags{}, true)
	if err != nil {
		return eris.Wrap(err, "Flow failed to select project in multiple-projects case")
	}
	c.updateProject(cfg, &proj, result)
	return nil
}

////////////////////////////////
// Need Existing Project      //
////////////////////////////////

func (c *Controller) handleNeedExistingProjectData(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
) error {
	projects, err := c.apiClient.GetProjects(ctx, cfg.OrganizationID)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	switch len(projects) {
	case 0: // No projects found
		return c.handleNeedExistingProjectCaseNoProjects()
	case 1: // One project found
		return c.handleNeedExistingProjectCaseOneProject(result, cfg, projects)
	default: // Multiple projects found
		return c.handleNeedExistingProjectCaseMultipleProjects(ctx, result, cfg)
	}
}

func (c *Controller) handleNeedExistingProjectCaseNoProjects() error {
	// TODO: printNoProjectsInOrganization()
	return ErrProjectSelectionCanceled
}

func (c *Controller) handleNeedExistingProjectCaseOneProject(
	result *models.CommandState,
	cfg *config.Config,
	projects []models.Project,
) error {
	c.updateProject(cfg, &projects[0], result)
	return nil
}

func (c *Controller) handleNeedExistingProjectCaseMultipleProjects(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
) error {
	if result.Organization == nil {
		return eris.New("Organization is nil")
	}

	// First check if we already have a selected project
	selectedProj, err := c.apiClient.GetProjectByID(ctx, result.Organization.ID, cfg.ProjectID)
	if err == nil && selectedProj.ID != "" {
		c.updateProject(cfg, &selectedProj, result)
		return nil
	}

	proj, err := c.projectHandler.Switch(ctx, models.SwitchProjectFlags{}, false)
	if err != nil {
		return eris.Wrap(err, "Flow failed to select project in existing multiple-projects case")
	}
	c.updateProject(cfg, &proj, result)
	return nil
}

////////////////////////////////
// Helper Functions           //
////////////////////////////////

// updateProject updates the project in the flow state and saves the config.
func (c *Controller) updateProject(cfg *config.Config, project *models.Project, result *models.CommandState) {
	cfg.ProjectID = project.ID
	cfg.CurrProjectName = project.Name
	result.Project = project
}
