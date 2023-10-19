package cardinal

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/tea/style"
)

func init() {
	// Register subcommands - `world cardinal [subcommand]`
	BaseCmd.AddCommand(createCmd, startCmd, restartCmd, purgeCmd, stopCmd)
}

var BaseCmd = &cobra.Command{
	Use:     "cardinal",
	Short:   "Manage your Cardinal game shard project",
	Long:    style.CLIHeader("World CLI — CARDINAL", "Manage your Cardinal game shard project"),
	GroupID: "Core",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			log.Fatal().Err(err).Msg("Failed to execute cardinal command")
		}
	},
}