package teacmd

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"pkg.world.dev/world-cli/config"
	"pkg.world.dev/world-cli/pkg/logger"
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

func (t *teaCmd) dockerCompose(args ...string) error {
	return t.dockerComposeWithCfg(config.Config{}, args...)
}

func (t *teaCmd) dockerComposeWithCfg(cfg config.Config, args ...string) error {
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
	_, err := t.terminal.ExecCmd(cmd)
	return err
	//return sh.RunWith(cfg.DockerEnv, "docker", args...)
}

// DockerStart starts a given docker container by name.
// Rebuilds the image if `build` is true
// Runs in detach mode if `detach` is true
// Runs with the debug docker compose, if `debug` is true
func (t *teaCmd) DockerStart(cfg config.Config, services []DockerService) error {
	if services == nil {
		return fmt.Errorf("no service names provided")
	}
	if err := t.prepareDirs(path.Join(cfg.RootDir, "cardinal")); err != nil {
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

	if err := t.dockerComposeWithCfg(cfg, dockerArgs("up", services, flags...)...); err != nil {
		return err
	}

	return nil
}

// DockerStartAll starts both cardinal and nakama
func (t *teaCmd) DockerStartAll(cfg config.Config) error {
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

	return t.DockerStart(cfg, services)
}

// DockerRestart restarts a given docker container by name, rebuilds the image if `build` is true
func (t *teaCmd) DockerRestart(cfg config.Config, services []DockerService) error {
	if services == nil {
		return fmt.Errorf("no service names provided")
	}
	if cfg.Build {
		if err := t.DockerStop(services); err != nil {
			return err
		}
		if err := t.DockerStart(cfg, services); err != nil {
			return err
		}
	} else {
		if err := t.dockerComposeWithCfg(cfg, dockerArgs("restart", services, "--build")...); err != nil {
			return err
		}
	}
	return nil
}

// DockerStop stops running specified docker containers (does not remove volumes).
// If you want to reset all the services state, use DockerPurge
func (t *teaCmd) DockerStop(services []DockerService) error {
	if services == nil {
		return fmt.Errorf("no service names provided")
	}
	if err := t.dockerCompose(dockerArgs("stop", services)...); err != nil {
		return err
	}
	return nil
}

// DockerStopAll stops all running docker containers (does not remove volumes).
func (t *teaCmd) DockerStopAll() error {
	return t.DockerStop([]DockerService{
		DockerServiceCardinal,
		DockerServiceCardinalDebug,
		DockerServiceNakama,
		DockerServiceNakamaDB,
		DockerServiceRedis,
	})
}

// DockerPurge stops and deletes all docker containers and data volumes
// This will completely wipe the state, if you only want to stop the containers, use DockerStop
func (t *teaCmd) DockerPurge() error {
	return t.dockerCompose("down", "--volumes")
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

func (t *teaCmd) prepareDirs(dirs ...string) error {
	for _, dir := range dirs {
		if err := t.prepareDir(dir); err != nil {
			return fmt.Errorf("failed to prepare dir %s: %w", dir, err)
		}
	}
	return nil
}

func (t *teaCmd) prepareDir(dir string) error {
	startDir, err := t.terminal.GetWd()
	if err != nil {
		return err
	}
	if err = t.terminal.Chdir(dir); err != nil {
		return err
	}
	if err = t.terminal.Rm("./vendor"); err != nil {
		return err
	}
	if _, err = t.terminal.Exec("go", "mod", "tidy"); err != nil {
		return err
	}
	if _, err = t.terminal.Exec("go", "mod", "vendor"); err != nil {
		return err
	}
	if err = t.terminal.Chdir(startDir); err != nil {
		return err
	}
	return nil
}
