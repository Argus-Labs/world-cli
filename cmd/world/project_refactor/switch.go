package project

import (
	"context"
	"strconv"
	"strings"

	"github.com/rotisserie/eris"
	cmdsetup "pkg.world.dev/world-cli/cmd/internal/controllers/cmd_setup"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/common/printer"
)

//nolint:gocognit,funlen // Belongs in a single function
func (h *Handler) Switch(
	ctx context.Context,
	state *models.CommandState,
	flags models.SwitchProjectFlags,
	createNew bool,
) (models.Project, error) {
	if h.configService.GetConfig().CurrRepoKnown {
		printer.Errorf("Cannot switch Project, current git working directory belongs to project: %s.",
			h.configService.GetConfig().CurrProjectName)
		return models.Project{}, ErrCannotCreateSwitchProject
	}

	// Get projects from selected organization
	projects, err := h.apiClient.GetProjects(ctx, h.configService.GetConfig().OrganizationID)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to get projects")
	}

	if len(projects) == 0 {
		if createNew {
			proj, err := h.Create(ctx, state, models.CreateProjectFlags{})
			if err != nil {
				return models.Project{}, eris.Wrap(err, "Failed to create project")
			}
			return proj, nil
		}
		printNoProjectsInOrganization()
		return models.Project{}, nil
	}

	// If slug is provided, select the project by slug
	if flags.Slug != "" {
		return h.switchBySlug(ctx, projects, flags.Slug)
	}

	// Display projects as a numbered list
	printer.NewLine(1)
	printer.Headerln("   Available Projects   ")
	for i, proj := range projects {
		printer.Infof("  %d. %s\n     └─ Slug: %s\n", i+1, proj.Name, proj.Slug)
	}

	inRepoRoot := false
	prompt := "Enter project number ('q' to quit)"
	if createNew {
		_, _, err = h.PreCreateUpdateValidation()
		if err != nil { // if in repo root, we can create a new project
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
			proj, err := h.Create(ctx, state, models.CreateProjectFlags{})
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

func (h *Handler) switchBySlug(ctx context.Context, projects []models.Project, slug string) (models.Project, error) {
	for _, project := range projects {
		if project.Slug == slug {
			err := h.saveToConfig(&project)
			if err != nil {
				return models.Project{}, eris.Wrap(err, "selectProjectBySlug")
			}
			err = h.showProjectList(ctx)
			if err != nil {
				return models.Project{}, eris.Wrap(err, "Failed to show project list")
			}
			return project, nil
		}
	}
	err := h.showProjectList(ctx)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to show project list")
	}
	printer.NewLine(1)
	printer.Errorln("Project not found in organization under the slug: " + slug)
	return models.Project{}, cmdsetup.ErrProjectSelectionCanceled
}
