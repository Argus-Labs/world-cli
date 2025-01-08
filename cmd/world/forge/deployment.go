package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/globalconfig"
)

var statusFailRegEx = regexp.MustCompile(`[^a-zA-Z0-9\. ]+`)

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

func status(ctx context.Context) error {
	globalConfig, err := globalconfig.GetGlobalConfig()
	if err != nil {
		return eris.Wrap(err, "Failed to get global config")
	}
	projectID := globalConfig.ProjectID
	if projectID == "" {
		printNoSelectedProject()
		return nil
	}
	// Get project details
	prj, err := getSelectedProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get project details")
	}

	statusURL := fmt.Sprintf("%s/api/deployment/%s", baseURL, projectID)
	result, err := sendRequest(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to get deployment status")
	}
	var response map[string]any
	err = json.Unmarshal(result, &response)
	if err != nil {
		return eris.Wrap(err, "Failed to unmarshal deployment status")
	}
	var data map[string]any
	if response["data"] != nil {
		data = response["data"].(map[string]any)
	}
	fmt.Println("Deployment Status")
	fmt.Println("-----------------")
	fmt.Printf("Project:      %s\n", prj.Name)
	fmt.Printf("Project Slug: %s\n", prj.Slug)
	fmt.Printf("Repository:   %s\n", prj.RepoURL)
	if data == nil {
		fmt.Printf("\n** Project has not been deployed **\n")
		return nil
	}
	if data["project_id"] != projectID {
		return eris.Errorf("Deployment status does not match project id %s", projectID)
	}
	if data["type"] != "deploy" {
		return eris.Errorf("Deployment status does not match type %s", data["type"])
	}
	executorID := data["executor_id"].(string)
	dt, dte := time.Parse(time.RFC3339, data["execution_time"].(string))
	if dte != nil {
		return eris.Wrapf(dte, "Failed to parse execution time %s", dt)
	}
	buildNumber := int(data["build_number"].(float64))
	bt, bte := time.Parse(time.RFC3339, data["build_start_time"].(string))
	if bte != nil {
		return eris.Wrapf(bte, "Failed to parse build start time %s", bt)
	}
	buildState := data["build_state"].(string)
	if buildState != "finished" {
		fmt.Printf("Build:        #%d started %s by %s - %s\n", buildNumber, dt.Format(time.RFC822), executorID, buildState)
		return nil
	}
	fmt.Printf("Build:        #%d on %s by %s\n", buildNumber, dt.Format(time.RFC822), executorID)
	fmt.Print("Health:       ")

	// fmt.Println()
	//	fmt.Println(string(result))

	healthURL := fmt.Sprintf("%s/api/health/%s", baseURL, projectID)
	result, err = sendRequest(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to get health")
	}
	err = json.Unmarshal(result, &response)
	if err != nil {
		return eris.Wrap(err, "Failed to unmarshal deployment status")
	}
	instances := response["data"].([]any)
	if len(instances) == 0 {
		fmt.Println("** No deployed instances found **")
		return nil
	}
	fmt.Printf("(%d deployed instances)\n", len(instances))
	currRegion := ""
	for _, instance := range instances {
		info := instance.(map[string]any)
		region := info["region"].(string)
		instanceNum := int(info["instance"].(float64))
		cardinalInfo := info["cardinal"].(map[string]any)
		nakamaInfo := info["nakama"].(map[string]any)
		cardinalURL := cardinalInfo["url"].(string)
		cardinalHost := strings.Split(cardinalURL, "/")[2]
		cardinalOK := cardinalInfo["ok"].(bool)
		cardinalResultCode := int(cardinalInfo["result_code"].(float64))
		cardinalResultStr := cardinalInfo["result_str"].(string)
		nakamaURL := nakamaInfo["url"].(string)
		nakamaHost := strings.Split(nakamaURL, "/")[2]
		nakamaOK := nakamaInfo["ok"].(bool)
		nakamaResultCode := int(nakamaInfo["result_code"].(float64))
		nakamaResultStr := nakamaInfo["result_str"].(string)

		if region != currRegion {
			currRegion = region
			fmt.Printf("• %s\n", currRegion)
		}
		fmt.Printf("  %d)", instanceNum)
		fmt.Printf("\tCardinal: %s - ", cardinalHost)
		if cardinalOK {
			fmt.Print("OK\n")
		} else if cardinalResultCode == 0 {
			fmt.Printf("FAIL %s\n", statusFailRegEx.ReplaceAllString(cardinalResultStr, ""))
		} else {
			fmt.Printf("FAIL %d %s\n", cardinalResultCode, statusFailRegEx.ReplaceAllString(cardinalResultStr, ""))
		}
		fmt.Printf("\tNakama:   %s - ", nakamaHost)
		if nakamaOK {
			fmt.Print("OK\n")
		} else if nakamaResultCode == 0 {
			fmt.Printf("FAIL %s\n", statusFailRegEx.ReplaceAllString(nakamaResultStr, ""))
		} else {
			fmt.Printf("FAIL %d %s\n", nakamaResultCode, statusFailRegEx.ReplaceAllString(nakamaResultStr, ""))
		}
	}
	//fmt.Println()
	//fmt.Println(string(result))

	return nil
}
