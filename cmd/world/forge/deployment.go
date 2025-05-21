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
	"pkg.world.dev/world-cli/common/printer"
)

const (
	DeploymentTypeDeploy   = "deploy"
	DeploymentTypeDestroy  = "destroy"
	DeploymentTypeReset    = "reset"
	DeploymentStatusFailed = "failed"
	DeploymentStatusPassed = "passed"
)

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

// Deployment a project.
func deployment(ctx context.Context, cmdState *CommandState, deployType string) error {
	if cmdState.Organization == nil || cmdState.Organization.ID == "" {
		printNoSelectedOrganization()
		return nil
	}
	organizationID := cmdState.Organization.ID

	var projectID string
	if cmdState.Project != nil && cmdState.Project.ID != "" {
		projectID = cmdState.Project.ID
	}

	// Ensure organization is not nil before this call.
	if cmdState.Project == nil || projectID == "" {
		org, err := getSelectedOrganization(ctx)
		if err != nil {
			return eris.Wrap(err, "Failed on deployment to get selected organization")
		}

		printer.Infof("Deploy requires a project created in World Forge: %s\n", org.Name)

		pID, err := createProject(ctx, &CreateProjectCmd{})
		if err != nil {
			return eris.Wrap(err, "Failed on deployment to create project")
		}
		projectID = pID.ID
	}

	// preview deployment
	err := previewDeployment(ctx, deployType, organizationID, projectID)
	if err != nil {
		return eris.Wrap(err, "Failed to preview deployment")
	}

	processTitle := map[string]string{
		DeploymentTypeDeploy:  "Deploying",
		DeploymentTypeDestroy: "Destroying",
		DeploymentTypeReset:   "Resetting",
	}

	// prompt user to confirm deployment
	printer.NewLine(1)
	printer.Headerln("   Confirm Deployment")
	printer.Infoln("Review the deployment details above.")
	prompt := fmt.Sprintf("\nDo you want to proceed with the %s? (Y/n): ", processTitle[deployType])

	confirmation := getInput(prompt, "n")

	if confirmation != "Y" {
		if confirmation == "y" {
			printer.Infoln("You need to put Y (uppercase) to confirm deployment")
			printer.NewLine(1)
			printer.Errorln("Deployment cancelled")
			return nil
		}
		printer.NewLine(1)
		printer.Errorln("Deployment cancelled")
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

	printer.NewLine(1)
	printer.Successf("Your %s is being processed!\n", deployType)
	printer.NewLine(1)
	printer.Infof("To check the status of your %s, run:\n", deployType)
	printer.Infoln("  $ 'world status'")

	return nil
}

//nolint:funlen, gocognit, gocyclo, cyclop // this is actually a straightforward function with a lot of error handling
func status(ctx context.Context, cmdState *CommandState) error {
	if cmdState.Project == nil || cmdState.Project.ID == "" {
		printNoSelectedProject()
		return nil
	}
	projectID := cmdState.Project.ID
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
	printer.Infoln(" Deployment Status ")
	printer.SectionDivider("-", 19)
	printer.Infof("Project:      %s\n", prj.Name)
	printer.Infof("Project Slug: %s\n", prj.Slug)
	printer.Infof("Repository:   %s\n", prj.RepoURL)
	if len(envMap) == 0 {
		printer.NewLine(1)
		printer.Infoln("** Project has not been deployed **")
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
			bnf, okInner := data["build_number"].(float64)
			if !okInner {
				return eris.New("Failed to unmarshal deployment build_number")
			}
			buildNumber := int(bnf)
			buildStartTimeStr, okInner := data["build_start_time"].(string)
			if !okInner {
				return eris.New("Failed to unmarshal deployment build_start_time")
			}
			bst, bte := time.Parse(time.RFC3339, buildStartTimeStr)
			if bte != nil {
				return eris.Wrapf(bte, "Failed to parse deployment build_start_time %s", buildStartTimeStr)
			}
			if bst.Before(dt) {
				bst = dt // we don't have a real build start time yet because build kite hasn't run yet
			}
			buildEndTimeStr, okInner := data["build_end_time"].(string)
			if !okInner {
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
				printer.Successf("Build:     [%s] #%d (duration %s) completed %s (%s ago) by %s\n",
					strings.ToUpper(env), buildNumber,
					formattedDuration(buildDuration),
					bet.Format(time.RFC822), formattedDuration(time.Since(bet)), executorID)
				shouldShowHealth[env] = true
			case DeploymentStatusFailed:
				printer.Errorf("Build:     [%s] #%d (duration %s) failed at %s (%s ago)\n",
					strings.ToUpper(env), buildNumber, formattedDuration(buildDuration),
					bet.Format(time.RFC822), formattedDuration(time.Since(bet)))
			default:
				printer.Infof("🔄 Build:     [%s] #%d started %s (%s ago) by %s - %s\n",
					strings.ToUpper(env), buildNumber,
					bst.Format(time.RFC822), formattedDuration(time.Since(bst)), executorID, buildState)
			}
		case DeploymentTypeDestroy:
			switch buildState {
			case DeploymentStatusPassed:
				printer.Successf("Destroyed: [%s] on %s by %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822), executorID)
			case DeploymentStatusFailed:
				printer.Errorf("Destroy:   [%s] failed on %s by %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822), executorID)
				// if destroy failed, continue on to show health
				shouldShowHealth[env] = true
			default:
				printer.Infof("🔄 Destroy:   [%s] started %s (%s ago) by %s - %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822),
					formattedDuration(time.Since(dt)), executorID, buildState)
			}
		case DeploymentTypeReset:
			// results can be "passed" or "failed", but either way continue to show the health
			switch buildState {
			case DeploymentStatusPassed:
				printer.Successf("Reset:     [%s] on %s by %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822), executorID)
				shouldShowHealth[env] = true
			case DeploymentStatusFailed:
				printer.Errorf("Reset:     [%s] failed on %s by %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822), executorID)
				// if destroy failed, continue on to show health
				shouldShowHealth[env] = true
			default:
				printer.Infof("🔄 Reset:     [%s] started %s (%s ago) by %s - %s\n",
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

	statusFailRegEx := regexp.MustCompile(`[^a-zA-Z0-9\. ]+`)
	for env, val := range envMap {
		if !shouldShowHealth[env] {
			continue
		}
		data, okay := val.(map[string]any)
		if !okay {
			return eris.Errorf("Failed to unmarshal response for environment %s", env)
		}
		instances, okay := data["deployed_instances"].([]any)
		if !okay {
			return eris.Errorf("Failed to unmarshal health data: expected array, got %T",
				data["deployed_instances"])
		}
		// ok will be true if everything is up. offline will be true if everything is down
		// neither will be set if status is mixed
		switch {
		case data["ok"] == true:
			printer.Successf("Health:    [%s] ", strings.ToUpper(env))
		case data["offline"] == true:
			printer.Errorf("Health:    [%s] ", strings.ToUpper(env))
		default:
			printer.Infof("⚠️ Health:    [%s] ", strings.ToUpper(env))
		}
		if len(instances) == 0 {
			printer.Infoln("** No deployed instances found **")
			return nil
		}
		printer.Infof("(%d deployed instances)\n", len(instances))
		currRegion := ""
		for _, instance := range instances {
			info, okayInner := instance.(map[string]any)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d info", instance)
			}
			region, okayInner := info["region"].(string)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d region", instance)
			}
			instancef, okayInner := info["instance"].(float64)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d instance number", instance)
			}
			instanceNum := int(instancef)
			cardinalInfo, okayInner := info["cardinal"].(map[string]any)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d cardinal data", instance)
			}
			nakamaInfo, okayInner := info["nakama"].(map[string]any)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d nakama data", instance)
			}
			cardinalURL, okayInner := cardinalInfo["url"].(string)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d cardinal url", instance)
			}
			cardinalHost := strings.Split(cardinalURL, "/")[2]
			cardinalOK, okayInner := cardinalInfo["ok"].(bool)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d cardinal ok flag", instance)
			}
			cardinalResultCodef, okayInner := cardinalInfo["result_code"].(float64)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d cardinal result_code", instance)
			}
			cardinalResultCode := int(cardinalResultCodef)
			cardinalResultStr, okayInner := cardinalInfo["result_str"].(string)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d cardinal result_str", instance)
			}
			nakamaURL, okayInner := nakamaInfo["url"].(string)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d nakama url", instance)
			}
			nakamaHost := strings.Split(nakamaURL, "/")[2]
			nakamaOK, okayInner := nakamaInfo["ok"].(bool)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d nakama ok", instance)
			}
			nakamaResultCodef, okayInner := nakamaInfo["result_code"].(float64)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d result_code", instance)
			}
			nakamaResultCode := int(nakamaResultCodef)
			nakamaResultStr, okayInner := nakamaInfo["result_str"].(string)
			if !okayInner {
				return eris.Errorf("Failed to unmarshal deployment instance %d result_str", instance)
			}
			if region != currRegion {
				currRegion = region
				printer.Infof("• %s\n", currRegion)
			}
			printer.Infof("  %d)", instanceNum)
			switch {
			case cardinalOK:
				printer.Successf("Cardinal: %s - OK\n", cardinalHost)
			case cardinalResultCode == 0:
				printer.Errorf("Cardinal: %s - FAIL %s\n", cardinalHost,
					statusFailRegEx.ReplaceAllString(cardinalResultStr, ""))
			default:
				printer.Errorf("Cardinal: %s - FAIL %d %s\n", cardinalHost, cardinalResultCode,
					statusFailRegEx.ReplaceAllString(cardinalResultStr, ""))
			}
			switch {
			case nakamaOK:
				printer.Successf("Nakama:   %s - OK\n", nakamaHost)
			case nakamaResultCode == 0:
				printer.Errorf("Nakama:   %s - FAIL %s\n", nakamaHost,
					statusFailRegEx.ReplaceAllString(nakamaResultStr, ""))
			default:
				printer.Errorf("Nakama:   %s - FAIL %d %s\n", nakamaHost, nakamaResultCode,
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
	printer.NewLine(1)
	printer.Headerln("   Deployment Preview")

	printer.NewLine(1)
	printer.Headerln("   Basic Information   ")
	printer.SectionDivider("-", 23)
	printer.Infof("Organization:    %s\n", response.Data.OrgName)
	printer.Infof("Org Slug:        %s\n", response.Data.OrgSlug)
	printer.Infof("Project:         %s\n", response.Data.ProjectName)
	printer.Infof("Project Slug:    %s\n", response.Data.ProjectSlug)

	printer.NewLine(1)
	printer.Headerln("     Configuration     ")
	printer.SectionDivider("-", 23)
	printer.Infof("Executor:        %s\n", response.Data.ExecutorName)
	printer.Infof("Deployment Type: %s\n", response.Data.DeploymentType)
	printer.Infof("Tick Rate:       %d\n", response.Data.TickRate)

	printer.NewLine(1)
	printer.Headerln("  Deployment Regions  ")
	printer.SectionDivider("-", 23)
	printer.Infof("%s\n", strings.Join(response.Data.Regions, ", "))

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
