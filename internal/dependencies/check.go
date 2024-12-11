package dependencies

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/rotisserie/eris"
)

var (
	// DependencyGo checks for Go installation
	DependencyGo = &Dependency{
		Name: "Go",
		VersionCheck: func() error {
			cmd := exec.Command("go", "version")
			return cmd.Run()
		},
	}

	// DependencyGit checks for Git installation
	DependencyGit = &Dependency{
		Name: "Git",
		VersionCheck: func() error {
			cmd := exec.Command("git", "--version")
			return cmd.Run()
		},
	}

	// DependencyDocker checks for Docker installation
	DependencyDocker = &Dependency{
		Name: "Docker",
		VersionCheck: func() error {
			cmd := exec.Command("docker", "--version")
			return cmd.Run()
		},
	}

	// DependencyDockerDaemon checks if Docker daemon is running
	DependencyDockerDaemon = &Dependency{
		Name: "Docker Daemon",
		VersionCheck: func() error {
			cmd := exec.Command("docker", "info")
			output, err := cmd.CombinedOutput()
			if err != nil {
				return eris.Wrap(err, "Docker daemon is not running")
			}
			if strings.Contains(string(output), "Cannot connect to the Docker daemon") {
				return eris.New("Docker daemon is not running")
			}
			return nil
		},
	}
)

// Dependency represents a system dependency
type Dependency struct {
	Name         string
	VersionCheck func() error
}

// Check verifies if all the given dependencies are installed and working
func Check(deps ...*Dependency) error {
	var missingDeps []string

	for _, dep := range deps {
		if err := dep.VersionCheck(); err != nil {
			missingDeps = append(missingDeps, dep.Name)
		}
	}

	if len(missingDeps) > 0 {
		return eris.New(fmt.Sprintf("Missing dependencies: %s", strings.Join(missingDeps, ", ")))
	}

	return nil
}
