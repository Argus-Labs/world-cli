package service

import (
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/logger"
)

func getRedisContainerName(cfg *config.Config) string {
	return cfg.DockerEnv["CARDINAL_NAMESPACE"] + "-redis"
}

func Redis(cfg *config.Config) Service {
	// Check cardinal namespace
	checkCardinalNamespace(cfg)

	redisPort := cfg.DockerEnv["REDIS_PORT"]
	if redisPort == "" {
		redisPort = "6379"
	}

	intPort, err := strconv.Atoi(redisPort)
	if err != nil {
		logger.Error("Failed to convert redis port to int, defaulting to 6379", err)
		intPort = 6379
	}
	exposedPorts := []int{intPort}

	return Service{
		Name: getRedisContainerName(cfg),
		Config: container.Config{
			Image:        "redis:latest",
			ExposedPorts: getExposedPorts(exposedPorts),
		},
		HostConfig: container.HostConfig{
			PortBindings:  newPortMap(exposedPorts),
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			Mounts:        []mount.Mount{{Type: mount.TypeVolume, Source: "data", Target: "/redis"}},
			NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}
