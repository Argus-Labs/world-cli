package forge

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/globalconfig"
)

// Deploy a project
func deploy(ctx context.Context) error {
	globalConfig, err := globalconfig.GetGlobalConfig()
	if err != nil {
		return eris.Wrap(err, "Failed to get global config")
	}

	projectID := globalConfig.ProjectID
	organizationID := globalConfig.OrganizationID

	if organizationID == "" {
		printNoSelectedOrganization()
		return nil
	}

	if projectID == "" {
		printNoSelectedProject()
		return nil
	}

	// Get organization details
	org, err := getSelectedOrganization(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization details")
	}

	// Get project details
	prj, err := getSelectedProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get project details")
	}

	fmt.Println("Deployment Details")
	fmt.Println("-----------------")
	fmt.Printf("Organization: %s\n", org.Name)
	fmt.Printf("Org Slug:     %s\n", org.Slug)
	fmt.Printf("Project:      %s\n", prj.Name)
	fmt.Printf("Project Slug: %s\n", prj.Slug)
	fmt.Printf("Repository:   %s\n\n", prj.RepoURL)

	deployURL := fmt.Sprintf("%s/api/organization/%s/project/%s/deploy", baseURL, organizationID, projectID)
	_, err = sendRequest(ctx, http.MethodPost, deployURL, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to deploy project")
	}

	fmt.Println("\n✨ Your deployment is being processed! ✨")
	fmt.Println("\nTo check the status of your deployment, run:")
	fmt.Println("  $ 'world forge deployment status'")

	return nil
}

// Destroy a project
func destroy(ctx context.Context) error {
	globalConfig, err := globalconfig.GetGlobalConfig()
	if err != nil {
		return eris.Wrap(err, "Failed to get global config")
	}

	projectID := globalConfig.ProjectID
	organizationID := globalConfig.OrganizationID

	if organizationID == "" {
		printNoSelectedOrganization()
		return nil
	}

	if projectID == "" {
		printNoSelectedProject()
		return nil
	}

	// Get organization details
	org, err := getSelectedOrganization(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization details")
	}

	// Get project details
	prj, err := getSelectedProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get project details")
	}

	fmt.Println("Project Details")
	fmt.Println("-----------------")
	fmt.Printf("Organization: %s\n", org.Name)
	fmt.Printf("Org Slug:     %s\n", org.Slug)
	fmt.Printf("Project:      %s\n", prj.Name)
	fmt.Printf("Project Slug: %s\n", prj.Slug)
	fmt.Printf("Repository:   %s\n\n", prj.RepoURL)

	fmt.Print("Are you sure you want to destroy this project? (y/N): ")
	response, err := getInput()
	if err != nil {
		return eris.Wrap(err, "Failed to read response")
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" {
		fmt.Println("Destroy cancelled")
		return nil
	}

	destroyURL := fmt.Sprintf("%s/api/organization/%s/project/%s/destroy", baseURL, organizationID, projectID)
	_, err = sendRequest(ctx, http.MethodPost, destroyURL, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to destroy project")
	}

	fmt.Println("\n🗑️  Your destroy request is being processed!")
	fmt.Println("\nTo check the status of your destroy request, run:")
	fmt.Println("  $ 'world forge deployment status'")

	return nil
}
