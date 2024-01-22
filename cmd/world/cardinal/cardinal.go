package cardinal

import (
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/internal/dependency"
	"pkg.world.dev/world-cli/internal/teacmd"
	"pkg.world.dev/world-cli/pkg/logger"
	"pkg.world.dev/world-cli/utils/tea/style"
	"pkg.world.dev/world-cli/utils/terminal"
)

type cardinal struct {
	terminal terminal.Terminal
	teaCmd   teacmd.TeaCmd
}

type Cardinal interface {
	GetBaseCmd() *cobra.Command
}

func New(terminal terminal.Terminal, teaCmd teacmd.TeaCmd) Cardinal {
	c := &cardinal{
		terminal: terminal,
		teaCmd:   teaCmd,
	}
	return c
}

// GetBaseCmd returns the base command
// BaseCmd is the base command for the cardinal subcommand
// Usage: `world cardinal`
func (c *cardinal) GetBaseCmd() *cobra.Command {
	baseCmd := &cobra.Command{
		Use:     "cardinal",
		Short:   "Manage your Cardinal game shard project",
		Long:    style.CLIHeader("World CLI — CARDINAL", "Manage your Cardinal game shard project"),
		GroupID: "Core",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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

	startCmd := c.StartCmd()
	devCmd := c.DevCmd()
	restartCmd := c.RestartCmd()
	purgeCmd := c.PurgeCmd()
	stopCmd := c.StopCmd()

	// Register subcommands - `world cardinal [subcommand]`
	baseCmd.AddCommand(startCmd, devCmd, restartCmd, purgeCmd, stopCmd)

	// Add --debug flag
	logger.AddLogFlag(startCmd, devCmd, restartCmd, purgeCmd, stopCmd)

	return baseCmd
}
