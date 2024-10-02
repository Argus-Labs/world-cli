package service

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"pkg.world.dev/world-cli/common/config"
)

func getJaegerContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-jaeger", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func Jaeger(cfg *config.Config) Service {
	exposedPorts := []int{14250, 16686}

	return Service{
		Name: getJaegerContainerName(cfg),
		Config: container.Config{
			Image: "jaegertracing/all-in-one:latest",
			Env: []string{
				"COLLECTOR_OTLP_ENABLED=false",
			},
		},
		HostConfig: container.HostConfig{
			PortBindings: newPortMap(exposedPorts),
			NetworkMode:  container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}
