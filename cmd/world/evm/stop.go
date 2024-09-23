package evm

import (
	"fmt"

	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the EVM base shard and DA layer client.",
	Long:  "Stop the EVM base shard and data availability layer client if they are running.",
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

		err = dockerClient.Stop(cfg, service.EVM, service.CelestiaDevNet)
		if err != nil {
			return err
		}

		fmt.Println("EVM successfully stopped")
		return nil
	},
}
