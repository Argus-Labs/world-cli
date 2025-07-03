package organization

import (
	"context"
	"strconv"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/common/printer"
)

var (
	ErrOrganizationSelectionCanceled = eris.New("Organization selection canceled")
	ErrCannotSwitchOrganization      = eris.New("Cannot switch organization, directory belongs to another project.")
	ErrOrganizationNotFoundWithSlug  = eris.New("Organization not found with slug: ")
)

func (h *Handler) Switch(ctx context.Context, flags models.SwitchOrganizationFlags,
) (models.Organization, error) {
	if h.configService.GetConfig().CurrRepoKnown {
		printer.Errorf("Cannot switch organization, current git working directory belongs to project: %s.",
			h.configService.GetConfig().CurrProjectName)
		return models.Organization{}, ErrCannotSwitchOrganization
	}

	// If slug is provided, select organization from slug
	if flags.Slug != "" {
		org, err := h.switchOrganizationFromSlug(ctx, flags.Slug)
		if err != nil {
			return models.Organization{}, eris.Wrap(err, "Failed command switch organization from slug")
		}
		return org, nil
	}

	orgs, err := h.apiClient.GetOrganizations(ctx)
	if err != nil {
		return models.Organization{}, eris.Wrap(err, "Failed to get organizations")
	}

	if len(orgs) == 0 {
		h.PrintNoOrganizations()
		return models.Organization{}, nil
	}

	selectedOrg, err := h.PromptForSwitch(ctx, orgs, false)
	if err != nil {
		return models.Organization{}, err
	}

	err = h.projectHandler.HandleSwitch(ctx)
	if err != nil {
		return models.Organization{}, err
	}

	return selectedOrg, nil
}

func (h *Handler) switchOrganizationFromSlug(ctx context.Context, slug string) (models.Organization, error) {
	orgs, err := h.apiClient.GetOrganizations(ctx)
	if err != nil {
		return models.Organization{}, eris.Wrap(err, "Failed to get organizations")
	}

	for _, org := range orgs {
		if org.Slug == slug {
			err = h.saveOrganization(org)
			if err != nil {
				return models.Organization{}, eris.Wrap(err, "Failed to save organization")
			}

			err = h.showOrganizationList(ctx)
			if err != nil {
				return models.Organization{}, err
			}

			err = h.projectHandler.HandleSwitch(ctx)
			if err != nil {
				return models.Organization{}, err
			}
			return org, nil
		}
	}

	printer.NewLine(1)
	printer.Errorln("Organization not found with slug: " + slug)
	return models.Organization{}, eris.Wrap(ErrOrganizationNotFoundWithSlug, slug)
}

func (h *Handler) showOrganizationList(ctx context.Context) error {
	selectedOrg, err := h.apiClient.GetOrganizationByID(ctx, h.configService.GetConfig().OrganizationID)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization")
	}

	organizations, err := h.apiClient.GetOrganizations(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization list")
	}

	printer.NewLine(1)
	printer.Headerln("  Organization Information  ")
	if selectedOrg.Name == "" {
		printer.Errorln("No organization selected")
		printer.NewLine(1)
		printer.Infoln("Use 'world organization switch' to choose an organization")
	} else {
		for _, org := range organizations {
			if org.ID == selectedOrg.ID {
				printer.Infof("• %s (%s) [SELECTED]\n", org.Name, org.Slug)
			} else {
				printer.Infof("  %s (%s)\n", org.Name, org.Slug)
			}
		}
	}
	return nil
}

//nolint:gocognit // Belongs in a single function
func (h *Handler) PromptForSwitch(
	ctx context.Context, orgs []models.Organization, enableCreation bool,
) (models.Organization, error) {
	// Display organizations as a numbered list
	printer.NewLine(1)
	printer.Headerln("   Available Organizations  ")
	for i, org := range orgs {
		printer.Infof("  %d. %s\n    └─ Slug: %s\n", i+1, org.Name, org.Slug)
	}

	// Get user input
	var input string
	var err error
	for {
		select {
		case <-ctx.Done():
			return models.Organization{}, ctx.Err()
		default:
			printer.NewLine(1)
			if enableCreation {
				input, err = h.inputService.Prompt(
					ctx,
					"Enter organization number ('c' to create new or 'q' to quit)",
					"",
				)
				if err != nil {
					return models.Organization{}, eris.Wrap(err, "Failed to get organization number")
				}
			} else {
				input, err = h.inputService.Prompt(ctx, "Enter organization number ('q' to quit)", "")
				if err != nil {
					return models.Organization{}, eris.Wrap(err, "Failed to get organization number")
				}
			}

			if input == "q" {
				return models.Organization{}, ErrOrganizationSelectionCanceled
			}

			if input == "c" && enableCreation {
				org, err := h.Create(ctx, models.CreateOrganizationFlags{})
				if err != nil {
					return models.Organization{}, eris.Wrap(err, "Failed to create organization")
				}
				return org, nil
			}

			// Parse selection
			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num > len(orgs) {
				printer.Errorf("Invalid selection. Please enter a number between 1 and %d\n", len(orgs))
				continue
			}

			selectedOrg := orgs[num-1]

			err = h.saveOrganization(selectedOrg)
			if err != nil {
				return models.Organization{}, eris.Wrap(err, "Failed to save organization")
			}

			printer.Successf("Selected organization: %s\n", selectedOrg.Name)
			return selectedOrg, nil
		}
	}
}
