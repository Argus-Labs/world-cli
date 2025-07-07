package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/common/printer"
	teaspinner "pkg.world.dev/world-cli/tea/component/spinner"
)

func (h *Handler) Deployment(
	ctx context.Context,
	organizationID string,
	project models.Project,
	deployType string,
) error {
	if organizationID == "" {
		printNoSelectedOrganization()
		return nil
	}

	// Ensure organization is not nil before this call.
	if project.ID == "" {
		org, err := h.apiClient.GetOrganizationByID(ctx, organizationID)
		if err != nil {
			return eris.Wrap(err, "Failed on deployment to get selected organization")
		}

		printer.Infof("Deploy requires a project created in World Forge: %s\n", org.Name)

		pID, err := h.projectHandler.Create(ctx, org, models.CreateProjectFlags{})
		if err != nil {
			return eris.Wrap(err, "Failed on deployment to create project")
		}
		project.ID = pID.ID
	}

	// preview deployment
	err := h.previewDeployment(ctx, organizationID, project.ID, deployType)
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

	confirmation, err := h.inputHandler.Confirm(ctx, prompt, "n")
	if err != nil {
		return eris.Wrap(err, "Failed to prompt user")
	}

	if !confirmation {
		printer.Errorln("Deployment cancelled")
		printer.NewLine(1)
		return nil
	}

	if deployType == DeploymentTypeForceDeploy {
		deployType = "deploy?force=true"
	}

	err = h.apiClient.DeployProject(ctx, organizationID, project.ID, deployType)
	if err != nil {
		return eris.Wrap(err, "Failed to deploy project")
	}

	env := DeployEnvPreview
	if deployType == DeploymentTypePromote {
		env = DeployEnvLive
	}

	// wait until the deployment is complete
	err = h.waitUntilDeploymentIsComplete(ctx, project, env, deployType)
	if err != nil {
		printer.NewLine(1)
		printer.Successf("Your %s is being processed!\n\n", deployType)
		printer.Infof("To check the status of your %s, run:\n", deployType)
		printer.Infoln("  $ 'world status'")
	}

	return nil
}

func (h *Handler) previewDeployment(ctx context.Context, organizationID, projectID string, deployType string) error {
	response, err := h.apiClient.PreviewDeployment(ctx, organizationID, projectID, deployType)
	if err != nil {
		return eris.Wrap(err, "Failed to preview deployment")
	}

	printer.NewLine(1)
	printer.Headerln("   Basic Information   ")
	printer.Infof("Organization:    %s\n", response.OrgName)
	printer.Infof("Org Slug:        %s\n", response.OrgSlug)
	printer.Infof("Project:         %s\n", response.ProjectName)
	printer.Infof("Project Slug:    %s\n", response.ProjectSlug)

	printer.NewLine(1)
	printer.Headerln("     Configuration     ")
	printer.Infof("Executor:        %s\n", response.ExecutorName)
	printer.Infof("Deployment Type: %s\n", response.DeploymentType)
	printer.Infof("Tick Rate:       %d\n", response.TickRate)

	printer.NewLine(1)
	printer.Headerln("  Deployment Regions  ")
	printer.Infof("%s\n", strings.Join(response.Regions, ", "))

	return nil
}

// nolint: gocognit // this is a complex function but it does what it needs to do
func (h *Handler) waitUntilDeploymentIsComplete(
	ctx context.Context,
	project models.Project,
	env string,
	deployType string,
) error {
	ctx, cancel := context.WithCancel(ctx)
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

	// spinnerCompleted will send a message to the spinner to stop and quit.
	spinnerCompleted := func(didComplete bool) {
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
			spinnerCompleted(false)
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

			deploys, err := h.getDeploymentStatus(ctx, project)
			if err != nil || deploys == nil {
				continue
			}
			if deploy, exists := deploys[env]; exists {
				printDeploymentStatus(deploy)
				// if shouldShowHealth(deploy) {
				// 	deployComplete = true // this changes the status message for the spinner
				// 	// just report health for the single environment
				// 	healthComplete, err := h.getAndPrintHealth(ctx, project, map[string]DeployInfo{
				// 		env: deploy,
				// 	})
				// 	if err != nil || !healthComplete {
				// 		continue
				// 	}
				// }
			}

			spinnerCompleted(true)
			return nil
		}
	}
}

// Returns a map of environment names to boolean values indicating whether the environment was
// successfully deployed.
// nolint: gocognit // this is a complex function but it does what it needs to do
func (h *Handler) getDeploymentStatus(ctx context.Context, project models.Project) (map[string]DeployInfo, error) {
	result, err := h.apiClient.GetDeploymentStatus(ctx, project.ID)
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
		data, ok := val.(map[string]any)
		if !ok {
			return nil, eris.Errorf("Failed to unmarshal response for environment %s", env)
		}
		if data["project_id"] != project.ID {
			return nil, eris.Errorf("Deployment status does not match project id %s", project.ID)
		}

		executorID, ok := data["created_by"].(string)
		if !ok {
			return nil, eris.New("Failed to unmarshal deployment created_by")
		}
		executorName, ok := data["executor_name"].(string)
		if ok {
			executorID = executorName
		}
		executionTimeStr, ok := data["created_at"].(string)
		if !ok {
			return nil, eris.New("Failed to unmarshal deployment created_at")
		}

		// Parse the timestamp and format it as yyyy-mm-dd hh:mm timezone
		executionTime, err := time.Parse(time.RFC3339, executionTimeStr)
		if err != nil {
			return nil, eris.Wrap(err, "Failed to parse deployment created_at timestamp")
		}
		executionTimeStr = executionTime.Format("2006-01-02 15:04 MST")
		deploymentStatus, ok := data["deployment_status"].(string)
		if !ok {
			return nil, eris.New("Failed to unmarshal deployment deployment_status")
		}
		deployStatus[env] = DeployInfo{
			DeployStatus: DeployStatus(deploymentStatus),
			DeployDisplay: fmt.Sprintf(
				"Pod `%s` %s at %s by `%s`",
				env,
				deploymentStatus,
				executionTimeStr,
				executorID,
			),
		}
	}
	return deployStatus, nil
}

// nolint: gocognit, gocyclo, cyclop, funlen // this is a complex function but it does what it needs to do
func (h *Handler) getAndPrintHealth(
	ctx context.Context,
	project models.Project,
	deployInfo map[string]DeployInfo,
) (bool, error) {
	result, err := h.apiClient.GetHealthStatus(ctx, project.ID)
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

func printDeploymentStatus(deployInfo DeployInfo) {
	switch deployInfo.DeployStatus {
	case DeployStatusCreated:
		printer.Successf("%s\n", deployInfo.DeployDisplay)
	case DeployStatusRemoved:
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
		return deployInfo.DeployStatus == DeployStatusCreated
	case DeploymentTypeDestroy:
		return deployInfo.DeployStatus == DeployStatusRemoved
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

func printNoSelectedOrganization() {
	printer.NewLine(1)
	printer.Headerln("   No Organization Selected   ")
	printer.Infoln("You don't have any organization selected.")
	printer.Info("Use ")
	printer.Notification("'world organization switch'")
	printer.Infoln(" to select one!")
}
