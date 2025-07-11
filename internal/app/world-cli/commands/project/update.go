package project

import (
	"context"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/internal/app/world-cli/clients/api"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
	"pkg.world.dev/world-cli/internal/pkg/printer"
)

func (h *Handler) Update(
	ctx context.Context,
	project models.Project,
	org models.Organization,
	flags models.UpdateProjectFlags,
) error {
	printer.Infof("Updating Project: %s [%s]\n", project.Name, project.Slug)

	repoPath, repoURL, err := h.PreCreateUpdateValidation(true)
	if err != nil {
		printRequiredStepsToCreateProject()
		return eris.Wrap(err, "Failed to validate project update")
	}

	regions, err := h.apiClient.GetListRegions(ctx, project.OrgID, project.ID)
	if err != nil {
		return eris.Wrap(err, "Failed to get available regions")
	}

	if flags.Name == "" {
		flags.Name = project.Name
	}
	if flags.Slug == "" {
		flags.Slug = project.Slug
	}

	// set update to true
	project.Update = true
	project.Name = flags.Name
	project.Slug = flags.Slug
	project.OrgID = org.ID
	project.RepoPath = repoPath
	project.RepoURL = repoURL

	printer.NewLine(1)
	printer.Headerln("  Project Update  ")

	// get project input
	err = h.getSetupInput(ctx, &project, regions)
	if err != nil {
		return eris.Wrap(err, "Failed to get project input")
	}

	printer.NewLine(1)
	printer.Infoln("Updating project...")

	// Send request
	prj, err := h.apiClient.UpdateProject(ctx, project.OrgID, project.ID, project)
	if err != nil {
		if eris.Is(err, api.ErrProjectSlugAlreadyExists) {
			printer.Errorf("Project already exists with slug: %s, please choose a different slug.\n", project.Slug)
			printer.NewLine(1)
		}
		return eris.Wrap(err, "Failed to update project")
	}

	h.configService.RemoveKnownProject(project.ID, project.OrgID)

	displayProjectDetails(&prj)
	printer.NewLine(1)
	printer.Successf("Project '%s [%s]' updated successfully!\n", prj.Name, prj.Slug)
	return nil
}
