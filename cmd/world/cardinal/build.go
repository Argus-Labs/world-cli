package cardinal

import (
	"fmt"

	"github.com/docker/docker/api/types/registry"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/tea/style"
)

const (
	flagPush     = "push"
	flagAuth     = "auth"
	flagUser     = "user"
	flagPass     = "pass"
	flagRegToken = "regtoken"
)

type BuildCmd struct {
	LogLevel  string `flag:"" help:"Set the log level for Cardinal"`
	Debug     bool   `flag:"" help:"Enable debugging"`
	Telemetry bool   `flag:"" help:"Enable tracing, metrics, and profiling"`
	Push      string `flag:"" help:"Push your cardinal image to a given image repository"`
}

func (c *BuildCmd) Run() error {
	_, err := config.GetConfig()
	if err != nil {
		return err
	}
	return nil
}

/////////////////
// Cobra Setup //
/////////////////

// buildCmd build your Cardinal game image.
// Usage: `world cardinal build`.
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Create optimized Docker images for your Cardinal game",
	Long: `Build and package your Cardinal game into production-ready Docker images.

This command creates the Cardinal (Game shard) Docker image with your game logic, 
optimized for deployment. You can optionally push the image to a registry with the --push flag.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.GetConfig()
		if err != nil {
			return err
		}
		// Parameters set at the command line overwrite toml values
		// if err := replaceBoolWithFlag(cmd, flagDebug, &cfg.Debug); err != nil {
		// 	return err
		// }

		// if err := replaceBoolWithFlag(cmd, flagTelemetry, &cfg.Telemetry); err != nil {
		// 	return err
		// }
		cfg.Timeout = -1

		// Replace cardinal log level using flag value if flag is set
		logLevel, err := cmd.Flags().GetString(flagLogLevel)
		if err != nil {
			return err
		}

		if logLevel != "" {
			zeroLogLevel, err := zerolog.ParseLevel(logLevel)
			if err != nil {
				return eris.Errorf("invalid value for flag %s: must be one of (%v)", flagLogLevel, validLogLevels())
			}
			cfg.DockerEnv[DockerCardinalEnvLogLevel] = zeroLogLevel.String()
		}

		if val, exists := cfg.DockerEnv[DockerCardinalEnvLogLevel]; !exists || val == "" {
			// Set default log level to 'info' if log level is not set
			cfg.DockerEnv[DockerCardinalEnvLogLevel] = zerolog.InfoLevel.String()
		} else if _, err := zerolog.ParseLevel(cfg.DockerEnv[DockerCardinalEnvLogLevel]); err != nil {
			// make sure the log level is valid when the flag is not set and using env var from config
			// Error when CARDINAL_LOG_LEVEL is not a valid log level
			return eris.Errorf("invalid value for %s env variable in the config file: must be one of (%v)",
				DockerCardinalEnvLogLevel, validLogLevels())
		}

		// Print out header
		printer.Infoln(style.CLIHeader("Cardinal", ""))
		// Print out service addresses
		printServiceAddress("Redis", cfg.DockerEnv["REDIS_ADDRESS"])
		// this can be changed in code by calling WithPort() on world options, but we have no way to detect that
		printServiceAddress("Cardinal", fmt.Sprintf("localhost:%s", CardinalPort))
		printer.NewLine(2)
		printer.Infoln("Building Cardinal game shard image...")
		printer.Infoln("This may take a few minutes.")

		group, ctx := errgroup.WithContext(cmd.Context())

		// Create docker client
		dockerClient, err := docker.NewClient(cfg)
		if err != nil {
			return err
		}
		defer dockerClient.Close()

		services := getCardinalServices(cfg)

		pushTo, _ := cmd.Flags().GetString(flagPush)
		pushAuth, _ := cmd.Flags().GetString(flagAuth)
		if pushAuth != "" {
			pushUser, _ := cmd.Flags().GetString(flagUser)
			pushPass, _ := cmd.Flags().GetString(flagPass)
			pushRegToken, _ := cmd.Flags().GetString(flagRegToken)
			if (pushUser != "" && pushPass != "") || pushRegToken != "" {
				authConfig := registry.AuthConfig{
					Username:      pushUser,
					Password:      pushPass,
					RegistryToken: pushRegToken,
				}
				pushAuth, _ = registry.EncodeAuthConfig(authConfig)
			}
		}

		// Build the World Engine stack
		group.Go(func() error {
			if err := dockerClient.Build(ctx, pushTo, pushAuth, services...); err != nil {
				return eris.Wrap(err, "Encountered an error with Docker")
			}
			return eris.Wrap(ErrGracefulExit, "Stack terminated")
		})

		// If any of the group's goroutines is terminated non-gracefully, we want to treat it as an error.
		if err := group.Wait(); err != nil && !eris.Is(err, ErrGracefulExit) {
			return err
		}

		return nil
	},
}

func buildCmdInit() {
	//registerEditorFlag(buildCmd, true)
	buildCmd.Flags().String(flagLogLevel, "",
		fmt.Sprintf("Set the log level for Cardinal. Must be one of (%v)", validLogLevels()))
	buildCmd.Flags().Bool(flagDebug, false, "Enable delve debugging")
	buildCmd.Flags().Bool(flagTelemetry, false, "Enable tracing, metrics, and profiling")
	buildCmd.Flags().String(flagPush, "", "Push your cardinal image to a given image repository")
}
