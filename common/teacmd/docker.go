package teacmd

import (
	"fmt"
	"github.com/magefile/mage/sh"
	"os"
	"strings"
)

type DockerService string

const (
	DockerServiceCardinal DockerService = "cardinal"
	DockerServiceNakama   DockerService = "nakama"
	DockerServicePostgres DockerService = "postgres"
	DockerServiceRedis    DockerService = "redis"
	DockerServiceEVM      DockerService = "evm"
	DockerServiceDA       DockerService = "celestia-devnet"
)

var dockerCompose = sh.RunCmd("docker", "compose")
var dockerComposeDebug = sh.RunCmd("docker", "compose -f ./.run/docker-compose-debug.yml")

// DockerStart starts a given docker container by name.
// Rebuilds the image if `build` is true
// Runs in detach mode if `detach` is true
// Runs with the debug docker compose, if `debug` is true
func DockerStart(build bool, debug bool, detach bool, timeout int, services ...DockerService) error {
	if len(services) == 0 {
		return fmt.Errorf("no service names provided")
	}
	if err := tidyAndVendorDirs("cardinal"); err != nil {
		return err
	}

	var flags []string
	if detach {
		flags = append(flags, "--detach")
	}
	if build {
		flags = append(flags, "--build")
	}
	if timeout > 0 {
		flags = append(flags, fmt.Sprintf("--wait-timeout %d", timeout))
	}

	if debug {
		if err := dockerComposeDebug(dockerArgs("up", services, flags...)...); err != nil {
			return err
		}
	} else {
		if err := dockerCompose(dockerArgs("up", services, flags...)...); err != nil {
			return err
		}
	}

	return nil
}

// DockerStartAll starts both cardinal and nakama
func DockerStartAll(build bool, debug bool, detach bool, timeout int) error {
	return DockerStart(build, debug, detach, timeout,
		DockerServiceCardinal, DockerServiceNakama, DockerServicePostgres, DockerServiceRedis)
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
		if err := DockerStart(build, false, false, -1, services...); err != nil {
			return err
		}
	} else {
		if err := dockerCompose(dockerArgs("restart", services, "--build")...); err != nil {
			return err
		}
	}
	return nil
}

// DockerStop stops running specified docker containers (does not remove volumes).
// If you want to reset all the services state, use DockerPurge
func DockerStop(services []DockerService) error {
	if services == nil {
		return fmt.Errorf("no service names provided")
	}
	if err := dockerCompose(dockerArgs("stop", services)...); err != nil {
		return err
	}
	return nil
}

// DockerStopAll stops all running docker containers (does not remove volumes).
func DockerStopAll() error {
	return DockerStop([]DockerService{DockerServiceCardinal, DockerServiceNakama, DockerServicePostgres, DockerServiceRedis})
}

// DockerPurge stops and deletes all docker containers and data volumes
// This will completely wipe the state, if you only want to stop the containers, use DockerStop
func DockerPurge() error {
	return dockerCompose("down", "--volumes")
}

// dockerArgs converts a string of docker args and slice of DockerService to a single slice of strings.
// We do this so we can pass variadic args cleanly.
func dockerArgs(args string, services []DockerService, flags ...string) []string {
	var res []string

	// split prefix and append them to slice of strings
	argsSlice := strings.Split(args, " ")
	res = append(res, argsSlice...)

	// append flags to slice of strings
	res = append(res, flags...)

	// convert DockerService to string and append them to slice of strings
	for _, s := range services {
		res = append(res, string(s))
	}

	return res
}

func tidyAndVendorDirs(dirs ...string) error {
	for _, dir := range dirs {
		if err := tidyAndVendorDir(dir); err != nil {
			return fmt.Errorf("failed to prepare dir %s: %w", dir, err)
		}
	}
	return nil
}

func tidyAndVendorDir(dir string) error {
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
