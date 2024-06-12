package cardinal

import (
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/teacmd"
)

// restartCmd restarts your Cardinal game shard stack
// Usage: `world cardinal restart`
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart your Cardinal game shard stack",
	Long: `Restart your Cardinal game shard stack.

This will restart the following Docker services:
- Cardinal (Core game logic)
- Nakama (Relay)`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.GetConfig(cmd)
		if err != nil {
			return err
		}
		cfg.Build = true
		if err := replaceBoolWithFlag(cmd, flagDebug, &cfg.Debug); err != nil {
			return err
		}

		if err := replaceBoolWithFlag(cmd, flagDetach, &cfg.Detach); err != nil {
			return err
		}

		if cfg.Debug {
			err = teacmd.DockerRestart(cfg, []teacmd.DockerService{
				teacmd.DockerServiceCardinalDebug,
				teacmd.DockerServiceNakama,
			})
		} else {
			err = teacmd.DockerRestart(cfg, []teacmd.DockerService{
				teacmd.DockerServiceCardinal,
				teacmd.DockerServiceNakama,
			})
		}

		if err != nil {
			return err
		}

		return nil
	},
}

/////////////////
// Cobra Setup //
/////////////////

func init() {
	restartCmd.Flags().Bool(flagDetach, false, "Run in detached mode")
}
