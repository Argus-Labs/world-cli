package cardinal

import (
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/logger"
)

func registerEditorFlag(cmd *cobra.Command, defaultEnable bool) {
	cmd.Flags().Bool("editor", defaultEnable, "Run Cardinal Editor, useful for prototyping and debugging")
}

func registerConfigAndVerboseFlags(cmds ...*cobra.Command) {
	for _, cmd := range cmds {
		config.AddConfigFlag(cmd)
		logger.AddVerboseFlag(cmd)
	}
}
