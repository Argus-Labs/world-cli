package tea_cmd

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/magefile/mage/sh"
	"os"
	"strings"
)

type DockerService string

const (
	DockerServiceCardinal  DockerService = "cardinal"
	DockerServiceNakama    DockerService = "nakama"
	DockerServicePostgres  DockerService = "postgres"
	DockerServiceRedis     DockerService = "redis"
	DockerServiceTestsuite DockerService = "testsuite"
)

type DockerOp int

const (
	DockerOpBuild DockerOp = iota
	DockerOpStart
	DockerOpStartTest
	DockerOpStartDebug
	DockerOpStartDetach
	DockerOpRestart
	DockerOpPurge
	DockerOpStop
)

type DockerCmdArgs struct {
	Op       DockerOp
	Build    bool
	Services []DockerService
}

type DockerFinishMsg struct {
	Err       error
	Operation DockerOp
}

// DockerCmd returns a tea.Cmd that runs a docker command
func DockerCmd(action DockerCmdArgs) tea.Cmd {
	return func() tea.Msg {
		switch action.Op {

		case DockerOpBuild:
			err := DockerBuild()
			return DockerFinishMsg{Err: err, Operation: DockerOpBuild}

		case DockerOpStart:
			err := DockerStart(action.Build, action.Services)
			return DockerFinishMsg{Err: err, Operation: DockerOpStart}

		case DockerOpStartTest:
			err := DockerStartTest()
			return DockerFinishMsg{Err: err, Operation: DockerOpStartTest}

		case DockerOpStartDebug:
			err := DockerStartDebug()
			return DockerFinishMsg{Err: err, Operation: DockerOpStartDebug}

		case DockerOpStartDetach:
			err := DockerStartDetach()
			return DockerFinishMsg{Err: err, Operation: DockerOpStartDetach}

		case DockerOpRestart:
			err := DockerRestart(action.Build, action.Services)
			return DockerFinishMsg{Err: err, Operation: DockerOpRestart}

		case DockerOpStop:
			err := DockerStop(action.Services)
			return DockerFinishMsg{Err: err, Operation: DockerOpStop}

		case DockerOpPurge:
			err := DockerPurge()
			return DockerFinishMsg{Err: err, Operation: DockerOpPurge}
		}

		return nil
	}
}

// DockerBuild builds all docker images
func DockerBuild() error {
	if err := prepareDirs("cardinal", "nakama"); err != nil {
		return err
	}
	if err := sh.Run("docker", "compose", "build"); err != nil {
		return err
	}
	return nil
}

// DockerStart starts a given docker container by name, rebuilds the image if `build` is true
func DockerStart(build bool, services []DockerService) error {
	if services == nil {
		return fmt.Errorf("no service names provided")
	}
	if err := prepareDirs("cardinal", "nakama"); err != nil {
		return err
	}
	if build {
		if err := sh.Run("docker", dockerArgs("compose up --build -d", services)...); err != nil {
			return err
		}
	} else {
		if err := sh.Run("docker", dockerArgs("compose up -d", services)...); err != nil {
			return err
		}
	}
	return nil
}

// DockerStartTest starts Nakama, Cardinal, and integration tests
func DockerStartTest() error {
	if err := DockerPurge(); err != nil {
		return err
	}
	if err := prepareDirs("testsuite", "cardinal", "nakama"); err != nil {
		return err
	}
	if err := sh.Run("docker", "compose", "up", "--build", "--abort-on-container-exit", "--exit-code-from", "testsuite", "--attach", "testsuite"); err != nil {
		return err
	}
	return nil
}

// DockerStartDebug starts Nakama and Cardinal in debug mode with Cardinal debugger listening on port 40000
// Note: Cardinal server will not run until a debugger is attached port 40000
func DockerStartDebug() error {
	if err := prepareDirs("cardinal", "nakama"); err != nil {
		return err
	}
	if err := sh.Run("docker", "compose", "-f", "docker-compose-debug.yml", "up", "--build", "cardinal", "nakama"); err != nil {
		return err
	}
	return nil
}

// DockerStartDetach starts Nakama and Cardinal with detach and wait-timeout 60s (useful for CI workflow)
func DockerStartDetach() error {
	if err := prepareDirs("cardinal", "nakama"); err != nil {
		return err
	}
	if err := sh.Run("docker", "compose", "up", "--detach", "--wait", "--wait-timeout", "60"); err != nil {
		return err
	}
	return nil
}

// DockerRestart restarts a given docker container by name, rebuilds the image if `build` is true
func DockerRestart(build bool, services []DockerService) error {
	if services == nil {
		return fmt.Errorf("no service names provided")
	}
	if build {
		if err := DockerStop(services); err != nil {
			return err
		}
		if err := DockerStart(build, services); err != nil {
			return err
		}
	} else {
		if err := sh.Run("docker", dockerArgs("compose restart", services)...); err != nil {
			return err
		}
	}
	return nil
}

// DockerStop stops running all docker containers (does not remove volumes).
// If you want to reset all the services state, use DockerPurge
func DockerStop(services []DockerService) error {
	if services == nil {
		return fmt.Errorf("no service names provided")
	}
	if err := sh.Run("docker", dockerArgs("compose stop", services)...); err != nil {
		return err
	}
	return nil
}

// DockerPurge stops and deletes all docker containers and data volumes
// This will completely wipe the state, if you only want to stop the containers, use DockerStop
func DockerPurge() error {
	return sh.Run("docker", "compose", "down", "--volumes")
}

// dockerArgs converts a string of docker args and slice of DockerService to a single slice of strings.
// We do this so we can pass variadic args cleanly.
func dockerArgs(args string, services []DockerService) []string {
	var res []string

	// split prefix and append them to slice of strings
	argsSlice := strings.Split(args, " ")
	res = append(res, argsSlice...)

	// convert DockerService to string and append them to slice of strings
	for _, s := range services {
		res = append(res, string(s))
	}

	return res
}

func prepareDirs(dirs ...string) error {
	for _, dir := range dirs {
		if err := prepareDir(dir); err != nil {
			return fmt.Errorf("failed to prepare dir %s: %w", dir, err)
		}
	}
	return nil
}

func prepareDir(dir string) error {
	if err := os.Chdir(dir); err != nil {
		return err
	}
	if err := sh.Rm("./vendor"); err != nil {
		return err
	}
	if err := sh.Run("go", "mod", "tidy"); err != nil {
		return err
	}
	if err := sh.Run("go", "mod", "vendor"); err != nil {
		return err
	}
	if err := os.Chdir(".."); err != nil {
		return err
	}
	return nil
}
