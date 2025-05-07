package evm

import (
	"context"
	"errors"
	"net"

	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/common/teacmd"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Launch your EVM blockchain environment",
	Long: `Start the EVM base shard with all required services for blockchain development.

This command initializes your Ethereum Virtual Machine environment, connecting it to 
the necessary data availability layer (Celestia). Perfect for developing and testing
smart contracts and blockchain applications.

Use --da-auth-token to pass in an authentication token directly, or --use-dev-da
to run with a local development data availability layer.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.GetConfig()
		if err != nil {
			return err
		}

		// Create docker client
		dockerClient, err := docker.NewClient(cfg)
		if err != nil {
			return err
		}
		defer dockerClient.Close()

		if err = validateDALayer(cmd, cfg, dockerClient); err != nil {
			return err
		}

		daToken, err := cmd.Flags().GetString(FlagDAAuthToken)
		if err != nil {
			return err
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
			return eris.Wrapf(err, "error starting %s docker container", teacmd.DockerServiceEVM)
		}

		// Stop the DA service if it was started in dev mode
		if cfg.DevDA {
			// using context background because cmd.Context() is already done
			err = dockerClient.Stop(context.Background(), service.CelestiaDevNet)
			if err != nil {
				return eris.Wrap(err, "Failed to stop DA service")
			}
		}
		return nil
	},
}

func Init() {
	startCmd.Flags().String(FlagDAAuthToken, "",
		"DA Auth Token that allows the rollup to communicate with the Celestia client.")
	startCmd.Flags().Bool(FlagUseDevDA, false, "Use a locally running DA layer")

	registerCommands()
}

// validateDevDALayer starts a locally running version of the DA layer, and replaces the DA_AUTH_TOKEN configuration
// variable with the token from the locally running container.
func validateDevDALayer(ctx context.Context, cfg *config.Config, dockerClient *docker.Client) error {
	cfg.Build = true
	cfg.Debug = false
	cfg.Detach = true
	cfg.Timeout = -1
	logger.Println("starting DA docker service for dev mode...")
	if err := dockerClient.Start(ctx, service.CelestiaDevNet); err != nil {
		return eris.Wrapf(err, "error starting %s docker container", daService)
	}
	logger.Println("started DA service...")

	daToken, err := getDAToken(ctx, cfg, dockerClient)
	if err != nil {
		return err
	}
	envOverrides := map[string]string{
		EnvDAAuthToken:   daToken,
		EnvDABaseURL:     net.JoinHostPort(string(daService), "26658"),
		EnvDANamespaceID: "67480c4a88c4d12935d4",
	}
	for key, value := range envOverrides {
		logger.Printf("overriding config value %q to %q\n", key, value)
		cfg.DockerEnv[key] = value
	}
	return nil
}

// validateProdDALayer assumes the DA layer is running somewhere else and validates the required world.toml
// variables are non-empty.
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
		errs = append(errs, eris.Errorf("missing %q", env))
	}
	if len(errs) > 0 {
		// Prepend an error describing the overall problem
		errs = append([]error{
			eris.New("the [evm] section of your config is missing some required variables"),
		}, errs...)
		return errors.Join(errs...)
	}
	return nil
}

func validateDALayer(cmd *cobra.Command, cfg *config.Config, dockerClient *docker.Client) error {
	devDA, err := cmd.Flags().GetBool(FlagUseDevDA)
	if err != nil {
		return err
	}
	if devDA {
		cfg.DevDA = true
		return validateDevDALayer(cmd.Context(), cfg, dockerClient)
	}
	return validateProdDALayer(cfg)
}

func getDAToken(ctx context.Context, cfg *config.Config, dockerClient *docker.Client) (string, error) {
	printer.Infoln("Getting DA token")

	containerName := service.CelestiaDevNet(cfg)

	_, err := dockerClient.Exec(ctx, containerName.Name,
		[]string{
			"mkdir",
			"-p",
			"/home/celestia/bridge/keys",
		})
	if err != nil {
		return "", eris.Wrap(err, "Failed to create keys directory")
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
		return "", eris.Wrapf(err, "Failed to get DA token")
	}

	if token == "" {
		return "", eris.New("got empty DA token")
	}
	return token, nil
}
