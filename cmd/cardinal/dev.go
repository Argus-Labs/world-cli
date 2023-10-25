package cardinal

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/cmd/tea/component"
	"pkg.world.dev/world-cli/utils"
)

/////////////////
// Cobra Setup //
/////////////////

func init() {
	BaseCmd.AddCommand(devCmd)
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		//total width/height doesn't matter here as soon as you put it into the bubbletea framework everything will resize to fit window.
		lowerLeftBox := component.NewServerStatusApp()
		lowerLeftBoxInfo := component.CreateBoxInfo(lowerLeftBox, 50, 30, component.WithBorder)
		triLayout := component.BuildTriLayoutHorizontal(0, 0, nil, lowerLeftBoxInfo, nil)
		_, _, _, err := utils.RunShellCommandReturnBuffers("cd cardinal && go run .", 1024)
		if err != nil {
			return err
		}
		p := tea.NewProgram(triLayout, tea.WithAltScreen())
		_, err = p.Run()
		if err != nil {
			return err
		}
		return nil
	},
}
