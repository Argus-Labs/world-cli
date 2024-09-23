package service

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/logger"
)

func getNakamaDBContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-nakama-db", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func NakamaDB(cfg *config.Config) Service {
	exposedPorts := []int{26257, 8080}

	// Set default password if not provided
	dbPassword := cfg.DockerEnv["DB_PASSWORD"]
	if dbPassword == "" {
		logger.Warn("Using default DB_PASSWORD, please change it.")
		dbPassword = "very_unsecure_password_please_change" //nolint:gosec // This is a default password
	}

	return Service{
		Name: getNakamaDBContainerName(cfg),
		Config: container.Config{
			Image: "cockroachdb/cockroach:latest-v23.1",
			Cmd:   []string{"start-single-node", "--insecure", "--store=attrs=ssd,path=/var/lib/cockroach/,size=20%"},
			Env: []string{
				"COCKROACH_DATABASE=nakama",
				"COCKROACH_USER=root",
				fmt.Sprintf("COCKROACH_PASSWORD=%s", dbPassword),
			},
			ExposedPorts: getExposedPorts(exposedPorts),
			Healthcheck: &container.HealthConfig{
				Test:     []string{"CMD", "curl", "-f", "http://localhost:8080/health?ready=1"},
				Interval: 3 * time.Second, //nolint:gomnd
				Timeout:  3 * time.Second, //nolint:gomnd
				Retries:  5,               //nolint:gomnd
			},
		},
		HostConfig: container.HostConfig{
			PortBindings:  newPortMap(exposedPorts),
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			Mounts: []mount.Mount{{Type: mount.TypeVolume, Source: cfg.DockerEnv["CARDINAL_NAMESPACE"],
				Target: "/var/lib/cockroach"}},
			NetworkMode: container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}
