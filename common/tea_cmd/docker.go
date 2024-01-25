package tea_cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/magefile/mage/sh"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/logger"
)

type DockerService string

const (
	DockerServiceCardinal      DockerService = "cardinal"
	DockerServiceNakama        DockerService = "nakama"
	DockerServiceNakamaDB      DockerService = "nakama-db"
	DockerServiceRedis         DockerService = "redis"
	DockerServiceEVM           DockerService = "evm"
	DockerServiceDA            DockerService = "celestia-devnet"
	DockerServiceCardinalDebug DockerService = "cardinal-debug"
)

func dockerCompose(args ...string) error {
	return dockerComposeWithCfg(config.Config{}, args...)
}

func dockerComposeWithCfg(cfg config.Config, args ...string) error {
	yml := path.Join(cfg.RootDir, "docker-compose.yml")
	args = append([]string{"compose", "-f", yml}, args...)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout

	// hide stderr if not in debug mode
	if logger.DebugMode {
		cmd.Stderr = os.Stderr
	}

	env := os.Environ()
	for k, v := range cfg.DockerEnv {
		env = append(env, k+"="+v)
	}
	if cfg.Debug {
		env = append(env, "CARDINAL_ADDR=cardinal-debug:3333")
	}

	cmd.Env = env
	return cmd.Run()
	//return sh.RunWith(cfg.DockerEnv, "docker", args...)
}

// DockerStart starts a given docker container by name.
// Rebuilds the image if `build` is true
// Runs in detach mode if `detach` is true
// Runs with the debug docker compose, if `debug` is true
func DockerStart(cfg config.Config, services []DockerService) error {
	if services == nil {
		return fmt.Errorf("no service names provided")
	}
	if err := prepareDirs(path.Join(cfg.RootDir, "cardinal")); err != nil {
		return err
	}

	var flags []string
	if cfg.Detach {
		flags = append(flags, "--detach")
	}
	if cfg.Build {
		flags = append(flags, "--build")
	}
	if cfg.Timeout > 0 {
		flags = append(flags, fmt.Sprintf("--wait-timeout %d", cfg.Timeout))
	}

	if err := dockerComposeWithCfg(cfg, dockerArgs("up", services, flags...)...); err != nil {
		return err
	}

	return nil
}

// DockerStartAll starts both cardinal and nakama
func DockerStartAll(cfg config.Config) error {
	services := []DockerService{
		DockerServiceNakama,
		DockerServiceNakamaDB,
		DockerServiceRedis,
	}

	if cfg.Debug {
		services = append(services, DockerServiceCardinalDebug)
	} else {
		services = append(services, DockerServiceCardinal)
	}

	return DockerStart(cfg, services)
}

// DockerRestart restarts a given docker container by name, rebuilds the image if `build` is true
func DockerRestart(cfg config.Config, services []DockerService) error {
	if services == nil {
		return fmt.Errorf("no service names provided")
	}
	if cfg.Build {
		if err := DockerStop(services); err != nil {
			return err
		}
		if err := DockerStart(cfg, services); err != nil {
			return err
		}
	} else {
		if err := dockerComposeWithCfg(cfg, dockerArgs("restart", services, "--build")...); err != nil {
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
	return DockerStop([]DockerService{
		DockerServiceCardinal,
		DockerServiceCardinalDebug,
		DockerServiceNakama,
		DockerServiceNakamaDB,
		DockerServiceRedis,
	})
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

func prepareDirs(dirs ...string) error {
	for _, dir := range dirs {
		if err := prepareDir(dir); err != nil {
			return fmt.Errorf("failed to prepare dir %s: %w", dir, err)
		}
	}
	return nil
}

func prepareDir(dir string) error {
	startDir, err := os.Getwd()
	if err != nil {
		return err
	}
	if err = os.Chdir(dir); err != nil {
		return err
	}
	if err = sh.Rm("./vendor"); err != nil {
		return err
	}
	if err = sh.Run("go", "mod", "tidy"); err != nil {
		return err
	}
	if err = sh.Run("go", "mod", "vendor"); err != nil {
		return err
	}
	if err = os.Chdir(startDir); err != nil {
		return err
	}
	return nil
}
