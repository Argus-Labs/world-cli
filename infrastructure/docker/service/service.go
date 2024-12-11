package service

import (
	"fmt"
	"strconv"

	"github.com/docker/go-connections/nat"
	"github.com/moby/moby/api/types"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"pkg.world.dev/world-cli/config"
	"pkg.world.dev/world-cli/logging"
)

var (
	// BuildkitSupport is a flag to check if buildkit is supported
	BuildkitSupport bool
)

type Builder func(cfg *config.Config) Service

// Service is a configuration for a docker container
// It contains the name of the container and a function to get the container and host config
type Service struct {
	Name string
	types.ContainerConfig
	container.HostConfig
	network.NetworkingConfig
	ocispec.Platform

	// Dependencies are other services that need to be pull before this service
	Dependencies []Service
	// Dockerfile is the content of the Dockerfile
	Dockerfile string
	// BuildTarget is the target build of the Dockerfile e.g. builder or runtime
	BuildTarget string
}

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

func checkCardinalNamespace(cfg *config.Config) {
	if cfg.DockerEnv["CARDINAL_NAMESPACE"] == "" {
		// Set default namespace if not provided
		logging.Warn("CARDINAL_NAMESPACE not provided, defaulting to defaultnamespace")
		cfg.DockerEnv["CARDINAL_NAMESPACE"] = "defaultnamespace"
	}
}
