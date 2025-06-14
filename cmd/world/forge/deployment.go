package forge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	"pkg.world.dev/world-cli/common/printer"
	teaspinner "pkg.world.dev/world-cli/tea/component/spinner"
)

const (
	DeploymentTypeDeploy      = "deploy"
	DeploymentTypeForceDeploy = "forceDeploy"
	DeploymentTypeDestroy     = "destroy"
	DeploymentTypeReset       = "reset"
	DeploymentTypePromote     = "promote"

	DeployStatusFailed  DeployStatus = "failed"
	DeployStatusPassed  DeployStatus = "passed"
	DeployStatusRunning DeployStatus = "running"

	DeployEnvPreview = "dev"
	DeployEnvLive    = "prod"
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

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusOffline   HealthStatus = "offline"
)

type DeployStatus string

type DeployInfo struct {
	DeployType    string
	DeployStatus  DeployStatus
	DeployDisplay string
}

// Deployment a project.
func deployment(fCtx ForgeContext, deployType string) error { //nolint:gocognit,funlen // will refactor later
	if fCtx.State.Organization == nil || fCtx.State.Organization.ID == "" {
		printNoSelectedOrganization()
		return nil
	}

	// Ensure organization is not nil before this call.
	if fCtx.State.Project == nil || fCtx.State.Project.ID == "" {
		org, err := getSelectedOrganization(fCtx)
		if err != nil {
			return eris.Wrap(err, "Failed on deployment to get selected organization")
		}

		printer.Infof("Deploy requires a project created in World Forge: %s\n", org.Name)

		pID, err := createProject(fCtx, &CreateProjectCmd{})
		if err != nil {
			return eris.Wrap(err, "Failed on deployment to create project")
		}
		fCtx.State.Project = &project{ID: pID.ID}
	}

	// preview deployment
	err := previewDeployment(fCtx, deployType)
	if err != nil {
		return eris.Wrap(err, "Failed to preview deployment")
	}

	processTitle := map[string]string{
		DeploymentTypeDeploy:      "Deploying",
		DeploymentTypeForceDeploy: "Force Deploying",
		DeploymentTypeDestroy:     "Destroying",
		DeploymentTypeReset:       "Resetting",
		DeploymentTypePromote:     "Promoting",
	}

	// prompt user to confirm deployment
	printer.NewLine(1)
	prompt := fmt.Sprintf("Do you want to proceed with the %s? (Y/n)", processTitle[deployType])

	confirmation := getInput(prompt, "n")

	if confirmation != "Y" {
		if confirmation == "y" {
			printer.Infoln("You need to put Y (uppercase) to confirm deployment")
			printer.Errorln("Deployment cancelled")
			printer.NewLine(1)
			return nil
		}
		printer.Errorln("Deployment cancelled")
		printer.NewLine(1)
		return nil
	}

	// build the image
	commitHash, reader, err := deploymentBuild(fCtx.Context, fCtx.State.Project)
	if err != nil {
		return eris.Wrap(err, "Failed to build image")
	}

	/* create multipart request */
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// add commit_hash to the request
	err = writer.WriteField("commit_hash", commitHash)
	if err != nil {
		return eris.Wrap(err, "Failed to write commit hash")
	}

	// add the image to the request
	part, err := writer.CreateFormFile("file", "image.tar")
	if err != nil {
		return eris.Wrap(err, "Failed to create form file")
	}
	_, err = io.Copy(part, reader)
	if err != nil {
		return eris.Wrap(err, "Failed to copy image to request")
	}
	writer.Close()
	/* end of multipart request */

	if deployType == DeploymentTypeForceDeploy {
		deployType = "deploy?force=true"
	}

	deployURL := fmt.Sprintf("%s/api/organization/%s/project/%s/%s", baseURL,
		fCtx.State.Organization.ID, fCtx.State.Project.ID, deployType)

	// Create request with proper Content-Type
	req, err := http.NewRequestWithContext(fCtx.Context, http.MethodPost, deployURL, body)
	if err != nil {
		return eris.Wrap(err, "Failed to create request")
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Add auth header
	if fCtx.Config != nil {
		prefix := "ArgusID "
		req.Header.Add("Authorization", prefix+fCtx.Config.Credential.Token)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return eris.Wrap(err, fmt.Sprintf("Failed to %s project", deployType))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return eris.Wrap(err, "Failed to read error response")
		}
		return eris.New(string(body))
	}

	env := DeployEnvPreview
	if deployType == DeploymentTypePromote {
		env = DeployEnvLive
	}

	err = waitUntilDeploymentIsComplete(fCtx, env, deployType)
	if err != nil {
		printer.NewLine(1)
		printer.Successf("Your %s is being processed!\n\n", deployType)
		printer.Infof("To check the status of your %s, run:\n", deployType)
		printer.Infoln("  $ 'world status'")
	}

	return nil
}

func status(fCtx ForgeContext) error {
	if fCtx.State.Project == nil || fCtx.State.Organization == nil {
		return nil
	}

	if fCtx.State.Project.ID == "" || fCtx.State.Organization.ID == "" {
		return nil
	}

	printer.NewLine(1)
	printer.Headerln("   Deployment Status   ")
	printer.Infof("Organization: %s\n", fCtx.State.Organization.Name)
	printer.Infof("Org Slug:     %s\n", fCtx.State.Organization.Slug)
	printer.Infof("Project:      %s\n", fCtx.State.Project.Name)
	printer.Infof("Project Slug: %s\n", fCtx.State.Project.Slug)
	printer.Infof("Repository:   %s\n", fCtx.State.Project.RepoURL)
	printer.NewLine(1)

	deployInfo, err := getDeploymentStatus(fCtx)
	if err != nil {
		return eris.Wrap(err, "Failed to get deployment status")
	}
	showHealth := false
	for env := range deployInfo {
		printDeploymentStatus(deployInfo[env])
		if shouldShowHealth(deployInfo[env]) {
			showHealth = true
		}
	}

	if showHealth {
		// don't care about healthComplete return because we are only doing this once
		_, err = getAndPrintHealth(fCtx, deployInfo)
		if err != nil {
			return eris.Wrap(err, "Failed to get health")
		}
	}
	return nil
}

// Returns a map of environment names to boolean values indicating whether the environment was
// successfully deployed.
// nolint: gocognit, gocyclo, cyclop, funlen // this is a complex function but it does what it needs to do
func getDeploymentStatus(fCtx ForgeContext) (map[string]DeployInfo, error) {
	statusURL := fmt.Sprintf("%s/api/deployment/%s", baseURL, fCtx.State.Project.ID)
	result, err := sendRequest(fCtx, http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get deployment status")
	}
	var response map[string]any
	err = json.Unmarshal(result, &response)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to unmarshal deployment response")
	}
	var envMap map[string]any
	if response["data"] != nil {
		// data = null is returned when there are no deployments, so we have to check for that before we
		// try to cast the response into a json object map, since this is not an error but the cast would
		// fail
		var ok bool
		envMap, ok = response["data"].(map[string]any)
		if !ok {
			return nil, eris.New("Failed to unmarshal deployment data")
		}
	}
	if len(envMap) == 0 {
		printer.Notificationln("** Project has not been deployed **")
		return nil, nil //nolint:nilnil // nil is a valid return value
	}
	deployStatus := map[string]DeployInfo{}
	for env, val := range envMap {
		deployStatus[env] = DeployInfo{
			DeployType:    "",
			DeployStatus:  DeployStatusFailed,
			DeployDisplay: "",
		}
		data, ok := val.(map[string]any)
		if !ok {
			return nil, eris.Errorf("Failed to unmarshal response for environment %s", env)
		}
		if data["project_id"] != fCtx.State.Project.ID {
			return nil, eris.Errorf("Deployment status does not match project id %s", fCtx.State.Project.ID)
		}
		deployType, ok := data["type"].(string)
		if !ok {
			return nil, eris.New("Failed to unmarshal deployment type")
		}
		if deployType != DeploymentTypeDeploy &&
			deployType != DeploymentTypeDestroy &&
			deployType != DeploymentTypeReset {
			return nil, eris.Errorf("Unknown deployment type %s", deployType)
		}
		executorID, ok := data["executor_id"].(string)
		if !ok {
			return nil, eris.New("Failed to unmarshal deployment executor_id")
		}
		executorName, ok := data["executor_name"].(string)
		if ok {
			executorID = executorName
		}
		executionTimeStr, ok := data["execution_time"].(string)
		if !ok {
			return nil, eris.New("Failed to unmarshal deployment execution_time")
		}
		dt, dte := time.Parse(time.RFC3339, executionTimeStr)
		if dte != nil {
			return nil, eris.Wrapf(dte, "Failed to parse deployment execution_time %s", executionTimeStr)
		}
		buildState, ok := data["build_state"].(string)
		if !ok {
			return nil, eris.New("Failed to unmarshal deployment build_state")
		}
		switch deployType {
		case DeploymentTypeDeploy:
			bnf, okInner := data["build_number"].(float64)
			if !okInner {
				return nil, eris.New("Failed to unmarshal deployment build_number")
			}
			buildNumber := int(bnf)
			buildStartTimeStr, okInner := data["build_start_time"].(string)
			if !okInner {
				return nil, eris.New("Failed to unmarshal deployment build_start_time")
			}
			bst, bte := time.Parse(time.RFC3339, buildStartTimeStr)
			if bte != nil {
				return nil, eris.Wrapf(bte, "Failed to parse deployment build_start_time %s", buildStartTimeStr)
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
				return nil, eris.Wrapf(bte, "Failed to parse deployment build_end_time %s", buildEndTimeStr)
			}
			if bet.Before(bst) {
				bet = bst // we don't know how long this took
			}
			buildDuration := bet.Sub(bst)
			// buildkite states (used with deployType deploy) are:
			//   creating, scheduled, running, passed, failing, failed, blocked, canceling, canceled, skipped, not_run

			switch buildState {
			case string(DeployStatusPassed):
				deployStatus[env] = DeployInfo{
					DeployType:   DeploymentTypeDeploy,
					DeployStatus: DeployStatusPassed,
					DeployDisplay: fmt.Sprintf("Build:     [%s] #%d (duration %s) completed %s (%s ago) by %s\n",
						envDisplayName(env), buildNumber,
						formattedDuration(buildDuration),
						bet.Format(time.RFC822), formattedDuration(time.Since(bet)), executorID),
				}
			case string(DeployStatusFailed):
				deployStatus[env] = DeployInfo{
					DeployType:   DeploymentTypeDeploy,
					DeployStatus: DeployStatusFailed,
					DeployDisplay: fmt.Sprintf("Build:     [%s] #%d (duration %s) failed at %s (%s ago)\n",
						envDisplayName(env), buildNumber, formattedDuration(buildDuration),
						bet.Format(time.RFC822), formattedDuration(time.Since(bet))),
				}
			default:
				deployStatus[env] = DeployInfo{
					DeployType:   DeploymentTypeDeploy,
					DeployStatus: DeployStatusRunning,
					DeployDisplay: fmt.Sprintf("Build:     [%s] #%d started %s (%s ago) by %s - %s\n",
						envDisplayName(env), buildNumber,
						bst.Format(time.RFC822), formattedDuration(time.Since(bst)), executorID, buildState),
				}
			}
		case DeploymentTypeDestroy:
			switch buildState {
			case string(DeployStatusPassed):
				deployStatus[env] = DeployInfo{
					DeployType:   DeploymentTypeDestroy,
					DeployStatus: DeployStatusPassed,
					DeployDisplay: fmt.Sprintf("Destroyed: [%s] on %s by %s\n",
						envDisplayName(env), dt.Format(time.RFC822), executorID),
				}
			case string(DeployStatusFailed):
				deployStatus[env] = DeployInfo{
					DeployType:   DeploymentTypeDestroy,
					DeployStatus: DeployStatusFailed,
					DeployDisplay: fmt.Sprintf("Destroy:   [%s] failed on %s by %s\n",
						envDisplayName(env), dt.Format(time.RFC822), executorID),
				}
			default:
				deployStatus[env] = DeployInfo{
					DeployType:   DeploymentTypeDestroy,
					DeployStatus: DeployStatusRunning,
					DeployDisplay: fmt.Sprintf("Destroy:   [%s] started %s (%s ago) by %s - %s\n",
						envDisplayName(env), dt.Format(time.RFC822),
						formattedDuration(time.Since(dt)), executorID, buildState),
				}
			}
		case DeploymentTypeReset:
			// results can be "passed" or "failed", but either way continue to show the health
			switch buildState {
			case string(DeployStatusPassed):
				deployStatus[env] = DeployInfo{
					DeployType:   DeploymentTypeReset,
					DeployStatus: DeployStatusPassed,
					DeployDisplay: fmt.Sprintf("Reset:     [%s] on %s by %s\n",
						envDisplayName(env), dt.Format(time.RFC822), executorID),
				}
			case string(DeployStatusFailed):
				deployStatus[env] = DeployInfo{
					DeployType:   DeploymentTypeReset,
					DeployStatus: DeployStatusFailed,
					DeployDisplay: fmt.Sprintf("Reset:     [%s] failed on %s by %s\n",
						envDisplayName(env), dt.Format(time.RFC822), executorID),
				}
			default:
				deployStatus[env] = DeployInfo{
					DeployType:   DeploymentTypeReset,
					DeployStatus: DeployStatusRunning,
					DeployDisplay: fmt.Sprintf("Reset:     [%s] started %s (%s ago) by %s - %s\n",
						envDisplayName(env), dt.Format(time.RFC822),
						formattedDuration(time.Since(dt)), executorID, buildState),
				}
			}
		default:
			return nil, eris.Errorf("Unknown deployment type %s", deployType)
		}
	}
	return deployStatus, nil
}

// nolint: gocognit, gocyclo, cyclop, funlen // this is a complex function but it does what it needs to do
func getAndPrintHealth(fCtx ForgeContext, deployInfo map[string]DeployInfo) (bool, error) {
	healthURL := fmt.Sprintf("%s/api/health/%s", baseURL, fCtx.State.Project.ID)
	result, err := sendRequest(fCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false, eris.Wrap(err, "Failed to get health")
	}
	var response map[string]any
	err = json.Unmarshal(result, &response)
	if err != nil {
		return false, eris.Wrap(err, "Failed to unmarshal health response")
	}
	if response["data"] == nil {
		return false, eris.New("Failed to unmarshal health data")
	}
	envMap, ok := response["data"].(map[string]any)
	if !ok {
		return false, eris.New("Failed to unmarshal health data")
	}

	healthComplete := true
	statusFailRegEx := regexp.MustCompile(`[^a-zA-Z0-9\. ]+`)
	for env, val := range envMap {
		if !shouldShowHealth(deployInfo[env]) {
			// only show health for environments that have been deployed
			continue
		}
		data, okay := val.(map[string]any)
		if !okay {
			return false, eris.Errorf("Failed to unmarshal response for environment %s", env)
		}
		instances, okay := data["deployed_instances"].([]any)
		if !okay {
			return false, eris.Errorf("Failed to unmarshal health data: expected array, got %T",
				data["deployed_instances"])
		}
		// ok will be true if everything is up. offline will be true if everything is down
		// neither will be set if status is mixed
		switch {
		case data["ok"] == true:
			printer.Successf("Health:    [%s] ", envDisplayName(env))
		case data["offline"] == true:
			printer.Errorf("Health:    [%s] ", envDisplayName(env))
			healthComplete = false
		default:
			printer.Infof("⚠️ Health:    [%s] ", envDisplayName(env))
			healthComplete = false
		}
		if len(instances) == 0 {
			printer.Infoln("** No deployed instances found **")
			continue
		}
		printer.Infof("(%d deployed instances)\n", len(instances))
		currRegion := ""
		for _, instance := range instances {
			info, okayInner := instance.(map[string]any)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d info", instance)
			}
			region, okayInner := info["region"].(string)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d region", instance)
			}
			instancef, okayInner := info["instance"].(float64)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d instance number", instance)
			}
			instanceNum := int(instancef)
			cardinalInfo, okayInner := info["cardinal"].(map[string]any)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d cardinal data", instance)
			}
			nakamaInfo, okayInner := info["nakama"].(map[string]any)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d nakama data", instance)
			}
			cardinalURL, okayInner := cardinalInfo["url"].(string)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d cardinal url", instance)
			}
			cardinalHost := strings.Split(cardinalURL, "/")[2]
			cardinalOK, okayInner := cardinalInfo["ok"].(bool)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d cardinal ok flag", instance)
			}
			cardinalResultCodef, okayInner := cardinalInfo["result_code"].(float64)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d cardinal result_code", instance)
			}
			cardinalResultCode := int(cardinalResultCodef)
			cardinalResultStr, okayInner := cardinalInfo["result_str"].(string)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d cardinal result_str", instance)
			}
			nakamaURL, okayInner := nakamaInfo["url"].(string)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d nakama url", instance)
			}
			nakamaHost := strings.Split(nakamaURL, "/")[2]
			nakamaOK, okayInner := nakamaInfo["ok"].(bool)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d nakama ok", instance)
			}
			nakamaResultCodef, okayInner := nakamaInfo["result_code"].(float64)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d result_code", instance)
			}
			nakamaResultCode := int(nakamaResultCodef)
			nakamaResultStr, okayInner := nakamaInfo["result_str"].(string)
			if !okayInner {
				return false, eris.Errorf("Failed to unmarshal deployment instance %d result_str", instance)
			}
			if region != currRegion {
				currRegion = region
				printer.Infof("• %s\n", currRegion)
			}
			printer.Infof("  %d) ", instanceNum)
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
			printer.Info("     ")
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
	return healthComplete, nil
}
func previewDeployment(fCtx ForgeContext, deployType string) error {
	deployURL := fmt.Sprintf("%s/api/organization/%s/project/%s/%s?preview=true",
		baseURL, fCtx.State.Organization.ID, fCtx.State.Project.ID, deployType)
	resultBytes, err := sendRequest(fCtx, http.MethodPost, deployURL, nil)
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
	printer.Headerln("   Basic Information   ")
	printer.Infof("Organization:    %s\n", response.Data.OrgName)
	printer.Infof("Org Slug:        %s\n", response.Data.OrgSlug)
	printer.Infof("Project:         %s\n", response.Data.ProjectName)
	printer.Infof("Project Slug:    %s\n", response.Data.ProjectSlug)

	printer.NewLine(1)
	printer.Headerln("     Configuration     ")
	printer.Infof("Executor:        %s\n", response.Data.ExecutorName)
	printer.Infof("Deployment Type: %s\n", response.Data.DeploymentType)
	printer.Infof("Tick Rate:       %d\n", response.Data.TickRate)

	printer.NewLine(1)
	printer.Headerln("  Deployment Regions  ")
	printer.Infof("%s\n", strings.Join(response.Data.Regions, ", "))

	return nil
}

// nolint: gocognit // this is a complex function but it does what it needs to do
func waitUntilDeploymentIsComplete(fCtx ForgeContext, env string, deployType string) error {
	ctx, cancel := context.WithCancel(fCtx.Context)
	defer cancel()

	// Spinner Setup
	spinnerExited := atomic.Bool{}
	var wg sync.WaitGroup
	wg.Add(1)

	spin := teaspinner.Spinner{
		Spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
		Cancel:  cancel,
	}
	spin.SetText(fmt.Sprintf("Waiting for %s to complete...", deployType))
	p := tea.NewProgram(&spin)

	// Run the spinner in a goroutine
	go func() {
		defer wg.Done()
		if _, err := p.Run(); err != nil {
			log.Error().Err(err).Msg("failed to run spinner")
			printer.Infoln(
				"Waiting for deployment to complete...",
			) // If spinner doesn't start, fallback to simple print.
		}
		spinnerExited.Store(true)
	}()

	// spinnnerCompleted will send a message to the spinner to stop and quit.
	spinnnerCompleted := func(didComplete bool) {
		if !spinnerExited.Load() {
			p.Send(teaspinner.LogMsg("spin: completed"))
			p.Send(tea.Quit())
			wg.Wait()
		}
		if didComplete {
			printer.Successln("Deployment completed!")
		} else {
			printer.Errorln("Deployment failed!")
		}
	}

	// Status Loop
	deployComplete := false
	for {
		select {
		case <-ctx.Done():
			spinnnerCompleted(false)
			return ctx.Err()
		case <-time.After(3 * time.Second):
			if !spinnerExited.Load() {
				switch {
				case !deployComplete:
					p.Send(teaspinner.LogMsg(fmt.Sprintf("Waiting for %s to complete...", deployType)))
				case deployType == "destroy":
					p.Send(teaspinner.LogMsg("Waiting for servers to be destroyed..."))
				default:
					p.Send(teaspinner.LogMsg("Waiting for servers to be healthy..."))
				}
			}

			deploys, err := getDeploymentStatus(fCtx)
			if err != nil || deploys == nil {
				continue
			}
			if deploy, exists := deploys[env]; exists {
				printDeploymentStatus(deploy)
				if shouldShowHealth(deploy) {
					deployComplete = true // this changes the status message for the spinner
					// just report health for the single environment
					healthComplete, err := getAndPrintHealth(fCtx, map[string]DeployInfo{
						env: deploy,
					})
					if err != nil || !healthComplete {
						continue
					}
				}
			}

			spinnnerCompleted(true)
			return nil
		}
	}
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

func printDeploymentStatus(deployInfo DeployInfo) {
	switch deployInfo.DeployStatus { //nolint:exhaustive // only success and failure have special messages
	case DeployStatusPassed:
		printer.Successf("%s\n", deployInfo.DeployDisplay)
	case DeployStatusFailed:
		printer.Errorf("%s\n", deployInfo.DeployDisplay)
	default:
		printer.Infof("🔄 %s\n", deployInfo.DeployDisplay)
	}
}

func shouldShowHealth(deployInfo DeployInfo) bool {
	switch deployInfo.DeployType {
	case DeploymentTypeDeploy:
		return deployInfo.DeployStatus == DeployStatusPassed
	case DeploymentTypeDestroy:
		return deployInfo.DeployStatus != DeployStatusPassed
	case DeploymentTypeReset:
		return true
	}
	return false
}

func envDisplayName(env string) string {
	switch env {
	case "dev":
		return "PREVIEW"
	case "prod":
		return "LIVE"
	default:
		return env
	}
}
