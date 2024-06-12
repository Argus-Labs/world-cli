package evm

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/teacmd"
)

var (
	// Docker compose seems to replace the hyphen with an underscore. This could be properly fixed by removing the hyphen
	// from celestia-devnet, or by investigating the docker compose documentation.
	daContainer = strings.ReplaceAll(string(daService), "-", "_")
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

		err = teacmd.DockerStart(cfg, services(teacmd.DockerServiceEVM))
		if err != nil {
			return fmt.Errorf("error starting %s docker container: %w", teacmd.DockerServiceEVM, err)
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
func validateDevDALayer(cfg *config.Config) error {
	cfg.Build = true
	cfg.Debug = false
	cfg.Detach = true
	cfg.Timeout = -1
	logger.Println("starting DA docker service for dev mode...")
	if err := teacmd.DockerStart(cfg, services(daService)); err != nil {
		return fmt.Errorf("error starting %s docker container: %w", daService, err)
	}

	if err := blockUntilContainerIsRunning(daContainer, 10*time.Second); err != nil { //nolint:gomnd
		return err
	}
	logger.Println("started DA service...")

	daToken, err := getDAToken()
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

func validateDALayer(cmd *cobra.Command, cfg *config.Config) error {
	devDA, err := cmd.Flags().GetBool(FlagUseDevDA)
	if err != nil {
		return err
	}
	if devDA {
		return validateDevDALayer(cfg)
	}
	return validateProdDALayer(cfg)
}

func getDAToken() (string, error) {
	// Create a new command
	maxRetries := 10
	cmdString := fmt.Sprintf("docker exec %s celestia bridge --node.store /home/celestia/bridge/ auth admin",
		daContainer)
	cmdParts := strings.Split(cmdString, " ")
	for retry := 0; retry < maxRetries; retry++ {
		logger.Println("attempting to get DA token...")

		cmd := exec.Command(cmdParts[0], cmdParts[1:]...) //nolint:gosec // not applicable
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Println("failed to get da token")
			logger.Printf("%d/%d retrying...\n", retry+1, maxRetries)
			time.Sleep(2 * time.Second) //nolint:gomnd
			continue
		}

		if bytes.Contains(output, []byte("\n")) {
			return "", fmt.Errorf("da token should be a single line. got %v", string(output))
		}
		if len(output) == 0 {
			return "", errors.New("got empty DA token")
		}
		return string(output), nil
	}
	return "", errors.New("timed out while getting DA token")
}

func blockUntilContainerIsRunning(targetContainer string, timeout time.Duration) error {
	timeoutAt := time.Now().Add(timeout)
	cmdString := "docker container inspect -f '{{.State.Running}}' " + targetContainer
	// This string will be returned by the above command when the container is running
	runningOutput := "'true'\n"
	cmdParts := strings.Split(cmdString, " ")
	for time.Now().Before(timeoutAt) {
		output, err := exec.Command(cmdParts[0], cmdParts[1:]...).CombinedOutput() //nolint:gosec // not applicable
		if err == nil && string(output) == runningOutput {
			return nil
		}
		time.Sleep(250 * time.Millisecond) //nolint:gomnd
	}
	return fmt.Errorf("timeout while waiting for %q to start", targetContainer)
}
