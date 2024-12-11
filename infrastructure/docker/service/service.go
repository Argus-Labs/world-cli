package service

import (
	"fmt"
	"strconv"

	"github.com/docker/go-connections/nat"

	globalconfig "pkg.world.dev/world-cli/config"
	logger "pkg.world.dev/world-cli/logging"
)

var (
	// BuildkitSupport is a flag to check if buildkit is supported
	BuildkitSupport bool
)

func getExposedPorts(ports []int) nat.PortSet {
	exposedPorts := make(nat.PortSet)
	for _, port := range ports {
		if port < 1 || port > 65535 {
			panic(fmt.Sprintf("invalid port %d, must be between 1 and 65535", port))
		}
		tcpPort := nat.Port(strconv.Itoa(port) + "/tcp")
		exposedPorts[tcpPort] = struct{}{}
	}
	return exposedPorts
}

func newPortMap(ports []int) nat.PortMap {
	portMap := make(nat.PortMap)
	for _, port := range ports {
		if port < 1 || port > 65535 {
			panic(fmt.Sprintf("invalid port %d, must be between 1 and 65535", port))
		}
		portStr := strconv.Itoa(port)
		tcpPort := nat.Port(portStr + "/tcp")
		portMap[tcpPort] = []nat.PortBinding{{HostPort: portStr}}
	}
	return portMap
}

func checkCardinalNamespace(cfg *globalconfig.Config) {
	if cfg.DockerEnv["CARDINAL_NAMESPACE"] == "" {
		// Set default namespace if not provided
		logger.Warn("CARDINAL_NAMESPACE not provided, defaulting to defaultnamespace")
		cfg.DockerEnv["CARDINAL_NAMESPACE"] = "defaultnamespace"
	}
}
