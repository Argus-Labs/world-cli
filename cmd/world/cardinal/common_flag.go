package cardinal

import "github.com/spf13/cobra"

func registerEditorFlag(cmd *cobra.Command, defaultEnable bool) {
	cmd.Flags().Bool("editor", defaultEnable, "Run Cardinal Editor, useful for prototyping and debugging")
}
