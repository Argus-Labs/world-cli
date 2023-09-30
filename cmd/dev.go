/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/utils"
)

func DevCommand(cmd *cobra.Command, args []string) error {
	//total width/height doesn't matter here as soon as you put it into the bubbletea framework everything will resize to fit window.
	lowerLeftBox := utils.NewServerStatusApp()
	lowerLeftBoxInfo := utils.CreateBoxInfo(lowerLeftBox, 50, 30, utils.WithBorder)
	triLayout := utils.BuildTriLayoutHorizontal(0, 0, nil, lowerLeftBoxInfo, nil)
	_, _, _, err := utils.RunShellCommandReturnBuffers("cd cardinal && go build && ./cardinal && cd ..", 1024)
	if err != nil {
		return err
	}
	p := tea.NewProgram(triLayout, tea.WithAltScreen())
	_, err = p.Run()
	if err != nil {
		return err
	}
	return nil
}

// devCmd represents the dev command
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: DevCommand,
}

func init() {
	rootCmd.AddCommand(devCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// devCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// devCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
