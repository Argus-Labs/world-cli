package cardinal

import (
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/teacmd"
)

/////////////////
// Cobra Setup //
/////////////////

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
		err := teacmd.DockerRestart(true, []teacmd.DockerService{
			teacmd.DockerServiceCardinal,
			teacmd.DockerServiceNakama,
		})
		if err != nil {
			return err
		}

		return nil
	},
}
