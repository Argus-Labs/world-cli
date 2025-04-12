package forge

import (
	"context"
	"fmt"
	"github.com/rotisserie/eris"
	"net/http"
	"strconv"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/globalconfig"
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

	fmt.Println("\n  Organization Information")
	fmt.Println("============================")
	if organization.Name == "" {
		fmt.Println("\nNo organization selected")
		fmt.Println("\nUse 'world forge organization switch' to choose an organization")
	} else {
		fmt.Println("\n Available Organizations:")
		fmt.Println("--------------------------")
		for _, org := range organizations {
			if org.ID == organization.ID {
				fmt.Printf("• %s (%s) [SELECTED]\n", org.Name, org.Slug)
			} else {
				fmt.Printf("  %s (%s)\n", org.Name, org.Slug)
			}
		}
	}
	return nil
}

func getSelectedOrganization(ctx context.Context) (organization, error) {
	// Get config
	config, err := GetCurrentConfigWithContext(ctx)
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
	fmt.Println("\n   Available Organizations")
	fmt.Println("=============================")
	for i, org := range orgs {
		fmt.Printf("  %d. %s\n    └─ Slug: %s\n", i+1, org.Name, org.Slug)
	}

	// Get user input
	for {
		select {
		case <-ctx.Done():
			return organization{}, ctx.Err()
		default:
			input := getInput("\nEnter organization number (or 'q' to quit)", "")

			if input == "q" {
				fmt.Println("\n❌ Organization selection canceled")
				return organization{}, eris.New("Organization selection canceled")
			}

			// Parse selection
			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num > len(orgs) {
				fmt.Printf("\n❌ Invalid selection. Please enter a number between 1 and %d\n", len(orgs))
				continue
			}

			selectedOrg := orgs[num-1]

			// Save organization to config file
			config, err := GetCurrentConfig()
			if err != nil {
				return organization{}, eris.Wrap(err, "Failed to get config")
			}
			config.OrganizationID = selectedOrg.ID
			err = globalconfig.SaveGlobalConfig(config)
			if err != nil {
				return organization{}, eris.Wrap(err, "Failed to save organization")
			}

			fmt.Printf("\n✅ Selected organization: %s\n", selectedOrg.Name)
			return selectedOrg, nil
		}
	}

	return organization{}, eris.New("Maximum attempts reached for selecting organization")
}

func handleProjectConfig(ctx context.Context) error {
	// Get projectID from config
	config, err := GetCurrentConfig()
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
	err = globalconfig.SaveGlobalConfig(config)
	if err != nil {
		return eris.Wrap(err, "Failed to save project")
	}

	// Show project list
	return showProjectList(ctx)
}

func createOrganization(ctx context.Context) (*organization, error) { //nolint:funlen
	var orgName, orgSlug, orgAvatarURL string

	// Get organization name
	fmt.Println("\n🏢 ✨ Create New Organization ✨")
	fmt.Println("==============================")
	for {
		orgName = getInput("\nEnter organization name", "")
		if orgName == "" {
			fmt.Printf("\nOrganization name is required\n")
			continue
		}
		break
	}

	// Get and validate organization slug
	for {
		// TODO: create default slug from name
		orgSlug = getInput("\nEnter organization slug", "")

		// Validate slug
		minLength := 3
		maxLength := 15
		orgSlug, err = slugToSaneCheck(orgSlug, minLength, maxLength)
		if err != nil {
			fmt.Printf("\n❌ Error: %s\n", err)
			continue
		}
		break
	}

	// Get and validate organization avatar URL
	for {
		orgAvatarURL = getInput("\nEnter organization avatar URL [none]", "")

		if orgAvatarURL == "" {
			fmt.Print("\nSkipped. No avatar URL will be used.\n")
			break
		}

		if !isValidURL(orgAvatarURL) {
			fmt.Printf("\n❌ Error: Invalid URL\n")
			continue
		}

		break
	}

	// Send request
	body, err := sendRequest(ctx, http.MethodPost, organizationURL, createOrgRequest{
		Name:      orgName,
		Slug:      orgSlug,
		AvatarURL: orgAvatarURL,
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
	config, err := GetCurrentConfig()
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get config")
	}
	config.OrganizationID = org.ID
	err = globalconfig.SaveGlobalConfig(config)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to save organization in config")
	}

	fmt.Printf("\nOrganization '%s' with slug '%s' created successfully!\n", orgName, orgSlug)
	// fmt.Printf("ID: %s\n", org.ID)
	return org, nil
}

func inviteUserToOrganization(ctx context.Context) error { //nolint:dupl // TODO: refactor
	fmt.Println("\n   Invite User to Organization")
	fmt.Println("=================================")
	userID := getInput("\nEnter user ID to invite", "")
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

	fmt.Printf("\nSuccessfully invited user %s to organization!\n", userID)
	fmt.Printf("Assigned role: %s\n", role)
	return nil
}

func updateUserRoleInOrganization(ctx context.Context) error { //nolint:dupl // TODO: refactor
	fmt.Println("\n  Update User Role in Organization ")
	fmt.Println("=====================================")
	userID := getInput("\nEnter user ID to update", "")

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

	fmt.Printf("\nSuccessfully updated role for user %s!\n", userID)
	fmt.Printf("New role: %s\n", role)
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
		fmt.Println("\n Role Assignment")
		fmt.Println("----------------")
		fmt.Printf("Available Roles: %s\n", opts)
		role := getInput("\nEnter organization role", "member")
		if allowNone && role == "none" {
			fmt.Print("\nWarning: Role \"none\" removes user from this organization")
			answer := getInput("\nConfirm removal? (Yes/no)", "no")
			if answer != "Yes" {
				fmt.Println("\n❌ User not removed")
				continue // let them try again
			}
			return role
		}
		if role == "admin" || role == "owner" || role == "member" {
			return role
		}
		fmt.Printf("\n❌ Error: Role must be one of %s\n", opts)
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
	// Confirmation prompt
	confirmation := getInput(prompt, "n")

	if confirmation != "Y" {
		if confirmation == "y" {
			fmt.Println("You need to enter Y (uppercase) to confirm creation")
			fmt.Println("\n❌ Organization creation canceled")
			return "", nil
		}
	}

	org, err := createOrganization(ctx)
	if err != nil {
		return "", eris.Wrap(err, "Failed to create organization")
	}
	return org.ID, nil
}
