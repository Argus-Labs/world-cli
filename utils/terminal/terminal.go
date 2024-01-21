package terminal

import (
	"os"
	"os/exec"
	"strings"

	"pkg.world.dev/world-cli/pkg/logger"
)

type terminal struct {
}

type Terminal interface {
	Exec(name string, args ...string) ([]byte, error)
	ExecCmd(cmd *exec.Cmd) ([]byte, error)
	GetWd() (string, error)
	Chdir(dir string) error
	Rm(path string) error
	Wait(cmd *exec.Cmd) error
}

func New() Terminal {
	return &terminal{}
}

func (t *terminal) Exec(name string, args ...string) ([]byte, error) {
	logger.Debugf("Executing: %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

func (t *terminal) ExecCmd(cmd *exec.Cmd) ([]byte, error) {
	logger.Debugf("Executing: %s %s\n", cmd.Path, strings.Join(cmd.Args, " "))
	return cmd.CombinedOutput()
}

func (t *terminal) GetWd() (string, error) {
	logger.Debugf("Executing: %s\n", "pwd")
	return os.Getwd()
}

func (t *terminal) Chdir(dir string) error {
	logger.Debugf("Executing: %s %s\n", "cd", dir)
	return os.Chdir(dir)
}

func (t *terminal) Rm(path string) error {
	logger.Debugf("Executing: %s %s\n", "rm", path)
	return os.RemoveAll(path)
}

func (t *terminal) Wait(cmd *exec.Cmd) error {
	logger.Debugf("Executing: %s %s\n", cmd.Path, strings.Join(cmd.Args, " "))
	return cmd.Wait()
}
