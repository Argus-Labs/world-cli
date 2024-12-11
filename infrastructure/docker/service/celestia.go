package service

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"

	"pkg.world.dev/world-cli/config"
)

func getCelestiaDevNetContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-celestia-devnet", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func CelestiaDevNet(cfg *config.Config) Service {
	// Check cardinal namespace
	checkCardinalNamespace(cfg)

	exposedPorts := []int{26657, 26658, 26659, 9090}

	return Service{
		Name: getCelestiaDevNetContainerName(cfg),
		Config: container.Config{
			Image:        "ghcr.io/rollkit/local-celestia-devnet:latest",
			ExposedPorts: getExposedPorts(exposedPorts),
			Healthcheck: &container.HealthConfig{
				Test:     []string{"CMD", "curl", "-f", "http://127.0.0.1:26659/head"},
				Interval: 1 * time.Second,
				Timeout:  1 * time.Second,
				Retries:  20, //nolint:gomnd
			},
		},
		HostConfig: container.HostConfig{
			PortBindings:  newPortMap(exposedPorts),
			RestartPolicy: container.RestartPolicy{Name: "on-failure"},
			NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}
