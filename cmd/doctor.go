package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"os/exec"
	"pkg.world.dev/world-cli/cmd/style"
)

type Dependency struct {
	Name string
	Cmd  *exec.Cmd
	Help string
}

var RequiredDependencies = []Dependency{
	{
		Name: "Git",
		Cmd:  exec.Command("git", "--version"),
		Help: `Git is required to clone the starter-game-template.
Learn how to install Git: https://github.com/git-guides/install-git`,
	},
	{
		Name: "Go",
		Cmd: exec.Command("go"+
			"", "version"),
		Help: `Go is required to build and run World Engine game shards.
Learn how to install Go: https://go.dev/doc/install`,
	},
	{
		Name: "Docker",
		Cmd:  exec.Command("docker", "--version"),
		Help: `Docker is required to build and run World Engine game shards.
Learn how to install Docker: https://docs.docker.com/engine/install/`,
	},
}

/////////////////
// Cobra Setup //
/////////////////

func init() {
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Checks that required dependencies for World CLI are installed",
	Long: `Checks that required dependencies for World CLI are installed.

World CLI requires the following dependencies to be installed:
- Git
- Go
- Docker`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(NewWorldDoctorModel())
		_, err := p.Run()
		if err != nil {
			return err
		}
		return nil
	},
}

//////////////////////
// Bubble Tea Model //
//////////////////////

type WorldDoctorModel struct {
	ListOutput string
	HelpOutput string
}

func NewWorldDoctorModel() WorldDoctorModel {
	return WorldDoctorModel{}
}

//////////////////////////
// Bubble Tea Lifecycle //
//////////////////////////

// Init returns an initial command for the application to run
func (m WorldDoctorModel) Init() tea.Cmd {
	var ListOutput string
	var HelpOutput string

	// Iterate over required dependencies and check if they are installed
	for _, dep := range RequiredDependencies {
		if !isInstalled(dep.Cmd) {
			// If the dependency is not installed, set the status to failed
			// and print the help message
			ListOutput += style.CrossIcon.Render() + " " + dep.Name + "\n"
			HelpOutput += dep.Help + "\n\n"
		} else {
			// If the dependency is installed, set the status to success
			ListOutput += style.TickIcon.Render() + " " + dep.Name + "\n"
		}
	}
	return SetOutputCmd(ListOutput, HelpOutput)
}

// Update handles incoming events and updates the model accordingly
func (m WorldDoctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}

	case SetOutputMsg:
		m.ListOutput = msg.ListOutput
		m.HelpOutput = msg.HelpOutput
		return m, tea.Quit
	}
	return m, nil
}

// View renders the model to the screen
func (m WorldDoctorModel) View() string {
	output := style.Container.Render("--- World CLI Doctor ---") + "\n\n"
	output += "Checking dependencies...\n"
	output += m.ListOutput + "\n"
	output += m.HelpOutput
	return output
}

/////////////////////
// Misc Functions //
////////////////////

// isInstalled checks whether a dependency is installed by running the given command
func isInstalled(cmd *exec.Cmd) bool {
	if err := cmd.Run(); err != nil {
		return false
	} else {
		return true
	}
}

/////////////////////////
// Bubble Tea Commands //
/////////////////////////

type SetOutputMsg struct {
	ListOutput string
	HelpOutput string
}

// SetOutputCmd sets the output of the doctor
func SetOutputCmd(listOutput string, helpOutput string) tea.Cmd {
	return func() tea.Msg {
		return SetOutputMsg{
			ListOutput: listOutput,
			HelpOutput: helpOutput,
		}
	}
}
