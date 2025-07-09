package project

import (
	"context"
	"fmt"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
	"pkg.world.dev/world-cli/common/printer"
)

func (h *Handler) Delete(
	ctx context.Context,
	project models.Project,
) error {
	// Print project details with fancy formatting
	printer.NewLine(1)
	printer.Headerln("   Project Deletion   ")
	printer.Infoln("Project Details:")
	printer.Infof("• Name: %s\n", project.Name)
	printer.Infof("• Slug: %s\n", project.Slug)

	// Warning message with fancy formatting
	printer.NewLine(1)
	printer.Headerln("  ⚠️WARNING!⚠️  ")
	printer.Infoln("This action will permanently delete:")
	printer.Infoln("• All deployments")
	printer.Infoln("• All logs")
	printer.Infoln("• All associated resources")
	printer.NewLine(1)

	prompt := fmt.Sprintf("Type 'Yes' to confirm deletion of '%s (%s)'", project.Name, project.Slug)
	confirmation, err := h.inputService.Prompt(ctx, prompt, "no")
	if err != nil {
		return eris.Wrap(err, "Failed to prompt for confirmation")
	}

	if confirmation != "Yes" {
		if confirmation == "yes" {
			printer.Errorln("You must type 'Yes' with uppercase Y to confirm deletion")
		}
		printer.Errorln("Project deletion canceled")
		return nil
	}

	// Send request
	err = h.apiClient.DeleteProject(ctx, project.OrgID, project.ID)
	if err != nil {
		return eris.Wrap(err, "Failed to delete project")
	}

	err = h.configService.RemoveKnownProject(project.ID, project.OrgID)
	if err != nil {
		printer.Errorln("Project deleted from backend, but failed to remove project from local config")
		printer.Errorln("Current Project directory might be corrupted")
		return eris.Wrap(err, "Failed to remove project from config")
	}

	printer.Successf("Project deleted: %s (%s)\n", project.Name, project.Slug)

	return nil
}
