package docker

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/client"

	"pkg.world.dev/world-cli/common/logger"
)

func contextPrint(title, titleColor, subject, object string) {
	titleStr := foregroundPrint(title, titleColor)
	arrowStr := foregroundPrint("→", "241")
	subjectStr := foregroundPrint(subject, "5")

	fmt.Printf("%s %s %s %s ", titleStr, arrowStr, subjectStr, object)
}

func foregroundPrint(text string, color string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(text)
}

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
