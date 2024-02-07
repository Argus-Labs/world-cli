package root

import (
	"fmt"
	"github.com/spf13/cobra"
)

var AppVersion string

// versionCmd print the version number of World CLI
// Usage: `world version`
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of World CLI",
	Long:  `Print the version number of World CLI`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("World CLI %s\n", AppVersion)
	},
}
