package project

import (
	"context"
	"strings"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/clients/repo"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/utils"
	"pkg.world.dev/world-cli/common/printer"
)

// Get list of projects in selected organization.
func (h *Handler) getListOfProjects(ctx context.Context) ([]models.Project, error) {
	selectedOrg, err := h.apiClient.GetOrganizationByID(ctx, h.configService.GetConfig().OrganizationID)
	if err != nil && !eris.Is(err, api.ErrNoOrganizationID) {
		return nil, eris.Wrap(err, "Failed to get organization")
	}

	if selectedOrg.ID == "" {
		printNoSelectedOrganization()
		return nil, nil
	}

	projects, err := h.apiClient.GetProjects(ctx, selectedOrg.ID)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get projects")
	}

	return projects, nil
}

// Show list of projects in selected organization.
func (h *Handler) showProjectList(ctx context.Context) error {
	projects, err := h.getListOfProjects(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	if len(projects) == 0 {
		h.PrintNoProjectsInOrganization()
		return nil
	}

	selectedProject, err := h.getSelectedProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get selected project")
	}

	printer.NewLine(1)
	printer.Headerln("   Project Information   ")
	if selectedProject.Name == "" {
		printer.Errorln("No project selected")
		printer.NewLine(1)
		printer.Infoln("Use 'world forge project switch' to choose a project")
	} else {
		for _, prj := range projects {
			if prj.ID == selectedProject.ID {
				printer.Infof("• %s (%s) [SELECTED]\n", prj.Name, prj.Slug)
			} else {
				printer.Infof("  %s (%s)\n", prj.Name, prj.Slug)
			}
		}
	}

	return nil
}

func (h *Handler) getSelectedProject(ctx context.Context) (models.Project, error) {
	selectedOrg, err := h.apiClient.GetOrganizationByID(ctx, h.configService.GetConfig().OrganizationID)
	if err != nil && !eris.Is(err, api.ErrNoOrganizationID) {
		return models.Project{}, eris.Wrap(err, "Failed to get organization")
	}

	if selectedOrg.ID == "" {
		printNoSelectedOrganization()
		return models.Project{}, nil
	}

	if h.configService.GetConfig().ProjectID == "" {
		projects, err := h.getListOfProjects(ctx)
		if err != nil {
			return models.Project{}, eris.Wrap(err, "Failed to get projects")
		}
		if len(projects) == 0 {
			h.PrintNoProjectsInOrganization()
		}
		return models.Project{}, nil
	}

	// Send request
	project, err := h.apiClient.GetProjectByID(ctx, selectedOrg.ID, h.configService.GetConfig().ProjectID)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to get project")
	}

	// Parse response
	return project, nil
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

	inRoot, err := utils.IsInWorldCardinalRoot()
	if err != nil {
		lastError = eris.Wrap(err, "Failed to check if in World project root")
	} else if !inRoot {
		if printError {
			printer.Errorln(" Not in a World project root")
		}
		lastError = utils.ErrNotInWorldCardinalRoot
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

func printNoSelectedProject() {
	printer.NewLine(1)
	printer.Headerln("   No Project Selected   ")
	printer.Infoln("You don't have any project selected.")
	printer.Info("Use ")
	printer.Notification("'world project switch'")
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
