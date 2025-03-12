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

const (
	DeploymentTypeDeploy   = "deploy"
	DeploymentTypeDestroy  = "destroy"
	DeploymentTypeReset    = "reset"
	DeploymentStatusFailed = "failed"
	DeploymentStatusPassed = "passed"
)

var statusFailRegEx = regexp.MustCompile(`[^a-zA-Z0-9\. ]+`)

type deploymentPreview struct {
	OrgName        string   `json:"org_name"`
	OrgSlug        string   `json:"org_slug"`
	ProjectName    string   `json:"project_name"`
	ProjectSlug    string   `json:"project_slug"`
	ExecutorName   string   `json:"executor_name"`
	DeploymentType string   `json:"deployment_type"`
	TickRate       int      `json:"tick_rate"`
	Regions        []string `json:"regions"`
}

// Deployment a project
func deployment(ctx context.Context, deployType string) error {
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

	// preview deployment
	err = previewDeployment(ctx, deployType, organizationID, projectID)
	if err != nil {
		return eris.Wrap(err, "Failed to preview deployment")
	}

	// prompt user to confirm deployment
	fmt.Println("\n🔄  Confirm Deployment ✨")
	fmt.Println("=========================")
	fmt.Println("\n🔍  Review the deployment details above.")
	fmt.Printf("\n❓ Do you want to proceed with the deployment? (Y/n): ")

	confirmation, err := getInput()
	if err != nil {
		return eris.Wrap(err, "Failed to read confirmation")
	}

	if confirmation != "Y" {
		if confirmation == "y" {
			fmt.Println("You need to put Y (uppercase) to confirm deployment")
			fmt.Println("\n❌ Deployment cancelled")
			return nil
		}
		fmt.Println("\n❌ Deployment cancelled")
		return nil
	}

	if deployType == "forceDeploy" {
		deployType = "deploy?force=true"
	}
	deployURL := fmt.Sprintf("%s/api/organization/%s/project/%s/%s", baseURL, organizationID, projectID, deployType)
	_, err = sendRequest(ctx, http.MethodPost, deployURL, nil)
	if err != nil {
		return eris.Wrap(err, fmt.Sprintf("Failed to %s project", deployType))
	}

	fmt.Printf("\n✨ Your %s is being processed! ✨\n", deployType)
	fmt.Printf("\nTo check the status of your %s, run:\n", deployType)
	fmt.Println("  $ 'world forge deployment status'")

	return nil
}

//nolint:funlen, gocognit, gocyclo, cyclop // this is actually a straightforward function with a lot of error handling
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
		return eris.Wrap(err, "Failed to unmarshal deployment response")
	}
	var envMap map[string]any
	if response["data"] != nil {
		// data = null is returned when there are no deployments, so we have to check for that before we
		// try to cast the response into a json object map, since this is not an error but the cast would
		// fail
		var ok bool
		envMap, ok = response["data"].(map[string]any)
		if !ok {
			return eris.New("Failed to unmarshal deployment data")
		}
	}
	fmt.Println("Deployment Status")
	fmt.Println("-----------------")
	fmt.Printf("Project:      %s\n", prj.Name)
	fmt.Printf("Project Slug: %s\n", prj.Slug)
	fmt.Printf("Repository:   %s\n", prj.RepoURL)
	if len(envMap) == 0 {
		fmt.Printf("\n** Project has not been deployed **\n")
		return nil
	}
	checkHealth := false
	shouldShowHealth := map[string]bool{}
	for env, val := range envMap {
		shouldShowHealth[env] = false
		data, ok := val.(map[string]any)
		if !ok {
			return eris.Errorf("Failed to unmarshal response for environment %s", env)
		}
		if data["project_id"] != projectID {
			return eris.Errorf("Deployment status does not match project id %s", projectID)
		}
		deployType, ok := data["type"].(string)
		if !ok {
			return eris.New("Failed to unmarshal deployment type")
		}
		if deployType != DeploymentTypeDeploy &&
			deployType != DeploymentTypeDestroy &&
			deployType != DeploymentTypeReset {
			return eris.Errorf("Unknown deployment type %s", deployType)
		}
		executorID, ok := data["executor_id"].(string)
		if !ok {
			return eris.New("Failed to unmarshal deployment executor_id")
		}
		executorName, ok := data["executor_name"].(string)
		if ok {
			executorID = executorName
		}
		executionTimeStr, ok := data["execution_time"].(string)
		if !ok {
			return eris.New("Failed to unmarshal deployment execution_time")
		}
		dt, dte := time.Parse(time.RFC3339, executionTimeStr)
		if dte != nil {
			return eris.Wrapf(dte, "Failed to parse deployment execution_time %s", executionTimeStr)
		}
		buildState, ok := data["build_state"].(string)
		if !ok {
			return eris.New("Failed to unmarshal deployment build_state")
		}
		switch deployType {
		case DeploymentTypeDeploy:
			bnf, ok := data["build_number"].(float64)
			if !ok {
				return eris.New("Failed to unmarshal deployment build_number")
			}
			buildNumber := int(bnf)
			buildStartTimeStr, ok := data["build_start_time"].(string)
			if !ok {
				return eris.New("Failed to unmarshal deployment build_start_time")
			}
			bst, bte := time.Parse(time.RFC3339, buildStartTimeStr)
			if bte != nil {
				return eris.Wrapf(bte, "Failed to parse deployment build_start_time %s", buildStartTimeStr)
			}
			if bst.Before(dt) {
				bst = dt // we don't have a real build start time yet because build kite hasn't run yet
			}
			buildEndTimeStr, ok := data["build_end_time"].(string)
			if !ok {
				buildEndTimeStr = buildStartTimeStr // we don't know how long this took
			}
			bet, bte := time.Parse(time.RFC3339, buildEndTimeStr)
			if bte != nil {
				return eris.Wrapf(bte, "Failed to parse deployment build_end_time %s", buildEndTimeStr)
			}
			if bet.Before(bst) {
				bet = bst // we don't know how long this took
			}
			buildDuration := bet.Sub(bst)
			// buildkite states (used with deployType deploy) are:
			//   creating, scheduled, running, passed, failing, failed, blocked, canceling, canceled, skipped, not_run

			switch buildState {
			case DeploymentStatusPassed:
				fmt.Printf("✅ Build:     [%s] #%d (duration %s) completed %s (%s ago) by %s\n",
					strings.ToUpper(env), buildNumber,
					formattedDuration(buildDuration),
					bet.Format(time.RFC822), formattedDuration(time.Since(bet)), executorID)
				shouldShowHealth[env] = true
			case DeploymentStatusFailed:
				fmt.Printf("❌ Build:     [%s] #%d (duration %s) failed at %s (%s ago)\n",
					strings.ToUpper(env), buildNumber, formattedDuration(buildDuration),
					bet.Format(time.RFC822), formattedDuration(time.Since(bet)))
			default:
				fmt.Printf("🔄 Build:     [%s] #%d started %s (%s ago) by %s - %s\n",
					strings.ToUpper(env), buildNumber,
					bst.Format(time.RFC822), formattedDuration(time.Since(bst)), executorID, buildState)
			}
		case DeploymentTypeDestroy:
			switch buildState {
			case DeploymentStatusPassed:
				fmt.Printf("✅ Destroyed: [%s] on %s by %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822), executorID)
			case DeploymentStatusFailed:
				fmt.Printf("❌ Destroy:   [%s] failed on %s by %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822), executorID)
				// if destroy failed, continue on to show health
				shouldShowHealth[env] = true
			default:
				fmt.Printf("🔄 Destroy:   [%s] started %s (%s ago) by %s - %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822),
					formattedDuration(time.Since(dt)), executorID, buildState)
			}
		case DeploymentTypeReset:
			// results can be "passed" or "failed", but either way continue to show the health
			switch buildState {
			case DeploymentStatusPassed:
				fmt.Printf("✅ Reset:     [%s] on %s by %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822), executorID)
				shouldShowHealth[env] = true
			case DeploymentStatusFailed:
				fmt.Printf("❌ Reset:     [%s] failed on %s by %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822), executorID)
				// if destroy failed, continue on to show health
				shouldShowHealth[env] = true
			default:
				fmt.Printf("🔄 Reset:     [%s] started %s (%s ago) by %s - %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822),
					formattedDuration(time.Since(dt)), executorID, buildState)
			}
		default:
			return eris.Errorf("Unknown deployment type %s", deployType)
		}
		if shouldShowHealth[env] {
			checkHealth = true
		}
	}

	if !checkHealth {
		return nil
	}

	healthURL := fmt.Sprintf("%s/api/health/%s", baseURL, projectID)
	result, err = sendRequest(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to get health")
	}
	err = json.Unmarshal(result, &response)
	if err != nil {
		return eris.Wrap(err, "Failed to unmarshal health response")
	}
	if response["data"] == nil {
		return eris.New("Failed to unmarshal health data")
	}
	envMap, ok := response["data"].(map[string]any)
	if !ok {
		return eris.New("Failed to unmarshal health data")
	}
	for env, val := range envMap {
		if !shouldShowHealth[env] {
			continue
		}
		data, ok := val.(map[string]any)
		if !ok {
			return eris.Errorf("Failed to unmarshal response for environment %s", env)
		}
		instances, ok := data["deployed_instances"].([]any)
		if !ok {
			return eris.Errorf("Failed to unmarshal health data: expected array, got %T",
				response["deployed_instances"])
		}
		// ok will be true if everything is up. offline will be true if everything is down
		// neither will be set if status is mixed
		switch {
		case data["ok"] == true:
			fmt.Printf("✅ Health:    [%s] ", strings.ToUpper(env))
		case data["offline"] == true:
			fmt.Printf("❌ Health:    [%s] ", strings.ToUpper(env))
		default:
			fmt.Printf("⚠️ Health:    [%s] ", strings.ToUpper(env))
		}
		if len(instances) == 0 {
			fmt.Println("** No deployed instances found **")
			return nil
		}
		fmt.Printf("(%d deployed instances)\n", len(instances))
		currRegion := ""
		for _, instance := range instances {
			info, ok := instance.(map[string]any)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d info", instance)
			}
			region, ok := info["region"].(string)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d region", instance)
			}
			instancef, ok := info["instance"].(float64)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d instance number", instance)
			}
			instanceNum := int(instancef)
			cardinalInfo, ok := info["cardinal"].(map[string]any)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d cardinal data", instance)
			}
			nakamaInfo, ok := info["nakama"].(map[string]any)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d nakama data", instance)
			}
			cardinalURL, ok := cardinalInfo["url"].(string)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d cardinal url", instance)
			}
			cardinalHost := strings.Split(cardinalURL, "/")[2]
			cardinalOK, ok := cardinalInfo["ok"].(bool)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d cardinal ok flag", instance)
			}
			cardinalResultCodef, ok := cardinalInfo["result_code"].(float64)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d cardinal result_code", instance)
			}
			cardinalResultCode := int(cardinalResultCodef)
			cardinalResultStr, ok := cardinalInfo["result_str"].(string)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d cardinal result_str", instance)
			}
			nakamaURL, ok := nakamaInfo["url"].(string)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d nakama url", instance)
			}
			nakamaHost := strings.Split(nakamaURL, "/")[2]
			nakamaOK, ok := nakamaInfo["ok"].(bool)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d nakama ok", instance)
			}
			nakamaResultCodef, ok := nakamaInfo["result_code"].(float64)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d result_code", instance)
			}
			nakamaResultCode := int(nakamaResultCodef)
			nakamaResultStr, ok := nakamaInfo["result_str"].(string)
			if !ok {
				return eris.Errorf("Failed to unmarshal deployment instance %d result_str", instance)
			}
			if region != currRegion {
				currRegion = region
				fmt.Printf("• %s\n", currRegion)
			}
			fmt.Printf("  %d)", instanceNum)
			switch {
			case cardinalOK:
				fmt.Printf("\t✅ Cardinal: %s - OK\n", cardinalHost)
			case cardinalResultCode == 0:
				fmt.Printf("\t❌ Cardinal: %s - FAIL %s\n", cardinalHost,
					statusFailRegEx.ReplaceAllString(cardinalResultStr, ""))
			default:
				fmt.Printf("\t❌ Cardinal: %s - FAIL %d %s\n", cardinalHost, cardinalResultCode,
					statusFailRegEx.ReplaceAllString(cardinalResultStr, ""))
			}
			switch {
			case nakamaOK:
				fmt.Printf("\t✅ Nakama:   %s - OK\n", nakamaHost)
			case nakamaResultCode == 0:
				fmt.Printf("\t❌ Nakama:   %s - FAIL %s\n", nakamaHost,
					statusFailRegEx.ReplaceAllString(nakamaResultStr, ""))
			default:
				fmt.Printf("\t❌ Nakama:   %s - FAIL %d %s\n", nakamaHost, nakamaResultCode,
					statusFailRegEx.ReplaceAllString(nakamaResultStr, ""))
			}
		}
	}
	return nil
}

func previewDeployment(ctx context.Context, deployType string, organizationID string, projectID string) error {
	deployURL := fmt.Sprintf("%s/api/organization/%s/project/%s/%s?preview=true",
		baseURL, organizationID, projectID, deployType)
	resultBytes, err := sendRequest(ctx, http.MethodPost, deployURL, nil)
	if err != nil {
		return eris.Wrap(err, fmt.Sprintf("Failed to %s project", deployType))
	}

	type deploymentPreviewResponse struct {
		Data deploymentPreview `json:"data"`
	}
	var response deploymentPreviewResponse
	err = json.Unmarshal(resultBytes, &response)
	if err != nil {
		return eris.Wrap(err, "Failed to unmarshal deployment preview")
	}
	fmt.Println("\n✨ Deployment Preview ✨")
	fmt.Println("=======================")
	fmt.Println("\n📋 Basic Information")
	fmt.Println("------------------")
	fmt.Printf("🏢 Organization:     %s\n", response.Data.OrgName)
	fmt.Printf("🔖 Org Slug:        %s\n", response.Data.OrgSlug)
	fmt.Printf("📁 Project:         %s\n", response.Data.ProjectName)
	fmt.Printf("🏷️  Project Slug:    %s\n", response.Data.ProjectSlug)

	fmt.Println("\n⚙️  Configuration")
	fmt.Println("--------------")
	fmt.Printf("👤 Executor:        %s\n", response.Data.ExecutorName)
	fmt.Printf("🚀 Deployment Type: %s\n", response.Data.DeploymentType)
	fmt.Printf("⚡ Tick Rate:       %d\n", response.Data.TickRate)

	fmt.Println("\n🌎 Deployment Regions")
	fmt.Println("------------------")
	fmt.Printf("📍 %s\n", strings.Join(response.Data.Regions, ", "))

	return nil
}

func formattedDuration(d time.Duration) string {
	const hoursPerDay = 24
	const minPerHour = 60
	const secPerMinute = 60
	if d.Hours() > hoursPerDay {
		return fmt.Sprintf("%dd %dh", int(d.Hours()/hoursPerDay), int(d.Hours())%hoursPerDay)
	}
	if d.Minutes() > minPerHour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%minPerHour)
	}
	if d.Seconds() > secPerMinute {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%secPerMinute)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
