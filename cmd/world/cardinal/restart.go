package cardinal

import (
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/config"
	"pkg.world.dev/world-cli/internal/teacmd"
	"pkg.world.dev/world-cli/pkg/logger"
)

/////////////////
// Cobra Setup //
/////////////////

// restartCmd restarts your Cardinal game shard stack
// Usage: `world cardinal restart`
func (c *cardinal) RestartCmd() *cobra.Command {
	restartCmd := &cobra.Command{
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

			if cfg.Debug {
				err = c.teaCmd.DockerRestart(cfg, []teacmd.DockerService{
					teacmd.DockerServiceCardinalDebug,
					teacmd.DockerServiceNakama,
				})
			} else {
				err = c.teaCmd.DockerRestart(cfg, []teacmd.DockerService{
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

	return restartCmd
}
