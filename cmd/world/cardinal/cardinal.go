package cardinal

import (
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/dependency"
	"pkg.world.dev/world-cli/tea/style"
)

func init() {
	// Register subcommands - `world cardinal [subcommand]`
	BaseCmd.AddCommand(createCmd, startCmd, devCmd, restartCmd, purgeCmd, stopCmd)
	BaseCmd.Flags().String("config", "", "a toml encoded config file")
}

// BaseCmd is the base command for the cardinal subcommand
// Usage: `world cardinal`
var BaseCmd = &cobra.Command{
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
			log.Fatal().Err(err).Msg("Failed to execute cardinal command")
		}
	},
}

func getConfig(cmd *cobra.Command) (cfg config.Config, err error) {
	if !cmd.Flags().Changed("config") {
		// The config flag was not set. Attempt to find the config via environment variables or in the local directory
		return config.LoadConfig("")
	}
	// The config flag was explicitly set
	configFile, err := cmd.Flags().GetString("config")
	if err != nil {
		return cfg, err
	}
	if configFile == "" {
		return cfg, errors.New("config cannot be empty")
	}
	return config.LoadConfig(configFile)

}
