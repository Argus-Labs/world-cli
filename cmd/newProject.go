package cmd

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-engine-cli/utils"
)

type model struct {
	spinner     spinner.Model
	projectName string
}

func (m model) View() string {
	loadingValue := fmt.Sprintf("%s Creating new project \"%s\"...", m.spinner.View(), m.projectName)
	return loadingValue
}

func initialModel(projectName string) model {
	return model{spinner: spinner.New(spinner.WithSpinner(spinner.Pulse)), projectName: projectName}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	p := tea.NewProgram(initialModel(projectName))
	go func() {
		utils.RunShellCmd(command, true)
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
	Use:   "new-project",
	Short: "Creates a new project for world engine",
	Long:  `Uses git clone to create a new project for world-engine from https://github.com/Argus-Labs/starter-game-template`,
	RunE: func(_ *cobra.Command, arg []string) error {
		if len(arg) != 1 {
			return errors.New("new-project requires a destination to create a new project.")
		}
		return CreateNewProject(arg[0])
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
