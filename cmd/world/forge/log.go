package forge

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"connectrpc.com/connect"
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/printer"
	logsv1 "pkg.world.dev/world-cli/gen/logs/v1"
	"pkg.world.dev/world-cli/gen/logs/v1/logsv1connect"
)

type logParams struct {
	organization organization
	project      project
	region       string
	env          string
}

func getLogParams(fCtx ForgeContext, region string, env string) (*logParams, error) {
	organization, err := getSelectedOrganization(fCtx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get selected organization")
	}

	project, err := getSelectedProject(fCtx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get selected project")
	}

	if region == "" {
		region, err = selectRegion(project)
		if err != nil {
			return nil, err
		}
	}

	if env == "" {
		envs, err := getListOfEnvironments(fCtx, project)
		if err != nil {
			return nil, err
		}
		if len(envs) == 0 {
			return nil, eris.New("No environments found")
		}

		env, err = selectEnvironment(envs)
		if err != nil {
			return nil, err
		}
	}

	return &logParams{
		organization: organization,
		project:      project,
		region:       region,
		env:          env,
	}, nil
}

func selectRegion(project project) (string, error) {
	// If there is only one region, return it
	if len(project.Config.Region) == 1 {
		return project.Config.Region[0], nil
	}

	// If there are multiple regions, print them and let the user choose
	printer.Infoln("Available regions:")
	for i, region := range project.Config.Region {
		printer.Infof("%d. %s\n", i+1, region)
	}

	inputStr := getInput("Choose a region", "1")
	inputInt, err := strconv.Atoi(inputStr)
	if err != nil {
		return "", eris.New("Invalid region")
	}
	if inputInt < 1 || inputInt > len(project.Config.Region) {
		return "", eris.New("Invalid region")
	}

	return project.Config.Region[inputInt-1], nil
}

func selectEnvironment(availableEnvs []string) (string, error) {
	// If there is only one environment, return it
	if len(availableEnvs) == 1 {
		return availableEnvs[0], nil
	}

	// If there are multiple environments, print them and let the user choose
	printer.NewLine(1)
	printer.Infoln("Available environments:")
	printer.Infof("%d. %s\n", 1, "PREVIEW")
	printer.Infof("%d. %s\n", 2, "LIVE")

	inputStr := getInput("Choose an environment", "1")
	inputInt, err := strconv.Atoi(inputStr)
	if err != nil {
		return "", eris.New("Invalid environment")
	}
	if inputInt < 1 || inputInt > 2 {
		return "", eris.New("Invalid environment")
	}

	if inputInt == 2 {
		return DeployEnvLive, nil
	}
	return DeployEnvPreview, nil
}

func getListOfEnvironments(fCtx ForgeContext, project project) ([]string, error) {
	// Get the list of environments from the health check endpoint
	statusURL := fmt.Sprintf("%s/api/health/%s", baseURL, project.ID)
	result, err := sendRequest(fCtx, http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get deployment status")
	}

	// Parse the response into a map of environment names to their status
	type DeploymentHealthCheckResult struct {
		OK      bool `json:"ok"`
		Offline bool `json:"offline"`
	}
	envMap, err := parseResponse[map[string]DeploymentHealthCheckResult](result)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse deployment status")
	}

	// Create a list of environments that are online
	envs := []string{}
	for env, envStatus := range *envMap {
		if envStatus.OK {
			envs = append(envs, env)
		}
	}

	return envs, nil
}

func confirmLogParams(params *logParams) error {
	printer.NewLine(1)
	printer.Infof("Showing logs for '%s-%s-cardinal' in '%s-%s'\n",
		params.organization.Slug, params.project.Slug, params.env, params.region)
	printer.Info("(Press Enter to continue | Ctrl+C to cancel/exit)")
	inputStr := getInput("", "")
	if inputStr != "" {
		return eris.New("Operation cancelled by user")
	}
	return nil
}

func createLogsClient(fCtx ForgeContext, params *logParams) (
	logsv1connect.LogsServiceClient,
	*connect.Request[logsv1.GetLogsRequest],
) {
	client := logsv1connect.NewLogsServiceClient(
		http.DefaultClient,
		rpcURL,
	)

	req := connect.NewRequest(&logsv1.GetLogsRequest{
		OrganizationSlug: params.organization.Slug,
		ProjectSlug:      params.project.Slug,
		Env:              params.env,
		Region:           params.region,
	})

	token := fCtx.Config.Credential.Token
	req.Header().Set("Authorization", token)

	return client, req
}

func streamLogs(ctx context.Context,
	client logsv1connect.LogsServiceClient,
	req *connect.Request[logsv1.GetLogsRequest],
) error {
	stream, err := client.GetLogs(ctx, req)
	if err != nil {
		return eris.Wrap(err, "Failed to connect to logs service")
	}

	for {
		select {
		case <-ctx.Done():
			err := stream.Close()
			if err != nil {
				return eris.Wrap(err, "Failed to close log stream")
			}
			return nil
		default:
			ok := stream.Receive()
			if !ok {
				if err := stream.Err(); err != nil {
					return eris.Wrap(err, "Failed to get logs")
				}
				// Stream ended normally
				return nil
			}
			log.Println(stream.Msg().GetLog())
		}
	}
}

func tailLogs(fCtx ForgeContext, region string, env string) error {
	params, err := getLogParams(fCtx, region, env)
	if err != nil {
		return err
	}

	if err := confirmLogParams(params); err != nil {
		return err
	}

	client, req := createLogsClient(fCtx, params)
	return streamLogs(fCtx.Context, client, req)
}
