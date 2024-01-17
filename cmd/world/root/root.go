package root

import (
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/config"

	"pkg.world.dev/world-cli/cmd/world/cardinal"
	"pkg.world.dev/world-cli/cmd/world/evm"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/tea/style"
)

func init() {
	// Enable case-insensitive commands
	cobra.EnableCaseInsensitive = true

	// Register groups
	rootCmd.AddGroup(&cobra.Group{ID: "Core", Title: "World CLI Commands:"})

	// Register base commands
	rootCmd.AddCommand(doctorCmd, versionCmd)

	// Register subcommands
	rootCmd.AddCommand(cardinal.BaseCmd)
	rootCmd.AddCommand(evm.EVMCmds())

	config.AddConfigFlag(rootCmd)

	// Add --debug flag
	logger.AddLogFlag(doctorCmd)
}

// rootCmd represents the base command
// Usage: `world`
var rootCmd = &cobra.Command{
	Use:   "world",
	Short: "A swiss army knife for World Engine projects",
	Long:  style.CLIHeader("World CLI", "A swiss army knife for World Engine projects"),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Errors(err)
	}
	// print log stack
	logger.PrintLogs()
}
