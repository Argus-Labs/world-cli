package docker

import (
	"context"
	"os"
	"strings"

	"github.com/docker/docker/client"

	"pkg.world.dev/world-cli/common/logger"
)

func checkBuildkitSupport(cli *client.Client) bool {
	ctx := context.Background()
	defer func() {
		err := cli.Close()
		if err != nil {
			logger.Error("Failed to close docker client", err)
		}
	}()

	// Get Docker server version
	version, err := cli.ServerVersion(ctx)
	if err != nil {
		logger.Warnf("Failed to get Docker server version: %v", err)
		return false
	}

	// Check if the Docker version supports BuildKit
	supportsBuildKit := strings.HasPrefix(version.Version, "18.09") || version.Version > "18.09"

	if !supportsBuildKit {
		return false
	}

	// Check if DOCKER_BUILDKIT environment variable is set to 1
	buildKitEnv := os.Getenv("DOCKER_BUILDKIT")
	return buildKitEnv == "1"
}
