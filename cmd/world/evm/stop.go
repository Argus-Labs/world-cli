package evm

import (
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/printer"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Shut down your EVM blockchain environment",
	Long: `Gracefully stop your EVM blockchain environment and associated services.

This command safely shuts down the EVM base shard and data availability layer client,
preserving your blockchain state while freeing up system resources. Use this when you're
done working with your EVM environment.`,
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

		err = dockerClient.Stop(cmd.Context(), service.EVM, service.CelestiaDevNet)
		if err != nil {
			return err
		}

		printer.Infoln("EVM successfully stopped")
		return nil
	},
}
