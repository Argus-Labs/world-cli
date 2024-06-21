package evm

import (
	"fmt"

	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/teacmd"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the EVM base shard and DA layer client.",
	Long:  "Stop the EVM base shard and data availability layer client if they are running.",
	RunE: func(_ *cobra.Command, _ []string) error {
		err := teacmd.DockerStop(services(teacmd.DockerServiceEVM, teacmd.DockerServiceDA))
		if err != nil {
			return err
		}

		fmt.Println("EVM successfully stopped")
		return nil
	},
}
