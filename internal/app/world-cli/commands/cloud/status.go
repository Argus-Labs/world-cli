package cloud

import (
	"context"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
	"pkg.world.dev/world-cli/internal/pkg/printer"
)

func (h *Handler) Status(ctx context.Context, organization models.Organization, project models.Project) error {
	printer.NewLine(1)
	printer.Headerln("   Deployment Status   ")
	printer.Infof("Organization: %s\n", organization.Name)
	printer.Infof("Org Slug:     %s\n", organization.Slug)
	printer.Infof("Project:      %s\n", project.Name)
	printer.Infof("Project Slug: %s\n", project.Slug)
	printer.Infof("Repository:   %s\n", project.RepoURL)
	printer.NewLine(1)

	deployInfo, err := h.getDeploymentStatus(ctx, project)
	if err != nil {
		return eris.Wrap(err, "Failed to get deployment status")
	}
	showHealth := false
	for env := range deployInfo {
		printDeploymentStatus(deployInfo[env])
		if shouldShowHealth(deployInfo[env]) {
			showHealth = true
		}
	}

	if showHealth {
		// don't care about healthComplete return because we are only doing this once
		_, err = h.getAndPrintHealth(ctx, project, deployInfo)
		if err != nil {
			return eris.Wrap(err, "Failed to get health")
		}
	}
	return nil
}
