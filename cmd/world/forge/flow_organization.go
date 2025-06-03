package forge

import (
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/printer"
)

var (
	ErrOrganizationSelectionCanceled = eris.New("Organization selection canceled")
	ErrOrganizationCreationCanceled  = eris.New("Organization creation canceled")
)

///////////////////////
// Need Organization //
///////////////////////

func (flow *initFlow) handleNeedOrgData(fCtx *ForgeContext) error {
	orgs, err := getListOfOrganizations(*fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organizations")
	}

	switch len(orgs) {
	case 0: // No organizations found
		return flow.handleNeedOrganizationCaseNoOrgs(fCtx)
	case 1: // One organization found
		return flow.handleNeedOrganizationCaseOneOrg(fCtx, orgs)
	default: // Multiple organizations found
		return flow.handleNeedOrganizationCaseMultipleOrgs(fCtx, orgs)
	}
}

func (flow *initFlow) handleNeedOrganizationCaseNoOrgs(fCtx *ForgeContext) error {
	for {
		printer.NewLine(1)
		printer.Infoln("No organizations found.")
		choice := getInput("Would you like to create one? (Y/n)", "Y")

		switch choice {
		case "Y":
			org, err := createOrganization(*fCtx, &CreateOrganizationCmd{})
			if err != nil {
				return eris.Wrap(err, "Flow failed to create organization in no-orgs case")
			}
			flow.updateOrganization(fCtx, org)
			return nil
		case "n":
			return ErrOrganizationCreationCanceled
		default:
			printer.Infoln("Please select capital 'Y' or lowercase 'n'")
			printer.NewLine(1)
		}
	}
}

func (flow *initFlow) handleNeedOrganizationCaseOneOrg(fCtx *ForgeContext, orgs []organization) error {
	printer.NewLine(1)
	printer.Infof("Found one organization: %s [%s]\n", orgs[0].Name, orgs[0].Slug)

	for {
		choice := getInput("Use this organization? (Y/n/c to create new)", "Y")

		switch choice {
		case "Y":
			flow.updateOrganization(fCtx, &orgs[0])
			return nil
		case "n":
			return ErrOrganizationSelectionCanceled
		case "c":
			org, err := createOrganization(*fCtx, &CreateOrganizationCmd{})
			if err != nil {
				return eris.Wrap(err, "Flow failed to create organization in one-org case")
			}
			flow.updateOrganization(fCtx, org)
			return nil
		default:
			printer.NewLine(1)
			printer.Infoln("Please select capital 'Y' or lowercase 'n'/'c'")
		}
	}
}

func (flow *initFlow) handleNeedOrganizationCaseMultipleOrgs(fCtx *ForgeContext, orgs []organization) error {
	org, err := promptForOrganization(*fCtx, orgs, true)
	if err != nil {
		return eris.Wrap(err, "Flow failed to prompt for organization in multiple-orgs case")
	}
	flow.updateOrganization(fCtx, &org)
	return nil
}

////////////////////////////////
// Need Existing Organization //
////////////////////////////////

func (flow *initFlow) handleNeedExistingOrgData(fCtx *ForgeContext) error {
	// No org selected, get list of organizations
	orgs, err := getListOfOrganizations(*fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organizations")
	}

	switch len(orgs) {
	case 0: // No organizations found
		return flow.handleNeedExistingOrganizationCaseNoOrgs()
	case 1: // One organization found
		return flow.handleNeedExistingOrganizationCaseOneOrg(fCtx, orgs)
	default: // Multiple organizations found
		return flow.handleNeedExistingOrganizationCaseMultipleOrgs(fCtx, orgs)
	}
}

func (flow *initFlow) handleNeedExistingOrganizationCaseNoOrgs() error {
	printNoOrganizations()
	return ErrOrganizationSelectionCanceled
}

func (flow *initFlow) handleNeedExistingOrganizationCaseOneOrg(fCtx *ForgeContext, orgs []organization) error {
	flow.updateOrganization(fCtx, &orgs[0])
	return nil
}

func (flow *initFlow) handleNeedExistingOrganizationCaseMultipleOrgs(fCtx *ForgeContext, orgs []organization) error {
	// First check if we already have a selected organization
	selectedOrg, err := getSelectedOrganization(*fCtx)
	if err == nil && selectedOrg.ID != "" {
		flow.updateOrganization(fCtx, &selectedOrg)
		return nil
	}

	org, err := promptForOrganization(*fCtx, orgs, false)
	if err != nil {
		return eris.Wrap(err, "Flow failed to prompt for organization in existing multiple-orgs case")
	}
	flow.updateOrganization(fCtx, &org)
	return nil
}

////////////////////////////////
// Helper Functions           //
////////////////////////////////

// updateOrganization updates the organization in the flow state and saves the config.
func (flow *initFlow) updateOrganization(fCtx *ForgeContext, org *organization) {
	fCtx.State.Organization = org
	flow.organizationStepDone = true
	fCtx.Config.OrganizationID = org.ID

	err := fCtx.Config.Save()
	if err != nil {
		printer.Notificationf("Warning: Failed to save config: %s", err)
		logger.Error(eris.Wrap(err, "Organization flow failed to save config"))
		// continue on, this is not fatal
	}
}
