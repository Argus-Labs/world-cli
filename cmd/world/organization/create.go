package organization

import (
	"context"
	"strings"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/utils"
	"pkg.world.dev/world-cli/common/printer"
)

const MaxOrgNameLen = 50

//nolint:gocognit,funlen // Belongs in a single function
func (h *Handler) Create(ctx context.Context, flags models.CreateOrganizationFlags) (models.Organization, error) {
	var orgName, orgSlug string
	var err error

	for {
		// Get organization name
		printer.NewLine(1)
		printer.Headerln("  Create New Organization  ")
		for {
			orgName, err = h.inputService.Prompt(ctx, "Enter organization name", flags.Name)
			if err != nil {
				return models.Organization{}, eris.Wrap(err, "Failed to get organization name")
			}
			err = utils.ValidateName(orgName, MaxOrgNameLen)
			if err != nil {
				printer.Errorf("%s\n", err)
				printer.NewLine(1)
				continue
			}
			break
		}

		// Used to create slug from name
		orgSlug = orgName
		if flags.Slug != "" {
			orgSlug = flags.Slug
		}

		// Get and validate organization slug
		for {
			minLength := 3
			maxLength := 15
			orgSlug = utils.CreateSlugFromName(orgSlug, minLength, maxLength)
			orgSlug, err = h.inputService.Prompt(ctx, "Enter organization slug", orgSlug)
			if err != nil {
				return models.Organization{}, eris.Wrap(err, "Failed to get organization slug")
			}

			// Validate slug
			orgSlug, err = utils.SlugToSaneCheck(orgSlug, minLength, maxLength)
			if err != nil {
				printer.Errorf("%s\n", err)
				printer.NewLine(1)
				continue
			}
			break
		}

		// Show confirmation
		printer.NewLine(1)
		printer.Headerln("  Organization Details  ")
		printer.Infof("Name: %s\n", orgName)
		printer.Infof("Slug: %s\n", orgSlug)

		// Get confirmation
		printer.NewLine(1)
		confirm, err := h.inputService.Confirm(ctx, "Create organization with these details? (Y/n)", "n")
		if err != nil {
			return models.Organization{}, eris.Wrap(err, "Failed to get confirmation")
		}
		if confirm {
			org, err := h.apiClient.CreateOrganization(ctx, orgName, orgSlug)
			if err != nil {
				if strings.Contains(err.Error(), api.ErrOrganizationSlugAlreadyExists.Error()) {
					printer.Errorf(
						"An Organization already exists with slug: %s, please choose a different slug.\n",
						orgSlug,
					)
					printer.NewLine(1)
				}
				return models.Organization{}, eris.Wrap(err, "Failed to create organization")
			}
			printer.NewLine(1)
			printer.Successf("Organization '%s' with slug '%s' created successfully!\n", org.Name, org.Slug)

			err = h.saveOrganization(org)
			if err != nil {
				return models.Organization{}, eris.Wrap(err, "Failed to save organization")
			}
			return org, nil
		}
	}
}
