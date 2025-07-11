package project

import (
	"context"
	"strconv"
	"strings"

	"github.com/rotisserie/eris"
	cmdsetup "pkg.world.dev/world-cli/internal/app/world-cli/controllers/cmd_setup"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
	"pkg.world.dev/world-cli/internal/pkg/printer"
)

//nolint:gocognit,funlen // Belongs in a single function
func (h *Handler) Switch(
	ctx context.Context,
	flags models.SwitchProjectFlags,
	org models.Organization,
	enableCreation bool,
) (models.Project, error) {
	if h.configService.GetConfig().CurrRepoKnown {
		printer.Errorf("Cannot switch Project, current git working directory belongs to project: %s.",
			h.configService.GetConfig().CurrProjectName)
		return models.Project{}, ErrCannotCreateSwitchProject
	}

	// Get projects from selected organization
	projects, err := h.apiClient.GetProjects(ctx, org.ID)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to get projects")
	}

	if len(projects) == 0 {
		if enableCreation {
			proj, err := h.Create(ctx, org, models.CreateProjectFlags{})
			if err != nil {
				return models.Project{}, eris.Wrap(err, "Failed to create project")
			}
			return proj, nil
		}
		h.PrintNoProjectsInOrganization()
		return models.Project{}, nil
	}

	// If slug is provided, select the project by slug
	if flags.Slug != "" {
		return h.switchBySlug(ctx, projects, org, flags.Slug)
	}

	// Display projects as a numbered list
	printer.NewLine(1)
	printer.Headerln("   Available Projects   ")
	for i, proj := range projects {
		printer.Infof("  %d. %s\n     └─ Slug: %s\n", i+1, proj.Name, proj.Slug)
	}

	inRepoRoot := false
	prompt := "Enter project number ('q' to quit)"
	if enableCreation {
		_, _, err = h.PreCreateUpdateValidation(false)
		if err == nil { // if in repo root, we can create a new project
			inRepoRoot = true
			prompt = "Enter project number ('c' to create new, 'q' to quit)"
		}
	}

	for {
		printer.NewLine(1)
		input, err := h.inputService.Prompt(ctx, prompt, "")
		if err != nil {
			return models.Project{}, eris.Wrap(err, "Failed to prompt")
		}
		input = strings.TrimSpace(input)
		if input == "q" {
			return models.Project{}, nil
		}

		if input == "c" && inRepoRoot {
			proj, err := h.Create(ctx, org, models.CreateProjectFlags{})
			if err != nil {
				return models.Project{}, eris.Wrap(err, "Failed to create project")
			}
			return proj, nil
		}

		// Parse selection
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(projects) {
			printer.Errorf("Please enter a number between 1 and %d\n", len(projects))
			continue
		}

		selectedProject := projects[num-1]

		err = h.saveToConfig(&selectedProject)
		if err != nil {
			return models.Project{}, eris.Wrap(err, "selectProject")
		}

		printer.NewLine(1)
		printer.Successf("Switched to project: %s\n", selectedProject.Name)
		return selectedProject, nil
	}
}

func (h *Handler) switchBySlug(
	ctx context.Context,
	projects []models.Project,
	org models.Organization,
	slug string,
) (models.Project, error) {
	for _, project := range projects {
		if project.Slug == slug {
			err := h.saveToConfig(&project)
			if err != nil {
				return models.Project{}, eris.Wrap(err, "selectProjectBySlug")
			}
			err = h.showProjectList(ctx, project, org)
			if err != nil {
				return models.Project{}, eris.Wrap(err, "Failed to show project list")
			}
			return project, nil
		}
	}
	err := h.showProjectList(ctx, models.Project{}, org)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to show project list")
	}
	printer.NewLine(1)
	printer.Errorln("Project not found in organization under the slug: " + slug)
	return models.Project{}, cmdsetup.ErrProjectSelectionCanceled
}

// HandleSwitch manages the project selection logic.
func (h *Handler) HandleSwitch(ctx context.Context, org models.Organization) error {
	projects, err := h.apiClient.GetProjects(ctx, org.ID)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	switch numProjects := len(projects); {
	case numProjects == 1:
		return h.handleSingleProject(ctx, projects[0], org)
	case numProjects > 1:
		return h.handleMultipleProjects(ctx, org)
	default:
		return h.handleNoProjects(ctx, org)
	}
}

func (h *Handler) handleSingleProject(
	ctx context.Context,
	project models.Project,
	org models.Organization,
) error {
	h.saveToConfig(&project)
	h.showProjectList(ctx, project, org)
	return nil
}

// handleMultipleProjects handles the case when there are multiple projects.
func (h *Handler) handleMultipleProjects(
	ctx context.Context,
	org models.Organization,
) error {
	project, err := h.Switch(ctx, models.SwitchProjectFlags{}, org, false)
	if err != nil {
		return eris.Wrap(err, "Failed to select project")
	}

	h.saveToConfig(&project)
	return nil
}

// handleNoProjects handles the case when there are no projects.
func (h *Handler) handleNoProjects(ctx context.Context, org models.Organization) error {
	_, _, err := h.PreCreateUpdateValidation(true)
	if err != nil {
		printRequiredStepsToCreateProject()
		return nil //nolint:nilerr // error here is representing a boolean
	}

	printer.NewLine(1)
	confirmation, err := h.inputService.Confirm(ctx, "Do you want to create a new project now? (y/n)", "Y")
	if err != nil {
		return eris.Wrap(err, "Failed to prompt")
	}

	if !confirmation {
		printer.Errorln("Project creation canceled")
		return nil
	}

	project, err := h.Create(ctx, org, models.CreateProjectFlags{})
	if err != nil {
		return eris.Wrap(err, "Failed to create project")
	}

	h.saveToConfig(&project)
	return nil
}
