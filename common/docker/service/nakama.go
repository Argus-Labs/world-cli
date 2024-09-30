package service

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"pkg.world.dev/world-cli/common/config"
)

func getNakamaContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-nakama", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func Nakama(cfg *config.Config) Service {
	// Check cardinal namespace
	checkCardinalNamespace(cfg)

	enableAllowList := cfg.DockerEnv["ENABLE_ALLOWLIST"]
	if enableAllowList == "" {
		enableAllowList = "false"
	}

	outgoingQueueSize := cfg.DockerEnv["OUTGOING_QUEUE_SIZE"]
	if outgoingQueueSize == "" {
		outgoingQueueSize = "64"
	}

	// Set default password if not provided
	dbPassword := cfg.DockerEnv["DB_PASSWORD"]
	if dbPassword == "" {
		dbPassword = "very_unsecure_password_please_change" //nolint:gosec // This is a default password
	}

	exposedPorts := []int{7349, 7350, 7351}

	return Service{
		Name: getNakamaContainerName(cfg),
		Config: container.Config{
			Image: "ghcr.io/argus-labs/world-engine-nakama:1.2.7",
			Env: []string{
				fmt.Sprintf("CARDINAL_CONTAINER=%s", getCardinalContainerName(cfg)),
				fmt.Sprintf("CARDINAL_ADDR=%s:4040", getCardinalContainerName(cfg)),
				fmt.Sprintf("CARDINAL_NAMESPACE=%s", cfg.DockerEnv["CARDINAL_NAMESPACE"]),
				fmt.Sprintf("DB_PASSWORD=%s", dbPassword),
				fmt.Sprintf("ENABLE_ALLOWLIST=%s", enableAllowList),
				fmt.Sprintf("OUTGOING_QUEUE_SIZE=%s", outgoingQueueSize),
			},
			Entrypoint: []string{
				"/bin/sh",
				"-ec",
				fmt.Sprintf("/nakama/nakama migrate up --database.address root:%s@%s:26257/nakama && /nakama/nakama --config /nakama/data/local.yml --database.address root:%s@%s:26257/nakama --socket.outgoing_queue_size=64 --logger.level INFO", //nolint:lll
					dbPassword,
					getNakamaDBContainerName(cfg),
					dbPassword,
					getNakamaDBContainerName(cfg)),
			},
			ExposedPorts: getExposedPorts(exposedPorts),
			Healthcheck: &container.HealthConfig{
				Test:     []string{"CMD", "/nakama/nakama", "healthcheck"},
				Interval: 1 * time.Second,
				Timeout:  1 * time.Second,
				Retries:  20, //nolint:gomnd
			},
		},
		HostConfig: container.HostConfig{
			PortBindings:  newPortMap(exposedPorts),
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
		Platform: ocispec.Platform{
			Architecture: "amd64",
			OS:           "linux",
		},
	}
}