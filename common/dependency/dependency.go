package dependency

import (
	"errors"
	"os/exec"

	"github.com/rotisserie/eris"
)

var (
	Git = Dependency{
		Name: "Git",
		Cmd:  exec.Command("git", "--version"),
		Help: `Git is required to clone the starter-game-template.
Learn how to install Git: https://github.com/git-guides/install-git`,
	}
	Go = Dependency{
		Name: "Go",
		Cmd:  exec.Command("go", "version"),
		Help: `Go is required to build and run World Engine game shards.
Learn how to install Go: https://go.dev/doc/install`,
	}
	Docker = Dependency{
		Name: "Docker",
		Cmd:  exec.Command("docker", "--version"),
		Help: `Docker is required to build and run World Engine game shards.
Learn how to install Docker: https://docs.docker.com/engine/install/`,
	}
	DockerCompose = Dependency{
		Name: "Docker Compose",
		Cmd:  exec.Command("docker", "compose", "version"),
		Help: `Docker Compose is required to build and run World Engine game shards.
Learn how to install Docker: https://docs.docker.com/engine/install/`,
	}
	DockerDaemon = Dependency{
		Name: "Docker daemon is running",
		Cmd:  exec.Command("docker", "info"),
		Help: `Docker daemon needs to be running.
If you use Docker Desktop, make sure that you have ran it`,
	}
	AlwaysFail = Dependency{
		Name: "Always fails",
		Cmd:  exec.Command("false"),
		Help: `This dependency check will always fail. It can be used for testing.`,
	}
)

type Dependency struct {
	Name string
	Cmd  *exec.Cmd
	Help string
}

func (d Dependency) Check() error {
	if err := d.Cmd.Run(); err != nil {
		return eris.Wrapf(err, "dependency check %q failed with", d.Name)
	}
	return nil
}

func Check(deps ...Dependency) error {
	errs := make([]error, 0, len(deps))
	for _, dep := range deps {
		errs = append(errs, dep.Check())
	}
	return errors.Join(errs...)
}
