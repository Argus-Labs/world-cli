package cardinal

import (
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
)

// restartCmd restarts your Cardinal game shard stack.
// Usage: `world cardinal restart`.
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Refresh your Cardinal game shard environment",
	Long: `Quickly restart your Cardinal game shard environment with the latest changes.

This command will rebuild and restart the following Docker services:
- Cardinal (Core game logic) - Your game's central processing engine
- Nakama (Relay) - Handles multiplayer communication and backend services`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.GetConfig()
		if err != nil {
			return err
		}
		cfg.Build = true
		if err := replaceBoolWithFlag(cmd, flagDebug, &cfg.Debug); err != nil {
			return err
		}

		if err := replaceBoolWithFlag(cmd, flagDetach, &cfg.Detach); err != nil {
			return err
		}

		// Create docker client
		dockerClient, err := docker.NewClient(cfg)
		if err != nil {
			return err
		}
		defer dockerClient.Close()

		err = dockerClient.Restart(cmd.Context(), getServices(cfg)...)
		if err != nil {
			return err
		}

		return nil
	},
}

/////////////////
// Cobra Setup //
/////////////////

func restartCmdInit() {
	restartCmd.Flags().Bool(flagDetach, false, "Run in detached mode")
}
