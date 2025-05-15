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

func SetAppVersion(version string) {
	AppVersion = version
}
