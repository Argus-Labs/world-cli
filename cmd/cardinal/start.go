package cardinal

import (
	"fmt"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/tea_cmd"
)

/////////////////
// Cobra Setup //
/////////////////

func init() {
	startCmd.Flags().Bool("build", true, "Rebuild the Docker images before starting")
	startCmd.Flags().Bool("debug", false, "Enable debug mode")
	startCmd.Flags().String("mode", "", "Run with special mode [detach/integration-test]")
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start your Cardinal game shard stack",
	Long: `Start your Cardinal game shard stack.

This will start the following Docker services:
- Cardinal (Core game logic)
- Nakama (Relay)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		buildFlag, err := cmd.Flags().GetBool("build")
		if err != nil {
			return err
		}

		debugFlag, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return err
		}

		modeFlag, err := cmd.Flags().GetString("mode")
		if err != nil {
			return err
		}

		// Don't allow running with special mode and debug mode
		if modeFlag != "" && debugFlag {
			return fmt.Errorf("cannot run with special mode and debug mode at the same time")
		}

		if modeFlag == "" {
			if debugFlag {
				err = tea_cmd.DockerStartDebug()
				if err != nil {
					return err
				}
			} else {
				err = tea_cmd.DockerStart(buildFlag, []tea_cmd.DockerService{tea_cmd.DockerServiceCardinal, tea_cmd.DockerServiceNakama})
				if err != nil {
					return err
				}
			}
		} else {
			// Start with special mode (detach/integration-test)
			switch modeFlag {
			case "detach":
				err = tea_cmd.DockerStartDetach()
				if err != nil {
					return err
				}
			case "integration-test":
				err = tea_cmd.DockerStartTest()
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown mode %s", modeFlag)
			}
		}

		return nil
	},
}
