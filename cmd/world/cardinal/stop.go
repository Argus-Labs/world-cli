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

// stopCmd stops your Cardinal game shard stack
// Usage: `world cardinal stop`
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop your Cardinal game shard stack",
	Long: `Stop your Cardinal game shard stack.

This will stop the following Docker services:
- Cardinal (Game shard)
- Nakama (Relay) + DB
- Redis (Cardinal dependency)`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.GetConfig(cmd)
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
