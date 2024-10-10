package service

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"

	"pkg.world.dev/world-cli/common/config"
)

var containerCmd = `sh -s <<EOF
cat > ./prometheus.yaml <<EON
global:
  scrape_interval:     __NAKAMA_METRICS_INTERVAL__s
  evaluation_interval: __NAKAMA_METRICS_INTERVAL__s

scrape_configs:
  - job_name: nakama
    metrics_path: /
    static_configs:
      - targets: ['__NAKAMA_CONTAINER__:9100']
EON
prometheus --config.file=./prometheus.yaml
EOF
`

func getPrometheusContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-prometheus", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func Prometheus(cfg *config.Config) Service {
	nakamaMetricsInterval := cfg.DockerEnv["NAKAMA_METRICS_INTERVAL"]
	if nakamaMetricsInterval == "" {
		nakamaMetricsInterval = "30"
	}

	exposedPorts := []int{9090}

	containerCmd = strings.ReplaceAll(containerCmd, "__NAKAMA_CONTAINER__", getNakamaContainerName(cfg))
	containerCmd = strings.ReplaceAll(containerCmd, "__NAKAMA_METRICS_INTERVAL__", nakamaMetricsInterval)

	return Service{
		Name: getPrometheusContainerName(cfg),
		Config: container.Config{
			Image:      "prom/prometheus:v2.54.1",
			Entrypoint: []string{"/bin/sh", "-c"},
			Cmd:        []string{containerCmd},
		},
		HostConfig: container.HostConfig{
			PortBindings: newPortMap(exposedPorts),
			NetworkMode:  container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}
