package cmdsetup

import (
	"context"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/pkg/clients/config"
	"pkg.world.dev/world-cli/cmd/pkg/models"
	"pkg.world.dev/world-cli/common/printer"
)

var (
	ErrOrganizationSelectionCanceled = eris.New("Organization selection canceled")
	ErrOrganizationCreationCanceled  = eris.New("Organization creation canceled")
)

///////////////////////
// Need Organization //
///////////////////////

func (s *Service) handleNeedOrgData(ctx context.Context, result *models.CommandState, cfg *config.Config) error {
	orgs, err := s.apiClient.GetOrganizations(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organizations")
	}

	switch len(orgs) {
	case 0: // No organizations found
		return s.handleNeedOrganizationCaseNoOrgs(ctx, result, cfg)
	case 1: // One organization found
		return s.handleNeedOrganizationCaseOneOrg(ctx, result, cfg, orgs)
	default: // Multiple organizations found
		return s.handleNeedOrganizationCaseMultipleOrgs(ctx, result, cfg, orgs)
	}
}

func (s *Service) handleNeedOrganizationCaseNoOrgs(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
) error {
	for {
		printer.NewLine(1)
		printer.Infoln("No organizations found.")
		choice, err := s.inputClient.Prompt(ctx, "Would you like to create one? (Y/n)", "Y")
		if err != nil {
			return eris.Wrap(err, "failed to get input")
		}

		switch choice {
		case "Y":
			org, err := s.organizationHandler.CreateOrganization(ctx, models.CreateOrganizationFlags{})
			if err != nil {
				return eris.Wrap(err, "Flow failed to create organization in no-orgs case")
			}
			s.updateOrganization(cfg, &org, result)
			return nil
		case "n":
			return ErrOrganizationCreationCanceled
		default:
			printer.Infoln("Please select capital 'Y' or lowercase 'n'")
			printer.NewLine(1)
		}
	}
}

func (s *Service) handleNeedOrganizationCaseOneOrg(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
	orgs []models.Organization,
) error {
	printer.NewLine(1)
	printer.Infof("Found one organization: %s [%s]\n", orgs[0].Name, orgs[0].Slug)

	for {
		choice, err := s.inputClient.Prompt(ctx, "Use this organization? (Y/n/c to create new)", "Y")
		if err != nil {
			return eris.Wrap(err, "failed to get input")
		}

		switch choice {
		case "Y":
			s.updateOrganization(cfg, &orgs[0], result)
			return nil
		case "n":
			return ErrOrganizationSelectionCanceled
		case "c":
			org, err := s.organizationHandler.CreateOrganization(ctx, models.CreateOrganizationFlags{})
			if err != nil {
				return eris.Wrap(err, "Flow failed to create organization in one-org case")
			}
			s.updateOrganization(cfg, &org, result)
			return nil
		default:
			printer.NewLine(1)
			printer.Infoln("Please select capital 'Y' or lowercase 'n'/'c'")
		}
	}
}

func (s *Service) handleNeedOrganizationCaseMultipleOrgs(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
	orgs []models.Organization,
) error {
	org, err := s.organizationHandler.PromptForOrganization(ctx, orgs, true)
	if err != nil {
		return eris.Wrap(err, "Flow failed to prompt for organization in multiple-orgs case")
	}
	s.updateOrganization(cfg, &org, result)
	return nil
}

////////////////////////////////
// Need Existing Organization //
////////////////////////////////

func (s *Service) handleNeedExistingOrgData(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
) error {
	// No org selected, get list of organizations
	orgs, err := s.apiClient.GetOrganizations(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organizations")
	}

	switch len(orgs) {
	case 0: // No organizations found
		return s.handleNeedExistingOrganizationCaseNoOrgs()
	case 1: // One organization found
		return s.handleNeedExistingOrganizationCaseOneOrg(result, cfg, orgs)
	default: // Multiple organizations found
		return s.handleNeedExistingOrganizationCaseMultipleOrgs(ctx, result, cfg, orgs)
	}
}

func (s *Service) handleNeedExistingOrganizationCaseNoOrgs() error {
	// TODO: printNoOrganizations()
	return ErrOrganizationSelectionCanceled
}

func (s *Service) handleNeedExistingOrganizationCaseOneOrg(
	result *models.CommandState,
	cfg *config.Config,
	orgs []models.Organization,
) error {
	s.updateOrganization(cfg, &orgs[0], result)
	return nil
}

func (s *Service) handleNeedExistingOrganizationCaseMultipleOrgs(
	ctx context.Context,
	result *models.CommandState,
	cfg *config.Config,
	orgs []models.Organization,
) error {
	// First check if we already have a selected organization
	selectedOrg, err := s.apiClient.GetOrganizationByID(ctx, cfg.OrganizationID)
	if err == nil && selectedOrg.ID != "" {
		s.updateOrganization(cfg, &selectedOrg, result)
		return nil
	}

	org, err := s.organizationHandler.PromptForOrganization(ctx, orgs, false)
	if err != nil {
		return eris.Wrap(err, "Flow failed to prompt for organization in existing multiple-orgs case")
	}
	s.updateOrganization(cfg, &org, result)
	return nil
}

////////////////////////////////
// Helper Functions           //
////////////////////////////////

// updateOrganization updates the organization in the flow state and saves the config.
func (s *Service) updateOrganization(cfg *config.Config, org *models.Organization, result *models.CommandState) {
	cfg.OrganizationID = org.ID
	result.Organization = org
}
