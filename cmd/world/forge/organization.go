package forge

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
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
	selectedOrg, err := getSelectedOrganization(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization")
	}

	orgList, err := getListOfOrganizations(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization list")
	}

	fmt.Println("Your organizations:")
	fmt.Println("------------------")
	for _, org := range orgList {
		if org.ID == selectedOrg.ID {
			fmt.Printf("* %s (%s) [SELECTED]\n", org.Name, org.Slug)
		} else {
			fmt.Printf("  %s (%s)\n", org.Name, org.Slug)
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
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nEnter organization number (or 'q' to quit): ")
		input, err := reader.ReadString('\n')
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
			fmt.Println("Invalid selection. Please enter a number between 1 and", len(orgs))
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
}

func createOrganization(ctx context.Context) (organization, error) {
	var orgName, orgSlug string

	// Get organization name
	fmt.Print("Enter organization name: ")
	if _, err := fmt.Scanln(&orgName); err != nil {
		return organization{}, eris.Wrap(err, "Failed to read organization name")
	}

	// Get and validate organization slug
	for {
		fmt.Print("Enter organization slug (5 characters, alphanumeric lowercase only): ")
		if _, err := fmt.Scanln(&orgSlug); err != nil {
			return organization{}, eris.Wrap(err, "Failed to read organization slug")
		}

		// Validate slug
		if len(orgSlug) != 5 { //nolint:gomnd
			fmt.Println("Error: Slug must be exactly 5 characters")
			continue
		}

		if !isAlphanumeric(orgSlug) {
			fmt.Println("Error: Slug must contain only lowercase letters (a-z) and numbers (0-9)")
			continue
		}

		break
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