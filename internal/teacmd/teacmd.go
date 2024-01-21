package teacmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"pkg.world.dev/world-cli/config"
	"pkg.world.dev/world-cli/utils/dependency"
	"pkg.world.dev/world-cli/utils/terminal"
)

type teaCmd struct {
	terminal terminal.Terminal
}

type TeaCmd interface {
	DockerStart(cfg config.Config, services []DockerService) error
	DockerStartAll(cfg config.Config) error
	DockerRestart(cfg config.Config, services []DockerService) error
	DockerStop(services []DockerService) error
	DockerStopAll() error
	DockerPurge() error

	GitCloneCmd(url string, targetDir string, initMsg string) (err error)

	CheckDependenciesCmd(deps []dependency.Dependency) tea.Cmd
	PrintDependencyStatus(depStatus []DependencyStatus) (string, string)
	PrettyPrintMissingDependency(depStatus []DependencyStatus) string
}

func New(terminal terminal.Terminal) TeaCmd {
	return &teaCmd{
		terminal: terminal,
	}
}
