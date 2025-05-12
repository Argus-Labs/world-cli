package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"pkg.world.dev/world-cli/common/config"
)

func getNakamaContainerName(cfg *config.Config) string {
	return cfg.DockerEnv["CARDINAL_NAMESPACE"] + "-nakama"
}

//nolint:cyclop,funlen // Removing Nakama in WEv2 no point refactoring
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

	traceEnabled := cfg.DockerEnv["NAKAMA_TRACE_ENABLED"]
	if traceEnabled == "" || !cfg.Telemetry {
		traceEnabled = "true"
	}

	traceSampleRate := cfg.DockerEnv["NAKAMA_TRACE_SAMPLE_RATE"]
	if traceSampleRate == "" {
		traceSampleRate = "0.6"
	}

	metricsEnabled := true
	if cfg.Telemetry {
		cfgMetricsEnabled, err := strconv.ParseBool(cfg.DockerEnv["NAKAMA_METRICS_ENABLED"])
		if err == nil {
			metricsEnabled = cfgMetricsEnabled
		}
	}

	nakamaImage := "ghcr.io/argus-labs/world-engine-nakama:latest"
	if cfg.DockerEnv["NAKAMA_IMAGE"] != "" {
		nakamaImage = cfg.DockerEnv["NAKAMA_IMAGE"]
	}

	platform := ocispec.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}
	if cfg.DockerEnv["NAKAMA_IMAGE_PLATFORM"] != "" {
		nakamaImagePlatform := strings.Split(cfg.DockerEnv["NAKAMA_IMAGE_PLATFORM"], "/")
		if len(nakamaImagePlatform) == 2 { //nolint:gomnd //2 is the expected length
			platform = ocispec.Platform{
				Architecture: nakamaImagePlatform[1],
				OS:           nakamaImagePlatform[0],
			}
		}
	}

	// prometheus metrics export is disabled if port is 0
	// src: https://heroiclabs.com/docs/nakama/getting-started/configuration/#metrics
	prometheusPort := 0
	if metricsEnabled {
		prometheusPort = 9100
	}

	exposedPorts := []int{7349, 7350, 7351}

	databaseAddress := fmt.Sprintf("postgres:%s@%s:5432/nakama", dbPassword, getNakamaDBContainerName(cfg))

	return Service{
		Name: getNakamaContainerName(cfg),
		Config: container.Config{
			Image: nakamaImage,
			Env: []string{
				"CARDINAL_CONTAINER=" + getCardinalContainerName(cfg),
				fmt.Sprintf("CARDINAL_ADDR=%s:4040", getCardinalContainerName(cfg)),
				"CARDINAL_NAMESPACE=" + cfg.DockerEnv["CARDINAL_NAMESPACE"],
				"DB_PASSWORD=" + dbPassword,
				"ENABLE_ALLOWLIST=" + enableAllowList,
				"OUTGOING_QUEUE_SIZE=" + outgoingQueueSize,
				"TRACE_ENABLED=" + traceEnabled,
				fmt.Sprintf("JAEGER_ADDR=%s:4317", getJaegerContainerName(cfg)),
				"JAEGER_SAMPLE_RATE=" + traceSampleRate,
			},
			Entrypoint: []string{
				"/bin/sh",
				"-ec",
				fmt.Sprintf(
					`/nakama/nakama migrate up --database.address %s && /nakama/nakama --database.address %s --config /nakama/data/local.yml --socket.outgoing_queue_size=64 --logger.level INFO --metrics.prometheus_port %d`, //nolint:lll
					databaseAddress,
					databaseAddress,
					prometheusPort,
				),
			},
			ExposedPorts: getExposedPorts(exposedPorts),
			Healthcheck: &container.HealthConfig{
				Test:     []string{"CMD", "/nakama/nakama", "healthcheck"},
				Interval: 1 * time.Second,
				Timeout:  1 * time.Second,
				Retries:  20,
			},
		},
		HostConfig: container.HostConfig{
			PortBindings:  newPortMap(exposedPorts),
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
		Platform: platform,
	}
}
