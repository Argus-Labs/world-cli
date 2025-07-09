package organization

import (
	"context"
	"strconv"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
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

	orgs, err := h.apiClient.GetOrganizations(ctx)
	if err != nil {
		return models.Organization{}, eris.Wrap(err, "Failed to get organizations")
	}

	if len(orgs) == 0 {
		h.PrintNoOrganizations()
		return models.Organization{}, nil
	}

	// If slug is provided, select organization from slug
	if flags.Slug != "" {
		org, err := h.switchOrganizationFromSlug(ctx, flags.Slug, orgs)
		if err != nil {
			return models.Organization{}, eris.Wrap(err, "Failed command switch organization from slug")
		}
		return org, nil
	}

	selectedOrg, err := h.PromptForSwitch(ctx, orgs, false)
	if err != nil {
		return models.Organization{}, err
	}

	err = h.projectHandler.HandleSwitch(ctx, selectedOrg)
	if err != nil {
		return models.Organization{}, err
	}

	return selectedOrg, nil
}

func (h *Handler) switchOrganizationFromSlug(
	ctx context.Context,
	slug string,
	orgs []models.Organization,
) (models.Organization, error) {
	for _, org := range orgs {
		if org.Slug == slug {
			err := h.saveOrganization(org)
			if err != nil {
				return models.Organization{}, eris.Wrap(err, "Failed to save organization")
			}

			h.showOrganizationList(org, orgs)

			err = h.projectHandler.HandleSwitch(ctx, org)
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
	var input, prompt string
	var err error
	for {
		printer.NewLine(1)
		if enableCreation {
			prompt = "Enter organization number ('c' to create new or 'q' to quit)"
		} else {
			prompt = "Enter organization number ('q' to quit)"
		}

		input, err = h.inputService.Prompt(ctx, prompt, "")
		if err != nil {
			return models.Organization{}, eris.Wrap(err, "Failed to get organization number")
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
