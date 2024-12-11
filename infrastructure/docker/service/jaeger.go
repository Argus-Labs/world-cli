package service

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	"pkg.world.dev/world-cli/config"
)

func getJaegerContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-jaeger", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func Jaeger(cfg *config.Config) Service {
	exposedPorts := []int{16686}

	return Service{
		Name: getJaegerContainerName(cfg),
		Config: container.Config{
			Image: "jaegertracing/all-in-one:1.61.0",
			ExposedPorts: getExposedPorts(exposedPorts),
			Env: []string{
				"SPAN_STORAGE_TYPE=badger",
				"BADGER_EPHEMERAL=false",
				"BADGER_DIRECTORY_VALUE=/badger/data",
				"BADGER_DIRECTORY_KEY=/badger/key",
				"QUERY_ADDITIONAL_HEADERS=Access-Control-Allow-Origin:*",
			},
			User: "root",
		},
		HostConfig: container.HostConfig{
			PortBindings: newPortMap(exposedPorts),
			NetworkMode:  container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeVolume,
					Source: cfg.DockerEnv["CARDINAL_NAMESPACE"],
					Target: "/badger",
				},
			},
		},
	}
}
