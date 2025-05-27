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

func (flow *initFlow) handleNeedOrgData() error {
	orgs, err := getListOfOrganizations(flow.context)
	if err != nil {
		return eris.Wrap(err, "Failed to get organizations")
	}

	switch len(orgs) {
	case 0: // No organizations found
		return flow.handleNeedOrganizationCaseNoOrgs()
	case 1: // One organization found
		return flow.handleNeedOrganizationCaseOneOrg(orgs)
	default: // Multiple organizations found
		return flow.handleNeedOrganizationCaseMultipleOrgs(orgs)
	}
}

func (flow *initFlow) handleNeedOrganizationCaseNoOrgs() error {
	for {
		printer.NewLine(1)
		printer.Infoln("No organizations found.")
		choice := getInput("Would you like to create one? (Y/n)", "Y")

		switch choice {
		case "Y":
			org, err := createOrganization(flow.context, &CreateOrganizationCmd{})
			if err != nil {
				return eris.Wrap(err, "Flow failed to create organization in no-orgs case")
			}
			flow.updateOrganization(org)
			return nil
		case "n":
			return ErrOrganizationCreationCanceled
		default:
			printer.Infoln("Please select capital 'Y' or lowercase 'n'")
			printer.NewLine(1)
		}
	}
}

func (flow *initFlow) handleNeedOrganizationCaseOneOrg(orgs []organization) error {
	printer.NewLine(1)
	printer.Infof("Found one organization: %s [%s]\n", orgs[0].Name, orgs[0].Slug)

	for {
		choice := getInput("Use this organization? (Y/n/c to create new)", "Y")

		switch choice {
		case "Y":
			flow.updateOrganization(&orgs[0])
			return nil
		case "n":
			return ErrOrganizationSelectionCanceled
		case "c":
			org, err := createOrganization(flow.context, &CreateOrganizationCmd{})
			if err != nil {
				return eris.Wrap(err, "Flow failed to create organization in one-org case")
			}
			flow.updateOrganization(org)
			return nil
		default:
			printer.NewLine(1)
			printer.Infoln("Please select capital 'Y' or lowercase 'n'/'c'")
		}
	}
}

func (flow *initFlow) handleNeedOrganizationCaseMultipleOrgs(orgs []organization) error {
	org, err := promptForOrganization(flow.context, orgs, true)
	if err != nil {
		return eris.Wrap(err, "Flow failed to prompt for organization in multiple-orgs case")
	}
	flow.updateOrganization(&org)
	return nil
}

////////////////////////////////
// Need Existing Organization //
////////////////////////////////

func (flow *initFlow) handleNeedExistingOrgData() error {
	// No org selected, get list of organizations
	orgs, err := getListOfOrganizations(flow.context)
	if err != nil {
		return eris.Wrap(err, "Failed to get organizations")
	}

	switch len(orgs) {
	case 0: // No organizations found
		return flow.handleNeedExistingOrganizationCaseNoOrgs()
	case 1: // One organization found
		return flow.handleNeedExistingOrganizationCaseOneOrg(orgs)
	default: // Multiple organizations found
		return flow.handleNeedExistingOrganizationCaseMultipleOrgs(orgs)
	}
}

func (flow *initFlow) handleNeedExistingOrganizationCaseNoOrgs() error {
	printNoOrganizations()
	return ErrOrganizationSelectionCanceled
}

func (flow *initFlow) handleNeedExistingOrganizationCaseOneOrg(orgs []organization) error {
	printer.NewLine(1)
	printer.Headerln("  Organization Information  ")
	printer.Infof("  %s (%s)\n", orgs[0].Name, orgs[0].Slug)
	flow.updateOrganization(&orgs[0])
	return nil
}

func (flow *initFlow) handleNeedExistingOrganizationCaseMultipleOrgs(orgs []organization) error {
	// First check if we already have a selected organization
	selectedOrg, err := getSelectedOrganization(flow.context)
	if err == nil && selectedOrg.ID != "" {
		// Show the org and project lists
		if err := showOrganizationList(flow.context); err != nil {
			// If we fail to show the org list, just use the selected org
			printer.NewLine(1)
			printer.Headerln("  Organization Information  ")
			printer.Infof("  Organization: %s (%s)\n", selectedOrg.Name, selectedOrg.Slug)
		}
		flow.updateOrganization(&selectedOrg)
		return nil
	}

	org, err := promptForOrganization(flow.context, orgs, false)
	if err != nil {
		return eris.Wrap(err, "Flow failed to prompt for organization in existing multiple-orgs case")
	}
	flow.updateOrganization(&org)
	return nil
}

////////////////////////////////
// Helper Functions           //
////////////////////////////////

// updateOrganization updates the organization in the flow state and saves the config.
func (flow *initFlow) updateOrganization(org *organization) {
	flow.State.Organization = org
	flow.organizationStepDone = true
	flow.config.OrganizationID = org.ID

	err := SaveForgeConfig(flow.config)
	if err != nil {
		printer.Notificationf("Warning: Failed to save config: %s", err)
		logger.Error(eris.Wrap(err, "Organization flow failed to save config"))
		// continue on, this is not fatal
	}
}
