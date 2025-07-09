package project

import (
	"context"
	"strings"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/world/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/world/internal/clients/repo"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
	"pkg.world.dev/world-cli/cmd/world/internal/utils/validate"
	"pkg.world.dev/world-cli/common/printer"
)

// Show list of projects in selected organization.
// Input: project is the project to highlight in the list.
// If project is empty, no project is selected when the list is shown.
func (h *Handler) showProjectList(ctx context.Context, project models.Project, org models.Organization) error {
	if org.ID == "" {
		printNoSelectedOrganization()
		return nil
	}

	projects, err := h.apiClient.GetProjects(ctx, org.ID)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	if len(projects) == 0 {
		h.PrintNoProjectsInOrganization()
		return nil
	}

	printer.NewLine(1)
	printer.Headerln("   Project Information   ")
	for _, prj := range projects {
		if prj.ID == project.ID {
			printer.Infof("• %s (%s) [SELECTED]\n", prj.Name, prj.Slug)
		} else {
			printer.Infof("  %s (%s)\n", prj.Name, prj.Slug)
		}
	}

	return nil
}

// PreCreateUpdateValidation returns the repo path and URL, and an error.
func (h *Handler) PreCreateUpdateValidation(printError bool) (string, string, error) {
	var lastError error

	repoPath, repoURL, err := h.repoClient.FindGitPathAndURL()
	if err != nil && !strings.Contains(err.Error(), repo.ErrNotInGitRepository.Error()) {
		lastError = eris.Wrap(err, "Failed to find git path and URL")
	} else if repoURL == "" { // Empty URL means not in a git repository or no remotes configured
		if printError {
			printer.Errorln(" Not in a git repository")
		}
		lastError = repo.ErrNotInGitRepository
	}

	inRoot, err := validate.IsInWorldCardinalRoot()
	if err != nil {
		lastError = eris.Wrap(err, "Failed to check if in World project root")
	} else if !inRoot {
		if printError {
			printer.Errorln(" Not in a World project root")
		}
		lastError = repo.ErrNotInWorldCardinalRoot
	}
	return repoPath, repoURL, lastError
}

// Get list of projects in selected organization.
func (h *Handler) getListOfAvailableRegionsForNewProject(ctx context.Context) ([]string, error) {
	selectedOrg, err := h.apiClient.GetOrganizationByID(ctx, h.configService.GetConfig().OrganizationID)
	if err != nil && !eris.Is(err, api.ErrNoOrganizationID) {
		return nil, eris.Wrap(err, "Failed to get organization")
	}
	if selectedOrg.ID == "" {
		printNoSelectedOrganization()
		return nil, nil
	}
	return h.apiClient.GetListRegions(ctx, selectedOrg.ID, nilUUID)
}

func printRequiredStepsToCreateProject() {
	printer.NewLine(1)
	printer.Headerln(" To create a new project follow these steps ")
	printer.Infoln("1. Move to the root of your World project.")
	printer.Infoln("   This is the directory that contains world.toml and the cardinal directory")
	printer.Infoln("2. Must be within a git repository")
	printer.Info("3. Use command ")
	printer.Notificationln("'world project create'")
}

func (h *Handler) PrintNoProjectsInOrganization() {
	printer.NewLine(1)
	printer.Headerln("   No Projects Found   ")
	printer.Infoln("You don't have any projects in this organization yet.")
	printRequiredStepsToCreateProject()
}

func printNoSelectedOrganization() {
	printer.NewLine(1)
	printer.Headerln("   No Organization Selected   ")
	printer.Infoln("You don't have any organization selected.")
	printer.Info("Use ")
	printer.Notification("'world organization switch'")
	printer.Infoln(" to select one!")
}

func (h *Handler) saveToConfig(project *models.Project) error {
	h.configService.GetConfig().ProjectID = project.ID
	err := h.configService.Save()
	if err != nil {
		return eris.Wrap(err, "Failed to save project configuration")
	}
	return nil
}
