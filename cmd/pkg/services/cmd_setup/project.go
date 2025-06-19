package cmdsetup

import (
	"context"
	"fmt"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/pkg/clients/config"
	"pkg.world.dev/world-cli/cmd/pkg/models"
	"pkg.world.dev/world-cli/common/printer"
)

var (
	ErrProjectSelectionCanceled = eris.New("Project selection canceled")
	ErrProjectCreationCanceled  = eris.New("Project creation canceled")
)

/////////////////////
// Need Project    //
/////////////////////

func (s *Service) handleNeedProjectData(ctx context.Context, result *models.CommandState, cfg *config.Config) error {
	projects, err := s.apiClient.GetProjects(ctx, cfg.OrganizationID)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	switch len(projects) {
	case 0: // No projects found
		return s.handleNeedProjectCaseNoProjects(ctx, result, cfg)
	case 1: // One project found
		return s.handleNeedProjectCaseOneProject(ctx, result, cfg, projects)
	default: // Multiple projects found
		return s.handleNeedProjectCaseMultipleProjects(ctx, result, cfg, projects)
	}
}

func (s *Service) handleNeedProjectCaseNoProjects(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
) error {
	for {
		// Must be in a valid directory to create a project
		_, _, err := s.projectHandler.ProjectPreCreateUpdateValidation()
		if err != nil {
			// TODO: printNoProjectsInOrganization()
			return ErrProjectCreationCanceled
		}

		choice := getInput("Would you like to create a new project? (Y/n)", "Y")

		switch choice {
		case "Y":
			proj, err := s.projectHandler.CreateProject(ctx, models.CreateProjectFlags{})
			if err != nil {
				return eris.Wrap(err, "Flow failed to create project in no-projects case")
			}
			s.updateProject(cfg, &proj, result)
			return nil
		case "n":
			return ErrProjectCreationCanceled
		default:
			printer.NewLine(1)
			printer.Infoln("Please select capital 'Y' or lowercase 'n'")
		}
	}
}

func (s *Service) handleNeedProjectCaseOneProject(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
	projects []models.Project,
) error {
	printer.NewLine(1)

	inRepoRoot := true
	prompt := fmt.Sprintf("Select project: %s [%s]? (Y/n/c to create new)", projects[0].Name, projects[0].Slug)
	_, _, err := s.projectHandler.ProjectPreCreateUpdateValidation()
	if err != nil {
		inRepoRoot = false
		prompt = fmt.Sprintf("Select project: %s [%s]? (Y/n)", projects[0].Name, projects[0].Slug)
	}

	for {
		choice := getInput(prompt, "Y")

		switch choice {
		case "Y":
			s.updateProject(cfg, &projects[0], result)
			return nil
		case "n":
			return ErrProjectSelectionCanceled
		case "c":
			if inRepoRoot {
				proj, err := s.projectHandler.CreateProject(ctx, models.CreateProjectFlags{})
				if err != nil {
					return eris.Wrap(err, "Flow failed to create project in one-project case")
				}
				s.updateProject(cfg, &proj, result)
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

func (s *Service) handleNeedProjectCaseMultipleProjects(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
	projects []models.Project,
) error {
	proj, err := s.projectHandler.PromptForProject(ctx, projects, true)
	if err != nil {
		return eris.Wrap(err, "Flow failed to select project in multiple-projects case")
	}
	s.updateProject(cfg, &proj, result)
	return nil
}

////////////////////////////////
// Need Existing Project      //
////////////////////////////////

func (s *Service) handleNeedExistingProjectData(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
) error {
	projects, err := s.apiClient.GetProjects(ctx, cfg.OrganizationID)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	switch len(projects) {
	case 0: // No projects found
		return s.handleNeedExistingProjectCaseNoProjects()
	case 1: // One project found
		return s.handleNeedExistingProjectCaseOneProject(result, cfg, projects)
	default: // Multiple projects found
		return s.handleNeedExistingProjectCaseMultipleProjects(ctx, result, cfg, projects)
	}
}

func (s *Service) handleNeedExistingProjectCaseNoProjects() error {
	// TODO: printNoProjectsInOrganization()
	return ErrProjectSelectionCanceled
}

func (s *Service) handleNeedExistingProjectCaseOneProject(
	result *models.CommandState,
	cfg *config.Config,
	projects []models.Project,
) error {
	s.updateProject(cfg, &projects[0], result)
	return nil
}

func (s *Service) handleNeedExistingProjectCaseMultipleProjects(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
	projects []models.Project,
) error {
	// First check if we already have a selected project
	selectedProj, err := s.apiClient.GetProjectByID(ctx, cfg.ProjectID)
	if err == nil && selectedProj.ID != "" {
		s.updateProject(cfg, &selectedProj, result)
		return nil
	}

	proj, err := s.projectHandler.PromptForProject(ctx, projects, false)
	if err != nil {
		return eris.Wrap(err, "Flow failed to select project in existing multiple-projects case")
	}
	s.updateProject(cfg, &proj, result)
	return nil
}

////////////////////////////////
// Helper Functions           //
////////////////////////////////

// updateProject updates the project in the flow state and saves the config.
func (s *Service) updateProject(cfg *config.Config, project *models.Project, result *models.CommandState) {
	cfg.ProjectID = project.ID
	cfg.CurrProjectName = project.Name
	result.Project = project
}
