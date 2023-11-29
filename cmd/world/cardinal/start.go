package cardinal

import (
	"fmt"

	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/tea_cmd"
)

/////////////////
// Cobra Setup //
/////////////////

const (
	flagBuild  = "build"
	flagDebug  = "debug"
	flagDetach = "detach"
)

func init() {
	startCmd.Flags().Bool(flagBuild, true, "Rebuild Docker images before starting")
	startCmd.Flags().Bool(flagDebug, false, "Run in debug mode")
	startCmd.Flags().Bool(flagDetach, false, "Run in detached mode")
}

// startCmd starts your Cardinal game shard stack
// Usage: `world cardinal start`
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start your Cardinal game shard stack",
	Long: `Start your Cardinal game shard stack.

This will start the following Docker services and its dependencies:
- Cardinal (Core game logic)
- Nakama (Relay)
- Redis (Cardinal dependency)
- Postgres (Nakama dependency)`,
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
