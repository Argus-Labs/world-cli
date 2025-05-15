package forge

import (
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/printer"
)

var (
	ErrProjectSelectionCanceled = eris.New("Project selection canceled")
	ErrProjectCreationCanceled  = eris.New("Project creation canceled")
)

/////////////////////
// Need Project    //
/////////////////////

func (flow *initFlow) handleNeedProjectData() error {
	projects, err := getListOfProjects(flow.State.Command.Context())
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	found := flow.getProjectByID(projects)
	if found {
		return nil
	}

	switch len(projects) {
	case 0: // No projects found
		return flow.handleNeedProjectCaseNoProjects()
	case 1: // One project found
		return flow.handleNeedProjectCaseOneProject(projects)
	default: // Multiple projects found
		return flow.handleNeedProjectCaseMultipleProjects()
	}
}

func (flow *initFlow) handleNeedProjectCaseNoProjects() error {
	for {
		printer.NewLine(1)
		printer.Infoln("No projects found.")
		choice := getInput("Would you like to create one? (Y/n)", "Y")

		switch choice {
		case "Y":
			proj, err := createProject(flow.State.Command.Context())
			if err != nil {
				return eris.Wrap(err, "Flow failed to create project in no-projects case")
			}
			flow.updateProject(proj)
			return nil
		case "n":
			return ErrProjectCreationCanceled
		default:
			printer.NewLine(1)
			printer.Infoln("Please select capital 'Y' or lowercase 'n'")
		}
	}
}

func (flow *initFlow) handleNeedProjectCaseOneProject(projects []project) error {
	printer.NewLine(1)
	printer.Infof("Project: %s [%s]\n", projects[0].Name, projects[0].Slug)

	for {
		choice := getInput("Use this project? (Y/n/c to create new)", "Y")

		switch choice {
		case "Y":
			flow.updateProject(&projects[0])
			return nil
		case "n":
			return ErrProjectSelectionCanceled
		case "c":
			proj, err := createProject(flow.State.Command.Context())
			if err != nil {
				return eris.Wrap(err, "Flow failed to create project in one-project case")
			}
			flow.updateProject(proj)
			return nil
		default:
			printer.Infoln("Please select capital 'Y' or lowercase 'n'/'c'")
			printer.NewLine(1)
		}
	}
}

func (flow *initFlow) handleNeedProjectCaseMultipleProjects() error {
	proj, err := selectProject(flow.State.Command.Context())
	if err != nil {
		return eris.Wrap(err, "Flow failed to select project in multiple-projects case")
	}
	if proj == nil {
		return ErrProjectSelectionCanceled
	}
	flow.updateProject(proj)
	return nil
}

////////////////////////////////
// Need Existing Project      //
////////////////////////////////

func (flow *initFlow) handleNeedExistingProjectData() error {
	projects, err := getListOfProjects(flow.State.Command.Context())
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	found := flow.getProjectByID(projects)
	if found {
		return nil
	}

	// First check if we already have a selected project
	selectedProj, err := getSelectedProject(flow.State.Command.Context())
	if err != nil {
		return eris.Wrap(err, "Failed to get selected project")
	}

	// If we have a selected project, use it
	if selectedProj.ID != "" {
		flow.updateProject(&selectedProj)
		return nil
	}

	switch len(projects) {
	case 0: // No projects found
		return flow.handleNeedExistingProjectCaseNoProjects()
	case 1: // One project found
		return flow.handleNeedExistingProjectCaseOneProject(projects)
	default: // Multiple projects found
		return flow.handleNeedExistingProjectCaseMultipleProjects()
	}
}

func (flow *initFlow) handleNeedExistingProjectCaseNoProjects() error {
	printer.NewLine(1)
	printer.Errorln("No projects found.")
	printer.Infoln("You must have at least one project to proceed.")
	return ErrProjectSelectionCanceled
}

func (flow *initFlow) handleNeedExistingProjectCaseOneProject(projects []project) error {
	printer.NewLine(1)
	printer.Infof("Project: %s [%s]\n", projects[0].Name, projects[0].ID)
	flow.updateProject(&projects[0])
	return nil
}

func (flow *initFlow) handleNeedExistingProjectCaseMultipleProjects() error {
	proj, err := selectProject(flow.State.Command.Context())
	if err != nil {
		return eris.Wrap(err, "Flow failed to select project in existing multiple-projects case")
	}
	if proj == nil {
		return ErrProjectSelectionCanceled
	}
	flow.updateProject(proj)
	return nil
}

////////////////////////////////
// Helper Functions           //
////////////////////////////////

// getProjectByID checks if a project with the ID from config exists in the provided list.
// If found, it sets the project in the flow state and returns true.
func (flow *initFlow) getProjectByID(projects []project) bool {
	if flow.config.ProjectID == "" {
		return false
	}

	for _, project := range projects {
		if flow.config.ProjectID == project.ID {
			flow.updateProject(&project)
			return true
		}
	}
	return false
}

// updateProject updates the project in the flow state and saves the config.
func (flow *initFlow) updateProject(project *project) {
	flow.State.Project = project
	flow.projectStepDone = true
	flow.AddKnownProject(project)

	flow.config.ProjectID = project.ID
	flow.config.CurrProjectName = project.Name
	flow.config.CurrRepoKnown = true

	err := SaveForgeConfig(flow.config)
	if err != nil {
		printer.Notificationf("Warning: Failed to save config: %s", err)
		logger.Error(eris.Wrap(err, "Project flow failed to save config"))
		// continue on, this is not fatal
	}
}
