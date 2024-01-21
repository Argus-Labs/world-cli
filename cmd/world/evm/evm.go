package evm

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/config"
	"pkg.world.dev/world-cli/internal/teacmd"
	"pkg.world.dev/world-cli/pkg/logger"
	"pkg.world.dev/world-cli/utils/terminal"
)

type evm struct {
	teaCmd   teacmd.TeaCmd
	terminal terminal.Terminal
}

type EVM interface {
	GetBaseCmd() *cobra.Command
}

const (
	FlagUseDevDA     = "dev"
	FlagDAAuthToken  = "da-auth-token"
	EnvDAAuthToken   = "DA_AUTH_TOKEN"
	EnvDABaseURL     = "DA_BASE_URL"
	EnvDANamespaceID = "DA_NAMESPACE_ID"

	daService = teacmd.DockerServiceDA
)

func New(terminal terminal.Terminal, teaCommand teacmd.TeaCmd) EVM {
	return &evm{
		teaCmd:   teaCommand,
		terminal: terminal,
	}
}

func (e *evm) GetBaseCmd() *cobra.Command {
	evmRootCmd := &cobra.Command{
		Use:   "evm",
		Short: "EVM base shard commands.",
		Long:  "Commands for provisioning the EVM Base Shard.",
	}

	evmRootCmd.AddGroup(&cobra.Group{
		ID:    "EVM",
		Title: "EVM Base Shard Commands",
	})

	startCmd := e.startEVM()
	stopCmd := e.stopAll()

	evmRootCmd.AddCommand(
		startCmd,
		stopCmd,
	)

	// Add --debug flag
	logger.AddLogFlag(startCmd, stopCmd)

	return evmRootCmd
}

var (
	// Docker compose seems to replace the hyphen with an underscore. This could be properly fixed by removing the hyphen
	// from celesta-devnet, or by investigating the docker compose documentation.
	daContainer = strings.ReplaceAll(string(daService), "-", "_")
)

func services(s ...teacmd.DockerService) []teacmd.DockerService {
	return s
}

// validateDevDALayer starts a locally running version of the DA layer, and replaces the DA_AUTH_TOKEN configuration
// variable with the token from the locally running container.
func (e *evm) validateDevDALayer(cfg config.Config) error {
	cfg.Build = true
	cfg.Debug = false
	cfg.Detach = true
	cfg.Timeout = -1
	logger.Println("starting DA docker service for dev mode...")
	if err := e.teaCmd.DockerStart(cfg, services(daService)); err != nil {
		return fmt.Errorf("error starting %s docker container: %w", daService, err)
	}

	if err := e.blockUntilContainerIsRunning(daContainer, 10*time.Second); err != nil {
		return err
	}
	logger.Println("started DA service...")

	daToken, err := e.getDAToken()
	if err != nil {
		return err
	}
	envOverrides := map[string]string{
		EnvDAAuthToken:   daToken,
		EnvDABaseURL:     fmt.Sprintf("http://%s:26658", daService),
		EnvDANamespaceID: "67480c4a88c4d12935d4",
	}
	for key, value := range envOverrides {
		logger.Printf("overriding config value %q to %q\n", key, value)
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

func (e *evm) validateDALayer(cmd *cobra.Command, cfg config.Config) error {
	devDA, err := cmd.Flags().GetBool(FlagUseDevDA)
	if err != nil {
		return err
	}
	if devDA {
		return e.validateDevDALayer(cfg)
	}
	return validateProdDALayer(cfg)
}

func (e *evm) startEVM() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the EVM base shard. Use --da-auth-token to pass in an auth token directly.",
		Long:  "Start the EVM base shard. Requires connection to celestia DA.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.SetDebugMode(cmd)

			cfg, err := config.GetConfig(cmd)
			if err != nil {
				return err
			}

			if err = e.validateDALayer(cmd, cfg); err != nil {
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

			err = e.teaCmd.DockerStart(cfg, services(teacmd.DockerServiceEVM))
			if err != nil {
				return fmt.Errorf("error starting %s docker container: %w", teacmd.DockerServiceEVM, err)
			}
			return nil
		},
	}
	cmd.Flags().String(FlagDAAuthToken, "", "DA Auth Token that allows the rollup to communicate with the Celestia client.")
	cmd.Flags().Bool(FlagUseDevDA, false, "Use a locally running DA layer")
	return cmd
}

func (e *evm) stopAll() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the EVM base shard and DA layer client.",
		Long:  "Stop the EVM base shard and data availability layer client if they are running.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.SetDebugMode(cmd)

			err := e.teaCmd.DockerStop(services(teacmd.DockerServiceEVM, teacmd.DockerServiceDA))
			if err != nil {
				return err
			}

			fmt.Println("EVM successfully stopped")
			return nil
		},
	}
	return cmd
}

func (e *evm) getDAToken() (token string, err error) {
	// Create a new command
	maxRetries := 10
	cmdString := fmt.Sprintf("docker exec %s celestia bridge --node.store /home/celestia/bridge/ auth admin", daContainer)
	cmdParts := strings.Split(cmdString, " ")
	for retry := 0; retry < maxRetries; retry++ {
		logger.Println("attempting to get DA token...")

		output, err := e.terminal.Exec(cmdParts[0], cmdParts[1:]...)
		if err != nil {
			logger.Println("failed to get da token")
			logger.Printf("%d/%d retrying...\n", retry+1, maxRetries)
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

func (e *evm) blockUntilContainerIsRunning(targetContainer string, timeout time.Duration) error {
	timeoutAt := time.Now().Add(timeout)
	cmdString := "docker container inspect -f '{{.State.Running}}' " + targetContainer
	// This string will be returned by the above command when the container is running
	runningOutput := "'true'\n"
	cmdParts := strings.Split(cmdString, " ")
	for time.Now().Before(timeoutAt) {
		output, err := e.terminal.Exec(cmdParts[0], cmdParts[1:]...)
		if err == nil && string(output) == runningOutput {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("timeout while waiting for %q to start", targetContainer)
}
