package teacmd

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	"pkg.world.dev/world-cli/common/dependency"
	"pkg.world.dev/world-cli/tea/style"
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
func CheckDependenciesCmd(deps []dependency.Dependency) tea.Cmd {
	return func() tea.Msg {
		var res []DependencyStatus
		var resErr error
		for _, dep := range deps {
			err := dep.Cmd.Run()
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
func PrintDependencyStatus(depStatus []DependencyStatus) (string, string) {
	var depList string
	var help string
	for _, dep := range depStatus {
		if dep.IsInstalled {
			depList += style.TickIcon.Render() + " " + dep.Name + "\n"
		} else {
			depList += style.CrossIcon.Render() + " " + dep.Name + "\n"
			help += dep.Help + "\n"
		}
	}
	return depList, help
}

func PrettyPrintMissingDependency(depStatus []DependencyStatus) string {
	depList, help := PrintDependencyStatus(depStatus)
	out := style.Container.Render("--- Found Missing Dependencies ---") + "\n\n"
	out += depList + "\n" + help + "\n"
	return out
}
