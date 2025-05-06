package root

import (
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/printer"
)

var AppVersion string

// versionCmd print the version number of World CLI.
// Usage: `world version`.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the current World CLI version",
	Long: `Show the exact version of World CLI you're currently using.
	
This information is useful when reporting issues, checking for updates,
or verifying compatibility with World Engine features.`,
	Run: func(_ *cobra.Command, _ []string) {
		printer.Infof("World CLI %s\n", AppVersion)
	},
}
