package service

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	globalconfig "pkg.world.dev/world-cli/config"
	"pkg.world.dev/world-cli/infrastructure/docker/types"
)

func getJaegerContainerName(cfg *globalconfig.Config) string {
	return fmt.Sprintf("%s-jaeger", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func Jaeger(cfg *globalconfig.Config) types.Service {
	exposedPorts := []int{16686}

	return types.Service{
		Name: getJaegerContainerName(cfg),
		Config: container.Config{
			Image: "jaegertracing/all-in-one:1.61.0",
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
			Mounts: []mount.Mount{{Type: mount.TypeVolume,
				Source: cfg.DockerEnv["CARDINAL_NAMESPACE"], Target: "/badger"}},
		},
	}
}
