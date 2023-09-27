/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os/exec"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-engine-cli/utils"
)

func newDoctorModel() utils.StatusCollection {
	res := utils.StatusCollection{
		Spinner:      spinner.New(spinner.WithSpinner(spinner.Pulse)),
		ShutdownChan: make(chan bool),
	}
	statuses := []*utils.StatusObject{
		utils.CreateNewStatus("docker", func(status *utils.StatusObject) {
			cmd := exec.Command("docker", "--version")
			// Run the command
			if err := cmd.Run(); err != nil {
				status.SetStatus(utils.FAILED)
			} else {
				status.SetStatus(utils.SUCCESS)
			}
		}),
		utils.CreateNewStatus("git", func(status *utils.StatusObject) {
			cmd := exec.Command("git", "--version")
			// Run the command
			if err := cmd.Run(); err != nil {
				status.SetStatus(utils.FAILED)
			} else {
				status.SetStatus(utils.SUCCESS)
			}
		}),
		utils.CreateNewStatus("golang", func(status *utils.StatusObject) {
			cmd := exec.Command("go", "version")
			// Run the command
			if err := cmd.Run(); err != nil {
				status.SetStatus(utils.FAILED)
			} else {
				status.SetStatus(utils.SUCCESS)
			}
		}),
	}
	res.Statuses = statuses
	return res
}

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Checks if required dependencies for world-cli are installed",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		model := newDoctorModel()
		p := tea.NewProgram(model)
		_, err := p.Run()
		if err != nil {
			return err
		}
		if model.IsAllChecked() {
			fmt.Println("All dependencies found.")
		} else {
			fmt.Println("Missing dependencies.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// doctorCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// doctorCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
