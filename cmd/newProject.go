package cmd

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-engine-cli/utils"
)

type newProjectModel struct {
	spinner     spinner.Model
	projectName string
}

func (m newProjectModel) View() string {
	loadingValue := fmt.Sprintf("%s Creating new project \"%s\"...", m.spinner.View(), m.projectName)
	return loadingValue
}

func newProjectInitialModel(projectName string) newProjectModel {
	return newProjectModel{spinner: spinner.New(spinner.WithSpinner(spinner.Pulse)), projectName: projectName}
}

func (m newProjectModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m newProjectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.QuitMsg:
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

// newProjectCmd represents the newProject command
var newProjectCmd = &cobra.Command{
	Use:   "new-project",
	Short: "Creates a new project for world engine",
	Long:  `Uses git clone to create a new project for world-engine from https://github.com/Argus-Labs/starter-game-template`,
	RunE: func(cmd *cobra.Command, arg []string) error {
		if len(arg) != 1 {
			msg := "new-project requires a destination to create a new project."
			return errors.New(msg)
		}
		command := fmt.Sprintf("git clone git@github.com:Argus-Labs/starter-game-template.git %s", arg[0])
		p := tea.NewProgram(newProjectInitialModel(arg[0]))
		go func() {
			utils.RunShellCmd(command, true, false)
			p.Quit()
		}()
		_, err := p.Run()
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Project created: %s, please change current working directory to that project to use this cli to monitor and start it.\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(newProjectCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// newProjectCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// newProjectCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
