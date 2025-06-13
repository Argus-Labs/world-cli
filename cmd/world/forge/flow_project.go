package forge

import (
	"fmt"

	"github.com/rotisserie/eris"
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
		// Must be in a valid directory to create a project
		_, _, err := projectPreCreateUpdateValidation()
		if err != nil {
			printNoProjectsInOrganization()
			return ErrProjectCreationCanceled
		}

		choice := getInput("Would you like to create a new project? (Y/n)", "Y")

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

	inRepoRoot := true
	prompt := fmt.Sprintf("Select project: %s [%s]? (Y/n/c to create new)", projects[0].Name, projects[0].Slug)
	_, _, err := projectPreCreateUpdateValidation()
	if err != nil {
		inRepoRoot = false
		prompt = fmt.Sprintf("Select project: %s [%s]? (Y/n)", projects[0].Name, projects[0].Slug)
	}

	for {
		choice := getInput(prompt, "Y")

		switch choice {
		case "Y":
			flow.updateProject(fCtx, &projects[0])
			return nil
		case "n":
			return ErrProjectSelectionCanceled
		case "c":
			if inRepoRoot {
				proj, err := createProject(*fCtx, &CreateProjectCmd{})
				if err != nil {
					return eris.Wrap(err, "Flow failed to create project in one-project case")
				}
				flow.updateProject(fCtx, proj)
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

func (flow *initFlow) handleNeedProjectCaseMultipleProjects(fCtx *ForgeContext) error {
	proj, err := selectProject(*fCtx, &SwitchProjectCmd{}, true)
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

	proj, err := selectProject(*fCtx, &SwitchProjectCmd{}, false)
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
}
