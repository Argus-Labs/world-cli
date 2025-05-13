package root

import (
	"pkg.world.dev/world-cli/common/printer"
)

var AppVersion string

// VersionCmd is the command to show the version of the CLI.
type VersionCmd struct {
	Check bool `help:"Check for the latest version of the CLI"`
}

func (c *VersionCmd) Run() error {
	printer.Infof("World CLI %s\n", AppVersion)
	if c.Check {
		return checkLatestVersion()
	}
	return nil
}

// versionCmd print the version number of World CLI.
// Usage: `world version`.
/*var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the current World CLI version",
	Long: `Show the exact version of World CLI you're currently using.

This information is useful when reporting issues, checking for updates,
or verifying compatibility with World Engine features.`,
	Run: func(_ *cobra.Command, _ []string) {
		printer.Infof("World CLI %s\n", AppVersion)
	},
}*/

func SetAppVersion(version string) {
	AppVersion = version
}
