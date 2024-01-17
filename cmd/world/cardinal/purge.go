package cardinal

import (
	"fmt"
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/tea_cmd"
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
		logger.SetDebugMode(cmd)
		err := tea_cmd.DockerPurge()
		if err != nil {
			return err
		}
		fmt.Println("Cardinal successfully purged")

		return nil
	},
}
