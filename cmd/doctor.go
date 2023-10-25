package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/cmd/action"
	"pkg.world.dev/world-cli/cmd/style"
)

var DoctorDeps = []action.Dependency{
	action.GitDependency,
	action.GoDependency,
	action.DockerDependency,
	action.DockerComposeDependency,
	action.DockerDaemonDependency,
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
	DepStatus    []action.DependencyStatus
	DepStatusErr error
}

func NewWorldDoctorModel() WorldDoctorModel {
	return WorldDoctorModel{}
}

//////////////////////////
// Bubble Tea Lifecycle //
//////////////////////////

// Init returns an initial command for the application to run
func (m WorldDoctorModel) Init() tea.Cmd {
	return action.CheckDependenciesCmd(DoctorDeps)
}

// Update handles incoming events and updates the model accordingly
func (m WorldDoctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case action.CheckDependenciesMsg:
		m.DepStatus = msg.DepStatus
		m.DepStatusErr = msg.Err
		return m, tea.Quit
	}
	return m, nil
}

// View renders the model to the screen
func (m WorldDoctorModel) View() string {
	depList, help := action.PrintDependencyStatus(m.DepStatus)
	out := style.Container.Render("--- World CLI Doctor ---") + "\n\n"
	out += "Checking dependencies...\n"
	out += depList + "\n" + help + "\n"
	return out
}
