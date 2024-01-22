package dependency

import (
	"errors"
	"fmt"
	"os/exec"

	"pkg.world.dev/world-cli/pkg/logger"
)

var (
	Git = dependency{
		Name: "Git",
		Cmd:  exec.Command("git", "--version"),
		Help: `Git is required to clone the starter-game-template.
Learn how to install Git: https://github.com/git-guides/install-git`,
	}
	Go = dependency{
		Name: "Go",
		Cmd:  exec.Command("go", "version"),
		Help: `Go is required to build and run World Engine game shards.
Learn how to install Go: https://go.dev/doc/install`,
	}
	Docker = dependency{
		Name: "Docker",
		Cmd:  exec.Command("docker", "--version"),
		Help: `Docker is required to build and run World Engine game shards.
Learn how to install Docker: https://docs.docker.com/engine/install/`,
	}
	DockerCompose = dependency{
		Name: "Docker Compose",
		Cmd:  exec.Command("docker", "compose", "version"),
		Help: `Docker Compose is required to build and run World Engine game shards.
Learn how to install Docker: https://docs.docker.com/engine/install/`,
	}
	DockerDaemon = dependency{
		Name: "Docker daemon is running",
		Cmd:  exec.Command("docker", "info"),
		Help: `Docker daemon needs to be running.
If you use Docker Desktop, make sure that you have ran it`,
	}
	AlwaysFail = dependency{
		Name: "Always fails",
		Cmd:  exec.Command("false"),
		Help: `This dependency check will always fail. It can be used for testing.`,
	}
)

type dependency struct {
	Name string
	Cmd  *exec.Cmd
	Help string
}

type Dependency interface {
	Check() error
	GetCmd() *exec.Cmd
	GetName() string
	GetHelp() string
}

func (d *dependency) Check() error {
	if err := d.Cmd.Run(); err != nil {
		return fmt.Errorf("dependency check %q failed with: %w", d.Name, err)
	}
	return nil
}

func Check(deps ...dependency) error {
	var errs []error
	for _, dep := range deps {
		err := dep.Check()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func (d *dependency) GetCmd() *exec.Cmd {
	return d.Cmd
}

func (d *dependency) GetName() string {
	return d.Name
}

func (d *dependency) GetHelp() string {
	return d.Help
}
