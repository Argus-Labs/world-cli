package cmd

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/utils"
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

func CreateNewProject(projectName string) error {
	command := fmt.Sprintf("git clone git@github.com:Argus-Labs/starter-game-template.git %s", projectName)
	p := tea.NewProgram(newProjectInitialModel(projectName))
	go func() {
		utils.RunShellCmd(command, true, false)
		p.Quit()
	}()
	_, err := p.Run()
	if err != nil {
		return err
	}
	return nil
}

// newProjectCmd represents the newProject command
var newProjectCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a new project for world engine",
	Long:  `Uses git clone to create a new project for world-engine from https://github.com/Argus-Labs/starter-game-template`,
	RunE: func(_ *cobra.Command, arg []string) error {
		if len(arg) != 1 {
			return errors.New("new-project requires a destination to create a new project.")
		}
		err := CreateNewProject(arg[0])
		if err != nil {
			return err
		}
		fmt.Printf("Created new project: \"%s\"\n", arg[0])
		fmt.Printf("To use this cli to control the project please set current working directory to \"%s\"\n", arg[0])
		return nil
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
