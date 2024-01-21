package teacmd

import (
	"errors"
	"pkg.world.dev/world-cli/utils/dependency"

	tea "github.com/charmbracelet/bubbletea"

	"pkg.world.dev/world-cli/utils/tea/style"
)

type DependencyStatus struct {
	dependency.Dependency
	IsInstalled bool
}

type CheckDependenciesMsg struct {
	Err       error
	DepStatus []DependencyStatus
}

// CheckDependenciesCmd Iterate through required dependencies and check if they are installed.
// Dispatch CheckDependenciesMsg if any dependency is missing.
func (t *teaCmd) CheckDependenciesCmd(deps []dependency.Dependency) tea.Cmd {
	return func() tea.Msg {
		var res []DependencyStatus
		var resErr error
		for _, dep := range deps {
			err := dep.Run()
			res = append(res, DependencyStatus{
				Dependency:  dep,
				IsInstalled: err == nil,
			})
			resErr = errors.Join(resErr, err)
		}

		return CheckDependenciesMsg{
			Err:       resErr,
			DepStatus: res,
		}
	}
}

// PrintDependencyStatus Return a string with dependency status list and help messages.
func (t *teaCmd) PrintDependencyStatus(depStatus []DependencyStatus) (string, string) {
	var depList string
	var help string
	for _, dep := range depStatus {
		if dep.IsInstalled {
			depList += style.TickIcon.Render() + " " + dep.GetName() + "\n"
		} else {
			depList += style.CrossIcon.Render() + " " + dep.GetName() + "\n"
			help += dep.GetHelp() + "\n"
		}
	}
	return depList, help
}

func (t *teaCmd) PrettyPrintMissingDependency(depStatus []DependencyStatus) string {
	depList, help := t.PrintDependencyStatus(depStatus)
	out := style.Container.Render("--- Found Missing Dependencies ---") + "\n\n"
	out += depList + "\n" + help + "\n"
	return out
}
