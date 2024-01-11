package cardinal

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/tea_cmd"
)

/////////////////
// Cobra Setup //
/////////////////

const (
	flagBuild    = "build"
	flagDebug    = "debug"
	flagDetach   = "detach"
	flagLogLevel = "log-level"

	// DockerCardinalEnvLogLevel Environment variable name for Docker
	DockerCardinalEnvLogLevel = "CARDINAL_LOG_LEVEL"
)

var (
	// ValidLogLevels Valid log levels for zerolog
	validLogLevels = strings.Join([]string{zerolog.DebugLevel.String(), zerolog.InfoLevel.String(), zerolog.WarnLevel.String(),
		zerolog.ErrorLevel.String(), zerolog.FatalLevel.String(), zerolog.PanicLevel.String(), zerolog.Disabled.String(),
		zerolog.TraceLevel.String()}, ", ")
)

func init() {
	startCmd.Flags().Bool(flagBuild, true, "Rebuild Docker images before starting")
	startCmd.Flags().Bool(flagDebug, false, "Run in debug mode")
	startCmd.Flags().Bool(flagDetach, false, "Run in detached mode")
	startCmd.Flags().String(flagLogLevel, "", "Set the log level")
}

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
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.GetConfig(cmd)
		if err != nil {
			return err
		}
		// Parameters set at the command line overwrite toml values
		if replaceBoolWithFlag(cmd, flagBuild, &cfg.Build); err != nil {
			return err
		}

		if replaceBoolWithFlag(cmd, flagDebug, &cfg.Debug); err != nil {
			return err
		}

		if replaceBoolWithFlag(cmd, flagDetach, &cfg.Detach); err != nil {
			return err
		}
		cfg.Timeout = -1

		// Replace cardinal log level using flag value if flag is set
		logLevel, err := cmd.Flags().GetString(flagLogLevel)
		if logLevel != "" {
			zeroLogLevel, err := zerolog.ParseLevel(logLevel)
			if err != nil {
				return fmt.Errorf("invalid value for flag %s: must be one of (%v)", flagLogLevel, validLogLevels)
			}
			cfg.DockerEnv[DockerCardinalEnvLogLevel] = zeroLogLevel.String()
		}

		if val, exists := cfg.DockerEnv[DockerCardinalEnvLogLevel]; !exists || val == "" {
			// Set default log level to 'info' if log level is not set
			cfg.DockerEnv[DockerCardinalEnvLogLevel] = zerolog.InfoLevel.String()
		} else if _, err := zerolog.ParseLevel(cfg.DockerEnv[DockerCardinalEnvLogLevel]); err != nil { // make sure the log level is valid when the flag is not set and using env var from config
			// Error when CARDINAL_LOG_LEVEL is not a valid log level
			return fmt.Errorf("invalid value for %s env variable in the config file: must be one of (%v)", DockerCardinalEnvLogLevel, validLogLevels)
		}

		fmt.Println("Starting Cardinal game shard...")
		fmt.Println("This may take a few minutes to rebuild the Docker images.")
		fmt.Println("Use `world cardinal dev` to run Cardinal faster/easier in development mode.")

		err = tea_cmd.DockerStartAll(cfg)
		if err != nil {
			return err
		}

		return nil
	},
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
