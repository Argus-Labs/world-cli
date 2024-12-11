package evm

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"

	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/config"
	"pkg.world.dev/world-cli/errors"
	"pkg.world.dev/world-cli/infrastructure/docker"
	"pkg.world.dev/world-cli/infrastructure/docker/service"
	"pkg.world.dev/world-cli/logging"
	"pkg.world.dev/world-cli/ui/commands"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the EVM base shard. Use --da-auth-token to pass in an auth token directly.",
	Long:  "Start the EVM base shard. Requires connection to celestia DA.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.GetConfig()
		if err != nil {
			return errors.WrapIf(err, "getting config")
		}

		// Create docker client
		dockerClient, err := docker.NewClient(cfg)
		if err != nil {
			return errors.WrapIf(err, "creating docker client")
		}
		defer dockerClient.Close()

		if err = validateDALayer(cmd, cfg, dockerClient); err != nil {
			return errors.WrapIf(err, "validating DA layer")
		}

		daToken, err := cmd.Flags().GetString(FlagDAAuthToken)
		if err != nil {
			return errors.WrapIf(err, "getting DA auth token flag")
		}
		if daToken != "" {
			cfg.DockerEnv[EnvDAAuthToken] = daToken
		}

		cfg.Build = true
		cfg.Debug = false
		cfg.Detach = false
		cfg.Timeout = 0

		err = dockerClient.Start(cmd.Context(), service.EVM)
		if err != nil {
			return errors.WrapIf(err, fmt.Sprintf("starting %s docker container", commands.DockerServiceEVM))
		}

		// Stop the DA service if it was started in dev mode
		if cfg.DevDA {
			// using context background because cmd.Context() is already done
			err = dockerClient.Stop(context.Background(), service.CelestiaDevNet)
			if err != nil {
				return errors.WrapIf(err, "stopping DA service")
			}
		}
		return nil
	},
}

func init() {
	startCmd.Flags().String(FlagDAAuthToken, "",
		"DA Auth Token that allows the rollup to communicate with the Celestia client.")
	startCmd.Flags().Bool(FlagUseDevDA, false, "Use a locally running DA layer")
}

func validateDevDALayer(ctx context.Context, cfg *config.Config, dockerClient *docker.Client) error {
	cfg.Build = true
	cfg.Debug = false
	cfg.Detach = true
	cfg.Timeout = -1
	logging.Println("starting DA docker service for dev mode...")
	if err := dockerClient.Start(ctx, service.CelestiaDevNet); err != nil {
		return errors.WrapIf(err, fmt.Sprintf("starting %s docker container", daService))
	}
	logging.Println("started DA service...")

	daToken, err := getDAToken(ctx, cfg, dockerClient)
	if err != nil {
		return errors.WrapIf(err, "getting DA token")
	}
	envOverrides := map[string]string{
		EnvDAAuthToken:   daToken,
		EnvDABaseURL:     net.JoinHostPort(string(daService), "26658"),
		EnvDANamespaceID: "67480c4a88c4d12935d4",
	}
	for key, value := range envOverrides {
		logging.Printf("overriding config value %q to %q\n", key, value)
		cfg.DockerEnv[key] = value
	}
	return nil
}

func validateProdDALayer(cfg *config.Config) error {
	requiredEnvVariables := []string{
		EnvDAAuthToken,
		EnvDABaseURL,
		EnvDANamespaceID,
	}
	errs := make([]error, 0)
	for _, env := range requiredEnvVariables {
		if len(cfg.DockerEnv[env]) > 0 {
			continue
		}
		errs = append(errs, errors.Errorf("missing required environment variable %q", env))
	}
	if len(errs) > 0 {
		// Prepend an error describing the overall problem
		errs = append([]error{
			errors.ErrInvalidConfig,
		}, errs...)
		return stderrors.Join(errs...)
	}
	return nil
}

func validateDALayer(cmd *cobra.Command, cfg *config.Config, dockerClient *docker.Client) error {
	devDA, err := cmd.Flags().GetBool(FlagUseDevDA)
	if err != nil {
		return errors.WrapIf(err, "getting use dev DA flag")
	}
	if devDA {
		cfg.DevDA = true
		return validateDevDALayer(cmd.Context(), cfg, dockerClient)
	}
	return validateProdDALayer(cfg)
}

func getDAToken(ctx context.Context, cfg *config.Config, dockerClient *docker.Client) (string, error) {
	fmt.Println("Getting DA token")

	containerName := service.CelestiaDevNet(cfg)

	_, err := dockerClient.Exec(ctx, containerName.Name,
		[]string{
			"mkdir",
			"-p",
			"/home/celestia/bridge/keys",
		})
	if err != nil {
		return "", errors.WrapIf(err, "creating keys directory")
	}

	token, err := dockerClient.Exec(ctx, containerName.Name,
		[]string{
			"celestia",
			"bridge",
			"--node.store",
			"/home/celestia/bridge/",
			"auth",
			"admin",
		})

	if err != nil {
		return "", errors.WrapIf(err, "executing DA token command")
	}

	if token == "" {
		return "", errors.Errorf("received empty DA token from celestia bridge")
	}
	return token, nil
}
