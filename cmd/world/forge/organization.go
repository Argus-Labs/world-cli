package forge

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

	fmt.Println("\n🏢 ✨ Organization Information ✨")
	fmt.Println("=================================")
	if organization.Name == "" {
		fmt.Println("\n❌ No organization selected")
		fmt.Println("\nℹ️  Use 'world forge org select' to choose an organization")
	} else {
		fmt.Println("\n📋 Available Organizations:")
		fmt.Println("---------------------------")
		for _, org := range organizations {
			if org.ID == organization.ID {
				fmt.Printf("🌟 %s (%s) [SELECTED]\n", org.Name, org.Slug)
			} else {
				fmt.Printf("📎 %s (%s)\n", org.Name, org.Slug)
			}
		}
	}
	return nil
}

func getSelectedOrganization(ctx context.Context) (organization, error) {
	// Get config
	config, err := globalconfig.GetGlobalConfig()
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
	fmt.Println("\n🏢 Available Organizations")
	fmt.Println("========================")
	fmt.Println("\n📋 Organization List:")
	fmt.Println("------------------")
	for i, org := range orgs {
		fmt.Printf("  %d. 🏛️  %s\n     └─ 🔖 Slug: %s\n", i+1, org.Name, org.Slug)
	}

	// Get user input
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			return organization{}, ctx.Err()
		default:
			fmt.Print("\n✨ Enter organization number (or 'q' to quit): ")
			input, err := getInput()
			if err != nil {
				return organization{}, eris.Wrap(err, "Failed to read input")
			}

			input = strings.TrimSpace(input)
			if input == "q" {
				fmt.Println("\n❌ Organization selection canceled")
				return organization{}, eris.New("Organization selection canceled")
			}

			// Parse selection
			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num > len(orgs) {
				fmt.Printf("\n❌ Invalid selection. Please enter a number between 1 and %d (attempt %d/%d)\n",
					len(orgs), attempts+1, maxAttempts)
				attempts++
				continue
			}

			selectedOrg := orgs[num-1]

			// Save organization to config file
			config, err := globalconfig.GetGlobalConfig()
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
	config, err := globalconfig.GetGlobalConfig()
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
	fmt.Print("\n👥 Enter organization name: ")
	orgName, err := getInput()
	if err != nil {
		return nil, eris.Wrap(err, "Failed to read organization name")
	}

	// Get and validate organization slug
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		fmt.Print("\n🔖 Enter organization slug (3-15 characters, " +
			"lowercase letters, numbers, and underscores allowed): ")
		orgSlug, err = getInput()
		if err != nil {
			return nil, eris.Wrap(err, "Failed to read organization slug")
		}

		// Validate slug
		minLength := 3
		maxLength := 15
		err = slugCheck(orgSlug, minLength, maxLength)
		if err != nil {
			fmt.Printf("\n❌ Error: %s (attempt %d/%d)\n", err, attempts+1, maxAttempts)
			attempts++
			continue
		}

		break
	}

	if attempts >= maxAttempts {
		return nil, eris.New("Maximum attempts reached for entering organization slug")
	}

	// Get and validate organization avatar URL
	attempts = 0
	maxAttempts = 5
	for attempts < maxAttempts {
		fmt.Print("\n🖼️  Enter organization avatar URL: ")
		orgAvatarURL, err = getInput()
		if err != nil {
			return nil, eris.Wrap(err, "Failed to read organization avatar URL")
		}

		if orgAvatarURL == "" {
			fmt.Println("\n❌ Organization avatar URL cannot be empty")
			attempts++
			continue
		}

		if !isValidURL(orgAvatarURL) {
			fmt.Printf("\n❌ Error: Invalid URL (attempt %d/%d)\n", attempts+1, maxAttempts)
			attempts++
			continue
		}

		break
	}

	if attempts >= maxAttempts {
		return nil, eris.New("Maximum attempts reached for entering organization avatar URL")
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
	config, err := globalconfig.GetGlobalConfig()
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get config")
	}
	config.OrganizationID = org.ID
	err = globalconfig.SaveGlobalConfig(config)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to select organization")
	}

	fmt.Printf("\n✨ Organization '%s' created successfully! ✨\n", orgName)
	fmt.Printf("🔖 Slug: %s\n", orgSlug)
	// fmt.Printf("🆔 ID: %s\n", org.ID)
	return org, nil
}

func inviteUserToOrganization(ctx context.Context) error { //nolint:dupl // TODO: refactor
	fmt.Println("\n👥 ✨ Invite User to Organization ✨")
	fmt.Println("=================================")
	fmt.Print("\n🆔 Enter user ID to invite: ")
	userID, err := getInput()
	if err != nil {
		return eris.Wrap(err, "Failed to read user ID")
	}

	if userID == "" {
		return eris.New("User ID cannot be empty")
	}

	role, err := getRoleInput(false)
	if err != nil {
		return eris.Wrap(err, "Failed to read role input")
	}

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

	fmt.Printf("\n✨ Successfully invited user %s to organization! ✨\n", userID)
	fmt.Printf("🎭 Assigned role: %s\n", role)
	return nil
}

func updateUserRoleInOrganization(ctx context.Context) error { //nolint:dupl // TODO: refactor
	fmt.Println("\n👥 ✨ Update User Role in Organization ✨")
	fmt.Println("=====================================")
	fmt.Print("\n🆔 Enter user ID to update: ")
	userID, err := getInput()
	if err != nil {
		return eris.Wrap(err, "Failed to read user ID")
	}

	if userID == "" {
		return eris.New("User ID cannot be empty")
	}

	role, err := getRoleInput(true)
	if err != nil {
		return eris.Wrap(err, "Failed to read role input")
	}

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

	fmt.Printf("\n✨ Successfully updated role for user %s! ✨\n", userID)
	fmt.Printf("🎭 New role: %s\n", role)
	return nil
}

func getRoleInput(allowNone bool) (string, error) {
	// Get and validate role
	attempts := 0
	maxAttempts := 5
	var opts string
	if allowNone {
		opts = "owner, admin, member, or none"
	} else {
		opts = "owner, admin, or member"
	}
	for attempts < maxAttempts {
		fmt.Println("\n🎭 Role Assignment")
		fmt.Println("----------------")
		fmt.Printf("Available Roles: %s\n", opts)
		fmt.Print("\n👉 Enter organization role [Enter for member]: ")
		role, err := getInput()
		if err != nil {
			return "", eris.Wrap(err, "Failed to read organization role")
		}
		attempts++
		// default to member
		if role == "" {
			fmt.Println("\n✨ Using default role of member")
			role = "member"
		}
		if allowNone && role == "none" {
			fmt.Print("\n⚠️  Warning: Role \"none\" removes user from this organization")
			fmt.Print("\n❓ Confirm removal? (Yes/no): ")
			answer, err := getInput()
			if err != nil {
				return "", eris.Wrap(err, "Failed to read remove confirmation")
			}
			if answer != "Yes" {
				fmt.Println("\n❌ User not removed")
				continue // let them try again
			}
			return role, nil
		}
		if role == "admin" || role == "owner" || role == "member" {
			return role, nil
		}
		fmt.Printf("\n❌ Error: Role must be one of %s (attempt %d/%d)\n",
			opts, attempts, maxAttempts)
	}
	return "", eris.New("Maximum attempts reached for entering role")
}

// handleOrganizationSelection manages the organization selection logic
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

// handleMultipleOrgs handles the case when there are multiple organizations
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

// handleNoOrgs handles the case when there are no organizations
func handleNoOrgs(ctx context.Context) (string, error) {
	// Confirmation prompt
	fmt.Printf("❓ You don't have any organizations. Do you want to create a new organization now? (Y/n): ")
	confirmation, err := getInput()
	if err != nil {
		return "", eris.Wrap(err, "Failed to read confirmation")
	}

	if confirmation != "Y" {
		if confirmation == "y" {
			fmt.Println("You need to put Y (uppercase) to confirm creation")
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
