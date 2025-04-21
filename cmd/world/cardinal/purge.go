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

// purgeCmd stops and resets the state of your Cardinal game shard.
// Usage: `world cardinal purge`.
var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Clean and reset your Cardinal environment completely",
	Long: `Reset your Cardinal game shard to a clean state by removing all data and containers.

This command stops all running Docker services and removes all associated Docker volumes,
giving you a fresh environment for development or testing. Use this when you want to start
from scratch or resolve persistent state issues.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.GetConfig()
		if err != nil {
			return err
		}

		// Create a new Docker client
		dockerClient, err := docker.NewClient(cfg)
		if err != nil {
			return err
		}
		defer dockerClient.Close()

		err = dockerClient.Purge(cmd.Context(), service.Nakama, service.Cardinal,
			service.NakamaDB, service.Redis, service.Jaeger, service.Prometheus)
		if err != nil {
			return err
		}
		fmt.Println("Cardinal successfully purged")

		return nil
	},
}
