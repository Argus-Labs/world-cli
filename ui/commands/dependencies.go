package commands

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"pkg.world.dev/world-cli/ui/component/dependency"
	"pkg.world.dev/world-cli/ui/style"
)

// DependencyStatus represents the status of a dependency check
type DependencyStatus struct {
	Name    string
	Version string
	Error   error
}

// CheckDependenciesMsg represents a message containing dependency check results
type CheckDependenciesMsg struct {
	DepStatus []DependencyStatus
	Err       error
}

// CheckDependenciesCmd returns a tea.Cmd that checks dependencies
func CheckDependenciesCmd(deps []dependency.Dependency) tea.Cmd {
	return func() tea.Msg {
		depStatus := make([]DependencyStatus, len(deps))
		for i, dep := range deps {
			version, err := dep.GetVersion()
			depStatus[i] = DependencyStatus{
				Name:    dep.Name,
				Version: version,
				Error:   err,
			}
		}

		// Check if any dependencies failed
		var err error
		for _, status := range depStatus {
			if status.Error != nil {
				err = errors.New("missing dependencies")
				break
			}
		}

		return CheckDependenciesMsg{
			DepStatus: depStatus,
			Err:       err,
		}
	}
}

// PrintDependencyStatus formats dependency status for display
func PrintDependencyStatus(status []DependencyStatus) (string, string) {
	var depList strings.Builder
	var help strings.Builder

	for _, dep := range status {
		if dep.Error != nil {
			depList.WriteString(style.CrossIcon.Render() + " " + dep.Name + "\n")
			help.WriteString("- " + dep.Name + ": " + dep.Error.Error() + "\n")
		} else {
			depList.WriteString(style.TickIcon.Render() + " " + dep.Name + " " + dep.Version + "\n")
		}
	}

	if help.Len() > 0 {
		help.WriteString("\nPlease install the missing dependencies and try again.")
	}

	return depList.String(), help.String()
}

// PrettyPrintMissingDependency formats missing dependency information
func PrettyPrintMissingDependency(status []DependencyStatus) string {
	var output strings.Builder
	output.WriteString(style.Container.Render("Missing Dependencies") + "\n\n")

	for _, dep := range status {
		if dep.Error != nil {
			output.WriteString(style.CrossIcon.Render() + " " + dep.Name + ": " + dep.Error.Error() + "\n")
		}
	}

	output.WriteString("\nPlease install the missing dependencies and try again.\n")
	return output.String()
}
