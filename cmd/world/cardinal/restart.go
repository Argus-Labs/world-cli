package cardinal

import (
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/tea_cmd"
)

/////////////////
// Cobra Setup //
/////////////////

func init() {
	restartCmd.Flags().Bool(flagDetach, false, "Run in detached mode")
}

// restartCmd restarts your Cardinal game shard stack
// Usage: `world cardinal restart`
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart your Cardinal game shard stack",
	Long: `Restart your Cardinal game shard stack.

This will restart the following Docker services:
- Cardinal (Core game logic)
- Nakama (Relay)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.SetDebugMode(cmd)

		cfg, err := config.GetConfig(cmd)
		if err != nil {
			return err
		}
		cfg.Build = true
		if replaceBoolWithFlag(cmd, flagDebug, &cfg.Debug); err != nil {
			return err
		}

		if replaceBoolWithFlag(cmd, flagDetach, &cfg.Detach); err != nil {
			return err
		}

		if cfg.Debug {
			err = tea_cmd.DockerRestart(cfg, []tea_cmd.DockerService{
				tea_cmd.DockerServiceCardinalDebug,
				tea_cmd.DockerServiceNakama,
			})
		} else {
			err = tea_cmd.DockerRestart(cfg, []tea_cmd.DockerService{
				tea_cmd.DockerServiceCardinal,
				tea_cmd.DockerServiceNakama,
			})
		}

		if err != nil {
			return err
		}

		return nil
	},
}
