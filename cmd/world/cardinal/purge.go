package cardinal

import (
	"fmt"

	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
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
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.GetConfig(cmd)
		if err != nil {
			return err
		}

		err = docker.Purge(cfg)
		if err != nil {
			return err
		}
		fmt.Println("Cardinal successfully purged")

		return nil
	},
}
