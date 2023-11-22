package cardinal

import (
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/teacmd"
)

/////////////////
// Cobra Setup //
/////////////////

// purgeCmd stops and resets the state of your Cardinal game shard
// Usage: `world cardinal purge`
var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Stop and reset the state of your Cardinal game shard",
	Long: `Stop and reset the state of your Cardinal game shard.
This command stop all Docker services and remove all Docker volumes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := teacmd.DockerPurge()
		if err != nil {
			return err
		}

		return nil
	},
}
