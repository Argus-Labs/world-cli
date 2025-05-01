package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rotisserie/eris"
	teaspinner "pkg.world.dev/world-cli/tea/component/spinner"
)

const (
	DeploymentTypeDeploy   = "deploy"
	DeploymentTypeDestroy  = "destroy"
	DeploymentTypeReset    = "reset"
	DeploymentStatusFailed = "failed"
	DeploymentStatusPassed = "passed"
)

var (
	statusFailRegEx = regexp.MustCompile(`[^a-zA-Z0-9\. ]+`)
	processTitle    = map[string]string{
		DeploymentTypeDeploy:  "Deploying",
		DeploymentTypeDestroy: "Destroying",
		DeploymentTypeReset:   "Resetting",
	}
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

type deploymentStatus struct {
	ProjectName      string
	ProjectSlug      string
	Repository       string
	Environments     map[string]map[string]any
	ShouldShowHealth map[string]bool
}

type healthStatus struct {
	Environments map[string]map[string]any
}

// Deployment a project.
func deployment(ctx context.Context, deployType string) error {
	globalConfig, err := GetCurrentConfigWithContext(ctx)
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
	fmt.Println("\n   Confirm Deployment")
	fmt.Println("========================")
	fmt.Println("\nReview the deployment details above.")
	prompt := fmt.Sprintf("\nDo you want to proceed with the %s? (Y/n): ", processTitle[deployType])

	confirmation := getInput(prompt, "n")

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

	if err := monitorDeployment(ctx, deployType, organizationID, projectID); err != nil {
		return eris.Wrap(err, fmt.Sprintf("Failed to %s project", deployType))
	}

	return nil
}

func monitorDeployment(ctx context.Context, deployType string, organizationID string, projectID string) error {
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
	spin.SetText(fmt.Sprintf("%s project...", processTitle[deployType]))
	p := tea.NewProgram(spin)

	var lastStatus *deploymentStatus

	// Start spinner
	go func() {
		defer wg.Done()
		if _, err := p.Run(); err != nil {
			fmt.Printf("%s project...\n", processTitle[deployType]) // If the spinner doesn't start, fallback to print
		}
		spinnerExited.Store(true)
	}()

	spinnerCompleted := func(didDeploy bool, showHealth bool) {
		if !spinnerExited.Load() {
			p.Send(teaspinner.LogMsg("spin: completed"))
			p.Send(tea.Quit())
			wg.Wait()
		}
		if didDeploy {
			fmt.Printf("✅ Deployment complete!\n")
		} else {
			fmt.Printf("❌ Deployment failed!\n")
		}
		if showHealth {
			// Start health check spinner
			monitorHealth(ctx, projectID)
		}
	}

	// Monitor deployment in background
	for {
		select {
		case <-ctx.Done():
			spinnerCompleted(false, false)
			return ctx.Err()
		case <-time.After(3 * time.Second):
			if !spinnerExited.Load() {
				p.Send(teaspinner.LogMsg("Checking deployment status..."))
			}
			status, err := collectDeploymentStatus(ctx, projectID)
			if err != nil {
				spinnerCompleted(false, status.ShouldShowHealth[])
				return err
			}

			if lastStatus == nil || deploymentStatusChanged(lastStatus, status) {
				statusUpdate := getDeploymentStatusSummary(status)
				p.Send(teaspinner.LogMsg(statusUpdate))
				lastStatus = status
			}

			// Check if deployment is complete
			if isDeploymentComplete(status) {
				spinnerCompleted(true)
				return nil
			}
		}
	}
}

func monitorHealth(ctx context.Context, projectID string) error {
	s := teaSpinner.NewSpinner()
	s.Message = "Checking deployment health..."

	done := make(chan bool)
	var lastHealth *healthStatus

	// Start spinner
	go func() {
		if err := s.Run(); err != nil {
			fmt.Printf("Error running spinner: %v\n", err)
		}
	}()

	// Monitor health in background
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.StopMessage = "Health monitoring cancelled"
				s.Stop()
				done <- true
				return
			case <-ticker.C:
				health, err := collectHealthStatus(ctx, projectID, depStatus.ShouldShowHealth)
				if err != nil {
					s.StopMessage = fmt.Sprintf("Health check error: %s", err)
					s.Stop()
					done <- true
					return
				}

				if lastHealth == nil || healthStatusChanged(lastHealth, health) {
					s.Message = getHealthStatusSummary(health)
					lastHealth = health
				}

				// Check if all health checks pass
				if isHealthCheckComplete(health) {
					s.StopMessage = "All services healthy!"
					s.Stop()

					// Display full status at the end
					displayDeploymentStatus(depStatus)
					displayHealthStatus(health, depStatus.ShouldShowHealth)

					done <- true
					return
				}
			}
		}
	}()

	<-done
	return nil
}

func collectDeploymentStatus(ctx context.Context, projectID string) (*deploymentStatus, error) {
	// Get project details
	prj, err := getSelectedProject(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get project details")
	}

	statusURL := fmt.Sprintf("%s/api/deployment/%s", baseURL, projectID)
	result, err := sendRequest(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get deployment status")
	}

	var response map[string]any
	err = json.Unmarshal(result, &response)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to unmarshal deployment response")
	}

	status := &deploymentStatus{
		ProjectName:      prj.Name,
		ProjectSlug:      prj.Slug,
		Repository:       prj.RepoURL,
		Environments:     make(map[string]map[string]any),
		ShouldShowHealth: make(map[string]bool),
	}

	if response["data"] != nil {
		envMap, ok := response["data"].(map[string]any)
		if !ok {
			return nil, eris.New("Failed to unmarshal deployment data")
		}

		for env, val := range envMap {
			data, ok := val.(map[string]any)
			if !ok {
				return nil, eris.Errorf("Failed to unmarshal response for environment %s", env)
			}

			status.Environments[env] = data

			// Determine if we should show health for this environment
			buildState, ok := data["build_state"].(string)
			if !ok {
				return nil, eris.New("Failed to unmarshal deployment build_state")
			}

			deployType, ok := data["type"].(string)
			if !ok {
				return nil, eris.New("Failed to unmarshal deployment type")
			}

			status.ShouldShowHealth[env] = false
			switch deployType {
			case DeploymentTypeDeploy:
				if buildState == DeploymentStatusPassed {
					status.ShouldShowHealth[env] = true
				}
			case DeploymentTypeDestroy:
				if buildState == DeploymentStatusFailed {
					status.ShouldShowHealth[env] = true
				}
			case DeploymentTypeReset:
				if buildState == DeploymentStatusPassed || buildState == DeploymentStatusFailed {
					status.ShouldShowHealth[env] = true
				}
			}
		}
	}

	return status, nil
}

func collectHealthStatus(ctx context.Context, projectID string, shouldShowHealth map[string]bool) (*healthStatus, error) {
	healthURL := fmt.Sprintf("%s/api/health/%s", baseURL, projectID)
	result, err := sendRequest(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get health")
	}

	var response map[string]any
	err = json.Unmarshal(result, &response)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to unmarshal health response")
	}

	health := &healthStatus{
		Environments: make(map[string]map[string]any),
	}

	if response["data"] == nil {
		return health, nil
	}

	envMap, ok := response["data"].(map[string]any)
	if !ok {
		return nil, eris.New("Failed to unmarshal health data")
	}

	for env, val := range envMap {
		if !shouldShowHealth[env] {
			continue
		}

		data, ok := val.(map[string]any)
		if !ok {
			return nil, eris.Errorf("Failed to unmarshal response for environment %s", env)
		}

		health.Environments[env] = data
	}

	return health, nil
}

func displayDeploymentStatus(status *deploymentStatus) {
	fmt.Println(" Deployment Status")
	fmt.Println("-------------------")
	fmt.Printf("Project:      %s\n", status.ProjectName)
	fmt.Printf("Project Slug: %s\n", status.ProjectSlug)
	fmt.Printf("Repository:   %s\n", status.Repository)

	if len(status.Environments) == 0 {
		fmt.Printf("\n** Project has not been deployed **\n")
		return
	}

	for env, data := range status.Environments {
		deployType, _ := data["type"].(string)
		executorID, _ := data["executor_id"].(string)
		executorName, ok := data["executor_name"].(string)
		if ok {
			executorID = executorName
		}

		executionTimeStr, _ := data["execution_time"].(string)
		dt, _ := time.Parse(time.RFC3339, executionTimeStr)
		buildState, _ := data["build_state"].(string)

		switch deployType {
		case DeploymentTypeDeploy:
			bnf, _ := data["build_number"].(float64)
			buildNumber := int(bnf)

			buildStartTimeStr, _ := data["build_start_time"].(string)
			bst, _ := time.Parse(time.RFC3339, buildStartTimeStr)
			if bst.Before(dt) {
				bst = dt // we don't have a real build start time yet because build kite hasn't run yet
			}

			buildEndTimeStr, ok := data["build_end_time"].(string)
			if !ok {
				buildEndTimeStr = buildStartTimeStr
			}
			bet, _ := time.Parse(time.RFC3339, buildEndTimeStr)
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
			default:
				fmt.Printf("🔄 Destroy:   [%s] started %s (%s ago) by %s - %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822),
					formattedDuration(time.Since(dt)), executorID, buildState)
			}
		case DeploymentTypeReset:
			switch buildState {
			case DeploymentStatusPassed:
				fmt.Printf("✅ Reset:     [%s] on %s by %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822), executorID)
			case DeploymentStatusFailed:
				fmt.Printf("❌ Reset:     [%s] failed on %s by %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822), executorID)
			default:
				fmt.Printf("🔄 Reset:     [%s] started %s (%s ago) by %s - %s\n",
					strings.ToUpper(env), dt.Format(time.RFC822),
					formattedDuration(time.Since(dt)), executorID, buildState)
			}
		}
	}
}

func displayHealthStatus(health *healthStatus, shouldShowHealth map[string]bool) {
	for env, data := range health.Environments {
		if !shouldShowHealth[env] {
			continue
		}

		instances, ok := data["deployed_instances"].([]any)
		if !ok {
			continue
		}

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
			continue
		}

		fmt.Printf("(%d deployed instances)\n", len(instances))
		currRegion := ""

		for _, instance := range instances {
			info, _ := instance.(map[string]any)
			region, _ := info["region"].(string)
			instancef, _ := info["instance"].(float64)
			instanceNum := int(instancef)

			cardinalInfo, _ := info["cardinal"].(map[string]any)
			nakamaInfo, _ := info["nakama"].(map[string]any)

			cardinalURL, _ := cardinalInfo["url"].(string)
			cardinalHost := strings.Split(cardinalURL, "/")[2]
			cardinalOK, _ := cardinalInfo["ok"].(bool)
			cardinalResultCodef, _ := cardinalInfo["result_code"].(float64)
			cardinalResultCode := int(cardinalResultCodef)
			cardinalResultStr, _ := cardinalInfo["result_str"].(string)

			nakamaURL, _ := nakamaInfo["url"].(string)
			nakamaHost := strings.Split(nakamaURL, "/")[2]
			nakamaOK, _ := nakamaInfo["ok"].(bool)
			nakamaResultCodef, _ := nakamaInfo["result_code"].(float64)
			nakamaResultCode := int(nakamaResultCodef)
			nakamaResultStr, _ := nakamaInfo["result_str"].(string)

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
}

func getDeploymentStatusSummary(status *deploymentStatus) string {
	for env, data := range status.Environments {
		deployType, _ := data["type"].(string)
		buildState, _ := data["build_state"].(string)

		return fmt.Sprintf("%s [%s]: %s", processTitle[deployType],
			strings.ToUpper(env), buildState)
	}
	return "Waiting for deployment status..."
}

func getHealthStatusSummary(health *healthStatus) string {
	healthyCounts := 0
	totalCounts := 0

	for env, data := range health.Environments {
		instances, ok := data["deployed_instances"].([]any)
		if ok {
			totalCounts += len(instances)
			for _, instance := range instances {
				info, _ := instance.(map[string]any)
				cardinalInfo, _ := info["cardinal"].(map[string]any)
				nakamaInfo, _ := info["nakama"].(map[string]any)

				cardinalOK, _ := cardinalInfo["ok"].(bool)
				nakamaOK, _ := nakamaInfo["ok"].(bool)

				if cardinalOK && nakamaOK {
					healthyCounts++
				}
			}
		}

		return fmt.Sprintf("Health check [%s]: %d/%d services healthy",
			strings.ToUpper(env), healthyCounts, totalCounts*2) // *2 for Cardinal+Nakama
	}

	return "Checking deployment health..."
}

func deploymentStatusChanged(old, new *deploymentStatus) bool {
	if len(old.Environments) != len(new.Environments) {
		return true
	}

	for env, oldData := range old.Environments {
		newData, exists := new.Environments[env]
		if !exists {
			return true
		}

		oldState, _ := oldData["build_state"].(string)
		newState, _ := newData["build_state"].(string)

		if oldState != newState {
			return true
		}
	}

	return false
}

func healthStatusChanged(old, new *healthStatus) bool {
	if len(old.Environments) != len(new.Environments) {
		return true
	}

	for env, oldData := range old.Environments {
		newData, exists := new.Environments[env]
		if !exists {
			return true
		}

		oldOK, _ := oldData["ok"].(bool)
		newOK, _ := newData["ok"].(bool)

		if oldOK != newOK {
			return true
		}
	}

	return false
}

func isDeploymentComplete(status *deploymentStatus) bool {
	for _, data := range status.Environments {
		buildState, _ := data["build_state"].(string)
		if buildState != DeploymentStatusPassed && buildState != DeploymentStatusFailed {
			return false
		}
	}
	return len(status.Environments) > 0
}

func isHealthCheckComplete(health *healthStatus) bool {
	for _, data := range health.Environments {
		ok, _ := data["ok"].(bool)
		if !ok {
			return false
		}
	}
	return len(health.Environments) > 0
}

func status(ctx context.Context) error {
	globalConfig, err := GetCurrentConfigWithContext(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get global config")
	}
	projectID := globalConfig.ProjectID
	if projectID == "" {
		printNoSelectedProject()
		return nil
	}

	status, err := collectDeploymentStatus(ctx, projectID)
	if err != nil {
		return eris.Wrap(err, "Failed to collect deployment status")
	}

	displayDeploymentStatus(status)

	checkHealth := false
	for _, shouldShow := range status.ShouldShowHealth {
		if shouldShow {
			checkHealth = true
			break
		}
	}

	if checkHealth {
		health, err := collectHealthStatus(ctx, projectID, status.ShouldShowHealth)
		if err != nil {
			return eris.Wrap(err, "Failed to collect health status")
		}

		displayHealthStatus(health, status.ShouldShowHealth)
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
	fmt.Println("\n   Deployment Preview")
	fmt.Println("========================")
	fmt.Println("\n   Basic Information")
	fmt.Println("------------------------")
	fmt.Printf("Organization:    %s\n", response.Data.OrgName)
	fmt.Printf("Org Slug:        %s\n", response.Data.OrgSlug)
	fmt.Printf("Project:         %s\n", response.Data.ProjectName)
	fmt.Printf("Project Slug:    %s\n", response.Data.ProjectSlug)

	fmt.Println("\n     Configuration")
	fmt.Println("------------------------")
	fmt.Printf("Executor:        %s\n", response.Data.ExecutorName)
	fmt.Printf("Deployment Type: %s\n", response.Data.DeploymentType)
	fmt.Printf("Tick Rate:       %d\n", response.Data.TickRate)

	fmt.Println("\n  Deployment Regions")
	fmt.Println("------------------------")
	fmt.Printf("%s\n", strings.Join(response.Data.Regions, ", "))

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
