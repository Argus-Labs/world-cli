package service

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"pkg.world.dev/world-cli/common/config"
)

func getPrometheusContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-prometheus", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func Prometheus(cfg *config.Config) Service {
	exposedPorts := []int{9090}

	return Service{
		Name: getPrometheusContainerName(cfg),
		Config: container.Config{
			Image: "prom/prometheus",
			Cmd:   []string{fmt.Sprintf("--config.file=%s", "/etc/prometheus/config.yaml")},
		},
		HostConfig: container.HostConfig{
			PortBindings: newPortMap(exposedPorts),
			Binds: []string{
				"/home/rmrt1n/Containers/dev40/projects/argus/starter-game-template/prometheus-config.yaml:/etc/prometheus/config.yaml:z",
			},
			NetworkMode: container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}
