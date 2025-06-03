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

func (flow *initFlow) handleNeedProjectData(fCtx *ForgeContext) error {
	projects, err := getListOfProjects(*fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	switch len(projects) {
	case 0: // No projects found
		return flow.handleNeedProjectCaseNoProjects(fCtx)
	case 1: // One project found
		return flow.handleNeedProjectCaseOneProject(fCtx, projects)
	default: // Multiple projects found
		return flow.handleNeedProjectCaseMultipleProjects(fCtx)
	}
}

func (flow *initFlow) handleNeedProjectCaseNoProjects(fCtx *ForgeContext) error {
	for {
		printNoProjectsInOrganization()
		printer.NewLine(1)
		choice := getInput("If conditions are met, would you like to create a new project? (Y/n)", "Y")

		switch choice {
		case "Y":
			proj, err := createProject(*fCtx, &CreateProjectCmd{})
			if err != nil {
				return eris.Wrap(err, "Flow failed to create project in no-projects case")
			}
			flow.updateProject(fCtx, proj)
			return nil
		case "n":
			return ErrProjectCreationCanceled
		default:
			printer.NewLine(1)
			printer.Infoln("Please select capital 'Y' or lowercase 'n'")
		}
	}
}

func (flow *initFlow) handleNeedProjectCaseOneProject(fCtx *ForgeContext, projects []project) error {
	printer.NewLine(1)
	printer.Infof("Project: %s [%s]\n", projects[0].Name, projects[0].Slug)

	for {
		choice := getInput("Use this project? (Y/n/c to create new)", "Y")

		switch choice {
		case "Y":
			flow.updateProject(fCtx, &projects[0])
			return nil
		case "n":
			return ErrProjectSelectionCanceled
		case "c":
			proj, err := createProject(*fCtx, &CreateProjectCmd{})
			if err != nil {
				return eris.Wrap(err, "Flow failed to create project in one-project case")
			}
			flow.updateProject(fCtx, proj)
			return nil
		default:
			printer.Infoln("Please select capital 'Y' or lowercase 'n'/'c'")
			printer.NewLine(1)
		}
	}
}

func (flow *initFlow) handleNeedProjectCaseMultipleProjects(fCtx *ForgeContext) error {
	proj, err := selectProject(*fCtx, &SwitchProjectCmd{})
	if err != nil {
		return eris.Wrap(err, "Flow failed to select project in multiple-projects case")
	}
	if proj == nil {
		return ErrProjectSelectionCanceled
	}
	flow.updateProject(fCtx, proj)
	return nil
}

////////////////////////////////
// Need Existing Project      //
////////////////////////////////

func (flow *initFlow) handleNeedExistingProjectData(fCtx *ForgeContext) error {
	projects, err := getListOfProjects(*fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	switch len(projects) {
	case 0: // No projects found
		return flow.handleNeedExistingProjectCaseNoProjects()
	case 1: // One project found
		return flow.handleNeedExistingProjectCaseOneProject(fCtx, projects)
	default: // Multiple projects found
		return flow.handleNeedExistingProjectCaseMultipleProjects(fCtx)
	}
}

func (flow *initFlow) handleNeedExistingProjectCaseNoProjects() error {
	printNoProjectsInOrganization()
	return ErrProjectSelectionCanceled
}

func (flow *initFlow) handleNeedExistingProjectCaseOneProject(fCtx *ForgeContext, projects []project) error {
	flow.updateProject(fCtx, &projects[0])
	return nil
}

func (flow *initFlow) handleNeedExistingProjectCaseMultipleProjects(fCtx *ForgeContext) error {
	// First check if we already have a selected project
	selectedProj, err := getSelectedProject(*fCtx)
	if err == nil && selectedProj.ID != "" {
		flow.updateProject(fCtx, &selectedProj)
		return nil
	}

	proj, err := selectProject(*fCtx, &SwitchProjectCmd{})
	if err != nil {
		return eris.Wrap(err, "Flow failed to select project in existing multiple-projects case")
	}
	if proj == nil {
		return ErrProjectSelectionCanceled
	}
	flow.updateProject(fCtx, proj)
	return nil
}

////////////////////////////////
// Helper Functions           //
////////////////////////////////

// updateProject updates the project in the flow state and saves the config.
func (flow *initFlow) updateProject(fCtx *ForgeContext, project *project) {
	fCtx.State.Project = project
	flow.projectStepDone = true

	fCtx.Config.ProjectID = project.ID
	fCtx.Config.CurrProjectName = project.Name

	err := fCtx.Config.Save()
	if err != nil {
		printer.Notificationf("Warning: Failed to save config: %s", err)
		logger.Error(eris.Wrap(err, "Project flow failed to save config"))
		// continue on, this is not fatal
	}
}
