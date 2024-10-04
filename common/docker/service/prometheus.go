package service

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"pkg.world.dev/world-cli/common/config"
)

func getPrometheusContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-prometheus", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func Prometheus(cfg *config.Config) Service {
	exposedPorts := []int{9090}

	return Service{
		Name: getPrometheusContainerName(cfg),
		Config: container.Config{
			Image:      "prom/prometheus:v2.54.1",
			Entrypoint: []string{"/bin/sh", "-c"},
			Cmd: []string{
				strings.Replace(`sh -s <<EOF
cat > ./prometheus.yaml <<EON
global:
  scrape_interval:     15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: nakama
    metrics_path: /
    static_configs:
    - targets: ['__NAKAMA_CONTAINER__:9100']
EON
prometheus --config.file=./prometheus.yaml
EOF`, "__NAKAMA_CONTAINER__", getNakamaContainerName(cfg), 1),
			},
		},
		HostConfig: container.HostConfig{
			PortBindings: newPortMap(exposedPorts),
			NetworkMode:  container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}
