package cardinal

import (
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/tea_cmd"
)

/////////////////
// Cobra Setup //
/////////////////

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop your Cardinal game shard stack",
	Long: `Stop your Cardinal game shard stack.

This will stop the following Docker services:
- Cardinal (Core game logic)
- Nakama (Relay)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := tea_cmd.DockerStop([]tea_cmd.DockerService{
			tea_cmd.DockerServiceCardinal,
			tea_cmd.DockerServiceNakama,
			tea_cmd.DockerServicePostgres,
			tea_cmd.DockerServiceRedis,
			tea_cmd.DockerServiceTestsuite,
		})
		if err != nil {
			return err
		}

		return nil
	},
}
