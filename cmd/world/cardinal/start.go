package cardinal

import (
	"fmt"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/teacmd"
)

/////////////////
// Cobra Setup //
/////////////////

// StartCmd returns a command that starts your Cardinal game shard stack.
func StartCmd() *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start your Cardinal game shard stack",
		Long: `Start your Cardinal game shard stack.

This will start the following Docker services and its dependencies:
- Cardinal (Core game logic)
- Nakama (Relay)
- Redis (Cardinal dependency)
- Postgres (Nakama dependency)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			buildFlag, err := cmd.Flags().GetBool("build")
			if err != nil {
				return err
			}

			debugFlag, err := cmd.Flags().GetBool("debug")
			if err != nil {
				return err
			}

			detachFlag, err := cmd.Flags().GetBool("detach")
			if err != nil {
				return err
			}

			fmt.Println("Starting Cardinal game shard...")
			fmt.Println("This may take a few minutes to rebuild the Docker images.")
			fmt.Println("Use `world cardinal dev` to run Cardinal faster/easier in development mode.")

			err = teacmd.DockerStartAll(buildFlag, debugFlag, detachFlag, -1)
			if err != nil {
				return err
			}

			return nil
		},
	}
	startCmd.Flags().Bool("build", true, "Rebuild Docker images before starting")
	startCmd.Flags().Bool("debug", false, "Run in debug mode")
	startCmd.Flags().Bool("detach", false, "Run in detached mode")
	return startCmd
}
