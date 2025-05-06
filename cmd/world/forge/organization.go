package forge

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/printer"
)

type organization struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	CreatedTime      string `json:"created_time"`
	UpdatedTime      string `json:"updated_time"`
	OwnerID          string `json:"owner_id"`
	Deleted          bool   `json:"deleted"`
	DeletedTime      string `json:"deleted_time"`
	BaseShardAddress string `json:"base_shard_address"`
	AvatarURL        string `json:"avatar_url"`
}

type createOrgRequest struct {
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	AvatarURL string `json:"avatar_url"`
}

func showOrganizationList(ctx context.Context) error {
	organization, err := getSelectedOrganization(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization")
	}

	organizations, err := getListOfOrganizations(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization list")
	}

	printer.NewLine(1)
	printer.Headerln("  Organization Information  ")
	if organization.Name == "" {
		printer.NewLine(1)
		printer.Errorln("No organization selected")
		printer.NewLine(1)
		printer.Infoln("Use 'world forge organization switch' to choose an organization")
	} else {
		printer.NewLine(1)
		printer.Infoln(" Available Organizations: ")
		printer.SectionDivider("-", 26)
		for _, org := range organizations {
			if org.ID == organization.ID {
				printer.Infof("• %s (%s) [SELECTED]\n", org.Name, org.Slug)
			} else {
				printer.Infof("  %s (%s)\n", org.Name, org.Slug)
			}
		}
	}
	return nil
}

func getSelectedOrganization(ctx context.Context) (organization, error) {
	// Get config
	config, err := GetCurrentForgeConfig()
	if err != nil {
		return organization{}, eris.Wrap(err, "Failed to get config")
	}

	if config.OrganizationID == "" {
		return organization{}, nil
	}

	// send request
	body, err := sendRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/%s", organizationURL, config.OrganizationID), nil)
	if err != nil {
		return organization{}, eris.Wrap(err, "Failed to get organization")
	}

	// parse response
	org, err := parseResponse[organization](body)
	if err != nil {
		return organization{}, eris.Wrap(err, "Failed to parse response")
	}

	return *org, nil
}

func getListOfOrganizations(ctx context.Context) ([]organization, error) {
	// Send request
	body, err := sendRequest(ctx, http.MethodGet, organizationURL, nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get organizations")
	}

	// Parse response
	orgs, err := parseResponse[[]organization](body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse response")
	}

	return *orgs, nil
}

func selectOrganization(ctx context.Context) (organization, error) {
	orgs, err := getListOfOrganizations(ctx)
	if err != nil {
		return organization{}, eris.Wrap(err, "Failed to get organizations")
	}

	if len(orgs) == 0 {
		printNoOrganizations()
		return organization{}, nil
	}

	selectedOrg, err := promptForOrganization(ctx, orgs)
	if err != nil {
		return organization{}, err
	}

	err = handleProjectConfig(ctx)
	if err != nil {
		return organization{}, err
	}

	return selectedOrg, nil
}

func promptForOrganization(ctx context.Context, orgs []organization) (organization, error) {
	// Display organizations as a numbered list
	printer.NewLine(1)
	printer.Headerln("   Available Organizations  ")
	for i, org := range orgs {
		printer.Infof("  %d. %s\n    └─ Slug: %s\n", i+1, org.Name, org.Slug)
	}

	// Get user input
	for {
		select {
		case <-ctx.Done():
			return organization{}, ctx.Err()
		default:
			printer.NewLine(1)
			input := getInput("Enter organization number (or 'q' to quit)", "")

			if input == "q" {
				printer.NewLine(1)
				printer.Errorln("Organization selection canceled")
				return organization{}, eris.New("Organization selection canceled")
			}

			// Parse selection
			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num > len(orgs) {
				printer.NewLine(1)
				printer.Errorf("Invalid selection. Please enter a number between 1 and %d\n", len(orgs))
				continue
			}

			selectedOrg := orgs[num-1]

			// Save organization to config file
			config, err := GetCurrentForgeConfig()
			if err != nil {
				return organization{}, eris.Wrap(err, "Failed to get config")
			}
			config.OrganizationID = selectedOrg.ID
			err = SaveForgeConfig(config)
			if err != nil {
				return organization{}, eris.Wrap(err, "Failed to save organization")
			}

			printer.NewLine(1)
			printer.Successf("Selected organization: %s\n", selectedOrg.Name)
			return selectedOrg, nil
		}
	}
}

func handleProjectConfig(ctx context.Context) error {
	// Get projectID from config
	config, err := GetCurrentForgeConfig()
	if err != nil {
		return eris.Wrap(err, "Failed to get config")
	}
	projectID := config.ProjectID

	// Handle project selection
	projectID, err = handleProjectSelection(ctx, projectID)
	if err != nil {
		return eris.Wrap(err, "Failed to select project")
	}

	// Save projectID to config
	config.ProjectID = projectID
	err = SaveForgeConfig(config)
	if err != nil {
		return eris.Wrap(err, "Failed to save project")
	}

	// Show project list
	return showProjectList(ctx)
}

//nolint:gocognit,funlen // Makes sense to keep in one function.
func createOrganization(ctx context.Context) (*organization, error) {
	var orgName, orgSlug, orgAvatarURL string

	for {
		// Get organization name
		printer.NewLine(1)
		printer.Headerln("  Create New Organization  ")
		for {
			orgName = getInput("Enter organization name", "")
			if orgName == "" {
				printer.NewLine(1)
				printer.Errorln("Organization name is required")
				continue
			}
			break
		}

		// Get and validate organization slug
		for {
			minLength := 3
			maxLength := 15
			orgSlug = CreateSlugFromName(orgName, minLength, maxLength)
			orgSlug = getInput("Enter organization slug", orgSlug)

			// Validate slug
			var err error
			orgSlug, err = slugToSaneCheck(orgSlug, minLength, maxLength)
			if err != nil {
				printer.NewLine(1)
				printer.Errorf("Error: %s\n", err)
				continue
			}
			break
		}

		// Get and validate organization avatar URL
		for {
			orgAvatarURL = getInput("Enter organization avatar URL [none]", "")

			if orgAvatarURL == "" {
				printer.NewLine(1)
				printer.Infoln("Skipped. No avatar URL will be used.")
				break
			}

			if !isValidURL(orgAvatarURL) {
				printer.NewLine(1)
				printer.Errorln("Invalid URL, leave empty to skip")
				continue
			}

			break
		}

		// Show confirmation
		printer.NewLine(1)
		printer.Headerln("  Organization Details  ")
		printer.Infof("Name: %s\n", orgName)
		printer.Infof("Slug: %s\n", orgSlug)
		if orgAvatarURL != "" {
			printer.Infof("Avatar URL: %s\n", orgAvatarURL)
		} else {
			printer.Infoln("Avatar URL: None")
		}

		// Get confirmation
		for redo := true; redo; {
			printer.NewLine(1)
			confirm := getInput("Create organization with these details? (Y/n)", "n")
			switch confirm {
			case "Y":
				return createOrgRequestAndSave(ctx, orgName, orgSlug, orgAvatarURL)
			case "n":
				redo = false
			default:
				printer.NewLine(1)
				printer.Errorln("Please enter capital 'Y' to confirm, 'n' to cancel, or 'redo' to start over")
			}
		}
	}
}

func createOrgRequestAndSave(ctx context.Context, name, slug, avatarURL string) (*organization, error) {
	// Send request
	body, err := sendRequest(ctx, http.MethodPost, organizationURL, createOrgRequest{
		Name:      name,
		Slug:      slug,
		AvatarURL: avatarURL,
	})
	if err != nil {
		return nil, eris.Wrap(err, "Failed to create organization")
	}

	// Parse response
	org, err := parseResponse[organization](body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse response")
	}

	// Select organization to config file
	config, err := GetCurrentForgeConfig()
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get config")
	}
	config.OrganizationID = org.ID
	err = SaveForgeConfig(config)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to save organization in config")
	}

	printer.NewLine(1)
	printer.Successf("Organization '%s' with slug '%s' created successfully!\n", name, slug)
	return org, nil
}

func inviteUserToOrganization(ctx context.Context) error { //nolint:dupl // TODO: refactor
	printer.NewLine(1)
	printer.Headerln("   Invite User to Organization   ")
	userID := getInput("Enter user ID to invite", "")
	if userID == "" {
		return eris.New("User ID cannot be empty")
	}

	role := getRoleInput(false)

	payload := map[string]string{
		"invited_user_id": userID,
		"role":            role,
	}

	org, err := getSelectedOrganization(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization")
	}

	if org.ID == "" {
		printNoSelectedOrganization()
		return nil
	}

	// Send request
	_, err = sendRequest(ctx, http.MethodPost, fmt.Sprintf("%s/%s/invite", organizationURL, org.ID), payload)
	if err != nil {
		return eris.Wrap(err, "Failed to invite user to organization")
	}

	printer.NewLine(1)
	printer.Successf("Successfully invited user %s to organization!\n", userID)
	printer.Infof("Assigned role: %s\n", role)
	return nil
}

func updateUserRoleInOrganization(ctx context.Context) error { //nolint:dupl // TODO: refactor
	printer.NewLine(1)
	printer.Headerln("  Update User Role in Organization  ")
	userID := getInput("Enter user ID to update", "")

	if userID == "" {
		return eris.New("User ID cannot be empty")
	}

	role := getRoleInput(true)

	payload := map[string]string{
		"target_user_id": userID,
		"role":           role,
	}

	org, err := getSelectedOrganization(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization")
	}

	if org.ID == "" {
		printNoSelectedOrganization()
		return nil
	}

	// Send request
	_, err = sendRequest(ctx, http.MethodPost, fmt.Sprintf("%s/%s/role", organizationURL, org.ID), payload)
	if err != nil {
		return eris.Wrap(err, "Failed to set user role in organization")
	}

	printer.NewLine(1)
	printer.Successf("Successfully updated role for user %s!\n", userID)
	printer.Infof("New role: %s\n", role)
	return nil
}

func getRoleInput(allowNone bool) string {
	// Get and validate role
	var opts string
	if allowNone {
		opts = "owner, admin, member, or none"
	} else {
		opts = "owner, admin, or member"
	}
	for {
		printer.NewLine(1)
		printer.Headerln(" Role Assignment  ")
		printer.Infof("Available Roles: %s\n", opts)
		role := getInput("Enter organization role", "member")
		if allowNone && role == "none" {
			printer.NewLine(1)
			printer.Infoln("Warning: Role \"none\" removes user from this organization")
			answer := getInput("Confirm removal? (Yes/no)", "no")
			if answer != "Yes" {
				printer.NewLine(1)
				printer.Errorln("User not removed")
				continue // let them try again
			}
			return role
		}
		if role == "admin" || role == "owner" || role == "member" {
			return role
		}
		printer.NewLine(1)
		printer.Errorf("Error: Role must be one of %s\n", opts)
	}
}

// handleOrganizationSelection manages the organization selection logic.
func handleOrganizationSelection(ctx context.Context, orgID string) (string, error) {
	orgs, err := getListOfOrganizations(ctx)
	if err != nil {
		return "", eris.Wrap(err, "Failed to get orgs")
	}

	switch numOrgs := len(orgs); {
	case numOrgs == 1:
		return orgs[0].ID, nil
	case numOrgs > 1:
		return handleMultipleOrgs(ctx, orgID, orgs)
	default:
		return handleNoOrgs(ctx)
	}
}

// handleMultipleOrgs handles the case when there are multiple organizations.
func handleMultipleOrgs(ctx context.Context, orgID string, orgs []organization) (string, error) {
	for _, org := range orgs {
		if org.ID == orgID {
			return orgID, nil
		}
	}

	org, err := selectOrganization(ctx)
	if err != nil {
		return "", eris.Wrap(err, "Failed to select organization")
	}
	return org.ID, nil
}

// handleNoOrgs handles the case when there are no organizations.
func handleNoOrgs(ctx context.Context) (string, error) {
	for redo := true; redo; {
		// Confirmation prompt
		printer.NewLine(1)
		confirmation := getInput("You don't have any organizations. Create a new one now? (Y/n)", "n")

		switch confirmation {
		case "Y":
			redo = false
		case "y":
			printer.NewLine(1)
			printer.Errorln("You need to enter Y (uppercase) to confirm creation")
		case "n":
			printer.NewLine(1)
			printer.Errorln("Organization creation canceled")
			return "", nil
		default:
			printer.NewLine(1)
			printer.Errorln("Invalid input")
		}
	}

	org, err := createOrganization(ctx)
	if err != nil {
		return "", eris.Wrap(err, "Failed to create organization")
	}
	return org.ID, nil
}
