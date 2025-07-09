package cmdsetup

import (
	"context"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
	"pkg.world.dev/world-cli/cmd/world/internal/services/config"
	"pkg.world.dev/world-cli/common/printer"
)

var (
	ErrOrganizationSelectionCanceled = eris.New("Organization selection canceled")
	ErrOrganizationCreationCanceled  = eris.New("Organization creation canceled")
)

///////////////////////
// Need Organization //
///////////////////////

func (c *Controller) handleNeedOrgData(ctx context.Context, result *models.CommandState, cfg *config.Config) error {
	orgs, err := c.apiClient.GetOrganizations(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organizations")
	}

	switch len(orgs) {
	case 0: // No organizations found
		return c.handleNeedOrganizationCaseNoOrgs(ctx, result, cfg)
	case 1: // One organization found
		return c.handleNeedOrganizationCaseOneOrg(ctx, result, cfg, orgs)
	default: // Multiple organizations found
		return c.handleNeedOrganizationCaseMultipleOrgs(ctx, result, cfg, orgs)
	}
}

func (c *Controller) handleNeedOrganizationCaseNoOrgs(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
) error {
	printer.NewLine(1)
	printer.Infoln("No organizations found.")
	confirm, err := c.inputService.Confirm(ctx, "Would you like to create one? (Y/n)", "Y")
	if err != nil {
		return eris.Wrap(err, "failed to get input")
	}

	if confirm {
		org, err := c.organizationHandler.Create(ctx, models.CreateOrganizationFlags{})
		if err != nil {
			return eris.Wrap(err, "Flow failed to create organization in no-orgs case")
		}
		c.updateOrganization(cfg, &org, result)
		return nil
	}

	return ErrOrganizationCreationCanceled
}

func (c *Controller) handleNeedOrganizationCaseOneOrg(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
	orgs []models.Organization,
) error {
	printer.NewLine(1)
	printer.Infof("Found one organization: %s [%s]\n", orgs[0].Name, orgs[0].Slug)

	for {
		choice, err := c.inputService.Prompt(ctx, "Use this organization? (Y/n/c to create new)", "Y")
		if err != nil {
			return eris.Wrap(err, "failed to get input")
		}

		switch choice {
		case "Y":
			c.updateOrganization(cfg, &orgs[0], result)
			return nil
		case "n":
			return ErrOrganizationSelectionCanceled
		case "c":
			org, err := c.organizationHandler.Create(ctx, models.CreateOrganizationFlags{})
			if err != nil {
				return eris.Wrap(err, "Flow failed to create organization in one-org case")
			}
			c.updateOrganization(cfg, &org, result)
			return nil
		default:
			printer.NewLine(1)
			printer.Infoln("Please select capital 'Y' or lowercase 'n'/'c'")
		}
	}
}

func (c *Controller) handleNeedOrganizationCaseMultipleOrgs(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
	orgs []models.Organization,
) error {
	org, err := c.organizationHandler.PromptForSwitch(ctx, orgs, true)
	if err != nil {
		return eris.Wrap(err, "Flow failed to prompt for organization in multiple-orgs case")
	}
	c.updateOrganization(cfg, &org, result)
	return nil
}

////////////////////////////////
// Need Existing Organization //
////////////////////////////////

func (c *Controller) handleNeedExistingOrgData(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
) error {
	// No org selected, get list of organizations
	orgs, err := c.apiClient.GetOrganizations(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organizations")
	}

	switch len(orgs) {
	case 0: // No organizations found
		return c.handleNeedExistingOrganizationCaseNoOrgs()
	case 1: // One organization found
		return c.handleNeedExistingOrganizationCaseOneOrg(result, cfg, orgs)
	default: // Multiple organizations found
		return c.handleNeedExistingOrganizationCaseMultipleOrgs(ctx, result, cfg, orgs)
	}
}

func (c *Controller) handleNeedExistingOrganizationCaseNoOrgs() error {
	c.organizationHandler.PrintNoOrganizations()
	return ErrOrganizationSelectionCanceled
}

func (c *Controller) handleNeedExistingOrganizationCaseOneOrg(
	result *models.CommandState,
	cfg *config.Config,
	orgs []models.Organization,
) error {
	c.updateOrganization(cfg, &orgs[0], result)
	return nil
}

func (c *Controller) handleNeedExistingOrganizationCaseMultipleOrgs(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
	orgs []models.Organization,
) error {
	// First check if we already have a selected organization
	selectedOrg, err := c.apiClient.GetOrganizationByID(ctx, cfg.OrganizationID)
	if err == nil && selectedOrg.ID != "" {
		c.updateOrganization(cfg, &selectedOrg, result)
		return nil
	}

	org, err := c.organizationHandler.PromptForSwitch(ctx, orgs, false)
	if err != nil {
		return eris.Wrap(err, "Flow failed to prompt for organization in existing multiple-orgs case")
	}
	c.updateOrganization(cfg, &org, result)
	return nil
}

////////////////////////////////
// Helper Functions           //
////////////////////////////////

// updateOrganization updates the organization in the flow state and saves the config.
func (c *Controller) updateOrganization(cfg *config.Config, org *models.Organization, result *models.CommandState) {
	cfg.OrganizationID = org.ID
	result.Organization = org
}
