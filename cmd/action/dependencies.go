package action

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"os/exec"
	"pkg.world.dev/world-cli/tea/style"
)

var (
	GitDependency = Dependency{
		Name: "Git",
		Cmd:  exec.Command("git", "--version"),
		Help: `Git is required to clone the starter-game-template.
Learn how to install Git: https://github.com/git-guides/install-git`,
	}
	GoDependency = Dependency{
		Name: "Go",
		Cmd: exec.Command("go"+
			"", "version"),
		Help: `Go is required to build and run World Engine game shards.
Learn how to install Go: https://go.dev/doc/install`,
	}
	DockerDependency = Dependency{
		Name: "Docker",
		Cmd:  exec.Command("docker", "--version"),
		Help: `Docker is required to build and run World Engine game shards.
Learn how to install Docker: https://docs.docker.com/engine/install/`,
	}
	DockerComposeDependency = Dependency{
		Name: "Docker Compose",
		Cmd:  exec.Command("docker", "compose", "version"),
		Help: `Docker Compose is required to build and run World Engine game shards.
Learn how to install Docker: https://docs.docker.com/engine/install/`,
	}
	DockerDaemonDependency = Dependency{
		Name: "Docker daemon is running",
		Cmd:  exec.Command("docker", "--version"),
		Help: `Docker daemon needs to be running.
If you use Docker Desktop, make sure that you have ran it`,
	}
)

type Dependency struct {
	Name string
	Cmd  *exec.Cmd
	Help string
}

type DependencyStatus struct {
	Dependency
	IsInstalled bool
}

type CheckDependenciesMsg struct {
	Err       error
	DepStatus []DependencyStatus
}

// CheckDependenciesCmd Iterate through required dependencies and check if they are installed.
// Dispatch CheckDependenciesMsg if any dependency is missing.
func CheckDependenciesCmd(deps []Dependency) tea.Cmd {
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
