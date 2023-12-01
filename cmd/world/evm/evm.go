package evm

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/tea_cmd"
)

func EVMCmds() *cobra.Command {
	evmRootCmd := &cobra.Command{
		Use:   "evm",
		Short: "EVM base shard commands.",
		Long:  "Commands for provisioning the EVM Base Shard.",
	}
	evmRootCmd.AddGroup(&cobra.Group{
		ID:    "EVM",
		Title: "EVM Base Shard Commands",
	})
	evmRootCmd.AddCommand(
		StartEVM(),
		StopAll(),
	)
	return evmRootCmd
}

const (
	FlagUseDevDA     = "dev"
	FlagDAAuthToken  = "da-auth-token"
	EnvDAAuthToken   = "DA_AUTH_TOKEN"
	EnvDABaseURL     = "DA_BASE_URL"
	EnvDANamespaceID = "DA_NAMESPACE_ID"

	daService = tea_cmd.DockerServiceDA
)

var (
	// Docker compose seems to replace the hyphen with an underscore. This could be properly fixed by removing the hyphen
	// from celesta-devnet, or by investigating the docker compose documentation.
	daContainer = strings.ReplaceAll(string(daService), "-", "_")
)

func services(s ...tea_cmd.DockerService) []tea_cmd.DockerService {
	return s
}

// validateDevDALayer starts a locally running version of the DA layer, and replaces the DA_AUTH_TOKEN configuration
// variable with the token from the locally running container.
func validateDevDALayer(cfg config.Config) error {
	cfg.Build = true
	cfg.Debug = false
	cfg.Detach = true
	cfg.Timeout = -1
	fmt.Println("starting DA docker service for dev mode...")
	if err := tea_cmd.DockerStart(cfg, services(daService)); err != nil {
		return fmt.Errorf("error starting %s docker container: %w", daService, err)
	}

	// TODO: Use `docker container inspect -f '{{.State.Running}}' celestia_devnet` to block until
	// the container is running.
	time.Sleep(3 * time.Second)
	fmt.Println("started DA service...")

	daToken, err := getDAToken()
	if err != nil {
		return err
	}
	envOverrides := map[string]string{
		EnvDAAuthToken:   daToken,
		EnvDABaseURL:     fmt.Sprintf("http://%s:26658", daService),
		EnvDANamespaceID: "67480c4a88c4d12935d4",
	}
	for key, value := range envOverrides {
		fmt.Printf("overriding config value %q to %q\n", key, value)
		cfg.DockerEnv[key] = value
	}
	return nil
}

// validateProdDALayer assumes the DA layer is running somewhere else and validates the required world.toml variables are
// non-empty.
func validateProdDALayer(cfg config.Config) error {
	requiredEnvVariables := []string{
		EnvDAAuthToken,
		EnvDABaseURL,
		EnvDANamespaceID,
	}
	var errs []error
	for _, env := range requiredEnvVariables {
		if len(cfg.DockerEnv[env]) > 0 {
			continue
		}
		errs = append(errs, fmt.Errorf("missing %q", env))
	}
	if len(errs) > 0 {
		// Prepend an error describing the overall problem
		errs = append([]error{
			fmt.Errorf("the [evm] section of your config is missing some required variables"),
		}, errs...)
		return errors.Join(errs...)
	}
	return nil
}

func validateDALayer(cmd *cobra.Command, cfg config.Config) error {
	devDA, err := cmd.Flags().GetBool(FlagUseDevDA)
	if err != nil {
		return err
	}
	if devDA {
		return validateDevDALayer(cfg)
	}
	return validateProdDALayer(cfg)
}

func StartEVM() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the EVM base shard. Use --da-auth-token to pass in an auth token directly.",
		Long:  "Start the EVM base shard. Requires connection to celestia DA.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.GetConfig(cmd)
			if err != nil {
				return err
			}

			if err = validateDALayer(cmd, cfg); err != nil {
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

			err = tea_cmd.DockerStart(cfg, services(tea_cmd.DockerServiceEVM))
			if err != nil {
				fmt.Errorf("error starting %s docker container: %w", tea_cmd.DockerServiceEVM, err)
			}
			return nil
		},
	}
	cmd.Flags().String(FlagDAAuthToken, "", "DA Auth Token that allows the rollup to communicate with the Celestia client.")
	cmd.Flags().Bool(FlagUseDevDA, false, "Use a locally running DA layer")
	return cmd
}

func StopAll() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the EVM base shard and DA layer client.",
		Long:  "Stop the EVM base shard and data availability layer client if they are running.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tea_cmd.DockerStop(services(tea_cmd.DockerServiceEVM, tea_cmd.DockerServiceDA))
		},
	}
	return cmd
}

func getDAToken() (token string, err error) {
	// Create a new command
	maxRetries := 10
	cmdString := fmt.Sprintf("docker exec %s celestia bridge --node.store /home/celestia/bridge/ auth admin", daContainer)
	cmdParts := strings.Split(cmdString, " ")
	for retry := 0; retry < maxRetries; retry++ {
		fmt.Println("attempting to get DA token...")

		cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("failed to get da token")
			fmt.Println("command was: ", cmd.String())
			fmt.Printf("output: %q\n", string(output))
			fmt.Println("error: ", err)
			fmt.Printf("%d/%d retrying...\n", retry, maxRetries)
			time.Sleep(2 * time.Second)
			continue
		}

		if bytes.Contains(output, []byte("\n")) {
			return "", fmt.Errorf("da token should be a single line. got %v", string(output))
		}
		if len(output) == 0 {
			return "", fmt.Errorf("got empty DA token")
		}
		return string(output), nil
	}
	return "", fmt.Errorf("timed out while getting DA token")

}
