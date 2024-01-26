package cardinal

import (
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/dependency"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/tea/style"
)

func init() {
	// Register subcommands - `world cardinal [subcommand]`
	BaseCmd.AddCommand(startCmd, devCmd, restartCmd, purgeCmd, stopCmd)
	// Add --debug flag
	logger.AddLogFlag(startCmd, devCmd, restartCmd, purgeCmd, stopCmd)
}

// BaseCmd is the base command for the cardinal subcommand
// Usage: `world cardinal`
var BaseCmd = &cobra.Command{
	Use:     "cardinal",
	Short:   "Manage your Cardinal game shard project",
	Long:    style.CLIHeader("World CLI — CARDINAL", "Manage your Cardinal game shard project"),
	GroupID: "Core",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return dependency.Check(
			dependency.Go,
			dependency.Git,
			dependency.Docker,
			dependency.DockerCompose,
			dependency.DockerDaemon,
		)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			logger.Fatalf("Failed to execute cardinal command : %s", err.Error())
		}
	},
}
