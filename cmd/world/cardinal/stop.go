package cardinal

import (
	"fmt"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/pkg/logger"
)

/////////////////
// Cobra Setup //
/////////////////

// stopCmd stops your Cardinal game shard stack
// Usage: `world cardinal stop`
func (c *cardinal) StopCmd() *cobra.Command {
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop your Cardinal game shard stack",
		Long: `Stop your Cardinal game shard stack.

This will stop the following Docker services:
- Cardinal (Game shard)
- Nakama (Relay) + DB
- Redis (Cardinal dependency)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.SetDebugMode(cmd)
			err := c.teaCmd.DockerStopAll()
			if err != nil {
				return err
			}

			fmt.Println("Cardinal successfully stopped")

			return nil
		},
	}

	return stopCmd
}
