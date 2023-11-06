package root

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// versionCmd print the version number of World CLI
// Usage: `world version`
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of World CLI",
	Long:  `Print the version number of World CLI`,
	Run: func(cmd *cobra.Command, args []string) {
		bi, ok := debug.ReadBuildInfo()
		if ok {
			fmt.Printf("World CLI %s\n", bi.Main.Version)
		} else {
			fmt.Printf("World CLI <unknown>\n")
		}
	},
}
