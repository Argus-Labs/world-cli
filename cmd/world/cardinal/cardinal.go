package cardinal

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/dependency"
	"pkg.world.dev/world-cli/tea/style"
)

// BaseCmd is the base command for the cardinal subcommand
func BaseCmd() *cobra.Command {
	base := &cobra.Command{
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
	},Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				log.Fatal().Err(err).Msg("Failed to execute cardinal command")
			}
		},
	}
	base.AddCommand(createCmd, StartCmd(), devCmd, restartCmd, purgeCmd, stopCmd)
	return base
}
