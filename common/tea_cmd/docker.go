package tea_cmd

import (
	"fmt"
	"github.com/magefile/mage/sh"
	"os"
	"strings"
)

type DockerService string

const (
	DockerServiceCardinal  DockerService = "cardinal"
	DockerServiceNakama    DockerService = "nakama"
	DockerServiceTestsuite DockerService = "testsuite"
)

// DockerPurge stops and deletes all docker containers and data volumes
// This will completely wipe the state, if you only want to stop the containers, use DockerStop
func DockerPurge() error {
	return sh.RunV("docker", "compose", "down", "--volumes")
}

// DockerStop stops running all docker containers (does not remove volumes).
// If you want to reset all the services state, use DockerPurge
func DockerStop(services []DockerService) error {
	if services == nil {
		return fmt.Errorf("no service names provided")
	}
	if err := sh.Run("docker", "compose", "stop", servicesToStr(services)); err != nil {
		return err
	}
	return nil
}

// DockerStart starts a given docker container by name, rebuilds the image if `build` is true
func DockerStart(build bool, services []DockerService) error {
	if services == nil {
		return fmt.Errorf("no service names provided")
	}
	if build {
		if err := sh.Run("docker", "compose", "up", "--build", "-d", servicesToStr(services)); err != nil {
			return err
		}
	} else {
		if err := sh.Run("docker", "compose", "up", "-d", servicesToStr(services)); err != nil {
			return err
		}
	}
	return nil
}

// DockerBuild builds all docker images
func DockerBuild() error {
	if err := prepareDirs("cardinal", "nakama"); err != nil {
		return err
	}
	if err := sh.RunV("docker", "compose", "build"); err != nil {
		return err
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
	if err := sh.RunV("docker", "compose", "up", "--build", "--abort-on-container-exit", "--exit-code-from", "testsuite", "--attach", "testsuite"); err != nil {
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
	if err := sh.RunV("docker", "compose", "-f", "docker-compose-debug.yml", "up", "--build", "cardinal", "nakama"); err != nil {
		return err
	}
	return nil
}

// DockerStartDetach starts Nakama and Cardinal with detach and wait-timeout 60s (useful for CI workflow)
func DockerStartDetach() error {
	if err := prepareDirs("cardinal", "nakama"); err != nil {
		return err
	}
	if err := sh.RunV("docker", "compose", "up", "--detach", "--wait", "--wait-timeout", "60"); err != nil {
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
		if err := sh.Run("docker", "compose", "restart", servicesToStr(services)); err != nil {
			return err
		}
	}
	return nil
}

// servicesToStr converts a slice of DockerService to a joined string separated by " "
func servicesToStr(services []DockerService) string {
	var res []string
	for _, s := range services {
		res = append(res, string(s))
	}
	return strings.Join(res, " ")
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
