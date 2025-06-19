package organization

import (
	"context"
	"strconv"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/common/printer"
)

var ErrOrganizationSelectionCanceled = eris.New("Organization selection canceled")

//nolint:revive // TODO: implement
func (h *Handler) Switch(ctx context.Context, state *models.CommandState, flags models.SwitchOrganizationFlags,
) (models.Organization, error) {
	return models.Organization{}, nil
}

//nolint:gocognit // Belongs in a single function
func (h *Handler) PromptForSwitch(
	ctx context.Context, state *models.CommandState, orgs []models.Organization, createNew bool,
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
			if createNew {
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

			if input == "c" && createNew {
				org, err := h.Create(ctx, state, models.CreateOrganizationFlags{})
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
