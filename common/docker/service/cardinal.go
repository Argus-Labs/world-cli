package service

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"pkg.world.dev/world-cli/common/config"

	_ "embed"
)

const (
	// mountCache is the Docker mount command to cache the go build cache
	mountCacheScript = `--mount=type=cache,target="/root/.cache/go-build"`
)

//go:embed cardinal.Dockerfile
var dockerfileContent string

func getCardinalContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-cardinal", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func Cardinal(cfg *config.Config) Service {
	// Check cardinal namespace
	checkCardinalNamespace(cfg)

	exposedPorts := []int{4040}

	runtime := "runtime"
	if cfg.Debug {
		runtime = "runtime-debug"
	}

	dockerfile := dockerfileContent
	if !BuildkitSupport {
		dockerfile = strings.ReplaceAll(dockerfile, mountCacheScript, "")
	}

	service := Service{
		Name: getCardinalContainerName(cfg),
		Config: container.Config{
			Image: cfg.DockerEnv["CARDINAL_NAMESPACE"],
			Env: []string{
				fmt.Sprintf("REDIS_ADDRESS=%s:6379", getRedisContainerName(cfg)),
				fmt.Sprintf("BASE_SHARD_SEQUENCER_ADDRESS=%s:9601", getEVMContainerName(cfg)),
			},
			ExposedPorts: getExposedPorts(exposedPorts),
		},
		HostConfig: container.HostConfig{
			PortBindings:  newPortMap(exposedPorts),
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
		Dockerfile:  dockerfile,
		BuildTarget: runtime,
		Dependencies: []Service{
			{
				Name: "golang:1.22-bookworm",
				Config: container.Config{
					Image: "golang:1.22-bookworm",
				},
			},
			{
				Name: "gcr.io/distroless/base-debian12",
				Config: container.Config{
					Image: "gcr.io/distroless/base-debian12",
				},
			},
		},
	}

	// Add debug options
	debug := cfg.Debug
	if debug {
		service.Config.ExposedPorts["40000/tcp"] = struct{}{}
		service.HostConfig.PortBindings["40000/tcp"] = []nat.PortBinding{{HostPort: "40000"}}
		service.HostConfig.CapAdd = []string{"SYS_PTRACE"}
		service.HostConfig.SecurityOpt = []string{"seccomp:unconfined"}
	}

	return service
}
