package service

import (
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/logger"
)

func getNakamaDBContainerName(cfg *config.Config) string {
	return cfg.DockerEnv["CARDINAL_NAMESPACE"] + "-nakama-db"
}

func NakamaDB(cfg *config.Config) Service {
	exposedPorts := []int{5432, 8080}

	// Set default password if not provided
	dbPassword := cfg.DockerEnv["DB_PASSWORD"]
	if dbPassword == "" {
		logger.Warn("Using default DB_PASSWORD, please change it.")
		dbPassword = "very_unsecure_password_please_change" //nolint:gosec // This is a default password
	}

	return Service{
		Name: getNakamaDBContainerName(cfg),
		Config: container.Config{
			Image: "postgres:12.2-alpine",
			Env: []string{
				"POSTGRES_DB=nakama",
				"POSTGRES_PASSWORD=" + dbPassword,
			},
			ExposedPorts: getExposedPorts(exposedPorts),
			Healthcheck: &container.HealthConfig{
				Test:     []string{"CMD", "pg_isready", "-U", "postgres", "-d", "nakama"},
				Interval: 3 * time.Second,
				Timeout:  3 * time.Second,
				Retries:  5,
			},
		},
		HostConfig: container.HostConfig{
			PortBindings:  newPortMap(exposedPorts),
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			Mounts: []mount.Mount{{Type: mount.TypeVolume, Source: cfg.DockerEnv["CARDINAL_NAMESPACE"],
				Target: "/var/lib/postgresql/data"}},
			NetworkMode: container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}
