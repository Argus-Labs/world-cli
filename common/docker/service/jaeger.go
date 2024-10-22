package service

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	"pkg.world.dev/world-cli/common/config"
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
			// hard-coding this since it won't change much and the defaults work. users most likely
			// won't need to change these. note that this storage configuration isn't recommended to
			// be used for production environment, but is good enough for local development.
			// for more info see: https://www.jaegertracing.io/docs/1.62/deployment/#span-storage-backends
			Env: []string{
				"SPAN_STORAGE_TYPE=badger",
				"BADGER_EPHEMERAL=false",
				"BADGER_DIRECTORY_VALUE=/badger/data",
				"BADGER_DIRECTORY_KEY=/badger/key",
				"QUERY_ADDITIONAL_HEADERS=Access-Control-Allow-Origin:*",
			},
		},
		HostConfig: container.HostConfig{
			PortBindings: newPortMap(exposedPorts),
			NetworkMode:  container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
			Mounts: []mount.Mount{{Type: mount.TypeVolume,
				Source: cfg.DockerEnv["CARDINAL_NAMESPACE"], Target: "/badger"}},
		},
	}
}
