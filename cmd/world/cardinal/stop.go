package cardinal

import (
	"fmt"

	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
)

/////////////////
// Cobra Setup //
/////////////////

// stopCmd stops your Cardinal game shard stack.
// Usage: `world cardinal stop`.
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Gracefully shut down your Cardinal game environment",
	Long: `Safely stop all running Cardinal game shard services without losing data.

This command will gracefully shut down the following Docker services:
- Cardinal (Game shard) - Your core game logic engine
- Nakama (Relay) + DB - Handles multiplayer and backend services
- Redis (Cardinal dependency) - In-memory data store

Use this command when you're done working with your Cardinal environment or 
need to free up system resources.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.GetConfig()
		if err != nil {
			return err
		}

		// Create docker client
		dockerClient, err := docker.NewClient(cfg)
		if err != nil {
			return err
		}
		defer dockerClient.Close()

		err = dockerClient.Stop(cmd.Context(), service.Nakama, service.Cardinal,
			service.NakamaDB, service.Redis, service.Jaeger, service.Prometheus)
		if err != nil {
			return err
		}

		fmt.Println("Cardinal successfully stopped")

		return nil
	},
}
