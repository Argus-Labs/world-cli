package teacmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"

	"github.com/magefile/mage/sh"
	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/config"
)

const (
	DockerServiceCardinal      DockerService = "cardinal"
	DockerServiceNakama        DockerService = "nakama"
	DockerServiceNakamaDB      DockerService = "nakama-db"
	DockerServiceRedis         DockerService = "redis"
	DockerServiceEVM           DockerService = "evm"
	DockerServiceDA            DockerService = "celestia-devnet"
	DockerServiceCardinalDebug DockerService = "cardinal-debug"
)

type DockerService string

func dockerCompose(args ...string) error {
	return dockerComposeWithCfg(&config.Config{}, args...)
}

func dockerComposeWithCfg(cfg *config.Config, args ...string) error {
	args = append([]string{"compose"}, args...)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	env := os.Environ()
	for k, v := range cfg.DockerEnv {
		env = append(env, k+"="+v)
	}
	if cfg.Debug {
		env = append(env, "CARDINAL_ADDR=cardinal-debug:4040")
		env = append(env, "CARDINAL_CONTAINER=cardinal-debug")
	}

	cmd.Env = env
	if err := cmd.Run(); err != nil {
		var exitCode *exec.ExitError
		if errors.As(err, &exitCode) {
			expectedExitCodes := []int{130, 137, 143}
			if slices.Contains(expectedExitCodes, exitCode.ExitCode()) {
				return nil
			}
			return err
		}
	}

	return nil
}

func DockerStart(cfg *config.Config, services []DockerService) error {
	if services == nil {
		return eris.New("no service names provided")
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

func DockerStartAll(cfg *config.Config) error {
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

func DockerRestart(cfg *config.Config, services []DockerService) error {
	if services == nil {
		return eris.New("no service names provided")
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

func DockerStop(services []DockerService) error {
	if services == nil {
		return eris.New("no service names provided")
	}
	if err := dockerCompose(dockerArgs("stop", services)...); err != nil {
		return err
	}
	return nil
}

func DockerStopAll() error {
	return DockerStop([]DockerService{
		DockerServiceCardinal,
		DockerServiceCardinalDebug,
		DockerServiceNakama,
		DockerServiceNakamaDB,
		DockerServiceRedis,
	})
}

func DockerPurge() error {
	return dockerCompose("down", "--volumes")
}

func dockerArgs(args string, services []DockerService, flags ...string) []string {
	argsSlice := strings.Split(args, " ")

	res := make([]string, 0, len(argsSlice)+len(services)+len(flags))

	res = append(res, argsSlice...)
	res = append(res, flags...)

	for _, s := range services {
		res = append(res, string(s))
	}

	return res
}

func prepareDirs(dirs ...string) error {
	for _, dir := range dirs {
		if err := prepareDir(dir); err != nil {
			return eris.Wrapf(err, "failed to prepare dir %s", dir)
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
	if err = sh.Run("go", "mod", "tidy"); err != nil {
		return err
	}
	if err = os.Chdir(startDir); err != nil {
		return err
	}
	return nil
}
