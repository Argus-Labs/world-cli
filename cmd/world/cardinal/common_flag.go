package cardinal

import (
	"github.com/spf13/cobra"

	globalconfig "pkg.world.dev/world-cli/config"
	logger "pkg.world.dev/world-cli/logging"
)

func registerEditorFlag(cmd *cobra.Command, defaultEnable bool) {
	cmd.Flags().Bool("editor", defaultEnable, "Run Cardinal Editor, useful for prototyping and debugging")
}

func registerConfigAndVerboseFlags(cmds ...*cobra.Command) {
	for _, cmd := range cmds {
		globalconfig.AddConfigFlag(cmd)
		logger.AddVerboseFlag(cmd)
	}
}
