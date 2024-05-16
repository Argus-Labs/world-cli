package cardinal

import (
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/dependency"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/tea/style"
)

// BaseCmd is the base command for the cardinal subcommand
// Usage: `world cardinal`
var BaseCmd = &cobra.Command{
	Use:     "cardinal",
	Short:   "Utilities for managing the Cardinal game shard",
	Long:    style.CLIHeader("World CLI — CARDINAL", "Manage your Cardinal game shard project"),
	GroupID: "core",
	PreRunE: func(_ *cobra.Command, _ []string) error {
		return dependency.Check(
			dependency.Go,
			dependency.Git,
			dependency.Docker,
			dependency.DockerCompose,
			dependency.DockerDaemon,
		)
	},
	Run: func(cmd *cobra.Command, _ []string) {
		if err := cmd.Help(); err != nil {
			logger.Fatalf("Failed to execute cardinal command : %s", err.Error())
		}
	},
}

func init() {
	// Register subcommands - `world cardinal [subcommand]`
	BaseCmd.AddCommand(startCmd, devCmd, restartCmd, purgeCmd, stopCmd)
	// Add --log-debug flag
	logger.AddLogFlag(startCmd, devCmd, restartCmd, purgeCmd, stopCmd)
}
