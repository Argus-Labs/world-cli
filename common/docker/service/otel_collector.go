package service

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"pkg.world.dev/world-cli/common/config"
)

func getOtelCollectorContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-otel-collector", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func OtelCollector(cfg *config.Config) Service {
	exposedPorts := []int{4317, 4318, 9464}
	return Service{
		Name: getOtelCollectorContainerName(cfg),
		Config: container.Config{
			Image: "otel/opentelemetry-collector-contrib",
		},
		HostConfig: container.HostConfig{
			PortBindings: newPortMap(exposedPorts),
			Binds: []string{
				"/home/rmrt1n/Containers/dev40/projects/argus/starter-game-template/collector-config.yaml:/etc/otelcol-contrib/config.yaml:z",
			},
			NetworkMode: container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}
