package cardinal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"pkg.world.dev/world-cli/common"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/tea/style"
)

/////////////////
// Cobra Setup //
/////////////////

const (
	flagBuild     = "build"
	flagDebug     = "debug"
	flagDetach    = "detach"
	flagLogLevel  = "log-level"
	flagEditor    = "editor"
	flagTelemetry = "telemetry"

	// DockerCardinalEnvLogLevel Environment variable name for Docker
	DockerCardinalEnvLogLevel = "CARDINAL_LOG_LEVEL"
)

var (
	// ValidLogLevels Valid log levels for zerolog
	validLogLevels = strings.Join([]string{
		zerolog.TraceLevel.String(),
		zerolog.DebugLevel.String(),
		zerolog.InfoLevel.String(),
		zerolog.WarnLevel.String(),
		zerolog.ErrorLevel.String(),
		zerolog.FatalLevel.String(),
		zerolog.PanicLevel.String(),
		zerolog.Disabled.String(),
	}, ", ")

	ErrGracefulExit = eris.New("Process gracefully exited")
)

// startCmd starts your Cardinal game shard stack
// Usage: `world cardinal start`
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start your Cardinal game shard stack",
	Long: `Start your Cardinal game shard stack.

This will start the following Docker services and its dependencies:
- Cardinal (Game shard)
- Nakama (Relay)
- Redis (Cardinal dependency)`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.GetConfig()
		if err != nil {
			return err
		}
		// Parameters set at the command line overwrite toml values
		if err := replaceBoolWithFlag(cmd, flagBuild, &cfg.Build); err != nil {
			return err
		}

		if err := replaceBoolWithFlag(cmd, flagDebug, &cfg.Debug); err != nil {
			return err
		}

		if err := replaceBoolWithFlag(cmd, flagDetach, &cfg.Detach); err != nil {
			return err
		}

		if err := replaceBoolWithFlag(cmd, flagTelemetry, &cfg.Telemetry); err != nil {
			return err
		}
		cfg.Timeout = -1

		// Replace cardinal log level using flag value if flag is set
		logLevel, err := cmd.Flags().GetString(flagLogLevel)
		if err != nil {
			return err
		}

		if logLevel != "" {
			zeroLogLevel, err := zerolog.ParseLevel(logLevel)
			if err != nil {
				return eris.Errorf("invalid value for flag %s: must be one of (%v)", flagLogLevel, validLogLevels)
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
				DockerCardinalEnvLogLevel, validLogLevels)
		}

		runEditor, err := cmd.Flags().GetBool(flagEditor)
		if err != nil {
			return err
		}

		// Print out header
		fmt.Println(style.CLIHeader("Cardinal", ""))

		// Print out service addresses
		printServiceAddress("Redis", cfg.DockerEnv["REDIS_ADDRESS"])
		// this can be changed in code by calling WithPort() on world options, but we have no way to detect that
		printServiceAddress("Cardinal", fmt.Sprintf("localhost:%s", CardinalPort))
		var editorPort int
		if runEditor {
			editorPort, err = common.FindUnusedPort(cePortStart, cePortEnd)
			if err != nil {
				return eris.Wrap(err, "Failed to find an unused port for Cardinal Editor")
			}
			printServiceAddress("Cardinal Editor", fmt.Sprintf("localhost:%d", editorPort))
		} else {
			printServiceAddress("Cardinal Editor", "[disabled]")
		}
		fmt.Println()

		fmt.Print("Press <ENTER> to continue...")
		_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')

		fmt.Println("\nStarting Cardinal game shard...")
		fmt.Println("This may take a few minutes to rebuild the Docker images.")
		fmt.Println("Use `world cardinal dev` to run Cardinal faster/easier in development mode.")

		group, ctx := errgroup.WithContext(cmd.Context())

		// Create docker client
		dockerClient, err := docker.NewClient(cfg)
		if err != nil {
			return err
		}
		defer dockerClient.Close()

		services := getServices(cfg)

		// Start the World Engine stack
		group.Go(func() error {
			if err := dockerClient.Start(ctx, services...); err != nil {
				return eris.Wrap(err, "Encountered an error with Docker")
			}
			return eris.Wrap(ErrGracefulExit, "Stack terminated")
		})

		// Start Cardinal Editor is flag is set to true
		if runEditor {
			group.Go(func() error {
				if err := startCardinalEditor(ctx, cfg.RootDir, cfg.GameDir, editorPort); err != nil {
					return eris.Wrap(err, "Encountered an error with Cardinal Editor")
				}
				return eris.Wrap(ErrGracefulExit, "Cardinal Editor terminated")
			})
		}

		// If any of the group's goroutines is terminated non-gracefully, we want to treat it as an error.
		if err := group.Wait(); err != nil && !eris.Is(err, ErrGracefulExit) {
			return err
		}

		return nil
	},
}

func init() {
	registerEditorFlag(startCmd, true)
	startCmd.Flags().Bool(flagBuild, true, "Rebuild Docker images before starting")
	startCmd.Flags().Bool(flagDetach, false, "Run in detached mode")
	startCmd.Flags().String(flagLogLevel, "",
		fmt.Sprintf("Set the log level for Cardinal. Must be one of (%v)", validLogLevels))
	startCmd.Flags().Bool(flagDebug, false, "Enable delve debugging")
	startCmd.Flags().Bool(flagTelemetry, false, "Enable tracing, metrics, and profiling")
}

// replaceBoolWithFlag overwrites the contents of vale with the contents of the given flag. If the flag
// has not been set, value will remain unchanged.
func replaceBoolWithFlag(cmd *cobra.Command, flagName string, value *bool) error {
	if !cmd.Flags().Changed(flagName) {
		return nil
	}
	newVal, err := cmd.Flags().GetBool(flagName)
	if err != nil {
		return err
	}
	*value = newVal
	return nil
}
