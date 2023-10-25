package cmd

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"pkg.world.dev/world-cli/cmd/cardinal"
	"pkg.world.dev/world-cli/tea/style"
)

func init() {
	// Enable case-insensitive commands
	cobra.EnableCaseInsensitive = true

	// Register groups
	RootCmd.AddGroup(&cobra.Group{ID: "Core", Title: "World CLI Commands:"})

	// Register base commands
	RootCmd.AddCommand(doctorCmd, versionCmd)

	// Register subcommands
	RootCmd.AddCommand(cardinal.BaseCmd)
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "world",
	Short: "A swiss army knife for World Engine projects",
	Long:  style.CLIHeader("World CLI", "A swiss army knife for World Engine projects"),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	if err := RootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Failed to execute root command")
	}
}
