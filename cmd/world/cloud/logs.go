package cloud

import (
	"context"
	"net/http"
	"strconv"

	"connectrpc.com/connect"
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
	"pkg.world.dev/world-cli/common/printer"
	logsv1 "pkg.world.dev/world-cli/gen/logs/v1"
	"pkg.world.dev/world-cli/gen/logs/v1/logsv1connect"
)

func (h *Handler) TailLogs(ctx context.Context, region string, env string) error {
	params, err := h.getLogParams(ctx, region, env)
	if err != nil {
		return err
	}

	if err := h.confirmLogParams(ctx, params); err != nil {
		return err
	}

	client, req := h.createLogsClient(params)
	return streamLogs(ctx, client, req)
}

func (h *Handler) getLogParams(ctx context.Context, region string, env string) (*logParams, error) {
	organization, err := h.apiClient.GetOrganizationByID(ctx, h.configService.GetConfig().OrganizationID)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get selected organization")
	}

	project, err := h.apiClient.GetProjectByID(ctx,
		h.configService.GetConfig().OrganizationID, h.configService.GetConfig().ProjectID)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get selected project")
	}

	if region == "" {
		region, err = h.selectRegion(ctx, project)
		if err != nil {
			return nil, err
		}
	}

	if env == "" {
		envs, err := h.getListOfEnvironments(ctx, project)
		if err != nil {
			return nil, err
		}
		if len(envs) == 0 {
			return nil, eris.New("No environments found")
		}

		env, err = h.selectEnvironment(ctx, envs)
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

func (h *Handler) selectRegion(ctx context.Context, project models.Project) (string, error) {
	// If there is only one region, return it
	if len(project.Config.Region) == 1 {
		return project.Config.Region[0], nil
	}

	// If there are multiple regions, print them and let the user choose
	printer.Infoln("Available regions:")
	for i, region := range project.Config.Region {
		printer.Infof("%d. %s\n", i+1, region)
	}

	inputStr, err := h.inputHandler.Prompt(ctx, "Choose a region", "1")
	if err != nil {
		return "", eris.Wrap(err, "Failed to prompt for region")
	}

	inputInt, err := strconv.Atoi(inputStr)
	if err != nil {
		return "", eris.New("Invalid region")
	}
	if inputInt < 1 || inputInt > len(project.Config.Region) {
		return "", eris.New("Invalid region")
	}

	return project.Config.Region[inputInt-1], nil
}

func (h *Handler) selectEnvironment(ctx context.Context, availableEnvs []string) (string, error) {
	// If there is only one environment, return it
	if len(availableEnvs) == 1 {
		return availableEnvs[0], nil
	}

	// If there are multiple environments, print them and let the user choose
	printer.NewLine(1)
	printer.Infoln("Available environments:")
	printer.Infof("%d. %s\n", 1, "PREVIEW")
	printer.Infof("%d. %s\n", 2, "LIVE")

	inputStr, err := h.inputHandler.Prompt(ctx, "Choose an environment", "1")
	if err != nil {
		return "", eris.Wrap(err, "Failed to prompt for environment")
	}
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

func (h *Handler) getListOfEnvironments(ctx context.Context, project models.Project) ([]string, error) {
	envMap, err := h.apiClient.GetDeploymentHealthStatus(ctx, project.ID)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get deployment health status")
	}

	// Create a list of environments that are online
	envs := []string{}
	for env, envStatus := range envMap {
		if envStatus.OK {
			envs = append(envs, env)
		}
	}

	return envs, nil
}

func (h *Handler) confirmLogParams(ctx context.Context, params *logParams) error {
	printer.NewLine(1)
	printer.Infof("Showing logs for '%s-%s-cardinal' in '%s-%s'\n",
		params.organization.Slug, params.project.Slug, params.env, params.region)
	printer.Info("(Press Enter to continue | Ctrl+C to cancel/exit)")
	inputStr, err := h.inputHandler.Prompt(ctx, "", "")
	if err != nil {
		return eris.Wrap(err, "Failed to prompt for confirmation")
	}
	if inputStr != "" {
		return eris.New("Operation cancelled by user")
	}
	return nil
}

func (h *Handler) createLogsClient(params *logParams) (
	logsv1connect.LogsServiceClient,
	*connect.Request[logsv1.GetLogsRequest],
) {
	client := logsv1connect.NewLogsServiceClient(
		http.DefaultClient,
		// TODO: Remove this once we have a proper RPC client
		h.apiClient.GetRPCBaseURL(),
	)

	req := connect.NewRequest(&logsv1.GetLogsRequest{
		OrganizationSlug: params.organization.Slug,
		ProjectSlug:      params.project.Slug,
		Env:              params.env,
		Region:           params.region,
	})

	token := h.configService.GetConfig().Credential.Token
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
			printer.Infoln(stream.Msg().GetLog())
		}
	}
}
