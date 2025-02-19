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
}

type createOrgRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
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

	fmt.Println("\n🏢 Organization Information")
	fmt.Println("-------------------------")
	if organization.Name == "" {
		fmt.Println("No organization selected")
	} else {
		fmt.Println("\nAvailable Organizations:")
		for _, org := range organizations {
			if org.ID == organization.ID {
				fmt.Printf("* %s (%s) [SELECTED]\n", org.Name, org.Slug)
			} else {
				fmt.Printf("  %s (%s)\n", org.Name, org.Slug)
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
		fmt.Println("You don't have any organizations yet.")
		fmt.Println("Use 'world forge organization create' to create one.")
		return organization{}, nil
	}

	// Display organizations as a numbered list
	fmt.Println("\nAvailable organizations:")
	for i, org := range orgs {
		fmt.Printf("%d. %s (%s)\n", i+1, org.Name, org.Slug)
	}

	// Get user input
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		fmt.Print("\nEnter organization number (or 'q' to quit): ")
		input, err := getInput()
		if err != nil {
			return organization{}, eris.Wrap(err, "Failed to read input")
		}

		input = strings.TrimSpace(input)
		if input == "q" {
			return organization{}, eris.New("Organization selection canceled")
		}

		// Parse selection
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(orgs) {
			fmt.Printf("Invalid selection. Please enter a number between 1 and %d\n", len(orgs))
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

		return selectedOrg, nil
	}

	return organization{}, eris.New("Maximum attempts reached for selecting organization")
}

func createOrganization(ctx context.Context) (organization, error) {
	var orgName, orgSlug string

	// Get organization name
	fmt.Print("Enter organization name: ")
	orgName, err := getInput()
	if err != nil {
		return organization{}, eris.Wrap(err, "Failed to read organization name")
	}

	// Get and validate organization slug
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		fmt.Print("Enter organization slug (5 characters, alphanumeric only): ")
		orgSlug, err = getInput()
		if err != nil {
			return organization{}, eris.Wrap(err, "Failed to read organization slug")
		}

		// Validate slug
		if len(orgSlug) != 5 { //nolint:gomnd
			fmt.Printf("Error: Slug must be exactly 5 characters (attempt %d/%d)\n", attempts+1, maxAttempts)
			attempts++
			continue
		}

		if !isAlphanumeric(orgSlug) {
			attempts++
			fmt.Printf("Error: Slug must contain only letters (a-z|A-Z) and numbers (0-9) "+
				"(attempt %d/%d)\n", attempts, maxAttempts)
			continue
		}

		break
	}

	if attempts >= maxAttempts {
		return organization{}, eris.New("Maximum attempts reached for entering organization slug")
	}

	// Send request
	body, err := sendRequest(ctx, http.MethodPost, organizationURL, createOrgRequest{
		Name: orgName,
		Slug: orgSlug,
	})
	if err != nil {
		return organization{}, eris.Wrap(err, "Failed to create organization")
	}

	// Parse response
	org, err := parseResponse[organization](body)
	if err != nil {
		return organization{}, eris.Wrap(err, "Failed to parse response")
	}

	return *org, nil
}

func inviteUserToOrganization(ctx context.Context) error {
	// Input user id
	fmt.Print("Enter user ID: ")
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

	fmt.Println("User invited to organization")
	return nil
}

func updateUserRoleInOrganization(ctx context.Context) error {
	// Input user id
	fmt.Print("Enter user ID: ")
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

	fmt.Println("User role changed in organization")
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
		fmt.Printf("Enter organization role (%s) [Enter for member]: ", opts)
		role, err := getInput()
		if err != nil {
			return "", eris.Wrap(err, "Failed to read organization role")
		}
		attempts++
		// default to member
		if role == "" {
			fmt.Println("Using default role of member.")
			role = "member"
		}
		if allowNone && role == "none" {
			fmt.Print("Role \"none\" removes user from this organization. Confirm removal? (Yes/no): ")
			answer, err := getInput()
			if err != nil {
				return "", eris.Wrap(err, "Failed to read remove confirmation")
			}
			if answer != "Yes" {
				fmt.Println("User not removed.")
				continue // let them try again
			}
			return role, nil
		}
		if role == "admin" || role == "owner" || role == "member" {
			return role, nil
		}
		fmt.Printf("Error: Role must be one of %s. (attempt %d/%d)\n",
			opts, attempts, maxAttempts)
	}
	return "", eris.New("Maximum attempts reached for entering role")
}
