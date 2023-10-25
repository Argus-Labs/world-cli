package cardinal

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common"
	"pkg.world.dev/world-cli/tea/component"
)

/////////////////
// Cobra Setup //
/////////////////

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "TODO",
	Long:  `TODO`,
	RunE: func(cmd *cobra.Command, args []string) error {
		//total width/height doesn't matter here as soon as you put it into the bubbletea framework everything will resize to fit window.
		lowerLeftBox := component.NewServerStatusApp()
		lowerLeftBoxInfo := component.CreateBoxInfo(lowerLeftBox, 50, 30, component.WithBorder)
		triLayout := component.BuildTriLayoutHorizontal(0, 0, nil, lowerLeftBoxInfo, nil)
		_, _, _, err := common.RunShellCommandReturnBuffers("cd cardinal && go run .", 1024)
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
