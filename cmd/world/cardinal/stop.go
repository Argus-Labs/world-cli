package cardinal

import (
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/tea_cmd"
)

/////////////////
// Cobra Setup //
/////////////////

// stopCmd stops your Cardinal game shard stack
// Usage: `world cardinal stop`
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop your Cardinal game shard stack",
	Long: `Stop your Cardinal game shard stack.

This will stop the following Docker services:
- Cardinal (Core game logic)
- Nakama (Relay)
- Redis (Cardinal dependency)
- Postgres (Nakama dependency)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := tea_cmd.DockerStopAll()
		if err != nil {
			return err
		}

		return nil
	},
}
