package evm

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/teacmd"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the EVM base shard. Use --da-auth-token to pass in an auth token directly.",
	Long:  "Start the EVM base shard. Requires connection to celestia DA.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.GetConfig(cmd)
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

		err = dockerClient.Start(cmd.Context(), cfg, service.EVM)
		if err != nil {
			return fmt.Errorf("error starting %s docker container: %w", teacmd.DockerServiceEVM, err)
		}

		// Stop the DA service if it was started in dev mode
		if cfg.DevDA {
			err = dockerClient.Stop(cfg, service.CelestiaDevNet)
			if err != nil {
				return eris.Wrap(err, "Failed to stop DA service")
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

// validateDevDALayer starts a locally running version of the DA layer, and replaces the DA_AUTH_TOKEN configuration
// variable with the token from the locally running container.
func validateDevDALayer(ctx context.Context, cfg *config.Config, dockerClient *docker.Client) error {
	cfg.Build = true
	cfg.Debug = false
	cfg.Detach = true
	cfg.Timeout = -1
	logger.Println("starting DA docker service for dev mode...")
	if err := dockerClient.Start(ctx, cfg, service.CelestiaDevNet); err != nil {
		return fmt.Errorf("error starting %s docker container: %w", daService, err)
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
		errs = append(errs, fmt.Errorf("missing %q", env))
	}
	if len(errs) > 0 {
		// Prepend an error describing the overall problem
		errs = append([]error{
			errors.New("the [evm] section of your config is missing some required variables"),
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
	fmt.Println("Getting DA token")

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
		return "", errors.New("got empty DA token")
	}
	return token, nil
}
